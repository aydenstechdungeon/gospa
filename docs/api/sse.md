# Server-Sent Events (SSE)

GoSPA provides a robust Server-Sent Events (SSE) system for real-time, one-way communication from the server to your clients. SSE is a lightweight alternative to WebSockets when bi-directional communication is not required.

## SSE Architecture

SSE in GoSPA is managed by the `SSEBroker`. It handles client connections, heartbeats to keep connections alive, and event distribution.

### Configuration
```go
app := gospa.New(gospa.Config{
    SSE: &fiber.SSEConfig{
        EventBufferSize:    100,
        HeartbeatInterval:  30 * time.Second,
        OnConnect:          func(c *fiber.SSEClient) { /* ... */ },
    },
})
```

## Sending Events

You can push events to specific topics or directly to individual clients.

```go
// Push to a global topic
app.SSE().Publish("notifications", fiber.SSEEvent{
    Event: "new_message",
    Data:  map[string]string{"from": "Alice", "text": "Hello!"},
})

// Push to a specific client
app.SSE().SendToClient(clientID, fiber.SSEEvent{
    Data: "Your account has been updated",
})
```

## Client-Side Consumption

GoSPA provides a standard way to consume SSE streams in your component's reactive logic.

```typescript
import { onMount } from "/_gospa/runtime.js";

onMount(() => {
    const eventSource = new EventSource("/_gospa/sse/notifications");
    
    eventSource.onmessage = (event) => {
        const data = JSON.parse(event.data);
        console.log("New SSE Event:", data);
    };

    return () => eventSource.close();
});
```

## Scalability and Distributed Environments

The `SSEBroker` can be configured with a `store.PubSub` backend (such as Redis) to synchronize events across multiple application instances. This ensures that a client connected to Server A will still receive events published by Server B.

```go
broker := fiber.NewSSEBroker(&fiber.SSEConfig{
    PubSub: store.NewRedisPubSub(redisClient),
})
```
