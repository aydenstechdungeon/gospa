# Realtime: SSE and WebSockets

GoSPA provides two primary realtime communication mechanisms: **Server-Sent Events (SSE)** for one-way server-to-client updates, and **WebSockets** for full-duplex state synchronization.

Implementation: `github.com/aydenstechdungeon/gospa/fiber` (`sse.go`).

## Server-Sent Events (SSE)

## Server-Side API

### `SSEConfig`

```go
type SSEConfig struct {
    EventBufferSize   int                      // per-client channel buffer (default 100)
    HeartbeatInterval time.Duration            // keepalive comments (default 30s; 0 = disable)
    OnConnect         func(*SSEClient)
    OnDisconnect      func(*SSEClient)
    PubSub            store.PubSub             // optional; defaults to in-memory
}
```

### `NewSSEBroker(config *SSEConfig) *SSEBroker`

Creates a new SSE broker. Pass `nil` to use defaults.

### `SetupSSE(app *fiber.App, broker *SSEBroker, basePath string, corsConfig *cors.Config)`

Registers SSE routes under `basePath`. Pass **`nil`** for `corsConfig` if you do not need CORS middleware on the SSE group.

| Route | Method | Description |
|-------|--------|-------------|
| `{basePath}/connect` | GET | EventSource stream; `clientId` query param selects/stores the client |
| `{basePath}/subscribe` | POST | JSON body subscribes a client to topics |
| `{basePath}/unsubscribe` | POST | JSON body unsubscribes |
| `{basePath}/stats` | GET | JSON `{"clientCount": ...}` |

```go
broker := fiber.NewSSEBroker(&fiber.SSEConfig{
    HeartbeatInterval: 30 * time.Second,
    OnConnect: func(c *fiber.SSEClient) {
        log.Printf("SSE client connected: %s", c.ID)
    },
})

fiber.SetupSSE(app.Fiber, broker, "/_sse", nil)
```

### `SSEClient`

```go
type SSEClient struct {
    ID          string
    Channel     chan SSEEvent
    ConnectedAt time.Time
    Metadata    map[string]any
    Topics      map[string]bool
}
```

### Broker methods

```go
broker.Subscribe(clientID string, topics ...string)
broker.Unsubscribe(clientID string, topics ...string)
broker.Send(clientID string, event SSEEvent) bool   // false if client missing / full
broker.Broadcast(event SSEEvent) int                // number of clients notified
broker.BroadcastToTopic(topic string, event SSEEvent) int
broker.Disconnect(clientID string)
broker.ClientCount() int
```

### `SSEEvent`

```go
type SSEEvent struct {
    ID    string // Optional event ID (for client reconnect resume)
    Event string // Event type (e.g. "notification", "update")
    Data  any    // Payload — will be JSON-encoded
    Retry int    // Optional retry delay (ms)
}
```

### `SSEHelper` (High-Level)

```go
helper := fiber.NewSSEHelper(broker)

// Send a notification
helper.Notify(clientID, map[string]string{"message": "Hello!"})

// Broadcast to all clients
helper.NotifyAll("System maintenance in 5 minutes")

// Send a state update
helper.Update(clientID, "count", 42)

// Send an alert
helper.Alert(clientID, "warning", "Low disk space")

// Report progress (0–100)
helper.Progress(clientID, 75, "Uploading...")
```

---

## Security

> **IMPORTANT:** `SSESubscribeHandler` verifies that the target `clientId` is connected, but does **NOT** verify that the HTTP requester is that client. Any caller who knows another client's ID can subscribe it to arbitrary topics.
>
> Always protect **subscribe/unsubscribe** HTTP handlers with authentication that ties the request to the `clientId` being modified. `SetupSSE` does not add auth for you—use Fiber groups/middleware around routes that need it, or validate inside your app before exposing `POST` subscribe URLs publicly.

---

## Client-Side

The browser uses the native `EventSource` API to connect and receive events. GoSPA does not ship a client-side SSE wrapper — the native API is sufficient:

```typescript
const clientId = crypto.randomUUID();

// Connect
const es = new EventSource(`/_sse/connect?clientId=${clientId}`);

// Generic message
es.onmessage = (ev) => console.log('message:', ev.data);

// Named event
es.addEventListener('notification', (ev) => {
    const data = JSON.parse(ev.data);
    showToast(data.message);
});

es.addEventListener('error', () => {
    console.warn('SSE connection lost, browser will reconnect automatically');
});
```

### Topic Subscription from Client

```typescript
// Subscribe to topics after connecting
await fetch('/_sse/subscribe', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ clientId, topics: ['alerts', 'updates'] }),
});
```

---

## Full Example

```go
package main

import (
    "fmt"
    "time"
    "github.com/aydenstechdungeon/gospa"
    "github.com/aydenstechdungeon/gospa/fiber"
)

func main() {
    app := gospa.New(gospa.Config{RoutesDir: "./routes"})

    broker := fiber.NewSSEBroker(&fiber.SSEConfig{
        HeartbeatInterval: 30 * time.Second,
        OnConnect: func(c *fiber.SSEClient) {
            fmt.Printf("Client connected: %s\n", c.ID)
        },
    })

    fiber.SetupSSE(app.Fiber, broker, "/_sse", nil)

    // Push an event from a background task
    go func() {
        for {
            time.Sleep(5 * time.Second)
            broker.Broadcast(fiber.SSEEvent{
                Event: "tick",
                Data:  map[string]int{"count": int(time.Now().Unix())},
            })
        }
    }()

    _ = app.Run(":3000")
}
```

---

## Heartbeat

When `HeartbeatInterval > 0`, the broker sends a comment line (`: heartbeat`) at the configured interval. This keeps proxies from closing idle connections. The browser `EventSource` ignores comment lines; they are invisible to `onmessage` handlers.

---

## WebSockets

WebSockets in GoSPA are primarily used for **State Synchronization**. When enabled, any state object marked as synced will automatically replicate changes between the server and all connected clients.

### High-Performance Serialization

GoSPA uses specialized serialization libraries to ensure minimal latency and CPU overhead:

1. **JSON (Default)**: Powered by `goccy/go-json`, which is significantly faster than the Go standard library's `encoding/json`.
2. **MessagePack**: A binary serialization format that reduces payload sizes and improves parsing performance. Enable it with:

```go
app := gospa.New(gospa.Config{
    SerializationFormat: "msgpack",
})
```

### Bandwidth Optimization

For large state objects or frequent updates, GoSPA provides built-in optimizations:

- **State Diffing**: Only transmits changed keys instead of the entire object.
- **GZIP Compression**: Automatically compresses large payloads using the browser's `DecompressionStream` API.

```go
app := gospa.New(gospa.Config{
    StateDiffing:  true,
    CompressState: true,
})
```

### Stability and Reliability

The GoSPA WebSocket client includes:
- **Auto-Reconnect**: Exponential backoff on connection loss.
- **Message Batching**: Coalesces rapid state changes into single network frames.
- **Binary Tags**: Strict field mapping for MessagePack ensures cross-language compatibility.
