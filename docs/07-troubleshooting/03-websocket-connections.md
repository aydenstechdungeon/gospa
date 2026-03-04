# Troubleshooting WebSocket & Real-Time Connections

## "WebSocket Connection Failed" Error

### Problem
The browser console shows:

```
WebSocket connection failed
Connection refused
```

Or the connection state remains `disconnected`.

### Cause
WebSocket support isn't enabled or properly configured on the server.

### Solution

#### 1. Enable WebSocket in Config

```go
app := gospa.New(gospa.Config{
    WebSocket: true,  // Enable WebSocket support
    WSPath: "/_gospa/ws",  // WebSocket endpoint path
})
```

#### 2. Ensure Middleware is Applied

```go
// Apply WebSocket middleware before starting server
app.Use(fiber.WebSocketMiddleware())
```

#### 3. Check Client Configuration

```javascript
import { initWebSocket } from '@gospa/runtime';

const ws = initWebSocket({
    url: 'ws://localhost:3000/_gospa/ws',
    reconnect: true,
    reconnectInterval: 1000,
    maxReconnectAttempts: 10,
});

await ws.connect();
```

---

## "Max Reconnect Attempts Reached"

### Problem
WebSocket repeatedly tries to connect but fails after max attempts:

```
[HMR] Disconnected
[HMR] Reconnecting... (1/10)
...
Max reconnect attempts reached
```

### Causes & Solutions

#### 1. Server Not Running
Ensure your Go server is started and listening on the correct port:

```bash
go run main.go
```

#### 2. Wrong URL/Path
Verify the WebSocket URL matches your server config:

```javascript
// Check your config
const wsUrl = window.location.protocol === 'https:' 
    ? 'wss://localhost:3000/_gospa/ws'  // HTTPS
    : 'ws://localhost:3000/_gospa/ws';   // HTTP
```

#### 3. Proxy/Router Blocking WebSocket
If behind Nginx or a load balancer, ensure WebSocket upgrade is allowed:

```nginx
# Nginx config
location /_gospa/ws {
    proxy_pass http://localhost:3000;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
}
```

#### 4. Firewall Blocking Port
Check that the WebSocket port (usually same as HTTP) isn't blocked:

```bash
# Test WebSocket connection
curl -i -N -H "Connection: Upgrade" \
     -H "Upgrade: websocket" \
     -H "Host: localhost:3000" \
     http://localhost:3000/_gospa/ws
```

---

## WebSocket Works on HTTP but Not HTTPS

### Problem
WebSocket connects fine on `http://localhost` but fails on `https://` production.

### Solution
Use `wss://` (secure WebSocket) for HTTPS sites:

```javascript
const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
const wsUrl = `${protocol}//${window.location.host}/_gospa/ws`;

const ws = initWebSocket({ url: wsUrl });
```

For self-signed certificates in development, you may need to accept the certificate first by visiting the HTTPS URL in your browser.

---

## "WebSocket is already in CONNECTING or OPEN state"

### Problem
Calling `connect()` multiple times throws an error or logs a warning.

### Solution
Check connection state before connecting:

```javascript
import { getWebSocket } from '@gospa/runtime';

const ws = getWebSocket();

// Only connect if not already connecting or connected
if (!ws || ws.getState() === 'disconnected') {
    await ws.connect();
}
```

Or use the singleton pattern:

```javascript
let wsPromise = null;

function ensureWebSocket() {
    if (!wsPromise) {
        wsPromise = initWebSocket({ url: 'ws://localhost:3000/_gospa/ws' })
            .connect();
    }
    return wsPromise;
}
```

---

## Messages Not Being Received

### Problem
WebSocket connects but no messages arrive from the server.

### Causes & Solutions

#### 1. Not Subscribed to Events

```javascript
const ws = initWebSocket({
    url: 'ws://localhost:3000/_gospa/ws',
    onMessage: (message) => {
        console.log('Received:', message);
    },
});
```

#### 2. Server Not Broadcasting
Ensure server is broadcasting messages:

```go
// Broadcast to all clients
app.Broadcast(map[string]interface{}{
    "type": "notification",
    "message": "Hello everyone!",
})

// Or broadcast state update
app.BroadcastState("counter", count.Get())
```

#### 3. Message Format Mismatch
Server and client must agree on message format:

```go
// Server sends
type StateMessage struct {
    Type        string      `json:"type"`
    ComponentID string      `json:"componentId,omitempty"`
    Key         string      `json:"key,omitempty"`
    Value       interface{} `json:"value,omitempty"`
}

app.Broadcast(StateMessage{
    Type:  "update",
    Key:   "count",
    Value: 42,
})
```

```javascript
// Client receives
ws.onMessage = (msg) => {
    if (msg.type === 'update' && msg.key === 'count') {
        console.log('Count is now:', msg.value);
    }
};
```

---

## Heartbeat/Keep-Alive Timeout

### Problem
Connection drops after periods of inactivity.

### Solution
Enable heartbeat in client config:

```javascript
const ws = initWebSocket({
    url: 'ws://localhost:3000/_gospa/ws',
    heartbeatInterval: 30000,  // Send ping every 30s
    heartbeatTimeout: 5000,    // Wait 5s for pong
});
```

Server-side configuration:

```go
app := gospa.New(gospa.Config{
    WebSocket:         true,
    WSHeartbeat:       30 * time.Second,
    WSHeartbeatTimeout: 10 * time.Second,
})
```

---

## Server-Sent Events (SSE) Not Working

### Problem
SSE connections fail or don't receive events.

### Solution

#### 1. Enable SSE in Config

```go
app := gospa.New(gospa.Config{
    EnableSSE: true,
    SSEPath:   "/_sse",
})
```

#### 2. Correct Client Usage

```javascript
const clientId = crypto.randomUUID();
const es = new EventSource(`/_sse/connect?clientId=${clientId}`);

es.onopen = () => console.log('SSE connected');

es.onmessage = (ev) => {
    console.log('Message:', JSON.parse(ev.data));
};

es.onerror = (err) => {
    console.error('SSE error:', err);
};

// Listen for named events
es.addEventListener('notification', (ev) => {
    const data = JSON.parse(ev.data);
    showToast(data.message);
});
```

#### 3. Check Event Format

```go
// Server must send proper SSE format
routing.SendToClient(clientId, routing.SSEEvent{
    Event: "notification",  // Event name (optional)
    Data:  map[string]string{"message": "Hello!"},
})
```

---

## WebSocket Authentication Issues

### Problem
WebSocket connects but server rejects messages due to authentication.

### Solution

#### 1. Use Middleware for Auth

```go
app.Use(func(c *fiber.Ctx) error {
    // Skip WebSocket upgrade path
    if strings.HasPrefix(c.Path(), "/_gospa/ws") {
        return c.Next()
    }
    
    // Regular auth check
    return authMiddleware(c)
})

// WebSocket-specific auth
app.Use(fiber.WebSocketMiddlewareWithAuth(func(token string) (bool, string) {
    userID, err := validateToken(token)
    return err == nil, userID
}))
```

#### 2. Send Auth Token on Connect

```javascript
const ws = initWebSocket({
    url: 'ws://localhost:3000/_gospa/ws',
    onOpen: () => {
        // Send auth token as first message
        ws.send({
            type: 'auth',
            token: localStorage.getItem('authToken'),
        });
    },
});
```

---

## Debugging WebSocket Connections

### Enable Debug Logging

```javascript
const ws = initWebSocket({
    url: 'ws://localhost:3000/_gospa/ws',
    debug: true,  // Logs all events to console
});
```

### Monitor in DevTools

1. **Network Tab**: Filter by `WS` to see WebSocket frames
2. **Console**: Look for `[GoSPA]` prefixed logs
3. **Application > Storage**: Check `sessionStorage` for queued messages

### Server-Side Logging

```go
app := gospa.New(gospa.Config{
    DevMode: true,  // Logs all WebSocket activity
})
```

---

## Common Mistakes

### Mistake 1: Blocking the Event Loop

```javascript
// BAD - Blocks other messages
ws.onMessage = (msg) => {
    heavyComputation(msg);  // Synchronous, blocks
};

// GOOD - Use async or yield
ws.onMessage = async (msg) => {
    await heavyComputation(msg);  // Non-blocking
};
```

### Mistake 2: Not Handling Reconnect

```javascript
// BAD - No reconnect logic
ws.onClose = () => {
    console.log('Connection closed');
};

// GOOD - Auto-reconnect enabled
const ws = initWebSocket({
    url: 'ws://localhost:3000/_gospa/ws',
    reconnect: true,
    maxReconnectAttempts: 10,
    onDisconnect: () => showReconnectingUI(),
    onConnect: () => hideReconnectingUI(),
});
```

### Mistake 3: Sending Before Connected

```javascript
// BAD - May fail if not connected yet
ws.send({ type: 'action', data: {} });

// GOOD - Wait for connection
await ws.connect();
ws.send({ type: 'action', data: {} });

// Or use queue (automatic in GoSPA)
const ws = initWebSocket({
    url: 'ws://localhost:3000/_gospa/ws',
    queueMessages: true,  // Queue until connected
});
```
