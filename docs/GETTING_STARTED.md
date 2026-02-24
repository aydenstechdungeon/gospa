# Getting Started with GoSPA

Complete guide to building reactive single-page applications with GoSPA.

## What is GoSPA?

GoSPA is a Go-based SPA framework that brings Svelte-like reactive primitives to server-side rendering. It combines:

- **Go + Templ** for server-side rendering
- **TypeScript Runtime** for client-side reactivity
- **WebSocket Sync** for real-time state synchronization
- **File-based Routing** with automatic code generation

## Prerequisites

- Go 1.21 or later
- Node.js 18+ (for client-side tooling)
- Basic understanding of Go and TypeScript

## Installation

### Install the CLI

```bash
go install github.com/aydenstechdungeon/gospa/cmd/gospa@latest
```

### Create a New Project

```bash
gospa create myapp
cd myapp
go mod tidy
```

### Project Structure

```
myapp/
├── main.go                 # Application entry point
├── gospa.yaml              # Application configuration
├── go.mod                  # Go dependencies
├── routes/                 # Route definitions
│   ├── layout.templ        # Root layout
│   ├── page.templ          # Home page
│   └── generated_routes.go # Auto-generated routing
├── static/                 # Static assets
└── .gospa/                 # Framework cache
```

## Development Workflow

### Start Development Server

```bash
gospa dev
```

This starts:
- Go server with hot reload
- TypeScript generation on file changes
- WebSocket server for state sync

### Build for Production

```bash
gospa build
```

Creates optimized production build.

## Your First Page

### Create a Route

Create `routes/about.templ`:

```go
package routes

templ AboutPage() {
	<div>
		<h1>About</h1>
		<p>This is the about page.</p>
	</div>
}
```

### Add Interactive State

Create `routes/counter.templ`:

```go
package routes

import (
	"github.com/aydenstechdungeon/gospa/templ"
	"github.com/aydenstechdungeon/gospa/state"
)

templ CounterPage() {
	<div data-gospa-component="counter">
		<h1>Counter</h1>
		<p data-bind="text:count">0</p>
		<button data-on="click:increment">+1</button>
	</div>
}

templ CounterState() *state.StateMap {
	sm := state.NewStateMap()
	sm.AddAny("count", 0)
	return sm
}

templ CounterActions() map[string]func() {
	return map[string]func(){
		"increment": func() {
			// Action handled by client runtime
		},
	}
}
```

## Client-Side Reactivity

### The Runtime

GoSPA includes a TypeScript runtime that provides Svelte-like reactivity:

```typescript
import { Rune, Derived, Effect, StateMap } from '@gospa/runtime'

// Create reactive state
const count = new Rune(0)

// Create derived value
const doubled = new Derived(() => count.get() * 2)

// React to changes
const effect = new Effect(() => {
  console.log('Count changed:', count.get())
  return () => console.log('Cleanup')
})

// Update state
count.set(5)
count.update(v => v + 1)
```

### DOM Bindings

The runtime automatically handles DOM bindings:

```html
<!-- Bind text content -->
<p data-bind="text:count">0</p>

<!-- Bind input value -->
<input data-bind="value:name" />

<!-- Two-way binding -->
<input data-model="name" />

<!-- Event handlers -->
<button data-on="click:increment">Click</button>

<!-- Conditional rendering -->
<div data-bind="if:isVisible">Hidden content</div>

<!-- List rendering -->
<ul data-bind="list:items" data-item-name="todo">
  <li>{ todo.text }</li>
</ul>
```

## Routing

### File-Based Routing

Routes are defined by files in the `routes/` directory:

| File | URL | Description |
|------|-----|-------------|
| `page.templ` | `/` | Home page |
| `about.templ` | `/about` | About page |
| `blog/index.templ` | `/blog` | Blog index |
| `blog/[slug].templ` | `/blog/:slug` | Dynamic route |
| `(auth)/login.templ` | `/login` | Grouped route |
| `layout.templ` | N/A | Layout wrapper |

### Layouts

Create `routes/layout.templ` for a root layout:

```go
package routes

templ Layout(content templ.Component) {
	<!DOCTYPE html>
	<html>
		<head>
			<title>My App</title>
			<script src="/static/runtime.js"></script>
		</head>
		<body>
			<nav>
				<a href="/">Home</a>
				<a href="/about">About</a>
			</nav>
			<main>
				@content
			</main>
		</body>
	</html>
}
```

### Dynamic Routes

Create `routes/blog/[slug].templ`:

```go
package routes

templ BlogPost(slug string) {
	<article>
		<h1>Post: { slug }</h1>
	</article>
}
```

Access via `/blog/my-post` - `slug` will be `"my-post"`.

### Route Parameters

Access URL parameters in your templates:

```go
package routes

import "github.com/aydenstechdungeon/gospa/routing"

templ SearchPage(params routing.Params) {
	<div>
		<h1>Search: { params.Get("q") }</h1>
	</div>
}
```

## State Management

### Server-Side State

Use Go's state primitives for server-side state:

```go
package routes

import "github.com/aydenstechdungeon/gospa/state"

templ TodoPage() *state.StateMap {
	sm := state.NewStateMap()
	sm.AddAny("items", []string{})
	sm.AddAny("newItem", "")
	return sm
}
```

### Client-Side State

The client runtime mirrors server primitives:

```typescript
import { Rune, StateMap, batch } from '@gospa/runtime'

// Create state map
const state = new StateMap()
state.add('count', new Rune(0))
state.add('items', new Rune<string[]>([]))

// Batch updates
batch(() => {
  state.get('count')?.set(0)
  state.get('items')?.set([])
})
```

### WebSocket Sync

Enable real-time state sync:

```typescript
import { WSClient, syncedRune } from '@gospa/runtime'

const ws = new WSClient({
  url: 'ws://localhost:3000/ws',
  onConnect: () => console.log('Connected'),
  onDisconnect: () => console.log('Disconnected')
})

// Create synced rune
const count = syncedRune('count', 0, ws)
```

## Events

### Server-Side Events

Define event handlers:

```go
package routes

templ ButtonEvents() map[string]interface{} {
	return map[string]interface{}{
		"click": func(e templ.Event) {
			// Handle click
		},
	}
}
```

### Client-Side Events

The runtime handles events automatically:

```typescript
import { on, delegate, debounce, throttle } from '@gospa/runtime'

// Direct event
const unsub = on(button, 'click', (e) => {
  console.log('Clicked')
})

// Delegated events
delegate(document, 'click', '.btn', (e) => {
  console.log('Button clicked')
})

// Debounced handler
const debouncedClick = debounce((e) => {
  console.log('Debounced')
}, 300)
```

## Transitions

Add animations with the transition system:

```typescript
import { fade, fly, slide, scale, blur } from '@gospa/runtime'

// Apply transition
const element = document.querySelector('.fade-in')
fade(element, { duration: 300, delay: 0 })

// Available transitions
fly(element, { y: 50, duration: 400 })
slide(element, { direction: 'left', duration: 300 })
scale(element, { from: 0.5, duration: 200 })
blur(element, { from: 10, duration: 300 })
```

## Configuration

### Application Config

Create `gospa.yaml`:

```yaml
app:
  name: myapp
  port: 3000

performance:
  compress_state: true
  state_diffing: true

runtime:
  simple: false
  websocket: true
```

### Runtime Selection

Choose between full and minimal runtime:

**Full Runtime** (~17KB):
- DOMPurify sanitization
- Full reactive primitives
- WebSocket client
- Transitions

**Minimal Runtime** (~11KB):
- Basic reactivity
- No sanitization
- No WebSocket
- No transitions

```json
{
  "runtime": {
    "mode": "minimal"
  }
}
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `gospa create <name>` | Create new project |
| `gospa dev` | Start development server |
| `gospa build` | Build for production |
| `gospa generate` | Generate routes and types |
| `gospa check` | Type check project |

## Next Steps

1. **[Configuration Reference](./CONFIGURATION.md)** - All configuration options
2. **[Client Runtime API](./CLIENT_RUNTIME.md)** - Complete TypeScript API
3. **[State Primitives](./STATE_PRIMITIVES.md)** - Go reactive primitives
4. **[CLI Reference](./CLI.md)** - All CLI commands
5. **[Runtime Selection](./RUNTIME.md)** - Choose the right runtime

## Common Patterns

### Counter Component

```go
// routes/counter.templ
package routes

templ CounterPage() {
	<div data-gospa-component="counter">
		<h2>Counter</h2>
		<p>Count: <span data-gospa-bind="count">0</span></p>
		<button data-gospa-on:click="decrement">-</button>
		<button data-gospa-on:click="increment">+</button>
	</div>
}
```

```typescript
// Client-side handler
document.querySelector('[data-gospa-component="counter"]')
```

### Todo List

```go
// routes/todos.templ
package routes

templ TodosPage() {
	<div data-gospa-component="todos">
		<h2>Todos</h2>
		<input 
			type="text" 
			data-gospa-bind:value="newTodo"
			placeholder="Add todo..."
		/>
		<button data-gospa-on:click="addTodo">Add</button>
		<ul data-gospa-each="todos">
			<li>
				<span data-gospa-bind="text"></span>
				<button data-gospa-on:click="removeTodo">×</button>
			</li>
		</ul>
	</div>
}
```

### Form Handling

```go
// routes/contact.templ
package routes

templ ContactPage() {
	<form data-gospa-component="contact-form">
		<input 
			type="text" 
			name="name"
			data-gospa-bind:value="name"
			required
		/>
		<input 
			type="email" 
			name="email"
			data-gospa-bind:value="email"
			required
		/>
		<textarea 
			name="message"
			data-gospa-bind:value="message"
		></textarea>
		<button type="submit" data-gospa-on:click="submit">Send</button>
	</form>
}
```

## Troubleshooting

### Port Already in Use

```bash
# Kill process on port 3000
lsof -i :3000 | grep LISTEN | awk '{print $2}' | xargs kill -9
```

### Types Not Generated

```bash
# Regenerate types
gospa generate
```

### WebSocket Connection Failed

Check that WebSocket is enabled in config:

```json
{
  "runtime": {
    "websocket": true
  }
}
```

### Hot Reload Not Working

Ensure you're running `gospa dev` not `go run main.go`.
