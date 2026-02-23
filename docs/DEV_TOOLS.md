# GoSPA Development Tools

GoSPA provides comprehensive development tools for debugging, hot reloading, and state inspection during development.

## Overview

The development tools in `fiber/dev.go` provide:

- **DevConfig**: Configuration for development mode
- **FileWatcher**: Hot reload on file changes
- **DevTools**: Development utilities and debugging
- **DebugMiddleware**: Request/response debugging
- **StateInspector**: Real-time state inspection

---

## DevConfig

Configuration options for development mode.

### Creating DevConfig

```go
import "github.com/gospa/gospa/fiber"

config := fiber.DevConfig{
    Enabled:         true,
    HotReload:       true,
    WatchPaths:      []string{".", "./routes", "./components"},
    IgnorePatterns:  []string{"*.log", ".git", "node_modules"},
    PollInterval:    100 * time.Millisecond,
    Debounce:        300 * time.Millisecond,
    OnReload:        func() { fmt.Println("Reloading...") },
    StateInspector:  true,
    DebugHeaders:    true,
    LogRequests:     true,
    ErrorStack:      true,
}
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `Enabled` | `bool` | `false` | Enable development mode |
| `HotReload` | `bool` | `true` | Enable hot reload |
| `WatchPaths` | `[]string` | `["."]` | Paths to watch for changes |
| `IgnorePatterns` | `[]string` | `[]` | Patterns to ignore |
| `PollInterval` | `time.Duration` | `100ms` | File system poll interval |
| `Debounce` | `time.Duration` | `300ms` | Debounce reload events |
| `OnReload` | `func()` | `nil` | Callback on reload |
| `StateInspector` | `bool` | `true` | Enable state inspector |
| `DebugHeaders` | `bool` | `true` | Add debug headers |
| `LogRequests` | `bool` | `true` | Log all requests |
| `ErrorStack` | `bool` | `true` | Include stack traces in errors |

### Usage with App

```go
app := gospa.NewApp(gospa.Config{
    DevMode: true,
    DevConfig: fiber.DevConfig{
        HotReload:  true,
        WatchPaths: []string{"./routes", "./components"},
    },
})
```

---

## FileWatcher

Monitor file changes and trigger hot reload.

### Creating a FileWatcher

```go
watcher := fiber.NewFileWatcher(fiber.FileWatcherConfig{
    Paths:          []string{"./routes", "./components"},
    IgnorePatterns: []string{"*.log", ".git"},
    PollInterval:   100 * time.Millisecond,
    Debounce:       300 * time.Millisecond,
})

// Start watching
watcher.Start()

// Stop watching
defer watcher.Stop()
```

### Event Handling

```go
watcher.OnChange(func(event fiber.FileEvent) {
    fmt.Printf("File changed: %s (%s)\n", event.Path, event.Op)
    
    switch event.Op {
    case fiber.FileCreate:
        // Handle create
    case fiber.FileWrite:
        // Handle modify
    case fiber.FileRemove:
        // Handle delete
    case fiber.FileRename:
        // Handle rename
    }
})

watcher.OnError(func(err error) {
    fmt.Printf("Watcher error: %v\n", err)
})
```

### FileEvent

```go
type FileEvent struct {
    Path string      // File path
    Op   FileOp      // Operation type
    Time time.Time   // Event time
}

type FileOp int

const (
    FileCreate FileOp = iota
    FileWrite
    FileRemove
    FileRename
)
```

### Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `Start()` | `Start()` | Start watching |
| `Stop()` | `Stop()` | Stop watching |
| `AddPath()` | `AddPath(path string)` | Add path to watch |
| `RemovePath()` | `RemovePath(path string)` | Remove watched path |
| `OnChange()` | `OnChange(fn func(FileEvent))` | Register change handler |
| `OnError()` | `OnError(fn func(error))` | Register error handler |
| `IsRunning()` | `IsRunning() bool` | Check if running |

---

## DevTools

Development utilities for debugging.

### Creating DevTools

```go
tools := fiber.NewDevTools(fiber.DevToolsConfig{
    StateKey:    "state",
    LogState:    true,
    LogEvents:   true,
    Performance: true,
})
```

### State Logging

```go
// Log current state
tools.LogState(stateMap)

// Log state diff
tools.LogStateDiff(oldState, newState)

// Log state change
tools.OnStateChange(func(key string, oldValue, newValue any) {
    fmt.Printf("State changed: %s = %v (was %v)\n", key, newValue, oldValue)
})
```

### Event Logging

```go
// Log all events
tools.LogEvents(true)

// Custom event handler
tools.OnEvent(func(event fiber.Event) {
    fmt.Printf("Event: %s -> %v\n", event.Name, event.Data)
})
```

### Performance Tracking

```go
// Start performance tracking
tools.StartPerf("operation")

// End and log
duration := tools.EndPerf("operation")
fmt.Printf("Operation took: %v\n", duration)

// Get all metrics
metrics := tools.GetPerfMetrics()
for name, metric := range metrics {
    fmt.Printf("%s: avg=%v, count=%d\n", name, metric.Avg, metric.Count)
}
```

### DevTools Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `LogState()` | `LogState(state *state.StateMap)` | Log state snapshot |
| `LogStateDiff()` | `LogStateDiff(old, new *state.StateMap)` | Log state changes |
| `OnStateChange()` | `OnStateChange(fn func(string, any, any))` | State change handler |
| `LogEvents()` | `LogEvents(enabled bool)` | Enable event logging |
| `OnEvent()` | `OnEvent(fn func(Event))` | Event handler |
| `StartPerf()` | `StartPerf(name string)` | Start timing |
| `EndPerf()` | `EndPerf(name string) time.Duration` | End timing |
| `GetPerfMetrics()` | `GetPerfMetrics() map[string]PerfMetric` | Get all metrics |
| `ResetPerf()` | `ResetPerf()` | Reset metrics |

---

## StateLogEntry

Individual state change log entry.

```go
type StateLogEntry struct {
    Timestamp time.Time   `json:"timestamp"`
    Key       string      `json:"key"`
    OldValue  any         `json:"oldValue"`
    NewValue  any         `json:"newValue"`
    Source    string      `json:"source"`  // "user", "system", "remote"
}
```

### State History

```go
// Get state history
history := tools.GetStateHistory()
for _, entry := range history {
    fmt.Printf("[%s] %s: %v -> %v\n", 
        entry.Timestamp, entry.Key, entry.OldValue, entry.NewValue)
}

// Clear history
tools.ClearStateHistory()

// Export history
json, _ := tools.ExportStateHistory()
```

---

## DebugMiddleware

Middleware for request/response debugging.

### Basic Usage

```go
app.Use(fiber.DebugMiddleware(fiber.DebugConfig{
    LogRequest:  true,
    LogResponse: true,
    LogHeaders:  true,
    LogBody:     false,  // Be careful with sensitive data
}))
```

### Configuration

```go
type DebugConfig struct {
    LogRequest   bool          // Log incoming requests
    LogResponse  bool          // Log outgoing responses
    LogHeaders   bool          // Include headers
    LogBody      bool          // Include body (caution!)
    MaxBodySize  int           // Max body size to log
    SkipPaths    []string      // Paths to skip
    RequestID    bool          // Add request ID
    Timing       bool          // Add timing headers
}
```

### Example Output

```
[DEBUG] Request: POST /api/users
  Headers:
    Content-Type: application/json
    X-Request-ID: abc123
  Body: {"name": "John"}
[DEBUG] Response: 201 Created (12.5ms)
  Headers:
    Content-Type: application/json
  Body: {"id": 1, "name": "John"}
```

---

## StateInspectorMiddleware

Middleware for real-time state inspection.

### Setup

```go
app.Use(fiber.StateInspectorMiddleware(fiber.StateInspectorConfig{
    StateKey:   "state",
    Endpoint:   "/__state",      // Inspection endpoint
    WebSocket:  "/__state/ws",   // WebSocket for live updates
    Auth:       true,            // Require auth in production
}))
```

### Accessing State Inspector

Navigate to `/__state` in your browser to see:

- Current state snapshot
- State change history
- Real-time updates via WebSocket
- State diff visualization

### WebSocket Protocol

```javascript
// Connect to state inspector
const ws = new WebSocket('ws://localhost:3000/__state/ws');

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    console.log('State update:', data);
    // { type: 'state_change', key: 'count', oldValue: 0, newValue: 1 }
};
```

---

## Hot Reload Integration

### Server-Side Hot Reload

```go
// In development mode
if config.DevMode {
    watcher := fiber.NewFileWatcher(fiber.FileWatcherConfig{
        Paths: []string{"./routes", "./components"},
    })
    
    watcher.OnChange(func(event fiber.FileEvent) {
        // Rebuild templates
        templ.ClearCache()
        
        // Notify clients
        fiber.BroadcastReload()
    })
    
    watcher.Start()
}
```

### Client-Side Hot Reload

The client runtime automatically connects to the hot reload WebSocket:

```typescript
// In client runtime
if (import.meta.hot) {
    import.meta.hot.on('reload', () => {
        window.location.reload();
    });
}
```

---

## Complete Example

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "github.com/gospa/gospa"
    "github.com/gospa/gospa/fiber"
    "github.com/gospa/gospa/state"
)

func main() {
    // Create app with dev config
    app := gospa.NewApp(gospa.Config{
        DevMode: true,
        DevConfig: fiber.DevConfig{
            HotReload:      true,
            WatchPaths:     []string{"./routes", "./components"},
            IgnorePatterns: []string{"*.log", ".git", "node_modules"},
            StateInspector: true,
            DebugHeaders:   true,
            LogRequests:    true,
            ErrorStack:     true,
            OnReload: func() {
                log.Println("Hot reload triggered")
            },
        },
    })
    
    // Setup dev tools
    devTools := fiber.NewDevTools(fiber.DevToolsConfig{
        StateKey:    "state",
        LogState:    true,
        LogEvents:   true,
        Performance: true,
    })
    
    // Track state changes
    devTools.OnStateChange(func(key string, oldVal, newVal any) {
        log.Printf("State: %s changed from %v to %v", key, oldVal, newVal)
    })
    
    // Add debug middleware
    app.Use(fiber.DebugMiddleware(fiber.DebugConfig{
        LogRequest:  true,
        LogResponse: true,
        LogHeaders:  true,
        Timing:      true,
    }))
    
    // Add state inspector
    app.Use(fiber.StateInspectorMiddleware(fiber.StateInspectorConfig{
        StateKey:  "state",
        Endpoint:  "/__state",
        WebSocket: "/__state/ws",
    }))
    
    // Example route with performance tracking
    app.Get("/api/data", func(c *fiber.Ctx) error {
        devTools.StartPerf("fetch_data")
        defer devTools.EndPerf("fetch_data")
        
        // ... handle request
        return c.JSON(data)
    })
    
    // Start server
    log.Fatal(app.Listen(":3000"))
}
```

---

## Best Practices

1. **Disable in production**: Always set `DevMode: false` in production
2. **Secure state inspector**: Use auth for state inspector endpoint
3. **Limit watch paths**: Only watch necessary directories
4. **Use debounce**: Prevent excessive reloads with proper debounce
5. **Log selectively**: Don't log sensitive data (bodies, headers)
6. **Monitor performance**: Use performance tracking to identify bottlenecks
7. **Clear history**: Periodically clear state history to prevent memory issues
8. **Handle errors**: Always handle watcher errors gracefully

---

## Troubleshooting

### Hot Reload Not Working

1. Check if `DevMode` is enabled
2. Verify watch paths are correct
3. Check ignore patterns aren't too broad
4. Ensure file system supports watching

### State Inspector Not Connecting

1. Verify WebSocket endpoint is accessible
2. Check for proxy/firewall issues
3. Ensure client runtime is loaded

### Performance Issues

1. Reduce logging verbosity
2. Increase poll interval
3. Limit watch paths
4. Disable features not needed
