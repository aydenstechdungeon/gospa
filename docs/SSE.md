# Server-Sent Events (SSE)

GoSPA provides first-class support for Server-Sent Events (SSE), enabling real-time, one-way push notifications from the server to the client. This is ideal for status updates, notifications, and real-time state synchronization without the overhead of WebSockets.

## Server-Side (Go)

The SSE implementation is provided by the `fiber` package via the `SSEBroker`.

### Setup

```go
import (
    "github.com/aydenstechdungeon/gospa/fiber"
    "time"
)

func main() {
    // 1. Create a broker
    broker := fiber.NewSSEBroker(&fiber.SSEConfig{
        EventBufferSize:   100,
        HeartbeatInterval: 30 * time.Second,
        OnConnect: func(client *fiber.SSEClient) {
            fmt.Printf("Client connected: %s\n", client.ID)
        },
    })

    app := fiber.New()

    // 2. Setup SSE routes
    fiber.SetupSSE(app, broker, "/_sse", nil)
}
```

### Broker API

| Method | Description |
|--------|-------------|
| `NewSSEBroker(config)` | Creates a new broker instance. |
| `Connect(clientID, metadata)` | Manually registers a client. |
| `Disconnect(clientID)` | Removes a client. |
| `Subscribe(clientID, topics...)` | Adds a client to specific topics. |
| `Unsubscribe(clientID, topics...)` | Removes a client from topics. |
| `Send(clientID, event)` | Sends an event to a specific client. |
| `Broadcast(event)` | Sends an event to all connected clients. |
| `BroadcastToTopic(topic, event)` | Sends an event to clients in a topic. |

### SSEHelper

A high-level helper for common notification patterns:

```go
helper := fiber.NewSSEHelper(broker)

// Send a basic notification
helper.Notify(clientID, map[string]string{"message": "Hello!"})

// Broadcast to everyone
helper.NotifyAll("System maintenance in 5 minutes")

// Send state update
helper.Update(clientID, "count", 42)

// Send an alert
helper.Alert(clientID, "warning", "Low disk space")

// Report progress
helper.Progress(clientID, 75, "Uploading...")
```

---

## Client-Side (TypeScript)

The client runtime provides an `SSEClient` to consume events with automatic reconnection and heartbeat monitoring.

### Setup

```typescript
import { connectSSE } from '@gospa/runtime';

const sse = connectSSE('notifications', {
  url: '/_sse/connect',
  autoReconnect: true,
  debug: true
});
```

### Event Handling

```typescript
// Listen for generic messages
sse.onMessage((ev) => {
  console.log('Received:', ev.data);
});

// Listen for custom event types
sse.on('notification', (ev) => {
  showToast(ev.data.message);
});

// Listen for state updates
sse.on('update', (ev) => {
  const { key, value } = ev.data;
  console.log(`Update ${key} to ${value}`);
});

// Unsubscribe
const unsub = sse.on('alert', handleAlert);
unsub(); // Stop listening
```

### Connection Management

```typescript
sse.onStateChange((state) => {
  console.log('Connection status:', state); // 'connecting', 'connected', 'disconnected', 'error'
});

sse.disconnect();
sse.connect();
```

### SSE Configuration

| Option | Default | Description |
|--------|---------|-------------|
| `url` | - | The SSE endpoint URL. |
| `autoReconnect` | `true` | Automatically reconnect on failure. |
| `maxRetries` | `5` | Maximum reconnection attempts. |
| `reconnectDelay` | `1000` | Initial delay before reconnecting (ms). |
| `heartbeatInterval` | `30000` | Expected heartbeat interval from server. |
| `debug` | `false` | Enable verbose logging. |

### SSEManager (Advanced)

If you need multiple SSE connections, you can use the `getSSEManager()` API:

```typescript
import { getSSEManager } from '@gospa/runtime';

const manager = getSSEManager();
manager.setDefaultConfig({ autoReconnect: true });

const main = manager.client('main', { url: '/_sse/main' });
const logs = manager.client('logs', { url: '/_sse/logs' });
```
