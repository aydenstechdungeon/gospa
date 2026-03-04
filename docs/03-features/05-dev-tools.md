# Hot Module Replacement (HMR)

GoSPA includes a built-in Hot Module Replacement (HMR) system that watches files for changes and notifies connected browser clients in real-time — without losing application state.

---

## How It Works

The HMR system has three parts:

1. **`HMRFileWatcher`** — polls the file system every 500ms for `.templ`, `.go`, `.ts`, `.js`, `.css`, `.html`, `.svelte`, `.vue`, `.jsx`, `.tsx` changes.
2. **`HMRManager`** — debounces change events, determines the update type, and broadcasts `HMRMessage` payloads over WebSocket to connected clients.
3. **Client script** — injected automatically by `HMRMiddleware`, connects over WebSocket and applies updates or triggers a full reload.

> **Note:** The file watcher uses polling (not `fsnotify`). For faster feedback loops, set `DebounceTime` to a lower value or replace the watcher with an inotify-based solution.

---

## Server-Side API

### `HMRConfig`

```go
type HMRConfig struct {
    Enabled      bool          // Enable the HMR system
    WatchPaths   []string      // Directories to watch (e.g. []string{"./routes", "./static"})
    IgnorePaths  []string      // Path substrings to ignore (e.g. []string{"node_modules", ".git"})
    DebounceTime time.Duration // Min time between two updates for the same file (default: 500ms)
    BroadcastAll bool          // Broadcast all updates, not just matching clients (unused currently)
}
```

### `NewHMRManager(config HMRConfig) *HMRManager`

Creates a new HMR manager. Does not start watching; call `Start()` to begin.

### `(*HMRManager) Start()`

Starts the file watcher and the change-processing goroutine.

### `(*HMRManager) Stop()`

Stops the file watcher and closes the change channel. Safe to call multiple times.

### `(*HMRManager) HMREndpoint() fiber.Handler`

Returns a Fiber handler that upgrades WebSocket connections for HMR clients. Register this at `/__hmr`.

```go
app.Get("/__hmr", hmr.HMREndpoint())
```

### `(*HMRManager) HMRMiddleware() fiber.Handler`

Returns middleware that injects the HMR client script into HTML responses. Must be used **after** the route handlers so the body is available.

```go
app.Use(hmr.HMRMiddleware())
```

### State Preservation

Clients can push state before a reload and retrieve it after:

```go
// Server: preserve module state sent from client
hmr.PreserveState(moduleID string, state any)

// Server: retrieve preserved state for a module
state, exists := hmr.GetState(moduleID string)

// Server: clear preserved state for a module
hmr.ClearState(moduleID string)
```

The client script sends `state-preserve` messages automatically via `window.__gospaPreserveState()` on `beforeunload`.

### `InitHMR(config HMRConfig) *HMRManager`

Initializes and stores a global HMR manager instance.

### `GetHMR() *HMRManager`

Returns the global HMR manager set by `InitHMR`.

---

## Message Types

| Type | Direction | Meaning |
|------|-----------|---------|
| `connected` | Server → Client | Welcome message on connect |
| `update` | Server → Client | A file changed; contains `path`, `moduleId`, `event`, `timestamp` |
| `reload` | Server → Client | Full page reload required |
| `error` | Server → Client | A server-side HMR error |
| `state-preserve` | Client → Server | Client is saving module state before reload |
| `state-request` | Client → Server | Client requests previously saved state |
| `error` | Client → Server | Client-side HMR error logged to server |

---

## Complete Setup Example

```go
package main

import (
    "time"
    "github.com/aydenstechdungeon/gospa"
    "github.com/aydenstechdungeon/gospa/fiber"
)

func main() {
    app := gospa.New(gospa.Config{
        RoutesDir: "./routes",
        DevMode:   true,
    })

    if app.Config.DevMode {
        hmr := fiber.InitHMR(fiber.HMRConfig{
            Enabled:      true,
            WatchPaths:   []string{"./routes", "./static"},
            IgnorePaths:  []string{"node_modules", ".git", "_templ.go"},
            DebounceTime: 500 * time.Millisecond,
        })
        hmr.Start()
        defer hmr.Stop()

        // WebSocket endpoint for HMR clients
        app.Fiber.Get("/__hmr", hmr.HMREndpoint())

        // Inject HMR script into HTML pages
        app.Fiber.Use(hmr.HMRMiddleware())
    }

    app.Listen(":3000")
}
```

---

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| Full reload instead of hot update | `.go` or `.templ` file changed requiring rebuild | Expected — Go templates require re-render |
| HMR doesn't connect over HTTPS | Script used `ws://` (now fixed — uses `wss://` on HTTPS) | Ensure you're on a recent GoSPA version |
| Changes not detected | File not in `WatchPaths` or path matches `IgnorePaths` | Review config; check console for `[HMR] Connected` |
| State lost across reload | `window.__gospaPreserveState` not implemented | Implement the state preservation hook |
