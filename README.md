# GoSPA

A Go framework for building reactive SPAs with server-side rendering. Brings Svelte-like reactivity to Go using Fiber and Templ.

## Features

- **Reactive Primitives** — `$state`, `$derived`, `$effect` equivalents in Go
- **File-Based Routing** — SvelteKit-style routing for `.templ` files
- **WebSocket Sync** — Real-time client-server state synchronization
- **Local State Mode** — Client-only reactivity without server roundtrips
- **Type Safety** — Compile-time template validation with Templ
- **Lightweight Runtime** — <15KB gzipped client JavaScript

## Installation

```bash
go get github.com/aydenstechdungeon/gospa
```

## Quick Start

### 1. Initialize Project

```bash
mkdir myapp && cd myapp
go mod init myapp
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
        <span data-bind="count">0</span>
        <button data-on="click:increment">+</button>
        <button data-on="click:decrement">−</button>
    </div>
}
```

### 4. Run

```bash
go run main.go
```

## Core Concepts

### State Modes

GoSPA supports two state management modes:

#### Local State (Client-Only)

State lives entirely in the browser. No server synchronization.

```templ
<div data-gospa-component="counter" data-gospa-local>
    <span data-bind="count">0</span>
    <button data-on="click:increment">+</button>
</div>
```

```go
// Register local-only handlers (no WebSocket broadcast)
fiber.RegisterLocalAction("increment", func(state *fiber.LocalState, payload json.RawMessage) {
    state.Update("count", func(v int) int { return v + 1 })
})
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
├ blog/
│   ├── layout.templ   → Layout for /blog/*
│   └ [id]/
│       └ page.templ   → /blog/:id
└ posts/
    └ [...rest]/
        └ page.templ   → /posts/* (catch-all)
```

### Client Runtime

```javascript
// Initialize component
GoSPA.init('component-id', { count: 0 })

// Reactive state
const count = new GoSPA.Rune(0)
const doubled = new GoSPA.Derived(() => count.get() * 2)

// Effects
new GoSPA.Effect(() => {
    console.log('Count:', count.get())
})

// DOM binding
GoSPA.bind('element-id', count)
```

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
│   ├── page.templ
│   └ about/
│       └ page.templ
├ components/          # Reusable .templ components
├ lib/                 # Shared Go code
│   └ state.go         # App state
├ main.go
└ go.mod
```

## API Reference

See [`docs/API.md`](docs/API.md) for complete API documentation.

## CLI

```bash
gospa create myapp    # Create new project
gospa generate        # Generate types and routes
gospa dev             # Development server with hot reload
gospa build           # Production build
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Browser                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              GoSPA Runtime (<15KB)                   │    │
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
| Runtime Size | <15KB | ~14KB | ~15KB | Varies |
| SSR | ✅ | ✅ | ❌ | ✅ |
| Type Safety | ✅ | ❌ | ❌ | ✅ |
| WebSocket | ✅ | ❌ | ❌ | ✅ |
| File Routing | ✅ | ❌ | ❌ | ✅ |
| Reactivity | ✅ | ❌ | ✅ | ✅ |

## License

MIT
