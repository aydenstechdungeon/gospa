# WebSocket & Real-time State Synchronization

GoSPA provides a powerful real-time system for synchronizing state between the server and all connected clients.

## High-Performance Broadcasting

The GoSPA WebSocket Hub is optimized for high-performance, topic-based broadcasting. Instead of iterating over all connected clients (which is O(N)), GoSPA uses a topic-based indexing system (O(1)) for targeted updates.

### Subscribing to Topics

Clients can subscribe to specific topics to receive targeted updates. This is managed automatically for reactive runes but can also be manually controlled:

```javascript
// Client-side subscription
ws.subscribe('topic-name');
```

### Server-side Broadcasting

On the server, you can broadcast messages to specific topics or sessions:

```go
// Broadcast to a specific topic
app.Hub.BroadcastToTopic("user-notifications", []byte(`{"type":"alert", "message":"New Message!"}`))

// Broadcast to a specific session
app.Hub.BroadcastToSession(sessionID, []byte(`{"type":"sync", "state":{...}}`))
```

## State Patching

GoSPA uses an efficient state patching mechanism. Instead of sending the full state on every change, only the modified keys and values are transmitted over the wire.

### Compression

WebSocket messages can be compressed using Gzip to reduce bandwidth usage. This is configurable in your `gospa.Config`:

```go
app := gospa.New(gospa.Config{
    EnableWebSocket: true,
    WSMaxMessageSize: 64 * 1024,
    CompressState: true, // Enable Gzip compression
})
```

## Security

The WebSocket system includes several security features to prevent attacks:

- **Rate Limiting**: Connections and actions are rate-limited per IP to prevent DoS attacks.
- **CSRF Protection**: WebSocket upgrades require a valid CSRF token.
- **JSON Depth Validation**: Prevents recursive JSON bombs by limiting the nesting depth of inbound messages.
- **Header Redaction**: Sensitive headers like `Authorization` and `Cookie` are automatically redacted in development error overlays.

## Scalability

GoSPA's WebSocket Hub is designed to scale horizontally. By using a `store.PubSub` backend (like Redis), broadcasts are automatically synchronized across multiple application processes and nodes.
