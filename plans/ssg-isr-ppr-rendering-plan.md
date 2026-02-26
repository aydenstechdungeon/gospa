# GoSPA Rendering Strategy Plan: SSG, ISR & PPR

## Overview

This plan extends the existing rendering strategy system in GoSPA to add **ISR** (Incremental Static Regeneration) and **PPR** (Partial Prerendering), alongside a **per-page strategy selector**. The current system only has `StrategySSR` and `StrategySSG`. This plan also covers proper documentation in `docs/`, `website/`, and `README.md`.

---

## Current State

| Component | Location | Status |
|---|---|---|
| `RenderStrategy` type + `StrategySSR`/`StrategySSG` consts | `routing/registry.go` | ✅ Exists |
| `RouteOptions.Strategy` field | `routing/registry.go` | ✅ Exists |
| `RegisterPageWithOptions` | `routing/registry.go` | ✅ Exists |
| SSG cache (FIFO, `ssgCache` map) | `gospa.go` | ✅ Exists |
| `CacheTemplates` config flag | `gospa.go` | ✅ Exists |
| ISR (TTL-based revalidation) | — | ❌ Missing |
| PPR (partial static shell + dynamic slots) | — | ❌ Missing |
| Per-page strategy selector in code gen | — | ❌ Missing |
| Docs for rendering strategies | `docs/`, `website/` | ❌ Missing |

---

## Architecture

```
RouteOptions.Strategy selects one of four strategies:
  StrategySSR  — render fresh on every request (default)
  StrategySSG  — render once, cache forever (FIFO eviction)
  StrategyISR  — render once, then revalidate after TTL expires
  StrategyPPR  — render static shell at startup, stream dynamic slots per-request
```

All four strategies share the same `renderRoute` dispatch path in `gospa.go`. The strategy is looked up via `routing.GetRouteOptions(route.Path)` and the resulting `RouteOptions` drives which cache branch executes.

---

## Phase 1 — Extend `routing/registry.go`

### 1.1 Add new strategy constants

Add `StrategyISR` and `StrategyPPR` alongside the existing two.

```go
const (
    StrategySSR RenderStrategy = "ssr"  // existing
    StrategySSG RenderStrategy = "ssg"  // existing
    StrategyISR RenderStrategy = "isr"  // new: revalidate after TTL
    StrategyPPR RenderStrategy = "ppr"  // new: static shell + dynamic slots
)
```

### 1.2 Extend `RouteOptions`

Add optional TTL for ISR and slot names for PPR:

```go
type RouteOptions struct {
    Strategy     RenderStrategy
    // ISR: how long the cached version is valid before background revalidation.
    // Zero means "revalidate on every request" (same as SSR).
    RevalidateAfter time.Duration

    // PPR: list of named slot component keys that are excluded from the static
    // shell and streamed dynamically per-request.
    DynamicSlots []string
}
```

No other changes are needed to the `Registry` struct itself — it already stores per-path `RouteOptions`.

---

## Phase 2 — ISR Cache Layer in `gospa.go`

ISR extends the existing `ssgCache`. Instead of caching forever, each entry also stores a generation timestamp. The first stale request triggers a **background revalidation goroutine** and immediately returns the stale cached page (stale-while-revalidate pattern).

### 2.1 New cache entry type

Replace the current `ssgCache map[string][]byte` with:

```go
type ssgEntry struct {
    html      []byte
    createdAt time.Time
}

// app fields:
ssgCache     map[string]ssgEntry
ssgCacheKeys []string
ssgCacheMu   sync.RWMutex
isrRevalidating sync.Map // key: string → bool (guard against duplicate goroutines)
```

### 2.2 ISR render path in `renderRoute`

```
1. opts = GetRouteOptions(route.Path)
2. if opts.Strategy == StrategyISR:
   a. RLock → check ssgCache[cacheKey]
   b. If hit AND age < opts.RevalidateAfter → serve cached HTML (fresh)
   c. If hit AND age >= opts.RevalidateAfter → serve cached HTML (stale), launch background revalidation goroutine (deduplicated via isrRevalidating)
   d. If miss → render synchronously, store entry with createdAt = now
3. Background goroutine:
   a. Re-render page to bytes
   b. Lock → update ssgCache[cacheKey] with new html + createdAt = now
   c. Delete key from isrRevalidating
```

The `isrRevalidating sync.Map` prevents multiple concurrent goroutines from rerendering the same page at the same time.

### 2.3 ISR eviction

ISR entries share the same FIFO eviction pool as SSG entries, bounded by `SSGCacheMaxEntries`. No separate limit is needed.

---

## Phase 3 — PPR: Static Shell + Dynamic Slots

PPR (Partial Prerendering) renders the **outer static shell** (everything not in a dynamic slot) once at startup and caches it. On each request, only the **named dynamic slots** are re-rendered and merged into the shell via a streaming replacement.

### 3.1 `DynamicSlot` templ helper (`templ/ppr.go`)

A new helper that marks a subtree as a dynamic slot. Server-side it emits a placeholder comment; client-side the runtime replaces it on arrival.

```go
// DynamicSlot wraps content that must be rendered dynamically per-request.
// In PPR mode, the shell is cached without this content.
// name must match a key in RouteOptions.DynamicSlots.
func DynamicSlot(name string, content templ.Component) templ.Component {
    return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
        if IsPPRShellBuild(ctx) {
            // Emit placeholder for shell
            fmt.Fprintf(w, `<!--gospa-slot:%s-->`, name)
            return nil
        }
        // Normal render: emit full content wrapped in marker
        fmt.Fprintf(w, `<div data-gospa-slot="%s">`, name)
        if err := content.Render(ctx, w); err != nil {
            return err
        }
        fmt.Fprint(w, `</div>`)
        return nil
    })
}

// context key used to signal shell-build phase
type pprShellKey struct{}

func WithPPRShellBuild(ctx context.Context) context.Context {
    return context.WithValue(ctx, pprShellKey{}, true)
}

func IsPPRShellBuild(ctx context.Context) bool {
    v, _ := ctx.Value(pprShellKey{}).(bool)
    return v
}
```

### 3.2 PPR shell cache in `gospa.go`

Add a separate shell cache (separate map, same eviction pool):

```go
pprShellCache map[string][]byte   // path → static shell HTML (with slot placeholders)
pprShellKeys  []string
```

### 3.3 PPR render path in `renderRoute`

```
1. opts = GetRouteOptions(route.Path); strategy == StrategyPPR
2. Check pprShellCache[cacheKey]:
   a. Miss → render with WithPPRShellBuild(ctx), store result as static shell
   b. Hit  → use cached shell

3. For each slot in opts.DynamicSlots:
   a. Re-render the relevant sub-component (looked up by slot name from a SlotRegistry, see §3.4)
   b. Produce slot HTML fragment

4. Stream response:
   a. Write shell up to <!--gospa-slot:name-->
   b. Write dynamic slot content
   c. Continue writing shell after placeholder
   d. Repeat for each slot
   e. Flush
```

### 3.4 PPR Slot Registry

A simple registry mapping slot names to component functions, registered alongside the page:

```go
// routing/registry.go addition
type SlotFunc func(props map[string]interface{}) templ.Component

// Per-page slot registry (stored in Registry, keyed by page path)
slots map[string]map[string]SlotFunc

func RegisterSlot(pagePath, slotName string, fn SlotFunc)
func GetSlot(pagePath, slotName string) SlotFunc
```

---

## Phase 4 — Per-Page Strategy Selector in Generated Code

Currently `gospa generate` produces `generated/routes.go` which calls `routing.RegisterPage`. Extend it to also emit `routing.RegisterPageWithOptions` calls.

### 4.1 Convention: `page.options.go` file

Each route can optionally have a `page.options.go` file next to `page.templ`:

```go
// routes/blog/[slug]/page.options.go
package blog

import (
    "time"
    "github.com/aydenstechdungeon/gospa/routing"
)

var PageOptions = routing.RouteOptions{
    Strategy:        routing.StrategyISR,
    RevalidateAfter: 5 * time.Minute,
}
```

The code generator (`routing/generator/`) reads `PageOptions` from each route package (via a naming convention, not reflection) and emits the correct `RegisterPageWithOptions` call in `generated/routes.go`.

### 4.2 Inline option via `init()` (always available, no code gen required)

Users can always skip code gen entirely and register manually in any `init()`:

```go
// routes/blog/[slug]/page.templ
func init() {
    routing.RegisterPageWithOptions("/blog/:slug", func(props map[string]interface{}) templ.Component {
        return Page(props)
    }, routing.RouteOptions{
        Strategy:        routing.StrategyISR,
        RevalidateAfter: 5 * time.Minute,
    })
}
```

This already works today with SSG. The plan just adds ISR/PPR options.

### 4.3 PPR slot inline registration

```go
func init() {
    routing.RegisterPageWithOptions("/dashboard", dashboardPage, routing.RouteOptions{
        Strategy:     routing.StrategyPPR,
        DynamicSlots: []string{"feed", "notifications"},
    })
    routing.RegisterSlot("/dashboard", "feed",          feedSlot)
    routing.RegisterSlot("/dashboard", "notifications", notificationsSlot)
}
```

---

## Phase 5 — Config-Level Defaults

Add a `DefaultRenderStrategy` field to `Config` so apps can set a global default without touching every page:

```go
// gospa.go Config struct addition:
DefaultRenderStrategy routing.RenderStrategy // overrides built-in SSR default
DefaultRevalidateAfter time.Duration         // ISR default TTL when not set per-page
```

`GetRouteOptions` continues to return the per-page override when set, falling back to these config defaults.

---

## Phase 6 — HTTP Cache Headers

Each strategy emits appropriate `Cache-Control` headers so CDNs and proxies can cooperate:

| Strategy | Cache-Control |
|---|---|
| SSR | `no-store` |
| SSG | `public, max-age=31536000, immutable` |
| ISR | `public, s-maxage=<TTL>, stale-while-revalidate=<TTL>` |
| PPR | Shell: `public, max-age=31536000` / Slots: `no-store` |

These are set in `renderRoute` before writing the response body.

---

## Phase 7 — Documentation

### 7.1 `docs/RENDERING.md` (new file)

Create `/home/a4bet/gospa/docs/RENDERING.md` with:

- Overview table: SSR / SSG / ISR / PPR comparison
- When to use each strategy
- Config reference (`CacheTemplates`, `SSGCacheMaxEntries`, `DefaultRenderStrategy`, `DefaultRevalidateAfter`)
- Per-page API (`RouteOptions`, `RegisterPageWithOptions`)
- ISR stale-while-revalidate explanation
- PPR slot API (`DynamicSlot`, `RegisterSlot`, `RouteOptions.DynamicSlots`)
- Code examples for all four strategies
- HTTP cache header behavior per strategy
- Interaction with Prefork mode (ISR/SSG cache is per-process without shared storage — document the limitation)

### 7.2 Update `docs/API.md`

- Update `RouteOptions` block at line ~576 to include `RevalidateAfter` and `DynamicSlots`
- Update `Config` block to include `DefaultRenderStrategy` and `DefaultRevalidateAfter`
- Add `RegisterSlot` / `GetSlot` to the Route Registry section
- Add `DynamicSlot`, `WithPPRShellBuild`, `IsPPRShellBuild` to the Templ Package section

### 7.3 `docs/CONFIGURATION.md`

Add a **Rendering Strategies** subsection documenting:
- `CacheTemplates` (already exists, clarify it applies to SSG and ISR both)
- `SSGCacheMaxEntries` (also covers ISR entries)
- `DefaultRenderStrategy`
- `DefaultRevalidateAfter`

### 7.4 Website: new docs page `website/routes/docs/rendering/page.templ`

Create a new route at `/docs/rendering` with:

- Section: Strategy Overview (comparison table)
- Section: SSR (default, how it works)
- Section: SSG (static generation, cache config)
- Section: ISR (TTL, stale-while-revalidate, background regen)
- Section: PPR (shell concept, slot registration, streaming)
- Section: Per-Page Configuration (inline `init()` + `page.options.go` convention)
- Section: Global Defaults (Config fields)
- Code examples for each strategy using `@components.CodeBlock`

Add the new page to the nav in `website/routes/docs/page.templ` and the sidebar layout.

### 7.5 Update `README.md`

Add **Rendering Strategies** subsection under **Core Concepts**:

```markdown
### Rendering Strategies

GoSPA supports four per-page rendering strategies:

| Strategy | When to Use |
|---|---|
| `StrategySSR` | Dynamic content, auth-gated pages (default) |
| `StrategySSG` | Fully static content, marketing pages |
| `StrategyISR` | Mostly static, refreshes every N minutes |
| `StrategyPPR` | Static shell with dynamic sections (e.g. dashboards) |

Select a strategy per-page in your route's `init()`:

​```go
func init() {
    routing.RegisterPageWithOptions("/pricing", PricingPage, routing.RouteOptions{
        Strategy:        routing.StrategyISR,
        RevalidateAfter: 10 * time.Minute,
    })
}
​```
```

Also update the Features bullet point (currently "Rendering Modes — Seamlessly mix CSR, SSR, and SSG per-page rendering strategies") to mention ISR and PPR.

Update the Comparison table to add ISR and PPR rows.

---

## Implementation Order

1. **`routing/registry.go`** — add `StrategyISR`, `StrategyPPR`, `RevalidateAfter`, `DynamicSlots` to `RouteOptions`; add `SlotFunc`, `RegisterSlot`, `GetSlot`
2. **`gospa.go`** — refactor `ssgCache` to `ssgEntry`, add `pprShellCache`; extend `renderRoute` with ISR and PPR branches; add HTTP cache headers; add config defaults
3. **`templ/ppr.go`** — `DynamicSlot`, `WithPPRShellBuild`, `IsPPRShellBuild`
4. **`docs/RENDERING.md`** — new file
5. **`docs/API.md`** — update `RouteOptions`, `Config`, route registry, templ sections
6. **`docs/CONFIGURATION.md`** — add rendering fields
7. **`website/routes/docs/rendering/page.templ`** — new website page
8. **`README.md`** — add Rendering Strategies section, update features + table

---

## Non-Goals (Out of Scope for This Plan)

- Global SSG ("build-time" pre-render CLI command) — covered by `runtime-performance-optimization-plan.md`
- Edge/CDN deployment helpers
- ISR persistence across restarts (cache is in-memory; acceptable since ISR revalidates by TTL anyway)
- PPR streaming over SSE (uses Fiber's existing `SetBodyStreamWriter`)
