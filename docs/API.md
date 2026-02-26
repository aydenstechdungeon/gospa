# GoSPA API Reference

## Table of Contents

- [GoSPA Package](#gospa-package)
- [State Package](#state-package)
- [Routing Package](#routing-package)
- [Fiber Package](#fiber-package)
- [Templ Package](#templ-package)
- [Client Runtime](#client-runtime)

---

## GoSPA Package

`github.com/aydenstechdungeon/gospa`

### App

The main GoSPA application.

```go
// Create new app
app := gospa.New(config gospa.Config)

// Start server
err := app.Run(":3000")
err := app.RunTLS(":443", "cert.pem", "key.pem")

// Graceful shutdown
err := app.Shutdown()

// Routing & Middleware
app.Scan()            // Scans routes directory
app.RegisterRoutes()  // Registers Fiber routes
app.Use(middleware)   // Adds global middleware
group := app.Group("/api") // Creates route group

// Access internals
hub := app.GetHub()
router := app.GetRouter()
fiberApp := app.GetFiber()

// Broadcast to all WebSocket clients
app.Broadcast([]byte("message"))
err := app.BroadcastState("key", value)

// Add routes manually
app.Get("/path", handler)
app.Post("/path", handler)
app.Put("/path", handler)
app.Delete("/path", handler)

// Static files
app.Static("/static", "./public")
```

### `Config`
The `Config` struct defines the application's configuration.

```go
type Config struct {
	// RoutesDir is the directory containing route files.
	RoutesDir string
	// RoutesFS is the filesystem containing route files (optional).
	RoutesFS fs.FS
	// DevMode enables development features.
	DevMode bool
	// RuntimeScript is the path to the client runtime script.
	RuntimeScript string
	// StaticDir is the directory for static files.
	StaticDir string
	// StaticPrefix is the URL prefix for static files.
	StaticPrefix string
	// AppName is the application name.
	AppName string
	// DefaultState is the initial state for new sessions.
	DefaultState map[string]interface{}
	// EnableWebSocket enables WebSocket support.
	EnableWebSocket bool
	// WebSocketPath is the WebSocket endpoint path.
	WebSocketPath string
	// WebSocketMiddleware allows injecting middleware before WebSocket upgrade.
	WebSocketMiddleware fiberpkg.Handler

	// Performance Options
	CompressState  bool // Gzip-compress outbound state payloads; client must decompress via DecompressionStream
	StateDiffing   bool // Only send changed state keys (patch messages) after the initial full snapshot
	CacheTemplates bool // Cache compiled templates
	SimpleRuntime  bool // Use lightweight runtime without DOMPurify

	// WebSocket Options
	WSReconnectDelay time.Duration // Initial reconnect delay (default 1s), passed to client runtime
	WSMaxReconnect   int           // Max reconnect attempts (default 10), passed to client runtime
	WSHeartbeat      time.Duration // Heartbeat ping interval (default 30s), passed to client runtime

	// Hydration Options
	HydrationMode    string // "immediate" | "lazy" | "visible" | "idle" (default: "immediate")
	HydrationTimeout int    // ms before force hydrate

	// Serialization Options
	StateSerializer   StateSerializerFunc // Custom serialization for outbound state
	StateDeserializer StateDeserializerFunc // Custom deserialization for inbound state

	// Routing Options
	DisableSPA bool // Disable SPA navigation completely
	SSR        bool // NOTE: planned, not yet implemented

	// Remote Action Options
	MaxRequestBodySize int    // Maximum allowed size for remote action request bodies
	RemotePrefix       string // Prefix for remote action endpoints

	// Security Options
	AllowedOrigins []string // Allowed CORS origins
	EnableCSRF     bool     // Enable CSRF (requires CSRFSetTokenMiddleware + CSRFTokenMiddleware)

	// Cache Options
	SSGCacheMaxEntries int // Max SSG cache entries (default 500; -1 = unbounded)

	// Distributed & Scaling Options
	Prefork bool          // Enables Fiber Prefork. WARNING: If true, you MUST use external Storage/PubSub to share state.
	Storage store.Storage // External Key-Value store (e.g., Redis) for Session and ClientState. Default is memory.
	PubSub  store.PubSub  // External messaging broker (e.g., Redis PubSub) for WebSocket broadcasting. Default is memory.
}
```

#### Key Options:
- `RoutesFS`: Allows embedding routes using `go:embed`.
- `CompressState`: Enables gzip+base64 compression of outbound WebSocket state payloads. Client decodes with the `DecompressionStream` browser API.
- `StateDiffing`: Sends only changed state keys as `"patch"` WebSocket messages after the initial full snapshot — reduces bandwidth for large states.
- `StateSerializer` / `StateDeserializer`: Custom hooks replacing JSON for WebSocket state encoding/decoding.
- `SimpleRuntime`: Reduces client bundle size by ~6KB by removing DOMPurify. ⚠️ Security Risk: Only enable SimpleRuntime for fully trusted content.
- `SimpleRuntimeSVGs`: Allows SVG elements in the simple runtime sanitizer. ⚠️ Security Risk: Only enable SimpleRuntimeSVGs for fully trusted content.
- `HydrationMode`: Controls when components become interactive. Values: `"immediate"` (default), `"lazy"`, `"visible"`, `"idle"`.
- `WSReconnectDelay`: Initial reconnect delay passed to the client WebSocket runtime. Default: 1s.
- `WSMaxReconnect`: Maximum reconnect attempts passed to the client. Default: 10.
- `WSHeartbeat`: Heartbeat ping interval passed to the client. Default: 30s.
- `SSR`: **Planned** — not yet implemented.
- `EnableCSRF`: Enables CSRF protection. Must wire up **both** `CSRFSetTokenMiddleware()` (issues cookie) **and** `CSRFTokenMiddleware()` (validates).
- `SSGCacheMaxEntries`: Caps the SSG page cache with FIFO eviction. Default 500.
- `Prefork`, `Storage`, `PubSub`: Used for horizontal scaling. See `store/redis` for the Redis implementation.

```go
// Default configuration
config := gospa.DefaultConfig()

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
rune.ID() string   // returns unique internal ID
// NOTE: rune.peek() does NOT exist — there is no non-tracking read on the server-side Rune.

// Write
rune.Set(value T)
rune.Update(fn func(T) T)
rune.SetAny(value any) error // for Settable interface

// Subscribe
unsubscribe := rune.Subscribe(func(newValue T) {})
unsubscribe := rune.SubscribeAny(func(newValue any) {})

// Serialization
data, err := rune.MarshalJSON()
```

**Interfaces**

```go
type Observable interface {
    GetAny() any
    SubscribeAny(func(any)) func()
}

type Settable interface {
    SetAny(value any) error
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

// Lifecycle
derived.Dispose()
```

**Helper Functions**

```go
// Single dependency
d := state.DerivedFrom(rune, func(v T) U { ... })

// Two dependencies
d := state.Derived2(rune1, rune2, func(v1 T1, v2 T2) U { ... })

// Three dependencies
d := state.Derived3(rune1, rune2, rune3, func(v1 T1, v2 T2, v3 T3) U { ... })
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

**Helper Functions**

```go
// Single dependency
cleanup := state.EffectOn(rune, func(v T) { ... })

// Watch multiple
cleanup := state.Watch(rune1, rune2, func(v1 T1, v2 T2) { ... })
cleanup := state.Watch3(rune1, rune2, rune3, func(v1 T1, v2 T2, v3 T3) { ... })
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

### StateMap

Collection of named reactive values.

```go
// Create
sm := state.NewStateMap()

// Add reactive value
sm.Add(key string, observable Observable)
sm.AddAny(key string, value any) error // Adds plain value as auto-rune

// Get value
obs, ok := sm.Get(key string)

// Remove
sm.Remove(key string)

// Iteration
sm.ForEach(func(key string, value Observable) { ... })

// Conversion
m := sm.ToMap() // returns map[string]any

// Serialization
json, err := sm.ToJSON()
data, err := sm.MarshalJSON()
sm.FromJSON(data []byte)

// Diff
diff := sm.Diff(other *StateMap) *StateDiff

// OnChange callback
sm.OnChange = func(key string, value any) { ... }
```

---

### StateValidator

Validates reactive state updates.

```go
validator := state.NewStateValidator()

// Register rules
validator.AddValidator("count", func(v any) error {
    if v.(int) < 0 { return errors.New("must be positive") }
    return nil
})

// Validate
err := validator.Validate("count", -1)
err := validator.ValidateAll(map[string]any{"count": 5})
```

---

### Serialization Types

```go
type StateMessage struct {
    Type        string      `json:"type"`
    ComponentID string      `json:"componentId"`
    State       interface{} `json:"state"`
    Timestamp   int64       `json:"timestamp"`
}

type StateSnapshot struct {
    ComponentID string
    State       StateMap
    Timestamp   int64
}

type StateDiff struct {
    Added   map[string]interface{}
    Removed map[string]interface{}
    Changed map[string]interface{}
}

// Constructors
msg := state.NewInitMessage(componentID string, state interface{})
msg := state.NewSyncMessage(componentID string, state interface{})
snapshot := state.NewSnapshot(componentID string, state StateMap)
```

---

## Routing Package

`github.com/aydenstechdungeon/gospa/routing`

### Router

File-based router that scans `.templ` files.

```go
// Create from directory
router := routing.NewRouter(routesDir string)

// Create from filesystem (hybrid approach)
router := routing.NewRouter(routesFS fs.FS)

// Scan routes directory
err := router.Scan()

// Match route
route, params := router.Match(path string)

// Match with layout chain
route, params, layouts := router.MatchWithLayout(path string)

// Get all routes
routes := router.GetRoutes()

// Get page routes only
pages := router.GetPages()

// Resolve layout chain for route
layouts := router.ResolveLayoutChain(route *Route)
```

**Route Structure**

```go
type Route struct {
    Path       string            // URL path
    FilePath   string            // Source .templ file
    Params     []string          // Dynamic param names
    IsCatchAll bool              // [...rest] route
    Type       RouteType         // page, layout, error, api
    Meta       map[string]string // Custom metadata
}

type RouteType int
const (
    RouteTypePage RouteType = iota
    RouteTypeLayout
    RouteTypeError
    RouteTypeAPI
)
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
routes := router.GetRoutes()

// Create route group
group := router.Group(prefix string, middleware ...Middleware)
group.GET("/subpath", handler)

// Register all routes to Fiber
router.RegisterToFiber(fiberApp *fiber.App)
```

**Handler Type**

```go
type Handler func(c *fiber.Ctx) error
type Middleware func(c *fiber.Ctx) error
```

---

### Params

Route parameter extraction and typed access.

```go
type Params map[string]string

// Basic access
value := params.Get("id")
value := params.GetDefault("id", "default")
exists := params.Has("id")

// Writing
params.Set("id", "123")

// Typed access (returns (T, error))
intVal, err := params.Int("count")
int64Val, err := params.Int64("id")
floatVal, err := params.Float64("price")
boolVal, err := params.Bool("active")

// Slice (for catch-all params, splits by '/')
sliceVal := params.Slice("path")

// Utility functions
params := routing.ExtractParams(c *fiber.Ctx, paramKeys []string)
queryParams := routing.QueryParams(c *fiber.Ctx)
```

---

### Route Registry

Register page and layout components.

```go
// Defined types
// type ComponentFunc func(props map[string]interface{}) templ.Component
// type LayoutFunc func(children templ.Component, props map[string]interface{}) templ.Component
// type SlotFunc func(props map[string]interface{}) templ.Component

// Register page component
routing.RegisterPage(path string, fn ComponentFunc)
routing.RegisterPageWithOptions(path string, fn ComponentFunc, opts RouteOptions)

// Register layout component
routing.RegisterLayout(path string, fn LayoutFunc)

// Register root layout
routing.RegisterRootLayout(fn LayoutFunc)

// Get registered components
pageFunc := routing.GetPage(path string)
layoutFunc := routing.GetLayout(path string)
rootLayoutFunc := routing.GetRootLayout()

// Remote actions
routing.RegisterRemoteAction(name string, fn RemoteActionFunc)
fn, ok := routing.GetRemoteAction(name string)

// PPR slot registration
routing.RegisterSlot(pagePath string, slotName string, fn SlotFunc)
slotFn := routing.GetSlot(pagePath string, slotName string)
```

---

### Route Options

```go
type RouteOptions struct {
    // Strategy controls how the page is rendered and cached.
    // Values: StrategySSR (default), StrategySSG, StrategyISR, StrategyPPR.
    Strategy   RenderStrategy

    // ISR only: duration after which the cached page is considered stale.
    // On a stale hit, the old page is returned immediately and a background
    // goroutine re-renders and updates the cache (stale-while-revalidate).
    // Zero means always revalidate (behaves like SSR).
    RevalidateAfter time.Duration

    // PPR only: names of dynamic slots excluded from the cached static shell.
    // Each name must match a SlotFunc registered via RegisterSlot for this path.
    DynamicSlots []string
}

const (
    StrategySSR RenderStrategy = "ssr" // fresh render per request (default)
    StrategySSG RenderStrategy = "ssg" // render once, cache forever
    StrategyISR RenderStrategy = "isr" // render once, revalidate after TTL
    StrategyPPR RenderStrategy = "ppr" // static shell + per-request dynamic slots
)

// Get options for route
opts := routing.GetRouteOptions(path string)
```

---

## Fiber Package

`github.com/aydenstechdungeon/gospa/fiber`

### Middleware

```go
// SPA middleware - initializes state and component ID
app.Use(fiber.SPAMiddleware(config fiber.Config))

// State injection into HTML responses
app.Use(fiber.StateMiddleware(config fiber.Config))

// Runtime script serving
app.Get("/_gospa/runtime.js", fiber.RuntimeMiddleware(simple bool))
app.Get("/_gospa/runtime.js", fiber.RuntimeMiddlewareWithContent(content []byte))

// SPA navigation detection
app.Use(fiber.SPANavigationMiddleware())
isSPA := fiber.IsSPANavigation(c *fiber.Ctx) bool

// CORS
app.Use(fiber.CORSMiddleware(allowedOrigins []string))

// Security headers
app.Use(fiber.SecurityHeadersMiddleware())

// CSRF protection — use BOTH middlewares:
// 1. CSRFSetTokenMiddleware issues the cookie on GET responses
// 2. CSRFTokenMiddleware validates the token on POST/PUT/DELETE/PATCH
app.Use(fiber.CSRFSetTokenMiddleware()) // must come before CSRFTokenMiddleware
app.Use(fiber.CSRFTokenMiddleware())

// Compression (Brotli + Gzip)
app.Use(fiber.BrotliGzipMiddleware(fiber.DefaultCompressionConfig()))

// Request logging
app.Use(fiber.RequestLoggerMiddleware())

// Panic recovery
app.Use(fiber.RecoveryMiddleware())
```

---

### WebSocket Hub

```go
// Create hub
hub := fiber.NewWSHub()

// Start hub (run in goroutine)
go hub.Run()

// Broadcast to all clients
hub.Broadcast <- []byte(message)

// Broadcast to specific clients
hub.BroadcastTo(clientIDs []string, message []byte)

// Broadcast except one
hub.BroadcastExcept(exceptID string, message []byte)

// Get client
client, ok := hub.GetClient(id string)

// Client count
count := hub.ClientCount()
```

---

### WebSocket Client

```go
// Create client
client := fiber.NewWSClient(id string, conn *websocket.Conn)

// Properties
client.ID        string
client.SessionID string
client.Conn      *websocket.Conn
client.State     *state.StateMap

// Methods
client.SendJSON(v interface{}) error
client.SendError(message string)
client.SendState()
client.SendInitWithSession(sessionToken string)
client.Close() error

// Read/Write pumps
client.ReadPump(hub *WSHub, onMessage func(*WSClient, WSMessage))
client.WritePump()
```

---

### WebSocket Configuration

```go
config := fiber.WebSocketConfig{
    Hub:         hub,                    // WebSocket hub
    OnConnect:   func(*WSClient) {},     // Connect callback
    OnDisconnect: func(*WSClient) {},    // Disconnect callback
    OnMessage:   func(*WSClient, WSMessage) {}, // Message handler
    GenerateID:  func() string {},       // ID generator
}

// Create handler
handler := fiber.WebSocketHandler(config)
```

---

### Action Handlers

```go
// Register action handler
fiber.RegisterActionHandler(name string, handler func(*WSClient, json.RawMessage))

// Register connect handler
fiber.RegisterOnConnectHandler(handler func(*WSClient))

// Get action handler
handler, ok := fiber.GetActionHandler(name string)
```

**Example**

```go
fiber.RegisterActionHandler("increment", func(client *fiber.WSClient, payload json.RawMessage) {
    GlobalCounter.Count++
    fiber.BroadcastState(hub, "count", GlobalCounter.Count)
})
```

---

### Session Management

```go
// Session store - maps tokens to client IDs
sessionStore := fiber.NewSessionStore()
token := sessionStore.CreateSession(clientID string)
clientID, ok := sessionStore.ValidateSession(token string)
sessionStore.RemoveSession(token string)
sessionStore.RemoveClientSessions(clientID string)

// Client state store - persists state by client ID
stateStore := fiber.NewClientStateStore()
stateStore.Save(clientID string, state *state.StateMap)
state, ok := stateStore.Get(clientID string)
stateStore.Remove(clientID string)

// Global instances
fiber.globalSessionStore
fiber.globalClientStateStore
```

---

### Utility Functions

```go
// Broadcast state to all clients
fiber.BroadcastState(hub *WSHub, key string, value interface{}) error

// Send to specific client
fiber.SendToClient(hub *WSHub, clientID string, message interface{}) error

// State sync HTTP handler
handler := fiber.StateSyncHandler(hub *WSHub)

// Component rendering
fiber.RenderComponent(c *fiber.Ctx, config Config, component templ.Component, name string) error

// State access
stateMap := fiber.GetState(c *fiber.Ctx, config Config)
componentID := fiber.GetComponentID(c *fiber.Ctx, config Config)
sessionState := fiber.GetSessionState(c *fiber.Ctx, config Config)
fiber.SetSessionState(c *fiber.Ctx, config Config, key string, value interface{})

// Response helpers
fiber.JSONResponse(c *fiber.Ctx, status int, data interface{}) error
fiber.JSONError(c *fiber.Ctx, status int, message string) error
fiber.ParseBody(c *fiber.Ctx, v interface{}) error
```

---

## Templ Package

`github.com/aydenstechdungeon/gospa/templ`

### Reactive Bindings

```go
// Create data binding attribute
attrs := templ.Bind(componentID, key string) templ.Attributes

// Create multiple bindings
attrs := templ.BindAll(componentID string, keys ...string) templ.Attributes
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
attrs := templ.On(event, componentID, handler string) templ.Attributes

// With options
attrs := templ.OnWithOpts(event, componentID, handler string, opts templ.EventOptions) templ.Attributes

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

### Component Helpers

```go
// Create component wrapper
comp := templ.NewComponent(name string, opts ...ComponentOption) *Component

// Options
templ.WithProps(props map[string]any) ComponentOption
templ.WithState(state *state.StateMap) ComponentOption

// Render
rendered := templ.RenderComponent(comp *Component, content templ.Component) templ.Component
```

---

## Client Runtime

### Rune Class

```javascript
const count = new GoSPA.Rune(0)

count.get()           // 0
count.set(5)
count.update(v => v + 1)

const unsub = count.subscribe((value, oldValue) => {
    console.log('Count:', value)
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

### StateMap Class

```javascript
const state = new GoSPA.StateMap()

state.set('count', 0)
state.get('count')     // Rune object
state.has('count')     // true
state.delete('count')
state.clear()

state.toJSON()         // { count: 0 }
state.fromJSON({ count: 5 })
```

---

### Batch Updates

```javascript
GoSPA.batch(() => {
    count.set(1)
    name.set('Alice')
})
```

---

### Watch

```javascript
// Watch single rune
const unsub = GoSPA.watch(count, (value, oldValue) => {
    console.log('Changed:', value)
})

// Watch multiple runes
const unsub = GoSPA.watch([count, name], (values, oldValues) => {
    console.log('Count:', values[0], 'Name:', values[1])
})
```

---

### Advanced State

```javascript
// Untrack - read without subscribing
const value = GoSPA.untrack(() => count.get())

// Raw rune - shallow reactivity
const raw = new GoSPA.RuneRaw({ name: 'Alice' })

// Snapshot - non-reactive copy
const snap = GoSPA.snapshot(count)

// Pre-effect - runs before DOM updates
const cleanup = new GoSPA.PreEffect(() => {
    console.log('Before DOM update')
})

// Effect root - manual lifecycle
const stop = GoSPA.effectRoot(() => {
    console.log('Effect running')
})

// Check if tracking
if (GoSPA.tracking()) { ... }
```

---

### Async State

```javascript
// Async derived
const data = new GoSPA.DerivedAsync(async () => {
    const res = await fetch('/api/data')
    return res.json()
})

data.get()       // undefined while loading
data.status      // 'idle' | 'pending' | 'success' | 'error'
data.isPending   // true/false
data.isSuccess   // true/false
data.isError     // true/false
data.error       // Error object if failed

// Resource
const res = new GoSPA.Resource(async () => {
    const response = await fetch('/api/data')
    return response.json()
})

await res.refetch()
res.reset()
res.data         // current data
res.status       // current status

// Reactive resource
const res = GoSPA.resourceReactive([userId], async () => {
    return fetch(`/api/users/${userId.get()}`).then(r => r.json())
})
```

---

### Debug

```javascript
// Inspect state changes (dev only)
GoSPA.inspect(count, name)
GoSPA.inspect(count).with((type, values) => {
    if (type === 'update') debugger
})

// Trace dependencies
GoSPA.inspect.trace('label')
```

---

### WebSocket Client

```javascript
// Initialize WebSocket
const ws = GoSPA.initWebSocket('ws://localhost:3000/_gospa/ws')

// Get client
const client = GoSPA.getWebSocketClient()

// Send action
GoSPA.sendAction('increment', { value: 1 })

// Synced rune (auto-syncs with server)
const count = GoSPA.syncedRune('count', 0)

// Apply state update
GoSPA.applyStateUpdate({ key: 'count', value: 5 })
```

---

### Navigation

```javascript
// Navigate
GoSPA.navigate('/about')
GoSPA.navigate('/about', { replace: true })

// History
GoSPA.back()
GoSPA.forward()
GoSPA.go(-2)  // Go back 2

// Prefetch
GoSPA.prefetch('/blog')      // Prefetch page into cache (30s TTL)

// State
GoSPA.getCurrentPath()
GoSPA.isNavigating()

// Events
GoSPA.onBeforeNavigate((from, to) => {
    // Return false to cancel
})
GoSPA.onAfterNavigate((from, to) => {
    console.log('Navigated to:', to)
})

// Initialize navigation
GoSPA.initNavigation()
GoSPA.destroyNavigation()

// Navigation state
const navState = GoSPA.createNavigationState()

// Global DOM Event (for vanilla JS)
document.addEventListener('gospa:navigated', (e) => {
    console.log('Navigated to:', e.detail.path)
})
```

---

### Transitions

```javascript
// Setup transitions on element
GoSPA.setupTransitions(element, {
    enter: 'fade',
    leave: 'slide'
})

// Built-in transitions
GoSPA.fade(element, { duration: 300 })
GoSPA.fly(element, { duration: 300, y: 50 })
GoSPA.slide(element, { duration: 300, direction: 'left' })
GoSPA.scale(element, { duration: 300, start: 0.8 })
GoSPA.blur(element, { duration: 300 })

// Crossfade between elements
GoSPA.crossfade(elementA, elementB, { duration: 300 })
```

---

### DOM Binding

```javascript
// Bind element to rune
GoSPA.bindElement('element-id', countRune)

// Two-way binding for inputs
GoSPA.bindTwoWay('input-id', nameRune)

// Conditional rendering
GoSPA.renderIf(conditionRune, element)

// List rendering
GoSPA.renderList(itemsRune, (item, index) => {
    return document.createElement('div')
})

// Register/unregister custom bindings
GoSPA.registerBinding('custom', (element, rune) => { ... })
GoSPA.unregisterBinding('custom')
```

---

### Events

```javascript
// Add event listener with auto-cleanup
GoSPA.on(element, 'click', handler)

// Remove all listeners
GoSPA.offAll(element)

// Debounce
const debounced = GoSPA.debounce(handler, 300)

// Throttle
const throttled = GoSPA.throttle(handler, 300)

// Event delegation
GoSPA.delegate(container, '.button', 'click', handler)

// Keyboard shortcuts
GoSPA.onKey(['ctrl', 's'], handler)
GoSPA.keys.ctrl  // Check if key is pressed
GoSPA.keys.all   // Set of all pressed keys

// Event transformers
GoSPA.transformers.preventDefault
GoSPA.transformers.stopPropagation
```

---

### Event Handling Patterns

GoSPA sanitizes HTML during SPA navigation using DOMPurify, which strips inline event handlers (`onclick`, `onmouseover`, etc.) for security. Use these patterns instead:

**Inside Components (data-on):**

```html
<div data-gospa-component="counter">
    <button data-on="click:increment">+</button>
    <span data-bind="count">0</span>
</div>
```

**Outside Components (data-action with global delegation):**

For elements outside `data-gospa-component` (like layouts, static content), use `data-action` attributes with global event delegation:

```html
<!-- In your template -->
<button data-action="copy-code">Copy</button>
<button data-action="toggle-menu">Menu</button>
```

```javascript
// In your root layout or main script
document.addEventListener('click', (e) => {
    const target = e.target.closest('[data-action]');
    if (!target) return;
    
    const action = target.getAttribute('data-action');
    
    switch (action) {
        case 'copy-code':
            const code = target.parentElement.querySelector('code');
            navigator.clipboard.writeText(code.innerText);
            break;
        case 'toggle-menu':
            document.getElementById('menu').classList.toggle('hidden');
            break;
    }
});
```

This pattern survives SPA navigation because the handler is attached at document level, not on individual elements.

---

### Component API

```javascript
// Initialize component
GoSPA.init({
    wsUrl: 'ws://localhost:3000/_gospa/ws',
    debug: true,
    hydration: {
        mode: 'immediate',  // 'immediate' | 'lazy' | 'visible'
        timeout: 5000
    }
})

// Create component
const comp = GoSPA.createComponent('counter', {
    count: new GoSPA.Rune(0)
})

// Get component
const comp = GoSPA.getComponent('counter')

// State access
const state = GoSPA.getState('counter')
GoSPA.setState('counter', { count: 5 })

// Call action
GoSPA.callAction('increment', { value: 1 })

// Destroy component
GoSPA.destroyComponent('counter')

// Auto-initialize from DOM
GoSPA.autoInit()
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

Available transitions: `fade`, `fly`, `slide`, `scale`, `blur`

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

---

## Islands & Partial Hydration

See [ISLANDS.md](ISLANDS.md) for full documentation.

```typescript
import { initIslands, hydrateIsland, getIslandManager } from './island.ts';

// Initialize — auto-discovers [data-gospa-island] elements
const manager = initIslands({
    moduleBasePath: '/islands',    // Where island JS files are served from
    defaultTimeout: 30000,
    debug: false,
    moduleLoader: async (name) => import(`./islands/${name}`), // optional custom loader
});

// Manually hydrate a lazy island
await hydrateIsland('Counter');  // by name or DOM id

// Check status
manager.isHydrated('counter-1');
manager.getIslands();       // all discovered islands
manager.discoverIslands();  // re-scan DOM (use after SPA nav)
manager.destroy();          // clean up observers

// Browser globals (available when island.ts is loaded)
window.__GOSPA_ISLAND_MANAGER__.init(config)
window.__GOSPA_ISLAND_MANAGER__.get()
window.__GOSPA_ISLAND_MANAGER__.hydrate(name)

// DOM event — fired on the island element after hydration
element.addEventListener('gospa:hydrated', (ev) => {
    console.log(ev.detail.island); // IslandElementData
});
```

### Hydration Modes (via `data-gospa-mode`)

| Mode | Trigger |
|------|---------|
| `immediate` | On discovery (default) |
| `visible` | Enters viewport (IntersectionObserver) |
| `idle` | Browser idle (requestIdleCallback) |
| `interaction` | mouseenter / touchstart / focusin / click |
| `lazy` | Manual call only |

### Island Module Shape

```typescript
// Both default export and named export are supported.
export default {
    hydrate(element: Element, props: Record<string, unknown>, state: Record<string, unknown>) {
        // Mount your UI here
    }
}
// OR
export function hydrate(element, props, state) { ... }
export function mount(element, props, state) { ... }  // alias for hydrate
```

---

## HMR (Hot Module Replacement)

See [HMR.md](HMR.md) for full documentation.

```go
// Server setup
hmr := fiber.InitHMR(fiber.HMRConfig{
    Enabled:      true,
    WatchPaths:   []string{"./routes", "./static"},
    IgnorePaths:  []string{"node_modules", ".git"},
    DebounceTime: 500 * time.Millisecond, // default
})
hmr.Start()
defer hmr.Stop()

// Register endpoints
app.Get("/__hmr", hmr.HMREndpoint())  // WebSocket upgrades
app.Use(hmr.HMRMiddleware())           // Inject client script into HTML

// State preservation (for modules that need it)
hmr.PreserveState(moduleID string, state any)
state, exists := hmr.GetState(moduleID string)
hmr.ClearState(moduleID string)

// Global accessor
hmr := fiber.GetHMR()
```

### HMRConfig Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Enabled` | bool | false | Master switch |
| `WatchPaths` | []string | — | Dirs to poll for changes |
| `IgnorePaths` | []string | — | Path substrings to skip |
| `DebounceTime` | Duration | 500ms | Min time between updates for same file |
| `BroadcastAll` | bool | false | Broadcast to all clients regardless of path |
