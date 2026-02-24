# Development Tools & Error Overlay

GoSPA includes an integrated suite of development tools to speed up the development cycle, including a powerful error overlay and a state inspector.

## Error Overlay

The error overlay provides a user-friendly interface for debugging server-side and client-side errors during development. It displays stack traces, source code snippets, and request information.

### Configuration

```go
config := fiber.ErrorOverlayConfig{
    Enabled:     true,
    ShowStack:   true,
    ShowRequest: true,
    Theme:       "dark",
    Editor:      "code", // Opens files in VS Code via vscode:// protocol
}

overlay := fiber.NewErrorOverlay(config)
```

### Usage

The overlay is automatically triggered by the framework's error handlers in development mode. When an error occurs, GoSPA renders a full-page HTML overlay instead of a generic 500 error.

---

## State Inspector

The State Inspector allows you to monitor reactive state changes across your application in real-time.

### Setup

```go
devTools := fiber.NewDevTools(fiber.DevConfig{
    Enabled: true,
    RoutesDir: "./routes",
})

// Add state inspector middleware
app.Use(fiber.StateInspectorMiddleware(devTools, config))

// Mount the dev panel UI
app.Get("/_gospa/dev", devTools.DevPanelHandler())
app.Get("/_gospa/dev/ws", devTools.DevToolsHandler()) // WS for real-time updates
```

### Features

- **Live Change Log**: See every `Rune` or `StateMap` update as it happens on both server and client.
- **Diff View**: Compare the "Before" and "After" state values.
- **Source Tracking**: Identify whether a state change originated from the "server" (via remote actions or SSR) or the "client".
- **Key Registry**: Browse all currently tracked reactive state keys.

---

## Debug Middleware

GoSPA provides a lightweight debug middleware that logs every request with its method, path, status, and processing time.

```go
app.Use(fiber.DebugMiddleware(devTools))
```

### Logging Format
`[GET] /docs/api 200 1.2ms`

---

## Client-Side Debugging

The runtime exposes internal state via the `window.__GOSPA__` object when the auto-init attribute is present.

```html
<html data-gospa-auto>
```

From the browser console, you can inspect:
- `__GOSPA__.components`: All active component instances.
- `__GOSPA__.globalState`: The root `StateMap`.
- `__GOSPA__.config`: Active runtime configuration.

### Manual Error Reporting

You can manually trigger the error overlay from the client:

```typescript
import { displayError } from '@gospa/runtime';

displayError({
    message: "Custom validation failed",
    type: "ValidationError",
    file: "main.ts",
    line: 42
});
```
