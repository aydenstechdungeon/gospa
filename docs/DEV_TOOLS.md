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

## Error Overlay (Planned)

A full-page HTML error overlay for development mode is **planned** but not yet implemented. The `fiber.ErrorOverlayConfig` struct and `fiber.NewErrorOverlay()` function are reserved for a future release.

---

## State Inspector (Planned)

The `fiber.DevTools`, `fiber.StateInspectorMiddleware`, `devTools.DevPanelHandler()`, and `devTools.DevToolsHandler()` APIs are **planned** but not yet implemented. For now, use the `window.__GOSPA__.globalState` object in the browser console to inspect live state.

---

## Request Logging

`fiber.RequestLoggerMiddleware()` is a no-op placeholder. For request logging, use Fiber's built-in logger:

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
