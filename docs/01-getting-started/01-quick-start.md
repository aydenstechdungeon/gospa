# Quick Start

Get your first reactive GoSPA application running in under five minutes.

## Prerequisites

- **Go 1.23+**
- **Bun** (for client-side builds)
- **Templ** CLI (`go install github.com/a-h/templ/cmd/templ@latest`)

## 1. Install GoSPA CLI

The CLI is the recommended way to manage GoSPA projects.

```bash
go install github.com/aydenstechdungeon/gospa/cmd/gospa@latest
```

## 2. Scaffold a Project

Create a new project using the `create` command. This sets up the directory structure and recommended configuration.

```bash
gospa create my-app
cd my-app
go mod tidy
```

## 3. Launch Development Server

GoSPA's dev server handles hot reloading, route generation, and runtime builds automatically.

```bash
gospa dev
```

Your app is now running at `http://localhost:3000`.

## 4. Define a Route

Routes are based on the file system. Create `routes/hello.templ`:

```go
package routes

templ HelloPage() {
    <div class="p-8 max-w-md mx-auto">
        <h1 class="text-4xl font-black italic tracking-tighter mb-4 underline decoration-[var(--accent-primary)]">
            Hello GoSPA
        </h1>
        <p class="text-lg text-[var(--text-secondary)]">
            You just created a route in seconds.
        </p>
    </div>
}
```

The CLI automatically detects this new file and registers the route `/hello`.

## 5. Add Reactive State

GoSPA uses a "state machine on the server" approach. Update `routes/hello.templ`:

```go
package routes

import (
    "github.com/aydenstechdungeon/gospa/state"
)

templ HelloPage() {
    <div data-gospa-component="hello" class="p-8">
        <h2 class="text-2xl font-bold">Counter: <span data-bind="count">0</span></h2>
        
        <button 
            data-on="click:increment"
            class="mt-4 px-6 py-2 bg-[var(--accent-primary)] text-white rounded-full font-bold hover:scale-105 transition-transform"
        >
            Increment
        </button>
    </div>
}

// HelloState defines the initial state for the 'hello' component
func HelloState() *state.StateMap {
    sm := state.NewStateMap()
    sm.AddAny("count", 0)
    return sm
}
```

### What's happening here?
1. `data-gospa-component`: Identifies this element as a reactive component.
2. `data-bind="count"`: Automatically updates the text content when the "count" state changes.
3. `data-on="click:increment"`: Maps a browser click to a server-side action (if using WebSockets) or a client-side transition.

## Next Steps

- **[Core Concepts: State](../02-core-concepts/03-state.md)** — Deep dive into Runes, Derived values, and Effects.
- **[Routing](../02-core-concepts/06-routing.md)** — Dynamic parameters and layout nesting.
- **[CLI Reference](../04-api-reference/03-cli.md)** — Master the `gospa` command.
