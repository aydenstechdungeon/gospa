# GoSPA API Reference

## Table of Contents

- [State Package](#state-package)
- [Routing Package](#routing-package)
- [Fiber Package](#fiber-package)
- [Templ Package](#templ-package)
- [Client Runtime](#client-runtime)

---

## State Package

`github.com/aydenstechdungeon/gospa/state`

### Rune[T]

Core reactive primitive. Holds a value and notifies subscribers on changes.

```go
// Create
rune := state.NewRune[T](initial T)

// Read
rune.Get() T
rune.GetAny() any  // for Observable interface

// Write
rune.Set(value T)
rune.Update(fn func(T) T)

// Subscribe
unsubscribe := rune.Subscribe(func(newValue T) {})
unsubscribe := rune.SubscribeAny(func(newValue any) {})
```

**Interface**

```go
type Observable interface {
    GetAny() any
    SubscribeAny(func(any)) func()
}
```

**Example**

```go
count := state.NewRune(0)

// Read
fmt.Println(count.Get()) // 0

// Write
count.Set(5)
count.Update(func(v int) int { return v + 1 })

// React
unsub := count.Subscribe(func(v int) {
    fmt.Println("Count:", v)
})
defer unsub()
```

---

### Derived[T]

Computed state that recalculates when dependencies change.

```go
// Create
derived := state.NewDerived[T](compute func() T)

// Read
derived.Get() T

// Subscribe
unsubscribe := derived.Subscribe(func(T) {})
```

**Example**

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

```go
// Create - returns cleanup function
cleanup := state.NewEffect(func() func() {
    // effect logic
    return func() {
        // cleanup logic
    }
})
```

**Example**

```go
count := state.NewRune(0)

cleanup := state.NewEffect(func() func() {
    fmt.Println("Count is:", count.Get())
    return func() {
        fmt.Println("Cleaning up")
    }
})
defer cleanup()

count.Set(1) // Prints: "Count is: 1"
```

---

### Batch

Batch multiple updates into single notification cycle.

```go
state.Batch(func() {
    count.Set(1)
    name.Set("Alice")
})
```

---

### Serialization

```go
// Types
type StateMap map[string]interface{}
type StateSnapshot struct {
    ComponentID string
    State       StateMap
    Timestamp   int64
}
type StateMessage struct {
    Type        string      `json:"type"`
    ComponentID string      `json:"componentId"`
    State       interface{} `json:"state"`
    Timestamp   int64       `json:"timestamp"`
}

// Constructors
func NewStateMap() *StateMap
func NewInitMessage(componentID string, state interface{}) *StateMessage
func NewSyncMessage(componentID string, state interface{}) *StateMessage
func NewSnapshot(componentID string, state StateMap) *StateSnapshot
```

---

## Routing Package

`github.com/aydenstechdungeon/gospa/routing`

### Auto Router

Scans `.templ` files and builds route tree.

```go
router := routing.NewAutoRouter(routesDir string)
router.Scan() error
routes := router.GetRoutes() []*Route
tree := router.GetRouteTree() *RouteNode
```

**Route Structure**

```go
type Route struct {
    Path       string
    FilePath   string
    Params     []string
    IsCatchAll bool
    Meta       map[string]string
}
```

**File Convention**

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
        └ page.templ   → /posts/*
```

---

### Manual Router

For programmatic route registration.

```go
router := routing.NewManualRouter()

// Register routes
router.GET(path string, handler Handler, middleware ...Middleware)
router.POST(path string, handler Handler, middleware ...Middleware)
router.PUT(path string, handler Handler, middleware ...Middleware)
router.DELETE(path string, handler Handler, middleware ...Middleware)
router.PATCH(path string, handler Handler, middleware ...Middleware)

// Get all routes
routes := router.GetRoutes() []*ManualRoute
```

**Handler Type**

```go
type Handler func(c *fiber.Ctx, params Params) error
type Middleware func(Handler) Handler
```

**Example**

```go
router := routing.NewManualRouter()

router.GET("/", func(c *fiber.Ctx, params routing.Params) error {
    return c.SendString("Hello")
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

```go
type Params map[string]string

func (p Params) Get(key string) string
func (p Params) GetDefault(key, defaultValue string) string
func ExtractParams(c *fiber.Ctx, paramKeys []string) Params
```

---

### Layouts

```go
// Create layout manager
lm := routing.NewLayoutManager()

// Register layout for path prefix
lm.Register(path string, layout Layout)

// Get layouts for path
layouts := lm.GetLayouts(path string) []Layout

// Root layout
routing.RegisterRootLayout(fn LayoutFunc)
routing.GetRootLayout() LayoutFunc
```

**Layout Interface**

```go
type Layout interface {
    Render(content templ.Component, params Params) templ.Component
}
```

---

## Fiber Package

`github.com/aydenstechdungeon/gospa/fiber`

### Middleware

```go
// Inject GoSPA runtime into HTML responses
app.Use(gospafiber.RuntimeMiddleware())
```

The middleware:
1. Serves runtime JavaScript at `/_gospa/runtime.js`
2. Injects runtime script tag into HTML responses
3. Adds GoSPA headers to all responses

---

### WebSocket Hub

```go
// Create hub
hub := fiber.NewWSHub()

// Start hub
go hub.Run()

// Broadcast to all clients
fiber.BroadcastState(hub, key string, value interface{}) error
```

---

### WebSocket Client

```go
// Create client
client := gospafiber.NewWSClient(id string, conn *websocket.Conn)

// Methods
client.Send(msg *state.StateMessage) error
client.Close() error
client.GetState() *state.StateMap
client.SetState(s *state.StateMap)
```

**Example**

```go
import "github.com/gofiber/websocket/v2"

app.Get("/ws", websocket.New(func(c *websocket.Conn) {
    client := gospafiber.NewWSClient(generateID(), c)
    defer client.Close()
    
    // Send initial state
    client.Send(state.NewInitMessage("app", initialState))
    
    // Handle incoming
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

### Action Handlers

Register server-side action handlers for synced state.

```go
// Register action handler
fiber.RegisterActionHandler(name string, handler func(*WSClient, json.RawMessage))

// Register connect handler
fiber.RegisterOnConnectHandler(handler func(*WSClient))
```

**Example**

```go
fiber.RegisterActionHandler("increment", func(client *fiber.WSClient, payload json.RawMessage) {
    GlobalCounter.Count++
    fiber.BroadcastState(hub, "count", GlobalCounter.Count)
})
```

---

## Templ Package

`github.com/aydenstechdungeon/gospa/templ`

### Reactive Bindings

```go
// Create data binding attribute
templ.Bind(componentID, key string) templ.Attributes

// Create multiple bindings
templ.BindAll(componentID string, keys ...string) templ.Attributes
```

**Example**

```templ
templ Counter(count int) {
    <div data-component="counter">
        <span { templ.Bind("counter", "count")... }>{ count }</span>
        <button { templ.On("click", "counter", "increment")... }>+</button>
    </div>
}
```

---

### Event Handlers

```go
// Create event handler attribute
templ.On(event, componentID, handler string) templ.Attributes

// With options
templ.OnWithOpts(event, componentID, handler string, opts EventOptions) templ.Attributes

type EventOptions struct {
    PreventDefault  bool
    StopPropagation bool
    Debounce        int // milliseconds
    Throttle        int // milliseconds
}
```

**Example**

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

### Rune Class

```javascript
const count = new GoSPA.Rune(0)

count.get()           // 0
count.set(5)
count.update(v => v + 1)

const unsub = count.subscribe(v => {
    console.log('Count:', v)
})
unsub() // stop listening
```

---

### Derived Class

```javascript
const count = new GoSPA.Rune(5)
const doubled = new GoSPA.Derived(() => count.get() * 2)

doubled.get() // 10
count.set(10)
doubled.get() // 20
```

---

### Effect Class

```javascript
const count = new GoSPA.Rune(0)

const cleanup = new GoSPA.Effect(() => {
    console.log('Count:', count.get())
    return () => console.log('Cleanup')
})

cleanup() // stop effect
```

---

### StateMap

```javascript
const state = new GoSPA.StateMap({
    count: 0,
    name: 'Alice'
})

state.get('count')
state.set('count', 5)

state.subscribe('count', (newValue) => {
    console.log('Count changed:', newValue)
})
```

---

### WebSocket Client

```javascript
const client = new GoSPA.WSClient('ws://localhost:3000/ws')

client.onConnect(() => {
    console.log('Connected')
})

client.onMessage((msg) => {
    console.log('Message:', msg)
})

client.send({
    type: 'sync',
    componentId: 'counter',
    state: { count: 5 }
})

client.disconnect()
```

---

### DOM Binding

```javascript
// Bind element to state
GoSPA.bind('element-id', rune)

// Bind multiple
GoSPA.bindAll({
    'count-display': countRune,
    'name-input': nameRune
})

// Two-way binding for inputs
GoSPA.bindInput('input-id', rune)
```

---

### Component Initialization

```javascript
// Initialize with server state
GoSPA.init('component-id', {
    count: 0,
    name: 'Alice'
})

// Register handler
GoSPA.registerHandler('increment', (state, event) => {
    state.update('count', v => v + 1)
})
```

---

### Partial Hydration

```html
<div data-gospa-static>
    <!-- No bindings or event listeners -->
    <p>Static content</p>
</div>
```

---

### Transitions

```html
<!-- Single transition -->
<div data-transition="fade" data-transition-params='{"duration": 300}'>
    Content
</div>

<!-- Separate in/out -->
<div data-transition-in="fly" data-transition-out="slide">
    Content
</div>
```

Available transitions: `fade`, `fly`, `slide`

---

### Local State Mode

Components with client-only state (no server sync).

```html
<div data-gospa-component="counter" data-gospa-local>
    <span data-bind="count">0</span>
    <button data-on="click:increment">+</button>
</div>
```

Local state handlers run entirely in the browser without WebSocket communication.
