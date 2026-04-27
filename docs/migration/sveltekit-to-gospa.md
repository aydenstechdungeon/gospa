# SvelteKit to GoSPA Migration Guide

This guide maps common SvelteKit patterns to GoSPA equivalents.

## Route data loading

- SvelteKit: `load(...)`
- GoSPA: `Load(c routing.LoadContext) (map[string]interface{}, error)` in `+page.server.go` or module script in `+page.gospa`.

Client helper:

```ts
import { loadRouteData } from "/_gospa/runtime.js";
const data = await loadRouteData("/dashboard");
```

## Form actions

- SvelteKit: `actions = { ... }` / `default`
- GoSPA: `ActionDefault` and `Action<Name>` server exports.

Client helper:

```ts
import { callRouteAction } from "/_gospa/runtime.js";
const result = await callRouteAction("/posts", "publish", formData);
```

Action response envelope:

- `code`: `SUCCESS | REDIRECT | FAIL`
- `data`, `validation`, `redirect`, `error`, `revalidate*`

## Redirect and fail semantics

- SvelteKit: `redirect(status, to)`, `fail(status, data)`
- GoSPA: `kit.Redirect(status, to)`, `kit.Fail(status, data)`

Typed HTTP errors:

- SvelteKit: `error(status, body)`
- GoSPA: `kit.Error(status, body)`

Helper markers: `kit.Error`

These are supported in `Load` and action exports.

## Parent data and dependency tracking

- SvelteKit: `parent()`, `depends(...)`, `untrack(...)`
- GoSPA:
  - `kit.Parent[T](c)` (immediate parent layout data)
  - `kit.Depends(keys...)`
  - `kit.Untrack(func() error)`

Helper markers: `kit.Parent`, `kit.Depends`, `kit.Untrack`

## Navigation lifecycle

- SvelteKit: `beforeNavigate`, `afterNavigate`, `goto`, preloading APIs
- GoSPA: same helper surface from GoSPA runtime module (`/_gospa/runtime.js`):
  - `beforeNavigate`, `afterNavigate`
  - `goto(path, options?)`
  - `preloadData(path)`, `preloadCode(path)`
  - `refresh(init?)`
  - `prefetchOnHover(selector, options?)`

## Invalidation

- SvelteKit: `invalidate(...)`, `invalidateAll()`
- GoSPA:
  - `invalidate(path)`
  - `invalidateTag(tag)`
  - `invalidateKey(key)`
  - `invalidateAll()`

## Reactivity mapping

- SvelteKit runes:
  - `$state` -> GoSPA `$state`
  - `$derived` -> GoSPA `$derived`
  - `$effect` -> GoSPA `$effect(func() { ... })` in Go SFC scripts

## SFC module script mapping

In `+page.gospa`:

```svelte
<script context="module" lang="go">
  func Load(c routing.LoadContext) (map[string]interface{}, error) { ... }
  func ActionDefault(c routing.LoadContext) (interface{}, error) { ... }
  func ActionSave(c routing.LoadContext) (interface{}, error) { ... }
</script>
```

Rules:

- Exports must be top-level functions (not methods).
- Action exports must be `Action<Name>` with uppercase suffix.
- Do not mix module exports with sibling `+page.server.go`/`+layout.server.go`.

## Practical migration checklist

1. Port `load` logic into `Load` function exports.
2. Port `actions` into `ActionDefault` / `Action<Name>`.
3. Replace client `goto`/preload calls with `/_gospa/runtime.js` helpers.
4. Switch invalidation to path/tag/key helpers.
5. Validate enhanced form behavior with `callRouteAction` and `enhanceForm`.
