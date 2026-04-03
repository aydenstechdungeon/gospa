# WebSocket and Performance Configuration

WebSocket settings and performance optimization options for GoSPA.

## WebSocket Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `EnableWebSocket` | `bool` | `true` | Enable real-time state synchronization via WebSocket |
| `WebSocketPath` | `string` | `"/_gospa/ws"` | Endpoint for WebSocket connections |
| `WebSocketMiddleware` | `fiber.Handler` | `nil` | Middleware to run before WebSocket upgrade |
| `WSReconnectDelay` | `time.Duration` | `0` | Delay before reconnecting; defaults to 1s in HTML |
| `WSMaxReconnect` | `int` | `0` | Max reconnection attempts; defaults to 10 in HTML |
| `WSHeartbeat` | `time.Duration` | `0` | Ping interval; defaults to 30s in HTML |
| `WSMaxMessageSize` | `int` | `65536` | Maximum payload size for WebSocket messages |
| `WSConnRateLimit` | `float64` | `1.5` | Refilling rate in connections per second |
| `WSConnBurst` | `float64` | `15.0` | Burst capacity for connection upgrades |

## Performance Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `CompressState` | `bool` | `false` | Enable zlib compression for WebSocket messages |
| `StateDiffing` | `bool` | `false` | Only send state diffs over WebSocket |
| `CacheTemplates` | `bool` | `false` | Enable template caching (recommended for production) |
| `SimpleRuntime` | `bool` | `false` | Use lightweight runtime without DOMPurify |
| `DisableSanitization` | `bool` | `false` | Trusts server-rendered HTML without DOMPurify |
| `NotificationBufferSize` | `int` | `1024` | Size of the state change notification queue |

## Example

```go
app := gospa.New(gospa.Config{
    EnableWebSocket:   true,
    WebSocketPath:     "/_gospa/ws",
    SerializationFormat: "msgpack",
    
    CompressState:  true,
    StateDiffing:   true,
    CacheTemplates: true,
})
```
