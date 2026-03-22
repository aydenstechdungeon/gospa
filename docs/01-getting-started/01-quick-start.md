# Getting Started with GoSPA

Complete guide to building reactive single-page applications with GoSPA.

## What is GoSPA?

GoSPA is a Go-based SPA framework that brings Svelte-like reactive primitives to server-side rendering. It combines:

- **Go + Templ** for server-side rendering
- **TypeScript Runtime** for client-side reactivity
- **WebSocket Sync** for real-time state synchronization
- **File-based Routing** with automatic code generation

## Prerequisites

- **Go 1.25+** (matches `go` directive in `go.mod`; use the latest stable toolchain)
- **Bun** (for client runtime build, CSS, and JS checks in this repo)
- Basic familiarity with Go and TypeScript

Use `gospa.DefaultConfig()` as your starting point in development, `gospa.ProductionConfig()` for hardened deployments, and `gospa.MinimalConfig()` when you want the smallest baseline.

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
gospa doctor
```

### Project Structure

```
myapp/
├── main.go                 # Application entry point
├── go.mod                  # Go dependencies
├── routes/                 # Route definitions
│   ├── layout.templ        # Root-level layout scaffold
│   └── page.templ          # Home page scaffold
├── static/                 # Static assets
│   └── css/
│       └── style.css       # Starter styles
└── .gitignore              # Common local build artifacts
```

#### Layout Files

GoSPA uses two types of layout files with different purposes:

| File | Purpose | Scope |
|------|---------|-------|
| `root_layout.templ` | Outer HTML shell (`<html>`, `<head>`, `<body>`) | Entire application |
| `layout.templ` | Nested layouts for sections | Route segment and children |

**`root_layout.templ`** — Place this at `routes/root_layout.templ` to define the outermost HTML document structure. It must include the GoSPA runtime script and is registered specially via `routing.RegisterRootLayout()`. Only one root layout exists per app.

**`layout.templ`** — Regular layouts that wrap pages. Create these in subdirectories (e.g., `routes/blog/layout.templ`) to wrap all pages in that section. Multiple nested layouts are supported.

**Layout Hierarchy Example:**
```
routes/
├── root_layout.templ     # Wraps entire app
├── layout.templ          # Optional: additional root wrapper
├── page.templ            # Home page
├── about/
│   └── page.templ        # About page (wrapped by root_layout)
└── dashboard/
    ├── layout.templ      # Dashboard sidebar/header
    └── page.templ        # Dashboard home
```

In this example, the dashboard page is wrapped first by `dashboard/layout.templ`, then by `root_layout.templ`.

## Development Workflow

### Start Development Server

```bash
gospa dev
```

This starts:
- the Go development server with reload support
- route/template generation as files change
- the GoSPA runtime with WebSocket state sync enabled by default

### Build for Production

```bash
gospa build
```

Creates an optimized production build using the Go toolchain plus Bun-based client runtime steps when the project includes a client package.

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

templ CounterPage() {
	<div 
		data-gospa-component="counter"
		data-gospa-state='{"count":0}'
	>
		<h1>Counter</h1>
		<p data-bind="count">0</p>
		<button 
			onclick="var r=__GOSPA__.getState('counter','count');r&&__GOSPA__.setState('counter','count',r.get()+1)"
		>
			+1
		</button>
	</div>
}
```

## Client-Side Reactivity

### The Runtime

GoSPA includes a TypeScript runtime that provides Svelte-like reactivity:

```typescript
import { Rune, Derived, Effect, StateMap } from '@gospa/runtime'

// Get or create a state instance
const state = __GOSPA__.getState('counter')
if (!state) return

// Get current value
const count = state.get('count')

// Set new value
state.set('count', count + 1)

// Or use the convenience method directly
__GOSPA__.setState('counter', 'count', newValue)
```

### DOM Bindings

The runtime automatically handles DOM bindings using `data-bind`:

```html
<!-- Bind text content -->
<p data-bind="count">0</p>

<!-- Two-way binding with input -->
<input data-bind:value="inputValue" />
```

Event handlers use standard onclick:

```html
<button onclick="...">Click</button>
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
| `gospa add <feature>` | Add plugins (tailwind, postcss, image, validation, seo, auth) |
| `gospa prune` | Remove unused state |
| `gospa clean` | Remove build artifacts |
| `gospa watch` | Build and watch for changes |

## Next Steps

1. **[Configuration Reference](./04-api-reference/02-configuration.md)** - All configuration options
2. **[Client Runtime API](./03-features/02-runtime-api.md)** - Complete TypeScript API
3. **[Core Concepts - State](./02-core-concepts/03-state.md)** - Go reactive primitives
4. **[CLI Reference](./04-api-reference/03-cli.md)** - All CLI commands
5. **[Client Runtime](./03-features/01-client-runtime.md)** - Choose the right runtime

## Common Patterns

### Counter Component

```go
// routes/counter.templ
package routes

templ CounterPage() {
	<div 
		data-gospa-component="counter"
		data-gospa-state='{"count":0}'
	>
		<h2>Counter</h2>
		<p>Count: <span data-bind="count">0</span></p>
		<button 
			onclick="var r=__GOSPA__.getState('counter','count');r&&__GOSPA__.setState('counter','count',r.get()-1)"
		>
			-
		</button>
		<button 
			onclick="var r=__GOSPA__.getState('counter','count');r&&__GOSPA__.setState('counter','count',r.get()+1)"
		>
			+
		</button>
	</div>
}
```

### Todo List

```go
// routes/todos.templ
package routes

templ TodosPage() {
	<div 
		data-gospa-component="todos"
		data-gospa-state='{"todos":[],"newTodo":""}'
	>
		<h2>Todos</h2>
		<input 
			type="text" 
			data-bind:value="newTodo"
			placeholder="Add todo..."
		/>
		<button 
			onclick="var s=__GOSPA__.getState('todos');if(!s)return;var v=s.get('newTodo');if(!v)return;var t=s.get('todos')||[];s.set('todos',[...t,{id:Date.now(),text:v,completed:false}]);s.set('newTodo','')"
		>
			Add
		</button>
		<ul data-bind="list:todos">
			<li>{ todo.text }</li>
		</ul>
	</div>
}
```

### Form Handling

```go
// routes/contact.templ
package routes

templ ContactPage() {
	<form 
		data-gospa-component="contact-form"
		data-gospa-state='{"name":"","email":"","message":""}'
	>
		<input 
			type="text" 
			name="name"
			data-bind:value="name"
			required
		/>
		<input 
			type="email" 
			name="email"
			data-bind:value="email"
			required
		/>
		<textarea 
			name="message"
			data-bind:value="message"
		></textarea>
		<button 
			type="submit"
			onclick="console.log('Form submitted')"
		>
			Send
		</button>
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
