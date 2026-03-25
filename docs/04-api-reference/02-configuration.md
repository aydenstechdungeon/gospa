# GoSPA Configuration Reference

Complete reference for all `gospa.Config` options. **Source of truth in code:** `type Config struct` in [`gospa.go`](https://github.com/aydenstechdungeon/gospa/blob/main/gospa.go). This page describes defaults applied in `gospa.New`, presets (`DefaultConfig`, `ProductionConfig`, `MinimalConfig`), and usage patterns.

**Website:** Topic pages at [gospa.onrender.com/docs/configuration](https://gospa.onrender.com/docs/configuration) mirror this content; edit the Markdown in this repo when updating docs.

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
| `Logger` | `*slog.Logger` | `slog.Default()` | Structured logger for framework and your handlers |

### State Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `DefaultState` | `map[string]interface{}` | `{}` | Initial state for new sessions |
| `SerializationFormat` | `string` | `"json"` | Serialization for WebSocket: `"json"` (goccy/go-json) or `"msgpack"` |
| `StateSerializer` | `StateSerializerFunc` | Auto | Overrides default outbound state serialization |
| `StateDeserializer` | `StateDeserializerFunc` | Auto | Overrides default inbound state deserialization |

### WebSocket Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `EnableWebSocket` | `bool` | `true` | Enable real-time state synchronization via WebSocket |
| `WebSocketPath` | `string` | `"/_gospa/ws"` | Endpoint for WebSocket connections |
| `WebSocketMiddleware` | `fiber.Handler` | `nil` | Middleware to run before WebSocket upgrade (for auth/sessions) |
| `WSReconnectDelay` | `time.Duration` | `0` in `DefaultConfig` | Passed to client; if zero when HTML is rendered, **1s** is injected. `ProductionConfig` sets **1s**. |
| `WSMaxReconnect` | `int` | `0` in `DefaultConfig` | If ≤0 when HTML is rendered, **10** is injected. `ProductionConfig` sets **10**. |
| `WSHeartbeat` | `time.Duration` | `0` in `DefaultConfig` | If zero when HTML is rendered, **30s** is injected. `ProductionConfig` sets **30s**. |
| `WSMaxMessageSize` | `int` | `65536` | Maximum payload size for WebSocket messages (64KB) |
| `WSConnRateLimit` | `float64` | `1.5` | Refilling rate in connections per second for WebSocket upgrades |
| `WSConnBurst` | `float64` | `15.0` | Burst capacity for WebSocket connection upgrades |

### Performance Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `CompressState` | `bool` | `false` | Enable zlib compression for WebSocket messages |
| `StateDiffing` | `bool` | `false` | Only send state diffs over WebSocket to save bandwidth |
| `CacheTemplates` | `bool` | `false` | Enable template caching for SSG, ISR, and PPR pages (recommended for production) |
| `SimpleRuntime` | `bool` | `false` | Use lightweight ~11KB runtime without DOMPurify |
| `SimpleRuntimeSVGs` | `bool` | `false` | Allow SVG/math elements in simple runtime (security risk for untrusted content) |
| `DisableSanitization` | `bool` | `false` | Trusts server-rendered HTML without DOMPurify (SvelteKit-like model) |
| `SSGCacheMaxEntries` | `int` | `500` | FIFO eviction limit shared by SSG, ISR, and PPR shell caches. `-1` = unbounded. |
| `SSGCacheTTL` | `time.Duration` | `0` | Expiration time for SSG cache entries. `0` means cache forever. |
| `NotificationBufferSize` | `int` | `1024` | Size of the asynchronous state change notification queue. |

### Rendering Strategy Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `DefaultRenderStrategy` | `routing.RenderStrategy` | `""` (StrategySSR) | Fallback strategy for pages that don't explicitly call `RegisterPageWithOptions` |
| `DefaultRevalidateAfter` | `time.Duration` | `0` | ISR TTL fallback for ISR pages that don't set `RouteOptions.RevalidateAfter` |
| `ISRSemaphoreLimit` | `int` | `10` | Limits concurrent background ISR revalidations to prevent resource exhaustion. |
| `ISRTimeout` | `time.Duration` | `60s` | Maximum time allowed for a single background ISR revalidation. |

### Hydration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `HydrationMode` | `string` | `""` | Hydration strategy: `"immediate"`, `"lazy"`, `"visible"`, `"idle"` |
| `HydrationTimeout` | `int` | `0` | Milliseconds before force hydrate |

### Routing Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `DisableSPA` | `bool` | `false` | Disable SPA navigation completely |

### Distributed & Scaling Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `Prefork` | `bool` | `false` | Enables Fiber Prefork. **CRITICAL**: Without external storage, state is isolated per process. |
| `Storage` | `store.Storage` | `memory` | External Key-Value store (e.g., Redis) for Session, ClientState, and caching of SSG/ISR/PPR pages. |
| `PubSub` | `store.PubSub` | `memory` | External messaging broker (e.g., Redis PubSub) for WebSocket broadcasting. |

### Remote Action Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `MaxRequestBodySize` | `int` | `4194304` (4MB) | Max size for remote action request bodies |
| `RemotePrefix` | `string` | `"/_gospa/remote"` | Prefix for remote action endpoints |
| `RemoteActionMiddleware` | `fiber.Handler` | `nil` | Optional middleware to enforce global auth/policy checks before remote actions |
| `AllowUnauthenticatedRemoteActions` | `bool` | `false` | If `true`, disables the production guard that blocks remote actions when no `RemoteActionMiddleware` is configured |

### Security Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `AllowedOrigins` | `[]string` | `[]` | Allowed CORS origins |
| `EnableCSRF` | `bool` | `true` | Enable automatic CSRF protection (wired by gospa.New) |
| `ContentSecurityPolicy` | `string` | built-in default | Optional CSP header; empty uses `fiber.DefaultContentSecurityPolicy` (`'self'` plus `'unsafe-inline'` for script and style—typical for GoSPA). Use `fiber.StrictContentSecurityPolicy` when you can avoid inline scripts. |
| `PublicOrigin` | `string` | `""` | Public base URL (e.g. `https://app.example.com`) for stable WebSocket URLs behind proxies; empty derives from the request |

### Navigation options (client)

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `NavigationOptions` | `NavigationOptions` | zero | Optional tuning: speculative prefetch, URL parse cache, view transitions, service worker helpers, etc. See `gospa.NavigationOptions` in `gospa.go`. |

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

### SerializationFormat

Sets the underlying format for all WebSocket communications. 

- `"json"` (Default): Uses the high-performance `goccy/go-json` library for 2-3x faster serialization than standard library.
- `"msgpack"`: binary MessagePack format for even smaller payloads and faster parsing on supported clients.

```go
SerializationFormat: "msgpack"
```

### StateSerializer / StateDeserializer

Custom serialization for state. Overrides the default selected by `SerializationFormat`.

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

### Rate Limiting (Built-in)

GoSPA includes an automatic, non-configurable token bucket rate limiter for WebSocket upgrades to prevent DoS attacks. The default limits are:
- **Burst Capacity:** 15 concurrent connection requests per IP
- **Refill Rate:** 1.5 tokens per second (~1 connection every 0.67 seconds)

If an IP exceeds this limit, they will receive a `429 Too Many Requests` response.

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

### NotificationBufferSize

Size of the state change notification queue. Increasing this can help prevent backpressure on the server when many clients are connected and state is changing rapidly.

```go
NotificationBufferSize: 2048,
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

See [Client runtime](../03-features/01-client-runtime.md) for runtime variants.

### DisableSanitization

Disables client-side HTML sanitization for SPA navigation. When enabled, GoSPA trusts server-rendered HTML (which is auto-escaped by Templ) without running it through DOMPurify. This provides a SvelteKit-like security model and slightly better performance.

```go
DisableSanitization: true
```

> ⚠️ **Warning**: You must manually ensure any user-generated content is sanitized on the server if you disable this option.

### SimpleRuntimeSVGs

Allow SVG and math elements in the simple runtime sanitizer. **Only use when you control ALL HTML content.**

```go
SimpleRuntime:    true,
SimpleRuntimeSVGs: true,  // Allow SVG/math elements in simple runtime
```

> ⚠️ **Security Warning**: SVG elements can contain embedded JavaScript and event handlers (e.g., `<svg onload="alert('xss')">`). Only enable this option if you completely trust all HTML content being rendered. Never enable when rendering user-generated content. This option has no effect when using the full runtime (when `SimpleRuntime: false`).

### ISR Options

Configure background revalidation limits for Incremental Static Regeneration (ISR):

```go
ISRSemaphoreLimit: 20,              // Up to 20 concurrent background renders
ISRTimeout:        5 * time.Minute, // Max 5 mins per render
```

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





### Prefork, Storage, and PubSub

Used for horizontal scaling and multi-core utilization via Fiber's `Prefork`.

> **⚠️ CRITICAL WARNING: Prefork Requires External Storage**
>
> When `Prefork: true` is enabled, GoSPA spawns multiple worker processes. Each process has **isolated memory**, meaning:
> - In-memory sessions are NOT shared between workers (users may lose login state)
> - WebSocket broadcasts only reach clients connected to the same worker
> - CSRF tokens stored in memory will fail validation across workers
> - SSG/ISR/PPR page caches are duplicated per worker instead of shared
>
> **You MUST provide external Storage and PubSub implementations** (e.g., Redis) when using Prefork:

```go
import "github.com/aydenstechdungeon/gospa/store/redis"

app := gospa.New(gospa.Config{
    Prefork: true,
    Storage: redis.NewStore(rdb),    // Required: shared session/state storage
    PubSub:  redis.NewPubSub(rdb),   // Required: cross-worker WebSocket broadcasts
})
```

Without external storage, your application will exhibit unpredictable behavior in production as requests are routed to different workers.

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

### RemoteActionMiddleware

Optional middleware executed for all remote action requests before the action handler. Use this for global authorization, tenant checks, and request policy enforcement.

In production (`DevMode: false`), GoSPA now blocks remote action calls if this middleware is not configured (secure-by-default). Set `AllowUnauthenticatedRemoteActions: true` only for explicitly public APIs.

```go
RemoteActionMiddleware: func(c *fiber.Ctx) error {
    // Example: enforce authenticated user context
    if c.Locals("user") == nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
    }
    return c.Next()
},
```

### AllowUnauthenticatedRemoteActions

By default, remote actions require `RemoteActionMiddleware` in production. This flag opts out of that guard.

```go
AllowUnauthenticatedRemoteActions: true, // Only for intentionally public remote actions
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

Enable automatic CSRF protection (wired by gospa.New). This is enabled by default and validates mutating requests (`POST/PUT/PATCH/DELETE`).

```go
EnableCSRF: true,
```

### ContentSecurityPolicy

Optional `Content-Security-Policy` header value. If empty, GoSPA uses **`fiber.DefaultContentSecurityPolicy`**: `default-src 'self'`, `frame-ancestors 'none'`, `object-src 'none'`, and **`'unsafe-inline'`** on `script-src` and `style-src` so inline state/bootstrap scripts work. Override with a stricter policy when your app allows it; `fiber.StrictContentSecurityPolicy` is the recommended starting point (see [Security](../03-features/04-security.md)).

### PublicOrigin

When set (e.g. `https://myapp.com`), WebSocket URLs in generated HTML use this origin’s host/scheme instead of inferring from the incoming request—useful behind reverse proxies or CDNs. Empty string keeps request-derived behavior via `getWSUrl`.

### NavigationOptions

Fine-tune client-side navigation behavior for maximum performance.

```go
NavigationOptions: gospa.NavigationOptions{
    SpeculativePrefetching: &gospa.NavigationSpeculativePrefetchingConfig{
        Enabled: ptr(true),
        TTL:     ptr(45000), // 45 seconds
    },
    URLParsingCache: &gospa.NavigationURLParsingCacheConfig{
        Enabled: ptr(true),
        MaxSize: ptr(500),
    },
}
```

Available sub-configs (see `NavigationOptions` in `gospa.go`):
- `SpeculativePrefetching`: Fetches links on hover/viewport entry.
- `URLParsingCache`: Caches internal URL parsing results.
- `IdleCallbackBatchUpdates`: Batches state updates during browser idle time.
- `LazyRuntimeInitialization`: Defers boot logic until first interaction.
- `ServiceWorkerNavigationCaching`: Offloads page caching to a Service Worker.
- `ViewTransitions`: Enables the View Transitions API for smooth page fades.

The TypeScript client also supports a **`progressBar`** option (defaults in `client/src/navigation.ts`). That field is **not** on the Go `NavigationOptions` struct yet; customize via client `setNavigationOptions` if you embed a custom runtime.

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
