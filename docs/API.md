# GoSPA API Reference

## Table of Contents

- [State Package](#state-package)
  - [Rune[T]](#runet)
  - [Derived[T]]#derivedt)
  - [Effect](#effect)
  - [Batch](#batch)
  - [Serialization](#serialization)
- [Routing Package](#routing-package)
  - [Auto Router](#auto-router)
  - [Manual Router](#manual-router)
  - [Route Parameters](#route-parameters)
  - [Layouts](#layouts)
- [Fiber Package](#fiber-package)
  - [Middleware](#middleware)
  - [WebSocket Client](#websocket-client)
- [Templ Package](#templ-package)
  - [Reactive Bindings](#reactive-bindings)
  - [Event Handlers](#event-handlers)
- [Client Runtime](#client-runtime)
  - [Rune Class](#rune-class)
  - [Derived Class](#derived-class)
  - [Effect Class](#effect-class)
  - [StateMap](#statemap)
  - [WebSocket Client](#websocket-client-1)
  - [DOM Binding](#dom-binding)
  - [Component Initialization](#component-initialization)
  - [Partial Hydration](#partial-hydration)
  - [Transitions](#transitions)
  - [Optimistic UI & State HMR](#optimistic-ui--state-hmr)

---

## State Package

`github.com/aydenstechdungeon/gospa/state`

### Rune[T]

The core reactive primitive that holds a value and notifies subscribers on changes.

#### Functions

```go
// Create a new Rune with an initial value
func NewRune[T any](initial T) *Rune[T]
```

#### Methods

```go
// Get the current value
func (r *Rune[T]) Get() T

// GetAny returns the current value as any (implements Observable)
func (r *Rune[T]) GetAny() any

// Set a new value and notify subscribers
func (r *Rune[T]) Set(value T)

// Subscribe to value changes
// Returns an unsubscribe function
func (r *Rune[T]) Subscribe(fn func(T)) func()

// SubscribeAny allows subscribing without knowing the type (implements Observable)
func (r *Rune[T]) SubscribeAny(fn func(any)) func()

// Update the value using a function
func (r *Rune[T]) Update(fn func(T) T)
```

#### Interface

Runes implement the `Observable` interface for generic state stores:

```go
type Observable interface {
    GetAny() any
    SubscribeAny(func(any)) func()
}
```

#### Example

```go
count := state.NewRune(0)

// Get value
fmt.Println(count.Get()) // 0

// Set value
count.Set(5)

// Subscribe to changes
unsubscribe := count.Subscribe(func(newValue int) {
    fmt.Println("Count changed to:", newValue)
})
defer unsubscribe()

// Update with function
count.Update(func(v int) int { return v + 1 })
```

---

### Derived[T]

Computed state that automatically recalculates when dependencies change.

#### Functions

```go
// Create a new Derived value
func NewDerived[T any](compute func() T) *Derived[T]
```

#### Methods

```go
// Get the current computed value
func (d *Derived[T]) Get() T

// Subscribe to computed value changes
func (d *Derived[T]) Subscribe(fn func(T)) func()
```

#### Example

```go
count := state.NewRune(5)
doubled := state.NewDerived(func() int {
    return count.Get() * 2
})

fmt.Println(doubled.Get()) // 10

count.Set(10)
fmt.Println(doubled.Get()) // 20
```

---

### Effect

Side effects that run when dependencies change.

#### Functions

```go
// Create a new effect
// Returns a cleanup function to stop the effect
func NewEffect(fn func() func()) func()
```

#### Example

```go
count := state.NewRune(0)

cleanup := state.NewEffect(func() func() {
    fmt.Println("Count is:", count.Get())
    return func() {
        fmt.Println("Cleaning up...")
    }
})
defer cleanup()

count.Set(1) // Prints: "Count is: 1"
```

---

### Batch

Batch multiple state updates into a single notification cycle.

#### Functions

```go
// Execute fn with batched updates
func Batch(fn func())
```

#### Example

```go
count := state.NewRune(0)
name := state.NewRune("")

// Both changes trigger only one notification each
state.Batch(func() {
    count.Set(1)
    name.Set("Alice")
})
```

---

### Serialization

Types and functions for serializing state to JSON for client transmission.

#### Types

```go
// Map of component states
type StateMap map[string]interface{}

// Snapshot of state at a point in time
type StateSnapshot struct {
    ComponentID string
    State       StateMap
    Timestamp   int64
}

// Message for client communication
type StateMessage struct {
    Type        string      `json:"type"`
    ComponentID string      `json:"componentId"`
    State       interface{} `json:"state"`
    Timestamp   int64       `json:"timestamp"`
}
```

#### Functions

```go
// Create a new state map
func NewStateMap() *StateMap

// Create an initialization message
func NewInitMessage(componentID string, state interface{}) *StateMessage

// Create a sync message
func NewSyncMessage(componentID string, state interface{}) *StateMessage

// Create a state snapshot
func NewSnapshot(componentID string, state StateMap) *StateSnapshot
```

---

## Routing Package

`github.com/aydenstechdungeon/gospa/routing`

### Auto Router

Automatically scans and registers routes from `.templ` files using `gospa-gen` (which parses Go files and spits out a `routes_registration.go` alongside an optionally typed `types.d.ts` file based on Go structs found in the package).

#### Functions

```go
// Create a new auto router
func NewAutoRouter(routesDir string) *AutoRouter

// Scan the routes directory and build route tree
func (ar *AutoRouter) Scan() error

// Get all discovered routes
func (ar *AutoRouter) GetRoutes() []*Route

// Get route tree for debugging
func (ar *AutoRouter) GetRouteTree() *RouteNode
```

#### Route Structure

```go
type Route struct {
    Path       string            // URL path
    FilePath   string            // Source file path
    Params     []string          // Dynamic parameters
    IsCatchAll bool              // Catch-all route flag
    Meta       map[string]string // Route metadata
}
```

#### File-Based Routing Convention

```
routes/
├── root_layout.templ    → Base HTML shell
├── page.templ           → /
├── about/
│   └── page.templ       → /about
├── blog/
│   ├── layout.templ     → Layout for /blog/*
│   └── [id]/
│       └── page.templ   → /blog/:id
└── posts/
    └── [...rest]/
        └── page.templ   → /posts/* (catch-all)
```

---

### Manual Router

For advanced use cases requiring manual route registration.

#### Functions

```go
// Create a new manual router
func NewManualRouter() *ManualRouter

// Register HTTP handlers
func (mr *ManualRouter) GET(path string, handler Handler, middleware ...Middleware) *ManualRoute
func (mr *ManualRouter) POST(path string, handler Handler, middleware ...Middleware) *ManualRoute
func (mr *ManualRouter) PUT(path string, handler Handler, middleware ...Middleware) *ManualRoute
func (mr *ManualRouter) DELETE(path string, handler Handler, middleware ...Middleware) *ManualRoute
func (mr *ManualRouter) PATCH(path string, handler Handler, middleware ...Middleware) *ManualRoute

// Get all registered routes
func (mr *ManualRouter) GetRoutes() []*ManualRoute
```

#### Handler Type

```go
type Handler func(c *fiber.Ctx, params Params) error
```

#### Middleware Type

```go
type Middleware func(Handler) Handler
```

#### ManualRoute Structure

```go
type ManualRoute struct {
    Method      string
    Path        string
    Handler     Handler
    Middleware  []Middleware
    Params      []string
}
```

#### Example

```go
router := routing.NewManualRouter()

router.GET("/", func(c *fiber.Ctx, params routing.Params) error {
    return c.SendString("Hello World")
})

router.GET("/users/:id", func(c *fiber.Ctx, params routing.Params) error {
    id := params.Get("id")
    return c.SendString("User: " + id)
})

// Register with Fiber
for _, route := range router.GetRoutes() {
    app.Add(route.Method, route.Path, func(c *fiber.Ctx) error {
        params := routing.ExtractParams(c, route.Params)
        return route.Handler(c, params)
    })
}
```

---

### Route Parameters

Extract and work with URL parameters.

#### Types

```go
type Params map[string]string
```

#### Functions

```go
// Get a parameter value
func (p Params) Get(key string) string

// Get with default value
func (p Params) GetDefault(key, defaultValue string) string

// Extract params from Fiber context
func ExtractParams(c *fiber.Ctx, paramKeys []string) Params
```

---

### Layouts

Manage nested layouts for routes.

#### Functions

```go
// Create a new layout manager
func NewLayoutManager() *LayoutManager

// Register a layout for a path prefix
func (lm *LayoutManager) Register(path string, layout Layout)

// Get layouts for a path (in order of specificity)
func (lm *LayoutManager) GetLayouts(path string) []Layout

// Register and Get Root Layouts (Global Base HTML Shell)
func RegisterRootLayout(fn LayoutFunc)
func GetRootLayout() LayoutFunc
```

#### Layout Interface

```go
type Layout interface {
    Render(content templ.Component, params Params) templ.Component
}
```

---

## Fiber Package

`github.com/aydenstechdungeon/gospa/fiber`

### Middleware

#### RuntimeMiddleware

Injects the GoSPA client runtime into HTML responses.

```go
func RuntimeMiddleware() fiber.Handler
```

#### Example

```go
app := fiber.New()
app.Use(gospafiber.RuntimeMiddleware())
```

The middleware:
1. Serves runtime JavaScript at `/_gospa/runtime.js`
2. Injects runtime script tag into HTML responses
3. Adds GoSPA headers to all responses

---

### WebSocket Client

Manage WebSocket connections for real-time state sync.

#### Functions

```go
// Create a new WebSocket client
func NewWSClient(id string, conn *websocket.Conn) *WSClient
```

#### WSClient Structure

```go
type WSClient struct {
    ID     string
    State  *state.StateMap
    Conn   *websocket.Conn
    mu     sync.RWMutex
}
```

#### Methods

```go
// Send a message to the client
func (c *WSClient) Send(msg *state.StateMessage) error

// Close the connection
func (c *WSClient) Close() error

// Get current state
func (c *WSClient) GetState() *state.StateMap

// Update state
func (c *WSClient) SetState(s *state.StateMap)
```

#### Example

```go
import "github.com/gofiber/websocket/v2"

app.Get("/ws", websocket.New(func(c *websocket.Conn) {
    client := gospafiber.NewWSClient(generateID(), c)
    defer client.Close()
    
    // Send initial state
    client.Send(state.NewInitMessage("app", initialState))
    
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

---

## Templ Package

`github.com/aydenstechdungeon/gospa/templ`

### Reactive Bindings

Create reactive bindings in Templ templates.

#### Functions

```go
// Create a data binding attribute
func Bind(componentID, key string) templ.Attributes

// Create multiple bindings
func BindAll(componentID string, keys ...string) templ.Attributes
```

#### Example

```templ
package main

import "github.com/aydenstechdungeon/gospa/templ"

templ Counter(count int) {
    <div data-component="counter">
        <span { templ.Bind("counter", "count")... }>{ count }</span>
        <button { templ.On("click", "counter", "increment")... }>+</button>
    </div>
}
```

---

### Event Handlers

Attach event handlers to elements.

#### Functions

```go
// Create an event handler attribute
func On(event, componentID, handler string) templ.Attributes

// Create an event handler with options
func OnWithOpts(event, componentID, handler string, opts EventOptions) templ.Attributes
```

#### EventOptions

```go
type EventOptions struct {
    PreventDefault bool
    StopPropagation bool
    Debounce       int // milliseconds
    Throttle       int // milliseconds
}
```

#### Example

```templ
templ Button() {
    <button { templ.On("click", "counter", "increment")... }>
        Click me
    </button>
    
    <form { templ.OnWithOpts("submit", "form", "handleSubmit", templ.EventOptions{
        PreventDefault: true,
    })... }>
        <input type="text" name="value" />
        <button type="submit">Submit</button>
    </form>
}
```

---

## Client Runtime

The JavaScript runtime provides client-side reactivity.

### Rune Class

```javascript
// Create a reactive value
const count = new GoSPA.Rune(0);

// Get value
count.get(); // 0

// Set value
count.set(5);

// Subscribe to changes
const unsubscribe = count.subscribe((newValue) => {
    console.log('Count:', newValue);
});
unsubscribe(); // Stop listening

// Update with function
count.update((v) => v + 1);
```

---

### Derived Class

```javascript
const count = new GoSPA.Rune(5);
const doubled = new GoSPA.Derived(() => count.get() * 2);

doubled.get(); // 10

count.set(10);
doubled.get(); // 20
```

---

### Effect Class

```javascript
const count = new GoSPA.Rune(0);

const cleanup = new GoSPA.Effect(() => {
    console.log('Count:', count.get());
    return () => {
        console.log('Cleanup');
    };
});

cleanup(); // Stop effect
```

---

### StateMap

```javascript
// Create a state map for a component
const state = new GoSPA.StateMap({
    count: 0,
    name: 'Alice'
});

// Get/set values
state.get('count');
state.set('count', 5);

// Subscribe to changes
state.subscribe('count', (newValue) => {
    console.log('Count changed:', newValue);
});
```

---

### WebSocket Client

```javascript
// Connect to server
const client = new GoSPA.WSClient('ws://localhost:3000/ws');

// Handle connection
client.onConnect(() => {
    console.log('Connected');
});

// Handle messages
client.onMessage((msg) => {
    console.log('Message:', msg);
});

// Send message
client.send({
    type: 'sync',
    componentId: 'counter',
    state: { count: 5 }
});

// Disconnect
client.disconnect();
```

---

### DOM Binding

```javascript
// Bind element to state
GoSPA.bind('element-id', rune);

// Bind multiple elements
GoSPA.bindAll({
    'count-display': countRune,
    'name-input': nameRune
});

// Create two-way binding for inputs
GoSPA.bindInput('input-id', rune);
```

---

### Component Initialization

```javascript
// Initialize component with server state
GoSPA.init('component-id', {
    count: 0,
    name: 'Alice'
});

// Register component handler
GoSPA.registerHandler('increment', (state, event) => {
    state.update('count', (v) => v + 1);
});
```

---

### Partial Hydration

To opt-out of reactivity and save memory on large static components, use `data-gospa-static`.

```html
<div data-gospa-static>
    <!-- Anything inside here is completely ignored by the client runtime -->
    <p>Static Content completely unaffected by bindings or event setups!</p>
</div>
```

---

### Transitions

Provide Svelte-like enter and exit animations to components via generic datasets (`fade`, `fly`, `slide`).

```html
<!-- Single transition -->
<div data-transition="fade" data-transition-params='{"duration": 300}'>Content</div>

<!-- Separated in and out transitions -->
<div data-transition-in="fly" data-transition-params='{"y": 30}' data-transition-out="slide">Content</div>
```

---

### Optimistic UI & State HMR

GoSPA's `syncedRune` and generic WebSockets implement **Optimistic Rollbacks** (attempting to set a state locally but instantly rolling back to the previous snapshot when rejected/dropped network). In development modes, server restarts trigger an automatic full-state **HMR Patch** across active sockets to prevent losing client scope state.
