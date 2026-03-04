# Server-Sent Events (SSE)

GoSPA provides first-class support for Server-Sent Events (SSE) via the `fiber.SSEBroker`. SSE enables real-time, server-to-client push without the full bidirectionality of WebSockets — ideal for notifications, status updates, and live counters.

---

## Server-Side API

### `SSEConfig`

```go
type SSEConfig struct {
    // EventBufferSize is the channel buffer for pending events per client (default: 100)
    EventBufferSize int
    // HeartbeatInterval is how often a keepalive comment is sent (default: 30s; 0 = disable)
    HeartbeatInterval time.Duration
    // OnConnect is called when a client connects
    OnConnect func(client *SSEClient)
    // OnDisconnect is called when a client disconnects
    OnDisconnect func(client *SSEClient)
}
```

### `NewSSEBroker(config *SSEConfig) *SSEBroker`

Creates a new SSE broker. Pass `nil` to use defaults.

### `SetupSSE(app *fiber.App, broker *SSEBroker, prefix string, middleware ...fiber.Handler)`

Registers all SSE routes under a path prefix:

| Route | Handler | Description |
|-------|---------|-------------|
| `GET  {prefix}/connect` | `SSEHandler` | Client opens EventSource connection |
| `POST {prefix}/subscribe` | `SSESubscribeHandler` | Subscribe client to topics |
| `POST {prefix}/unsubscribe` | `SSEUnsubscribeHandler` | Unsubscribe client from topics |

```go
broker := fiber.NewSSEBroker(&fiber.SSEConfig{
    HeartbeatInterval: 30 * time.Second,
    OnConnect: func(c *fiber.SSEClient) {
        log.Printf("SSE client connected: %s", c.ID)
    },
})

fiber.SetupSSE(app.Fiber, broker, "/_sse")
```

### `SSEClient`

```go
type SSEClient struct {
    ID       string            // Unique client ID (UUID)
    Topics   []string          // Currently subscribed topics
    Metadata map[string]string // Arbitrary metadata (e.g. user ID, set on connect)
}
```

### Broker Methods

```go
// Subscribe clientID to one or more topics
broker.Subscribe(clientID string, topics ...string)

// Unsubscribe clientID from one or more topics
broker.Unsubscribe(clientID string, topics ...string)

// Send an event to a single client
broker.Send(clientID string, event SSEEvent) error

// Broadcast an event to all connected clients
broker.Broadcast(event SSEEvent)

// Broadcast an event to all clients subscribed to a topic
broker.BroadcastToTopic(topic string, event SSEEvent)

// Disconnect a client
broker.Disconnect(clientID string)
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
> Always place `SSESubscribeHandler` behind authentication middleware that validates the session identity matches the requested `clientId`:
>
> ```go
> fiber.SetupSSE(app.Fiber, broker, "/_sse", myAuthMiddleware)
> ```

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

    fiber.SetupSSE(app.Fiber, broker, "/_sse")

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

    app.Listen(":3000")
}
```

---

## Heartbeat

When `HeartbeatInterval > 0`, the broker sends a comment line (`: heartbeat`) at the configured interval. This keeps proxies from closing idle connections. The browser `EventSource` ignores comment lines; they are invisible to `onmessage` handlers.
