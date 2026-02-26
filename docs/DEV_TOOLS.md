# Development Tools

GoSPA exposes debug utilities via the `window.__GOSPA__` global and several server-side helpers designed exclusively for **development mode**. Do not enable these in production.

---

## Client-Side Debug Object

When the `data-gospa-auto` attribute is present on `<html>`, the runtime exposes itself on `window.__GOSPA__`:

```html
<html data-gospa-auto>
```

From the browser console you can inspect:

```js
// All active component instances (Map<string, ComponentInstance>)
__GOSPA__.components

// The root StateMap holding global state
__GOSPA__.globalState

// The active runtime configuration
__GOSPA__.config

// Functions
__GOSPA__.init(options)
__GOSPA__.createComponent(def, element?, isLocal?)
__GOSPA__.destroyComponent(id)
__GOSPA__.getComponent(id)
__GOSPA__.getState(componentId, key)
__GOSPA__.setState(componentId, key, value)
__GOSPA__.callAction(componentId, action, ...args)
__GOSPA__.bind(componentId, element, binding, key, options?)
__GOSPA__.autoInit()
```

---

## HMR Integration

During development the HMR system logs to the browser console:

```
[HMR] Connected
[HMR] Update: routes/docs/page
[HMR] Full reload required
[HMR] Disconnected, reconnecting...
```

See [HMR.md](HMR.md) for full setup instructions.

---

## Server-Side Debug Logging

The framework uses the standard `log` package for server-side output. Key log messages:

| Message | Source | Meaning |
|---------|--------|---------|
| `Client connected: <id>` | `fiber/websocket.go` | WebSocket client registered |
| `Client disconnected: <id>` | `fiber/websocket.go` | WebSocket client unregistered |
| `Render error: <err>` | `gospa.go` | Template render failed |
| `Streaming render error: <err>` | `gospa.go` | Streaming render failed |
| `CRITICAL: crypto/rand.Read failed` | `fiber/websocket.go` | OS CSPRNG failure (should never happen) |
| `[HMR] Client error: <msg>` | `fiber/hmr.go` | Client reported an HMR error |
| `Failed to read initial message` | `fiber/websocket.go` | Client connected but timed out on auth |

---

## GOSPA_DEBUG Environment Variable

`routing.RegisterPageWithOptions` previously emitted a debug print on every page registration; it has been removed. To re-enable verbose registration logging, set the `GOSPA_DEBUG` environment variable:

```bash
GOSPA_DEBUG=1 go run .
```

> **Note:** This env variable is a convention — individual packages must check it manually if you add custom debug output.

---

## Error Overlay

The `fiber.ErrorOverlay` renders a rich full-page HTML error overlay for development mode, showing the error message, type, file location, stack trace, and request details.

### Configuration

```go
config := fiber.ErrorOverlayConfig{
    Enabled:     true,
    ShowStack:   true,     // Display stack traces
    ShowRequest: true,     // Show request details
    ShowCode:    true,     // Show source code snippets
    Theme:       "dark",   // "dark" or "light"
    Editor:      "code",   // "code" (VS Code), "idea" (JetBrains), "sublime"
}

overlay := fiber.NewErrorOverlay(config)
```

### Automatic DevMode Wiring

When `DevMode: true` is set in `gospa.Config`, a development error handler is automatically registered. Any unhandled error that returns HTML renders the overlay instead of a plain JSON response.

```go
app := gospa.New(gospa.Config{
    DevMode: true,
    // ...
})
```

### Manual Usage

Render the overlay HTML for any Go `error`:

```go
// With request context
overlayHTML := overlay.RenderOverlay(err, c.Request())

// Without request context (e.g., background goroutines)
overlayHTML := overlay.RenderOverlay(err, nil)

// Send as response
c.Status(500)
c.Set("Content-Type", "text/html; charset=utf-8")
return c.SendString(overlayHTML)
```

### `ErrorOverlayConfig` Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Enabled` | `bool` | `true` | Enable the overlay |
| `ShowStack` | `bool` | `true` | Render Go stack traces |
| `ShowRequest` | `bool` | `true` | Include method, URL, and query params |
| `ShowCode` | `bool` | `true` | Show code snippet (if available) |
| `Theme` | `string` | `"dark"` | `"dark"` or `"light"` |
| `Editor` | `string` | `"code"` | `"code"`, `"idea"`, or `"sublime"` — sets the click-to-open protocol |
| `EditorPort` | `int` | `0` | Optional port for local editor server |

---

## State Inspector

`fiber.DevTools` provides a real-time WebSocket-backed state inspector panel for development. It tracks every state key and value change across server and client.

### Setup

```go
devTools := fiber.NewDevTools(fiber.DevConfig{
    Enabled:   true,
    RoutesDir: "./routes",
})
devTools.Start()
defer devTools.Stop()

// Add state inspector middleware (before routes)
app.Use(fiber.StateInspectorMiddleware(devTools, config))

// Mount the dev panel UI at /_gospa/dev
app.Get("/_gospa/dev", devTools.DevPanelHandler())

// Mount WebSocket endpoint for real-time updates
app.Get("/_gospa/dev/ws", devTools.DevToolsHandler())
```

Navigate to `/_gospa/dev` in the browser to open the panel.

### `DevConfig` Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Enabled` | `bool` | `false` | Enable development tools |
| `RoutesDir` | `string` | `"routes"` | Directory to watch for changes |
| `ComponentsDir` | `string` | `"components"` | Components directory to watch |
| `WatchPaths` | `[]string` | `[]` | Additional paths to watch |
| `IgnorePaths` | `[]string` | `["node_modules", ".git", ...]` | Paths to skip |
| `Debounce` | `time.Duration` | `100ms` | Debounce interval for file changes |
| `OnReload` | `func()` | `nil` | Called when files change |
| `StateKey` | `string` | `"gospa.state"` | Context key for state |

### `DevTools` API

```go
// Lifecycle
devTools.Start()  // starts file watcher and dev tools
devTools.Stop()   // stops file watcher

// State logging
devTools.LogStateChange(key, oldValue, newValue, source)
// source: "client" or "server"

// Query log
entries := devTools.GetStateLog()     // []StateLogEntry
keys    := devTools.GetStateKeys()    // []string

// Handlers (registered to Fiber)
devTools.DevPanelHandler()  // fiber.Handler — serves the dev panel HTML
devTools.DevToolsHandler()  // fiber.Handler — WebSocket for real-time updates
```

### Panel Features

The dev panel at `/_gospa/dev` provides:
- **Live Change Log** — every state key change as it happens (server or client origin)
- **Diff View** — before and after values
- **Source Tracking** — whether update came from `"server"` or `"client"`
- **Key Registry** — all currently tracked reactive state keys

### Debug Middleware

`fiber.DebugMiddleware` logs every request with method, path, HTTP status, and processing time:

```go
app.Use(fiber.DebugMiddleware(devTools))
// Output: [GET] /docs/api 200 1.2ms
```

---

## Request Logging

`fiber.RequestLoggerMiddleware()` logs every request with method, path, status code, and duration:

```go
app.Use(fiber.RequestLoggerMiddleware())
// Output: [GET] /docs/api 200 1.234ms
```

For more advanced structured logging (JSON, fields, sampling), use Fiber's built-in logger middleware:

```go
import "github.com/gofiber/fiber/v2/middleware/logger"

app.Fiber.Use(logger.New())
```

---

## Recovery Middleware

`fiber.RecoveryMiddleware()` catches panics and returns a 500 JSON response:

```go
app.Fiber.Use(fiber.RecoveryMiddleware())
```

Use this in all environments to prevent a single panicking goroutine from crashing the server.
