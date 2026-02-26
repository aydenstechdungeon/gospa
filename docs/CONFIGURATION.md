# GoSPA Configuration Reference

Complete reference for all `gospa.Config` options. This is the single source of truth for configuring your GoSPA application.

## Quick Reference

```go
app := gospa.New(gospa.Config{
    RoutesDir: "./routes",
    DevMode:   true,
    AppName:   "My App",
})
```

## All Configuration Options

### Basic Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `RoutesDir` | `string` | `"./routes"` | Directory containing route files |
| `RoutesFS` | `fs.FS` | `nil` | Filesystem for routes (takes precedence over RoutesDir) |
| `DevMode` | `bool` | `false` | Enable development features (logging, print routes) |
| `RuntimeScript` | `string` | `"/_gospa/runtime.js"` | Path to client runtime script |
| `StaticDir` | `string` | `"./static"` | Directory for static files |
| `StaticPrefix` | `string` | `"/static"` | URL prefix for static files |
| `AppName` | `string` | `"GoSPA App"` | Application name (used in default layout) |

### State Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `DefaultState` | `map[string]interface{}` | `{}` | Initial state for new sessions |
| `StateSerializer` | `StateSerializerFunc` | JSON | Custom state serialization function |
| `StateDeserializer` | `StateDeserializerFunc` | JSON | Custom state deserialization function |

### WebSocket Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `EnableWebSocket` | `bool` | `true` | Enable real-time state synchronization via WebSocket |
| `WebSocketPath` | `string` | `"/_gospa/ws"` | Endpoint for WebSocket connections |
| `WebSocketMiddleware` | `fiber.Handler` | `nil` | Middleware to run before WebSocket upgrade (for auth/sessions) |
| `WSReconnectDelay` | `time.Duration` | `0` | Initial delay before reconnecting on failure |
| `WSMaxReconnect` | `int` | `0` | Maximum number of reconnect attempts (0 for unlimited) |
| `WSHeartbeat` | `time.Duration` | `0` | Interval for heartbeat messages to keep connection alive |

### Performance Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `CompressState` | `bool` | `false` | Enable zlib compression for WebSocket messages |
| `StateDiffing` | `bool` | `false` | Only send state diffs over WebSocket to save bandwidth |
| `CacheTemplates` | `bool` | `false` | Enable template caching for SSG, ISR, and PPR pages (recommended for production) |
| `SimpleRuntime` | `bool` | `false` | Use lightweight ~11KB runtime without DOMPurify |
| `SimpleRuntimeSVGs` | `bool` | `false` | Allow SVG/math elements in simple runtime (security risk for untrusted content) |
| `SSGCacheMaxEntries` | `int` | `500` | FIFO eviction limit shared by SSG, ISR, and PPR shell caches. `-1` = unbounded. |

### Rendering Strategy Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `DefaultRenderStrategy` | `routing.RenderStrategy` | `""` (StrategySSR) | Fallback strategy for pages that don't explicitly call `RegisterPageWithOptions` |
| `DefaultRevalidateAfter` | `time.Duration` | `0` | ISR TTL fallback for ISR pages that don't set `RouteOptions.RevalidateAfter` |

### Hydration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `HydrationMode` | `string` | `""` | Hydration strategy: `"immediate"`, `"lazy"`, `"visible"`, `"idle"` |
| `HydrationTimeout` | `int` | `0` | Milliseconds before force hydrate |

### Routing Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `DisableSPA` | `bool` | `false` | Disable SPA navigation completely |
| `SSR` | `bool` | `false` | Global SSR mode |

### Distributed & Scaling Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `Prefork` | `bool` | `false` | Enables Fiber Prefork. WARNING: Without external storage, state is isolated per process. |
| `Storage` | `store.Storage` | `memory` | External Key-Value store (e.g., Redis) for Session, ClientState, and caching of SSG/ISR/PPR pages. |
| `PubSub` | `store.PubSub` | `memory` | External messaging broker (e.g., Redis PubSub) for WebSocket broadcasting. |

### Remote Action Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `MaxRequestBodySize` | `int` | `4194304` (4MB) | Max size for remote action request bodies |
| `RemotePrefix` | `string` | `"/_gospa/remote"` | Prefix for remote action endpoints |

### Security Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `AllowedOrigins` | `[]string` | `[]` | Allowed CORS origins |
| `EnableCSRF` | `bool` | `false` | Enable automatic CSRF protection |

---

## Detailed Option Descriptions

### RoutesDir

Directory containing your `.templ` route files. GoSPA scans this directory to build the route tree.

```go
RoutesDir: "./routes"
```

### RoutesFS

Alternative: provide an `fs.FS` filesystem for routes. Takes precedence over `RoutesDir` if provided. Useful for embedding routes in the binary.

```go
//go:embed routes
var routesFS embed.FS

app := gospa.New(gospa.Config{
    RoutesFS: routesFS,
})
```

### DevMode

Enables development features:
- Request logging
- Route printing on startup
- Debug mode in client runtime

```go
DevMode: true
```

### RuntimeScript

Path to the client runtime script. Defaults to the embedded runtime. Override to use a custom build.

```go
RuntimeScript: "/static/js/runtime.js"
```

### StaticDir & StaticPrefix

Configure static file serving:

```go
StaticDir:    "./public",     // Directory containing static files
StaticPrefix: "/assets",      // Serve at /assets/*
```

### DefaultState

Initial state for new sessions. This state is synced to the client on initial load.

```go
DefaultState: map[string]interface{}{
    "user":    nil,
    "theme":   "light",
    "counter": 0,
}
```

### StateSerializer / StateDeserializer

Custom serialization for state. Useful for complex types or compression.

```go
StateSerializer: func(state interface{}) ([]byte, error) {
    return json.Marshal(state)
},

StateDeserializer: func(data []byte, target interface{}) error {
    return json.Unmarshal(data, target)
},
```

### WebSocketPath

WebSocket endpoint path. Must match client configuration.

```go
WebSocketPath: "/_gospa/ws"
```

### WebSocketMiddleware

Middleware that runs before WebSocket upgrade. Use for authentication, session validation, etc.

```go
WebSocketMiddleware: func(c *fiber.Ctx) error {
    // Check session/auth before upgrade
    token := c.Get("Authorization")
    if !validateToken(token) {
        return c.Status(401).SendString("Unauthorized")
    }
    return c.Next()
},
```

### WSReconnectDelay

Initial delay before reconnecting after WebSocket disconnect. The delay increases exponentially with each failed attempt.

```go
WSReconnectDelay: 100 * time.Millisecond,
```

### WSMaxReconnect

Maximum number of reconnection attempts. Set to 0 for unlimited attempts.

```go
WSMaxReconnect: 10,  // Give up after 10 attempts
```

### WSHeartbeat

Interval for sending heartbeat/ping messages to keep the connection alive.

```go
WSHeartbeat: 30 * time.Second,
```

### CompressState

Enable compression for WebSocket messages. Reduces bandwidth for large state updates.

```go
CompressState: true,
```

### StateDiffing

Only send state differences over WebSocket instead of full state. Reduces bandwidth when state changes are small.

```go
StateDiffing: true,
```

### CacheTemplates

Cache compiled templates in memory. Enables SSG, ISR, and PPR page caching. Required for all three strategies to operate; without it every request is rendered fresh (SSR behaviour).

```go
CacheTemplates: true,
```

### SimpleRuntime

Use the lightweight runtime without DOMPurify sanitization. **Faster but less secure.** Only use when you control all HTML content.

```go
SimpleRuntime: true,  // ~30% smaller runtime, no DOMPurify
```

See [Runtime Selection Guide](./RUNTIME.md) for details.

### SimpleRuntimeSVGs

Allow SVG and math elements in the simple runtime sanitizer. **Only use when you control ALL HTML content.**

```go
SimpleRuntime:    true,
SimpleRuntimeSVGs: true,  // Allow SVG/math elements in simple runtime
```

> ⚠️ **Security Warning**: SVG elements can contain embedded JavaScript and event handlers (e.g., `<svg onload="alert('xss')">`). Only enable this option if you completely trust all HTML content being rendered. Never enable when rendering user-generated content. This option has no effect when using the full runtime (when `SimpleRuntime: false`).

### HydrationMode

Control when components hydrate (become interactive):

- `"immediate"` - Hydrate as soon as possible (default)
- `"lazy"` - Hydrate during browser idle time
- `"visible"` - Hydrate when component enters viewport
- `"idle"` - Hydrate when browser is idle (uses requestIdleCallback)

```go
HydrationMode: "visible",  // Hydrate when scrolled into view
```

### HydrationTimeout

Milliseconds before forcing hydration. Used with `"lazy"` and `"idle"` modes to ensure hydration happens even if the browser never becomes idle.

```go
HydrationMode:    "idle",
HydrationTimeout: 2000,  // Force hydrate after 2 seconds
```

### DisableSPA

Disable SPA navigation completely. All navigation will trigger full page loads.

```go
DisableSPA: true,  // Traditional multi-page app behavior
```

### SSR

Enable global Server-Side Rendering mode.

```go
SSR: true,
```

### Prefork, Storage, and PubSub

Used for horizontal scaling and multi-core utilization via Fiber's `Prefork`. If `Prefork: true` is enabled, you **must** provide external implementations for `Storage` and `PubSub` (e.g. Redis), otherwise state and WebSockets will be isolated to individual child processes. This also enables a shared global cache for SSG, ISR, and PPR pages across processes.

```go
import "github.com/aydenstechdungeon/gospa/store/redis"

app := gospa.New(gospa.Config{
    Prefork: true,
    Storage: redis.NewStore(rdb),
    PubSub:  redis.NewPubSub(rdb),
})
```

### MaxRequestBodySize

Maximum allowed size for remote action request bodies in bytes.

```go
MaxRequestBodySize: 10 * 1024 * 1024,  // 10MB
```

### RemotePrefix

URL prefix for remote action endpoints.

```go
RemotePrefix: "/api/remote",  // Remote actions at /api/remote/:name
```

### AllowedOrigins

CORS allowed origins. Set to allow cross-origin requests.

```go
AllowedOrigins: []string{
    "https://example.com",
    "https://app.example.com",
},
```

### EnableCSRF

Enable automatic CSRF protection. Adds CSRF token to forms and validates on POST requests.

```go
EnableCSRF: true,
```

---

## Complete Example

```go
package main

import (
    "time"
    
    "github.com/aydenstechdungeon/gospa"
)

func main() {
    app := gospa.New(gospa.Config{
        // Basic
        RoutesDir: "./routes",
        AppName:   "My Application",
        DevMode:   true,
        
        // State
        DefaultState: map[string]interface{}{
            "theme": "dark",
            "user":  nil,
        },
        
        // WebSocket
        EnableWebSocket:    true,
        WebSocketPath:      "/_gospa/ws",
        WSReconnectDelay:   100 * time.Millisecond,
        WSMaxReconnect:     10,
        WSHeartbeat:        30 * time.Second,
        
        // Performance
        CompressState:  true,
        StateDiffing:   true,
        CacheTemplates: true,
        SimpleRuntime:  false,  // Use full runtime with DOMPurify
        
        // Rendering Strategy Defaults
        DefaultRenderStrategy:  routing.StrategyISR,
        DefaultRevalidateAfter: 10 * time.Minute,
        SSGCacheMaxEntries:     1000,
        
        // Hydration
        HydrationMode:    "visible",
        HydrationTimeout: 3000,
        
        // Security
        AllowedOrigins: []string{"https://myapp.com"},
        EnableCSRF:     true,
        
        // Remote Actions
        MaxRequestBodySize: 8 * 1024 * 1024,  // 8MB
    })
    
    if err := app.Run(":3000"); err != nil {
        panic(err)
    }
}
```

---

## Default Configuration

The `DefaultConfig()` function returns sensible defaults:

```go
config := gospa.DefaultConfig()
// Returns:
// - RoutesDir: "./routes"
// - RuntimeScript: "/_gospa/runtime.js"
// - StaticDir: "./static"
// - StaticPrefix: "/static"
// - AppName: "GoSPA App"
// - EnableWebSocket: true
// - WebSocketPath: "/_gospa/ws"
// - RemotePrefix: "/_gospa/remote"
// - MaxRequestBodySize: 4MB
```

---

## Environment-Specific Configuration

### Development

```go
app := gospa.New(gospa.Config{
    DevMode:     true,
    AppName:     "My App (Dev)",
    RoutesDir:   "./routes",
    SimpleRuntime: false,  // Full sanitization for dev
})
```

### Production

```go
app := gospa.New(gospa.Config{
    DevMode:         false,
    AppName:         "My App",
    RoutesDir:       "./routes",
    CompressState:   true,
    StateDiffing:    true,
    CacheTemplates:  true,
    EnableCSRF:      true,
    AllowedOrigins:  []string{"https://myapp.com"},
    WSHeartbeat:     30 * time.Second,
})
```

### High-Performance

```go
app := gospa.New(gospa.Config{
    DevMode:         false,
    SimpleRuntime:   true,   // No DOMPurify - only if you control all content!
    CompressState:   true,
    StateDiffing:    true,
    CacheTemplates:  true,
    HydrationMode:   "lazy",
})
```
