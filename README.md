# GoSPA 
![gospa1 128](https://github.com/user-attachments/assets/9e9d126d-8c91-465a-a5c0-968e41c095fb)
![gospa2 128](https://github.com/user-attachments/assets/338c5be2-9ce1-4f7a-a389-bfe176c6a9d6)


A Go framework for building reactive SPAs with server-side rendering. Brings Svelte-like reactivity to Go using Fiber and Templ.

## Features

- **Reactive Primitives** — `Rune[T]`, `Derived[T]`, `Effect` - Svelte-like reactivity in Go
- **File-Based Routing** — SvelteKit-style routing for `.templ` files
- **WebSocket Sync** — Real-time client-server state synchronization
- **Session Management** — Secure session persistence with `SessionStore` and `ClientStateStore`
- **Type Safety** — Compile-time template validation with Templ
- **Lightweight Runtime** — ~15KB default runtime (trusts server), ~35KB secure runtime with DOMPurify for user-generated content.
- **Remote Actions** — Type-safe server functions callable directly from the client.
- **Error Handling** — Global error boundaries, panic recovery, and error overlay in dev mode.
- **Security** — Built-in CSRF protection, customizable CORS origins, and strict XSS prevention.
- **Rendering Modes** — Mix SSR, SSG, ISR, and PPR per-page rendering strategies.

## Installation

```bash
go get github.com/aydenstechdungeon/gospa
```

## Quick Start

### 1. Initialize Project

```bash
go install github.com/aydenstechdungeon/gospa/cmd/gospa@latest
gospa create myapp
cd myapp
go mod tidy
```

### 2. Create Main File

```go
// main.go
package main

import (
    "log"
    _ "myapp/routes" // Import routes to trigger init()
    
    "github.com/aydenstechdungeon/gospa"
)

func main() {
    app := gospa.New(gospa.Config{
        RoutesDir: "./routes",
        DevMode:   true,
        AppName:   "myapp",
    })

    if err := app.Run(":3000"); err != nil {
        log.Fatal(err)
    }
}
```

### 3. Create a Page

```templ
// routes/page.templ
package routes

templ Page() {
    <div data-gospa-component="counter" data-gospa-state='{"count":0}'>
        <h1>Counter</h1>
        <span data-bind="text:count">0</span>
        <button data-on="click:increment">+</button>
        <button data-on="click:decrement">−</button>
    </div>
}
```

### 4. Run

```bash
gospa generate
go run .
```
or
```bash
gospa dev
```

## Core Concepts

### State Modes

GoSPA supports two state management modes:

#### Local State (Client-Only)

State lives entirely in the browser. No server synchronization.

```templ
<div data-gospa-component="counter" data-gospa-local>
    <span data-bind="text:count">0</span>
    <button data-on="click:increment">+</button>
</div>
```

#### Synced State (Client-Server)

State synchronizes across all connected clients via WebSocket.

```go
// Server-side handler with broadcast
fiber.RegisterActionHandler("increment", func(client *fiber.WSClient, payload json.RawMessage) {
    GlobalCounter.Count++
    fiber.BroadcastState(hub, "count", GlobalCounter.Count)
})
```

### Reactive Primitives

#### Rune[T] — Reactive State

```go
count := state.NewRune(0)
count.Get()           // 0
count.Set(5)          // notifies subscribers
count.Update(func(v int) int { return v + 1 })
```

#### Derived[T] — Computed State

```go
count := state.NewRune(5)
doubled := state.NewDerived(func() int {
    return count.Get() * 2
})
doubled.Get() // 10
```

#### Effect — Side Effects

```go
cleanup := state.NewEffect(func() func() {
    fmt.Println("Count:", count.Get())
    return func() { fmt.Println("cleanup") }
})
defer cleanup()
```

### File-Based Routing

```
routes/
├ root_layout.templ    → Base HTML shell
├ page.templ           → /
├ about/
│   └ page.templ       → /about
├ (auth)/              → Grouped routes
│   ├ layout.templ
│   ├ login/
│   │   └ page.templ   → /login
│   └ register/
│       └ page.templ   → /register
├ blog/
│   ├── layout.templ   → Layout for /blog/*
│   └ [id]/
│       └ page.templ   → /blog/:id
└ posts/
    └ [...rest]/
        └ page.templ   → /posts/* (catch-all)
```

#### Embedded Routes (Production)

For production, you can embed your routes into the binary using `go:embed`. This allows for a zero-dependency, single-binary distribution.

```go
// prod.go
//go:embed routes/*
var embeddedRoutes embed.FS

// main.go
var routesFS fs.FS
if devMode {
    routesFS = os.DirFS("./routes") // Real files for hot-reloading
} else {
    // Use the embedded files for production
    sub, _ := fs.Sub(embeddedRoutes, "routes")
    routesFS = sub
}

app := gospa.New(gospa.Config{
    RoutesFS: routesFS,
    DevMode:  devMode,
    // ...
})
```

### Client Runtime

```javascript
// Initialize component
GoSPA.init({ wsUrl: 'ws://localhost:3000/_gospa/ws' })

// Reactive state
const count = new GoSPA.Rune(0)
const doubled = new GoSPA.Derived(() => count.get() * 2)

// Effects
new GoSPA.Effect(() => {
    console.log('Count:', count.get())
})

// Remote action helper
GoSPA.remote('saveData', { count: count.get() })
```

#### Runtime Variants: Default vs Secure

GoSPA follows a **"trust the server"** security model (similar to SvelteKit). The default runtime trusts server-rendered HTML because Templ auto-escapes all dynamic content.

**Default Runtime (`gospa`) — Recommended for most apps:**
```typescript
import { init } from '@gospa/client';
init(); // ~15KB, no sanitizer needed
```

**Secure Runtime (`gospa/runtime-secure`) — For user-generated content:**
```typescript
import { init, sanitize } from '@gospa/client/runtime-secure';
init(); // ~35KB, includes DOMPurify

// Sanitize user-generated HTML
const clean = await sanitize(userComment);
```

| Runtime | Size | Sanitizer | Use Case |
|---------|------|-----------|----------|
| `gospa` | ~15KB | None (trusts server) | Most apps with CSP |
| `gospa/runtime-secure` | ~35KB | DOMPurify | Apps with user-generated HTML |

**When do you need the secure runtime?**
- User comments with HTML formatting
- Forums, wikis, social media apps
- Rich text editors (WYSIWYG)
- Any user-generated HTML content

**When is the default runtime sufficient?**
- Server-rendered Templ templates
- Text content from your database
- JSON data
- Any content already escaped by Templ on the server

See [`docs/03-features/01-client-runtime.md`](docs/03-features/01-client-runtime.md) for the complete runtime selection guide.

### Remote Actions

Remote Actions allow you to define type-safe server functions that can be invoked seamlessly from the client without manually managing HTTP endpoints.

```go
import (
    "context"
    "github.com/aydenstechdungeon/gospa/routing"
)

// Register on server
routing.RegisterRemoteAction("saveData", func(ctx context.Context, input interface{}) (interface{}, error) {
    // Type assert input to access data
    data, ok := input.(map[string]interface{})
    if !ok {
        return nil, errors.New("invalid input")
    }
    
    // Process data securely on the server
    id, _ := data["id"].(float64) // JSON numbers parse as float64
    
    return map[string]interface{}{
        "status": "success",
        "id":     int(id),
    }, nil
})

// Configure endpoint restrictions
app := gospa.New(gospa.Config{
    RemotePrefix:       "/api/rpc",
    MaxRequestBodySize: 1024 * 1024, // Limit body to 1MB
})
```

```typescript
// Call from client
import { remote } from '@gospa/client';

const result = await remote('saveData', { id: 123 });

if (result.ok) {
    console.log('Success:', result.data);
} else {
    console.error('Error:', result.error, 'Code:', result.code);
    // Handle specific error codes programmatically
    if (result.code === 'ACTION_NOT_FOUND') {
        console.error('Action does not exist');
    }
}
```

### Application Security

GoSPA comes with secure defaults, but robust configurations exist for production use to secure cross-origin requests and mitigate CSRF attacks:

```go
app := gospa.New(gospa.Config{
    // ...
    AllowedOrigins: []string{"https://myapp.com", "https://api.myapp.com"},
})
```

> **Security By Default:** 
> When you set `EnableCSRF: true`, GoSPA automatically wires both the token issuer 
> and validator middlewares for you. The built-in client remote helper sends the `X-CSRF-Token` header automatically for same-origin requests.

### Rendering Strategies

GoSPA supports four per-page rendering strategies:

| Strategy | When to Use |
|----------|-------------|
| `StrategySSR` | Auth-gated pages, per-user content, real-time data (default) |
| `StrategySSG` | Fully static: marketing, docs, landing pages |
| `StrategyISR` | Mostly static, refresh every N minutes (stale-while-revalidate) |
| `StrategyPPR` | Static shell with dynamic inner sections (app dashboards) |

Select a strategy per-page in your `init()`:

```go
import (
    "time"
    "github.com/aydenstechdungeon/gospa/routing"
)

func init() {
    // ISR: serve stale, revalidate in background every 5 minutes
    routing.RegisterPageWithOptions("/pricing", pricingPage, routing.RouteOptions{
        Strategy:        routing.StrategyISR,
        RevalidateAfter: 5 * time.Minute,
    })

    // PPR: cache nav/footer shell, re-render feed slot per-request
    routing.RegisterPageWithOptions("/dashboard", dashboardPage, routing.RouteOptions{
        Strategy:     routing.StrategyPPR,
        DynamicSlots: []string{"feed"},
    })
    routing.RegisterSlot("/dashboard", "feed", feedSlot)
}
```

Enable caching in your app config:

```go
app := gospa.New(gospa.Config{
    CacheTemplates:         true,
    DefaultRenderStrategy:  routing.StrategyISR,  // app-wide fallback
    DefaultRevalidateAfter: 10 * time.Minute,
})
```

See [`docs/02-core-concepts/02-rendering.md`](docs/02-core-concepts/02-rendering.md) for full rendering documentation.

### Partial Hydration

Opt out of reactivity for static content:

```html
<div data-gospa-static>
    <h1>Static content — no bindings or event listeners</h1>
</div>
```

### Transitions

```html
<div data-transition="fade" data-transition-params='{"duration": 300}'>
    Fades in and out
</div>

<div data-transition-in="fly" data-transition-out="slide">
    Different enter/exit animations
</div>
```

## Project Structure

```
myapp/
├ routes/              # Auto-routed .templ files
│   ├── root_layout.templ  # Root HTML shell (optional)
│   ├── layout.templ       # Root-level layout (optional)
│   ├── page.templ         # Home page
│   └ about/
│       └ page.templ
├ components/          # Reusable .templ components (optional)
├ lib/                 # Shared Go code (optional)
│   └ state.go         # App state
├ main.go
└ go.mod
```

### Layout Files

GoSPA supports two types of layout files:

| File | Purpose | Scope |
|------|---------|-------|
| `root_layout.templ` | Outer HTML shell with `<html>`, `<head>`, `<body>` | Entire application |
| `layout.templ` | Nested layouts for sections | Route segment and children |

**`root_layout.templ`** — The outermost wrapper for your app. Must include the HTML document structure and GoSPA runtime script. There can only be one root layout (at `routes/root_layout.templ`). Requires `routing.RegisterRootLayout()`.

**`layout.templ`** — Regular layouts that wrap pages within a route segment. You can have multiple nested layouts (e.g., `routes/blog/layout.templ` wraps all `/blog/*` pages).

```
routes/
├── root_layout.templ     # (Optional) Wraps entire app (Requires routing.RegisterRootLayout())
├── layout.templ          # Optional root-level layout
├── page.templ            # Home page (/)
├── about/
│   └── page.templ        # About page (/about)
└── blog/
    ├── layout.templ      # Wraps all blog pages
    └── page.templ        # Blog index (/blog)
```

If no `root_layout.templ` exists, GoSPA provides a minimal default HTML wrapper.

## API Reference

See [`docs/04-api-reference/`](docs/04-api-reference/) for complete API documentation.

## CLI

```bash
gospa create myapp    # Create new project
gospa generate        # Generate types and routes
gospa dev             # Development server with hot reload
gospa build           # Production build
```

Run any command with `--help` (for example `gospa build --help`) to see all available options and flags.

For more details, see [`docs/04-api-reference/03-cli.md`](docs/04-api-reference/03-cli.md).

## Plugin Ecosystem

GoSPA includes a powerful plugin system for extending build and development workflows.

### Built-in Plugins

| Plugin | Description | Commands |
|--------|-------------|----------|
| **Tailwind** | CSS processing with Tailwind CSS v4 | `gospa add:tailwind` (alias: `at`), `gospa tailwind:build` (alias: `tb`), `gospa tailwind:watch` (alias: `tw`) |
| **PostCSS** | Advanced CSS with plugins (autoprefixer, typography, forms) | `gospa add:postcss` (alias: `ap`), `gospa postcss:build` (alias: `pb`), `gospa postcss:watch` (alias: `pw`), `gospa postcss:config` (alias: `pc`) |
| **Image** | Image optimization and responsive variants | `gospa image:optimize` (alias: `io`), `gospa image:clean` (alias: `ic`), `gospa image:sizes` (alias: `is`) |
| **Validation** | Form validation (Valibot client + Go validator server) | `gospa validation:generate` (alias: `vg`), `gospa validation:create` (alias: `vc`), `gospa validation:list` (alias: `vl`) |
| **SEO** | Sitemap, robots.txt, meta tags, structured data | `gospa seo:generate` (alias: `sg`), `gospa seo:meta` (alias: `sm`), `gospa seo:structured` (alias: `ss`) |
| **Auth** | OAuth2, JWT sessions, TOTP/OTP authentication | `gospa auth:generate` (alias: `ag`), `gospa auth:secret` (alias: `as`), `gospa auth:otp` (alias: `ao`), `gospa auth:backup` (alias: `ab`), `gospa auth:verify` (alias: `av`) |
| **QRCode** | QR code generation with customizable options | Programmatic API only (no CLI commands) |

### Configuration

Plugins are configured in `gospa.yaml`:

```yaml
plugins:
  tailwind:
    input: ./styles/main.css
    output: ./static/css/output.css
  image:
    input: ./images
    output: ./static/images
    formats: [webp, jpeg]
    widths: [320, 640, 1280, 1920]
  auth:
    jwt_secret: ${JWT_SECRET}
    oauth_providers: [google]
    google_client_id: ${GOOGLE_CLIENT_ID}
    google_client_secret: ${GOOGLE_CLIENT_SECRET}
```

### Plugin Hooks

Plugins integrate at key lifecycle points:

- `BeforeGenerate` / `AfterGenerate` — Code generation
- `BeforeDev` / `AfterDev` — Development server
- `BeforeBuild` / `AfterBuild` — Production build

The current repository documents the built-in plugin model only. Dynamic plugin installation and external shared-library loading are not part of the checked-in CLI surface.

### Creating Custom Plugins

```go
package myplugin

import "github.com/aydenstechdungeon/gospa/plugin"

type MyPlugin struct{}

func (p *MyPlugin) Name() string { return "my-plugin" }
func (p *MyPlugin) Init() error { return nil }
func (p *MyPlugin) Dependencies() []plugin.Dependency {
    return []plugin.Dependency{
        {Name: "some-go-package", Type: plugin.DepGo},
        {Name: "some-bun-package", Type: plugin.DepBun},
    }
}
func (p *MyPlugin) OnHook(hook plugin.Hook, ctx map[string]interface{}) error {
    // Handle lifecycle hooks
    return nil
}
func (p *MyPlugin) Commands() []plugin.Command {
    return []plugin.Command{
        {Name: "my-plugin:run", Alias: "mp", Description: "Run my plugin"},
    }
}
```

See [`docs/04-api-reference/04-plugins.md`](docs/04-api-reference/04-plugins.md) for complete plugin documentation.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Browser                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              GoSPA Runtime (<15KB*)                   │    │
│  │  ┌─────────┐ ┌──────────┐ ┌─────────┐ ┌──────────┐ │    │
│  │  │  Rune   │ │ Derived  │ │ Effect  │ │WebSocket │ │    │
│  │  └────┬────┘ └────┬─────┘ └────┬────┘ └────┬─────┘ │    │
│  │       └───────────┴────────────┴──────────┘        │    │
│  │                      │                              │    │
│  │              ┌───────┴───────┐                     │    │
│  │              │  DOM Binder   │                     │    │
│  │              └───────────────┘                     │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                               │
                               │ WebSocket / HTTP
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                       Go Server                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                    Fiber App                         │    │
│  │  ┌──────────────┐  ┌──────────────┐                 │    │
│  │  │   Runtime    │  │   WebSocket  │                 │    │
│  │  │  Middleware  │  │   Handler    │                 │    │
│  │  └──────────────┘  └──────────────┘                 │    │
│  └─────────────────────────────────────────────────────┘    │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                  State Package                       │    │
│  │  ┌─────────┐ ┌──────────┐ ┌─────────┐ ┌──────────┐ │    │
│  │  │  Rune   │ │ Derived  │ │ Effect  │ │  Batch   │ │    │
│  │  └─────────┘ └──────────┘ └─────────┘ └──────────┘ │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

## Comparison

| Feature | GoSPA | HTMX | Alpine | SvelteKit |
|---------|-------|------|--------|-----------|
| Language | Go | HTML | JS | JS/TS |
| Runtime Size | ~15KB (default) | ~14KB | ~15KB | Varies |
| SSR | ✅ | ✅ | ❌ | ✅ |
| SSG | ✅ | ❌ | ❌ | ✅ |
| ISR | ✅ | ❌ | ❌ | ✅ |
| PPR | ✅ | ❌ | ❌ | ✅ |
| Type Safety | ✅ | ❌ | ❌ | ✅ |
| WebSocket | ✅ | ❌ | ❌ | ✅ |
| File Routing | ✅ | ❌ | ❌ | ✅ |
| Reactivity | ✅ | ❌ | ✅ | ✅ |

## License

[Apache License 2.0](LICENSE)
