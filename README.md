# GoSPA

A modern SPA framework for Go that brings Svelte-like reactivity to server-side rendering with Fiber and Templ.

## Overview

GoSPA enables developers to build Single Page Applications using Go with:

- **Svelte rune-like reactive primitives** - `$state`, `$derived`, `$effect` equivalents in Go
- **Automatic file-based routing** - SvelteKit-style routing for `.templ` files
- **WebSocket state synchronization** - Real-time client-server differential state sync
- **Generic State Stores** - Strong typing using Go Generics wrapped in an `Observable` interface
- **TypeScript Definition Generation** - Export `.go` structs into `.d.ts` types automatically
- **State HMR** - Automatically patch state in dev-mode across application reloads
- **Partial Hydration** - Opt-out of reactivity for static zones using `data-gospa-static`
- **Transition Engine** - Svelte-like animation engine (`data-transition="fade"`) natively in runtime
- **Lightweight client runtime** - <15KB gzipped, replaces HTMX/Alpine/Petite Vue
- **Type-safe templating** - Compile-time template validation with Templ

## Installation

```bash
go get github.com/aydenstechdungeon/gospa
```

## Quick Start

### 1. Create a new project

```bash
mkdir myapp && cd myapp
go mod init myapp
```

### 2. Create a simple page

```go
// main.go
package main

import (
    "log"
    
    gofiber "github.com/gofiber/fiber/v2"
    "github.com/aydenstechdungeon/gospa/fiber"
    "github.com/aydenstechdungeon/gospa/state"
)

type PageState struct {
    Count int `json:"count"`
}

func main() {
    app := gofiber.New()
    
    // Inject GoSPA runtime
    app.Use(gospafiber.RuntimeMiddleware())
    
    // Your page route
    app.Get("/", func(c *gofiber.Ctx) error {
        pageState := PageState{Count: 0}
        return c.Render("index", gofiber.Map{
            "State": pageState,
        })
    })
    
    log.Fatal(app.Listen(":3000"))
}
```

### 3. Run the server

```bash
go run main.go
```

## Core Concepts

### State Management

GoSPA provides Svelte-like reactive primitives:

#### Rune[T] - Reactive State

```go
import "github.com/aydenstechdungeon/gospa/state"

// Create a reactive value
count := state.NewRune(0)

// Get value
value := count.Get()

// Set value (notifies subscribers)
count.Set(5)

// Subscribe to changes
unsubscribe := count.Subscribe(func(newValue int) {
    fmt.Println("Count changed to:", newValue)
})
defer unsubscribe()
```

#### Derived[T] - Computed State

```go
// Create derived state that updates when dependencies change
count := state.NewRune(5)
doubled := state.NewDerived(func() int {
    return count.Get() * 2
})

fmt.Println(doubled.Get()) // 10
count.Set(10)
fmt.Println(doubled.Get()) // 20
```

#### Effect - Side Effects

```go
count := state.NewRune(0)

// Run side effect when count changes
cleanup := state.NewEffect(func() func() {
    fmt.Println("Count is:", count.Get())
    return func() {
        fmt.Println("Cleanup on next run or unsubscribe")
    }
})
defer cleanup()
```

#### Batch Updates

```go
count := state.NewRune(0)
name := state.NewRune("")

// Batch multiple updates - subscribers notified once
state.Batch(func() {
    count.Set(1)
    name.Set("Alice")
})
```

### Routing

#### File-Based Routing (Automatic)

Create `.templ` files in the `routes/` directory:

```
routes/
├── root_layout.templ    → Base HTML shell
├── page.templ           → /
├── about/
│   └── page.templ       → /about
├── blog/
│   ├── layout.templ     → Shared layout for /blog/*
│   └── [id]/
│       └── page.templ   → /blog/:id
└── posts/
    └── [...rest]/
        └── page.templ   → /posts/* (catch-all)
```

#### Manual Routing

For advanced use cases:

```go
import "github.com/aydenstechdungeon/gospa/routing"

router := routing.NewManualRouter()

// Register routes
router.GET("/", homeHandler)
router.GET("/users/:id", userHandler)
router.POST("/api/data", createDataHandler)

// Get all routes for registration
for _, route := range router.GetRoutes() {
    fmt.Printf("%s %s\n", route.Method, route.Path)
}
```

```

### Client Runtime & Partial Hydration

The client runtime provides:

- **State synchronization** via WebSocket
- **DOM binding** with reactive updates
- **Event handling** with server callbacks
- **Optimistic updates** with automatic rollbacks on network failure
- **State HMR** - Hot-swaps internal states automatically across local dev rebuilds
- **Partial Hydration** - Skip evaluating or binding static trees of the DOM with `data-gospa-static`

#### Partial Hydration Example

```html
<div data-gospa-static>
    <h1>This huge tree will not be bound by GoSPA and saves memory!</h1>
    <p>And it will never have recursive event listeners attached either.</p>
</div>
```

#### Transitions

You can add animations to DOM elements seamlessly when they mount or unmount via transition data-attributes. Included primitives: `fade`, `fly`, and `slide`.

```html
<div data-transition="fade" data-transition-params='{"duration": 300}'>
    I fade in and out!
</div>

<div data-transition-in="fly" data-transition-params='{"y": 50}' data-transition-out="slide">
    I fly in from the bottom, but I slide up on removal!
</div>
```

#### Runtime Features

```javascript
// Initialize component with server state
GoSPA.init('component-id', { count: 0 });

// Create reactive state
const count = new GoSPA.Rune(0);

// Create derived state
const doubled = new GoSPA.Derived(() => count.get() * 2);

// Create effect
const cleanup = new GoSPA.Effect(() => {
    console.log('Count:', count.get());
});

// Bind to DOM
GoSPA.bind('element-id', count);

// WebSocket sync
GoSPA.connect('ws://localhost:3000/ws');
```

### Fiber Integration

#### Middleware

```go
// Inject runtime JavaScript
app.Use(gospafiber.RuntimeMiddleware())
```

#### WebSocket Handler

```go
import "github.com/gofiber/websocket/v2"

app.Get("/ws", websocket.New(func(c *websocket.Conn) {
    client := gospafiber.NewWSClient("client-id", c)
    defer client.Close()
    
    // Send initial state
    client.Conn.WriteJSON(state.NewInitMessage("app", initialState))
    
    // Handle incoming messages
    for {
        _, msg, err := c.ReadMessage()
        if err != nil {
            break
        }
        // Process message...
    }
}))
```

## Project Structure

```
myapp/
├── routes/              # Auto-routed .templ files
│   ├── page.templ       # Home page
│   ├── about/
│   │   └── page.templ   # /about
│   └── blog/
│       └── [id]/
│           └── page.templ  # /blog/:id
├── components/          # Reusable .templ components
├── lib/                 # Shared Go code
│   └── state.go         # App-wide state
├── main.go
└── go.mod
```

## Examples

See the [`examples/`](./examples) directory:

- **Counter** - Basic reactive counter
- **Blog** - Dynamic routes with parameters

## API Reference

### State Package

```go
// Rune - reactive value
rune := state.NewRune[T](initial T)
rune.Get() T
rune.Set(value T)
rune.Subscribe(fn func(T)) func() // returns unsubscribe

// Derived - computed value
derived := state.NewDerived[T](compute func() T)
derived.Get() T

// Effect - side effect
effect := state.NewEffect(fn func() func()) func()

// Batch - batch updates
state.Batch(fn func())
```

### Routing Package

```go
// Manual router
router := routing.NewManualRouter()
router.GET(path string, handler routing.Handler, middleware ...routing.Middleware)
router.POST(path string, handler routing.Handler, middleware ...routing.Middleware)
router.GetRoutes() []*routing.ManualRoute
```

### Fiber Package

```go
// Middleware
gospafiber.RuntimeMiddleware() fiber.Handler

// WebSocket client
client := gospafiber.NewWSClient(id string, conn *websocket.Conn)
```

### State Serialization

```go
// Create state messages for client
initMsg := state.NewInitMessage(componentID string, state interface{})
syncMsg := state.NewSyncMessage(componentID string, state interface{})
```

## CLI Tool

```bash
# Create new project
gospa create myapp

# Generate types and routes
gospa generate

# Development server with hot reload
gospa dev

# Production build
gospa build
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
│  ┌─────────────────────────────────────────────────────┐    │
│  │                 Routing Package                      │    │
│  │  ┌──────────────┐  ┌──────────────┐                 │    │
│  │  │  Auto Router │  │ Manual Router│                 │    │
│  │  └──────────────┘  └──────────────┘                 │    │
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

MIT License

## Contributing

Contributions welcome! Please read the contributing guidelines first.
