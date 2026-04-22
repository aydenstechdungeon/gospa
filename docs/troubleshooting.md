# Troubleshooting Runtime Initialization

## "GoSPA is not defined" Error

### Problem
When calling `GoSPA.remote()` or other GoSPA functions from inline scripts or event handlers, you get:

```
Uncaught ReferenceError: GoSPA is not defined
```

### Cause
The GoSPA runtime hasn't been initialized yet. The `window.GoSPA` global is created by the runtime's `init()` function, which is called automatically when the page loads if properly configured.

### Solution

#### 1. Ensure `data-gospa-auto` is on your `<html>` tag

The runtime checks for this attribute to auto-initialize:

```html
<!DOCTYPE html>
<html lang="en" data-gospa-auto>
<head>
    <title>My App</title>
</head>
<body>
    <!-- Your content -->
</body>
</html>
```

#### 2. Use GoSPA's root layout

If using a custom layout, ensure it includes the runtime script. The default root layout handles this automatically:

```templ
package routes

templ RootLayout() {
    <!DOCTYPE html>
    <html data-gospa-auto>
        <head>
            <title>My App</title>
        </head>
        <body>
            { children... }
            <!-- Runtime script is auto-injected here by GoSPA -->
        </body>
    </html>
}
```

#### 3. Manual initialization (advanced)

If not using `data-gospa-auto`, manually initialize in a module script:

```html
<script type="module">
    import * as GoSPA from '/_gospa/runtime.js';
    GoSPA.init({
        wsUrl: 'ws://localhost:3000/_gospa/ws',
        debug: false
    });
</script>
```

#### 4. Wait for DOM ready

If calling from a regular script tag, ensure the runtime has loaded:

```html
<script>
    // Wait for GoSPA to be available
    function waitForGoSPA(callback, maxAttempts = 50) {
        let attempts = 0;
        const check = () => {
            if (typeof GoSPA !== 'undefined') {
                callback();
            } else if (attempts < maxAttempts) {
                attempts++;
                setTimeout(check, 100);
            } else {
                console.error('GoSPA failed to load');
            }
        };
        check();
    }

    waitForGoSPA(() => {
        // Now safe to use GoSPA
        GoSPA.remote('myAction', {});
    });
</script>
```

### How GoSPA is Created

The `window.GoSPA` object is created in `client/src/runtime-core.ts` when `init()` is called:

```typescript
// Create the public GoSPA global object
const GoSPA = {
    config,
    components,
    globalState,
    init,
    createComponent,
    destroyComponent,
    getComponent,
    getState,
    setState,
    callAction,
    bind,
    autoInit,
    // Remote actions
    remote,
    remoteAction,
    configureRemote,
    getRemotePrefix,
    // State primitives
    get Rune() { return Rune; },
    get Derived() { return Derived; },
    get Effect() { return Effect; },
    // Utility functions
    batch,
    effect,
    watch,
    // Events
    get on() { return on; },
    get offAll() { return offAll; },
    get debounce() { return debounce; },
    get throttle() { return throttle; }
};

// Expose to window as the primary public API
(window as any).GoSPA = GoSPA;
```

## "__GOSPA__ is not defined" vs "GoSPA is not defined"

There are two different globals:

- **`window.GoSPA`** - The public API (what you should use)
- **`window.__GOSPA__`** - Internal debugging object (same content, different name)

Always use `GoSPA` (without underscores) in your application code.

## ES Module Alternative

Instead of relying on the global, import directly from the runtime:

```html
<script type="module">
    import * as GoSPA from '/_gospa/runtime.js';
    
    // No need for GoSPA global
    const result = await GoSPA.remote('myAction', {});
</script>
```
This approach:
- Works immediately (no waiting for init)
- Is tree-shakeable
- Works better with TypeScript
- Doesn't pollute global scope

## Checking Runtime Status

To verify the runtime loaded correctly:

```javascript
// In browser console
console.log(typeof GoSPA);  // Should print "object"
console.log(Object.keys(GoSPA));  // Should list available methods
```

## Common Mistakes

### Mistake 1: Calling before DOM is ready

```javascript
// BAD - Script runs before runtime loads
goSPA.remote('action', {});  // Note: wrong case too!

// GOOD - Wait for page to load
window.addEventListener('DOMContentLoaded', () => {
    GoSPA.remote('action', {});
});
```

### Mistake 2: Wrong case

```javascript
// BAD - Wrong case
goSPA.remote('action', {});
Gospa.remote('action', {});

// GOOD
GoSPA.remote('action', {});
```

### Mistake 3: Script type="module" without import

```html
<!-- BAD - Module scripts don't see globals automatically -->
<script type="module">
    GoSPA.remote('action', {});  // May fail
</script>

<!-- GOOD - Either import or use regular script -->
<script type="module">
    import * as GoSPA from '/_gospa/runtime.js';
    GoSPA.remote('action', {});
</script>

<!-- Or -->
<script>
    GoSPA.remote('action', {});
</script>
```

## CSP Nonce Errors ("Refused to execute script")

### Problem
You see CSP errors in the browser console and runtime/client scripts do not execute:

```text
Refused to execute inline script because it violates the following Content Security Policy directive...
```

### Cause
One of these is usually true:
- `script-src` is missing `'nonce-{nonce}'`.
- A custom `<script>` tag in your layout is missing the per-request nonce.
- A static/hardcoded nonce is being reused instead of the request nonce.

### Solution

#### 1. Configure CSP with the nonce placeholder

```go
cspPolicy := "default-src 'self'; script-src 'self' 'nonce-{nonce}'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self' wss: https:; frame-ancestors 'none'; base-uri 'self'; form-action 'self';"
app.Fiber.Use(gospafiber.SecurityHeadersMiddleware(cspPolicy))
```

#### 2. Add nonce to custom scripts in your layout

```templ
<script src="/static/js/islands.js" type="module" nonce={ gospatempl.GetNonce(ctx) }></script>
<script type="module" nonce={ gospatempl.GetNonce(ctx) }>
  // custom bootstrap logic
</script>
```

#### 3. Verify header and tag nonces match

Check DevTools:
- Response header `Content-Security-Policy` includes `script-src ... 'nonce-<value>'`.
- Rendered `<script>` tags on the same response use that same nonce value.
# Troubleshooting Remote Actions

## "ACTION_NOT_FOUND" Error

### Problem
Calling a remote action returns:

```json
{
    "error": "Remote action not found",
    "code": "ACTION_NOT_FOUND"
}
```

### Cause
The action hasn't been registered on the server yet.

### Solution

#### 1. Ensure action is registered in `init()` or package-level

```go
package routes

import (
    "context"
    "github.com/aydenstechdungeon/gospa/routing"
)

func init() {
    routing.RegisterRemoteAction("saveData", func(ctx context.Context, rc routing.RemoteContext, input any) (any, error) {
        return "saved", nil
    })
}
```

The `init()` function runs automatically when the package is imported. Make sure your routes package is imported in `main.go`:

```go
package main

import (
    _ "yourapp/routes"  // Import for side effects (init)
)
```

#### 2. Check action name matches exactly

```javascript
// Client calls 'saveData'
GoSPA.remote('saveData', {})

// Server must register 'saveData' (case-sensitive)
routing.RegisterRemoteAction("saveData", ...)
```

Names are case-sensitive. `saveData` ≠ `SaveData` ≠ `savedata`.

#### 3. Verify registration order

Remote actions must be registered before the server starts. Register them in `init()` functions which run before `main()`.

## "INVALID_JSON" Error

### Problem
```json
{
    "error": "Invalid input JSON",
    "code": "INVALID_JSON"
}
```

### Cause
The request body isn't valid JSON (parse error), or the body failed decoding after the nesting check.

### Solution
Ensure you're sending proper JSON:

```javascript
// BAD - Manually stringifying json which causes double stringification
GoSPA.remote('action', '{"name": "value"}')

// GOOD - This is automatically handled by GoSPA
GoSPA.remote('action', { name: "value" })

// If calling manually with fetch:
fetch('/_gospa/remote/action', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: "value" })  // Must stringify!
})
```

## "JSON_TOO_DEEP" Error

### Problem
```json
{
    "error": "JSON nesting too deep",
    "code": "JSON_TOO_DEEP"
}
```

### Cause
The remote action JSON body exceeds the maximum nesting depth (framework limit; currently **64** levels). Deeply nested objects are rejected before your handler runs.

### Solution
- Flatten payloads or split data across multiple actions.
- Do not send adversarial or accidentally recursive JSON structures.

## "REMOTE_AUTH_REQUIRED" Error (production)

### Problem
```json
{
    "error": "Remote actions require RemoteActionMiddleware in production",
    "code": "REMOTE_AUTH_REQUIRED"
}
```

### Cause
With `DevMode: false`, GoSPA blocks remote actions unless you either configure **`RemoteActionMiddleware`** or set **`AllowUnauthenticatedRemoteActions: true`** (intentionally public endpoints only).

### Solution
```go
app := gospa.New(gospa.Config{
    DevMode: false,
    RemoteActionMiddleware: func(c *fiber.Ctx) error {
        // your auth / session check
        return c.Next()
    },
})
```

## "REQUEST_TOO_LARGE" Error

### Problem
```json
{
    "error": "Request body too large",
    "code": "REQUEST_TOO_LARGE"
}
```

### Cause
The request body exceeds `MaxRequestBodySize` (default: 4MB).

### Solution
Increase the limit in your config:

```go
app := gospa.New(gospa.Config{
    MaxRequestBodySize: 10 * 1024 * 1024,  // 10MB
})
```

Or reduce your payload size by:
- Sending file uploads via direct POST instead of remote actions
- Compressing data before sending
- Chunking large payloads

## CSRF Token Errors

### Problem
```json
{
    "error": "CSRF token mismatch"
}
```

### Cause
CSRF protection is enabled but the token is missing/invalid.

### Solution

#### 1. Ensure middleware is in correct order

```go
app := gospa.New(gospa.Config{
    EnableCSRF: true,
})
```

With `EnableCSRF: true`, GoSPA wires the middleware automatically. You only need to add `CSRFSetTokenMiddleware()` and `CSRFTokenMiddleware()` yourself if you are building a custom Fiber stack outside the default app setup.

#### 2. Check cookies are enabled

The client reads the `csrf_token` cookie. If cookies are disabled, remote actions will fail.

#### 3. Verify token is being sent

Check browser dev tools:
1. Look for `csrf_token` cookie in Application → Cookies
2. Check that `X-CSRF-Token` header is sent in the request. The built-in `remote()` helper sends it automatically for same-origin requests.


## "unauthorized" Error (Global Remote Middleware)

### Problem
Your **`RemoteActionMiddleware`** returned an unauthorized response (exact JSON depends on your handler).

### Cause
A `RemoteActionMiddleware` blocked the request before the action handler ran (e.g. missing session).

### Solution
Make sure your middleware allows authenticated requests to continue:

```go
app := gospa.New(gospa.Config{
    RemoteActionMiddleware: func(c *fiber.Ctx) error {
        if c.Locals("user") == nil {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
        }
        return c.Next()
    },
})
```

## "ACTION_FAILED" Error

### Problem
```json
{
    "error": "Internal server error",
    "code": "ACTION_FAILED"
}
```

### Cause
Your action handler returned an error.

### Solution
Add logging to your action:

```go
routing.RegisterRemoteAction("processData", func(ctx context.Context, rc routing.RemoteContext, input any) (any, error) {
    log.Printf("Received input: %+v from IP: %s", input, rc.IP)
    
    data, ok := input.(map[string]any)
    if !ok {
        return nil, fmt.Errorf("expected object, got %T", input)
    }
    
    result, err := process(data)
    if err != nil {
        log.Printf("Process failed: %v", err)
        return nil, fmt.Errorf("processing failed: %w", err)
    }
    
    return result, nil
})
```

## Network Errors (status: 0)

### Problem
```javascript
{
    ok: false,
    status: 0,
    code: "NETWORK_ERROR",
    error: "Failed to fetch"
}
```

### Causes & Solutions

#### 1. Server is not running
Check that your Go server is running and accessible.

#### 2. Wrong URL/path
Default remote actions path is `/_gospa/remote/:name`:

```go
app := gospa.New(gospa.Config{
    RemotePrefix: "/_gospa/remote",  // Default
})
```

If you change this, ensure the client knows:

```javascript
import { configureRemote } from '@gospa/client';

configureRemote({ prefix: '/api/rpc' });
```

#### 3. CORS issues
If calling from a different origin:

```go
app := gospa.New(gospa.Config{
    AllowedOrigins: []string{"https://example.com"},
})
```

#### 4. Ad blockers
Some ad blockers may block requests to `/_gospa/*`. Try:
- Using a custom `RemotePrefix` without "gospa"
- Testing with ad blockers disabled

## Input Type Handling

### Problem
JSON numbers become `float64`, not `int`:

```go
routing.RegisterRemoteAction("add", func(ctx context.Context, rc routing.RemoteContext, input any) (any, error) {
    // BAD - This will panic!
    num := input.(int)  // JSON numbers are float64
    
    // GOOD - Handle float64
    num, ok := input.(float64)
    if !ok {
        return nil, errors.New("expected number")
    }
    return int(num) + 1, nil
})
```

### Solution
Use a struct with proper JSON tags:

```go
type AddInput struct {
    A int `json:"a"`
    B int `json:"b"`
}

routing.RegisterRemoteAction("add", func(ctx context.Context, rc routing.RemoteContext, input any) (any, error) {
    var data AddInput
    
    // Convert map to struct
    if err := mapstructure.Decode(input, &data); err != nil {
        return nil, err
    }
    
    return data.A + data.B, nil
})
```

## Timeouts

### Problem
```javascript
{
    ok: false,
    status: 0,
    code: "TIMEOUT",
    error: "Request timeout"
}
```

### Solution
Increase the timeout:

```javascript
const result = await GoSPA.remote('slowAction', data, {
    timeout: 60000  // 60 seconds (default is 30s)
});
```

Or make your action faster by:
- Moving heavy work to goroutines
- Using background jobs for long operations
- Returning immediately and polling for results

## Debugging Remote Actions

### Enable Debug Logging (Server)

```go
app := gospa.New(gospa.Config{
    DevMode: true,  // Enables request logging
})
```

### Log All Remote Action Calls

Wrap your actions with logging:

```go
func loggedAction(name string, fn routing.RemoteActionFunc) routing.RemoteActionFunc {
    return func(ctx context.Context, rc routing.RemoteContext, input any) (any, error) {
        log.Printf("[RemoteAction:%s] IP: %s Input: %+v", name, rc.IP, input)
        start := time.Now()
        
        result, err := fn(ctx, rc, input)
        
        log.Printf("[RemoteAction:%s] Duration: %v, Error: %v", 
            name, time.Since(start), err)
        return result, err
    }
}

routing.RegisterRemoteAction("process", loggedAction("process", func(ctx context.Context, rc routing.RemoteContext, input any) (any, error) {
    // Your logic here
    return nil, nil
}))
```

### Check Browser DevTools

1. **Network tab**: Look for the POST request to `/_gospa/remote/*`
2. **Console**: Look for JavaScript errors
3. **Application > Cookies**: Verify `csrf_token` exists

## Common Mistakes

### Mistake 1: Not handling context cancellation

```go
// BAD - May hang indefinitely
routing.RegisterRemoteAction("slow", func(ctx context.Context, rc routing.RemoteContext, input any) (any, error) {
    result := slowOperation()  // No timeout handling
    return result, nil
})

// GOOD - Respect context
routing.RegisterRemoteAction("slow", func(ctx context.Context, rc routing.RemoteContext, input any) (any, error) {
    done := make(chan any)
    go func() {
        done <- slowOperation()
    }()
    
    select {
    case result := <-done:
        return result, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
})
```

### Mistake 2: Not validating authorization

```go
// BAD - No auth check
routing.RegisterRemoteAction("deleteUser", func(ctx context.Context, rc routing.RemoteContext, input any) (any, error) {
    return deleteUser(input.(string)), nil
})

// GOOD - Check permissions (could check rc.SessionID or context)
routing.RegisterRemoteAction("deleteUser", func(ctx context.Context, rc routing.RemoteContext, input any) (any, error) {
    user := auth.GetUser(ctx)  // Extract from context
    if !user.IsAdmin {
        return nil, errors.New("unauthorized")
    }
    return deleteUser(input.(string)), nil
})
```
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

## Allow Insecure WebSockets in Production/Testing

### Problem
You see "CRITICAL: PublicOrigin must be set in production" errors, or WebSocket connection to `ws://...` fails because the browser blocks insecure connections on secure pages.

### Solution

In some testing or special production environments (e.g., local preview of a build), you may want to allow insecure `ws://` connections even when the framework thinks it's in production.

#### 1. Configuration (Recommended for specific builds)
Set `AllowInsecureWS: true` in your `gospa.Config`:

```go
app := gospa.New(gospa.Config{
    // ... other config
    AllowInsecureWS: true, // Allow ws:// even if not in DevMode
})
```

#### 2. Environment Variable (Quick override)
Set `GOSPA_WS_INSECURE=1` before starting your server:

```bash
# Run the built binary with insecure WebSocket allowed
GOSPA_WS_INSECURE=1 ./dist/server
```

#### 3. Allow Specific Ports (New in v0.1.32)

By default, GoSPA allows insecure `ws://` connections on port **3000** even if the page is served over HTTPS. You can configure this list:

```go
app := gospa.New(gospa.Config{
    // Allow insecure WS on port 3000 (default) and 8080
    AllowPortsWithInsecureWS: []int{3000, 8080},
})
```

This is particularly useful when using a service like Localtunnel or Ngrok that provides an HTTPS URL for your application, but your WebSocket connection is still being made to your local development machine.

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
# Troubleshooting HMR & Development Server

## "[HMR] Disconnected" or Connection Issues

### Problem
Browser console shows:

```
[HMR] Disconnected
[HMR] Reconnecting... (1/10)
Max reconnect attempts reached
```

### Causes & Solutions

#### 1. Dev Server Not Running

Ensure you're running in development mode:

```bash
go run main.go  # Dev mode is default
gospa dev       # Using CLI
```

#### 2. Wrong HMR Port/Path

Check your HMR configuration:

```go
app := gospa.New(gospa.Config{
    DevMode: true,
    HMR: gospa.HMRConfig{
        Enabled: true,
        WSPath:  "/_gospa/hmr/ws",
    },
})
```

Client automatically detects the path, but verify in DevTools Network tab.

#### 3. HTTPS/WSS Mismatch

For HTTPS development sites, ensure WSS is used:

```javascript
// Automatically handled by GoSPA, but verify:
const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
const hmrUrl = `${protocol}//${window.location.host}/_gospa/hmr/ws`;
```

---

## Full Page Reload Instead of Hot Update

### Problem
Changes trigger a full browser reload instead of seamless hot update.

### Causes & Solutions

| Symptom | Cause | Fix |
|---------|-------|-----|
| `.go` or `.templ` file changed | Requires server rebuild | Expected - Go templates need re-render |
| State lost on reload | No state preservation hook | Implement `window.__gospaPreserveState` |
| CSS changes reload page | File in `IgnorePaths` | Check HMR config |

#### Implementing State Preservation

```javascript
// In your app initialization
window.__gospaPreserveState = () => {
    // Return serializable state
    return {
        formData: document.getElementById('myForm').value,
        scrollPosition: window.scrollY,
    };
};

window.__gospaRestoreState = (state) => {
    // Restore after reload
    if (state.formData) {
        document.getElementById('myForm').value = state.formData;
    }
    if (state.scrollPosition) {
        window.scrollTo(0, state.scrollPosition);
    }
};
```

---

## CSS/Style Changes Not Applying

### Problem
Modifying CSS files doesn't trigger updates, or styles don't refresh.

### Solutions

#### 1. Check CSS is Being Watched

```go
app := gospa.New(gospa.Config{
    DevMode: true,
    HMR: gospa.HMRConfig{
        Enabled:     true,
        WatchPaths:  []string{"./static/css", "./styles"},
        IgnorePaths: []string{"./node_modules"},
    },
})
```

#### 2. Verify CSS Link Tag

Ensure CSS is linked directly (not via JS injection):

```html
<!-- Good - HMR can detect this -->
<link rel="stylesheet" href="/static/css/app.css">

<!-- Bad - HMR may miss this -->
<script>
    document.head.appendChild(Object.assign(
        document.createElement('link'),
        { rel: 'stylesheet', href: '/static/css/app.css' }
    ));
</script>
```

#### 3. Hard Reload if Needed

Sometimes cached styles interfere. Force clear:

```bash
# Clear browser cache
Ctrl+Shift+R  # Chrome/Windows
Cmd+Shift+R   # Chrome/Mac
```

---

## HMR Works for Some Files but Not Others

### Problem
Some file types update, others don't.

### Solutions

#### 1. Check File Extensions

GoSPA HMR watches these by default:
- `.go` - Server files (triggers rebuild)
- `.templ` - Template files (triggers rebuild)
- `.css` - Styles (hot update)
- `.js`, `.ts` - JavaScript (hot update)

Add custom extensions:

```go
app := gospa.New(gospa.Config{
    HMR: gospa.HMRConfig{
        Extensions: []string{".go", ".templ", ".css", ".scss", ".js", ".ts"},
    },
})
```

#### 2. Verify File is in Watched Directory

```go
// Add additional watch paths
app := gospa.New(gospa.Config{
    HMR: gospa.HMRConfig{
        WatchPaths: []string{
            "./routes",
            "./components",
            "./static",
            "./assets",  // Add custom paths
        },
    },
})
```

---

## Dev Server Port Already in Use

### Problem

```
listen tcp :3000: bind: address already in use
```

### Solutions

#### 1. Change Port in Config

```go
app := gospa.New(gospa.Config{
    Port: 3001,  // Use different port
})
```

#### 2. Kill Existing Process

```bash
# Find and kill process on port 3000
lsof -ti:3000 | xargs kill -9  # macOS/Linux
netstat -ano | findstr :3000   # Windows (then taskkill /PID <id>)
```

---

## Slow Dev Server Startup

### Problem
Development server takes too long to start.

### Solutions

#### 1. Disable Unused Features

```go
app := gospa.New(gospa.Config{
    DevMode:     true,
    WebSocket:   false,  // Disable if not needed
    EnableSSE:   false,  // Disable if not needed
    EnableCSRF:  false,  // Disable in dev
    CompressState: false, // Disable compression
})
```

#### 2. Reduce Watch Scope

```go
app := gospa.New(gospa.Config{
    HMR: gospa.HMRConfig{
        WatchPaths: []string{"./routes"},  // Narrow scope
        IgnorePaths: []string{
            "./node_modules",
            "./.git",
            "./tmp",
            "./vendor",
        },
    },
})
```

---

## "Too Many Open Files" Error

### Problem

```
fatal error: pipe failed: too many open files
```

### Solution
Increase file descriptor limit:

```bash
# macOS
ulimit -n 10240

# Linux (temporary)
ulimit -n 65536

# Linux (permanent) - add to /etc/security/limits.conf
* soft nofile 65536
* hard nofile 65536
```

---

## Template Changes Not Reflecting

### Problem
Modifying `.templ` files doesn't show changes.

### Solutions

#### 1. Ensure Templ is Installed

```bash
go install github.com/a-h/templ/cmd/templ@latest
```

#### 2. Regenerate Templates

```bash
templ generate  # Or use gospa CLI which handles this
gospa generate
```

#### 3. Check Generated Files Are Being Watched

```go
app := gospa.New(gospa.Config{
    HMR: gospa.HMRConfig{
        WatchPaths: []string{
            "./routes",
            "./_templ",  // Generated templates
        },
    },
})
```

---

## DevTools Shows No HMR WebSocket

### Problem
Can't find HMR WebSocket connection in DevTools.

### Debugging Steps

1. **Check Console for `[HMR]` logs**
   - Should see `[HMR] Connected` on successful connection

2. **Network Tab > WS Filter**
   - Look for WebSocket to `/_gospa/hmr/ws`
   - Check messages being sent/received

3. **Verify DevMode is Enabled**

```javascript
// In browser console
console.log(window.__GOSPA_HMR__);  // Should show HMR state
```

4. **Check for Errors**

```javascript
// Enable debug logging
window.__GOSPA_DEBUG__ = true;
```

---

## Environment Variables Not Reloading

### Problem
Changes to `.env` files require full server restart.

### Solution

This is expected behavior. GoSPA doesn't hot-reload environment variables. Use a restart:

```bash
# Use air for auto-restart on .env changes
air --build.cmd "go run main.go" --build.include_ext "go,templ,env"
```

Or implement config hot-reload:

```go
// Custom config watcher
func watchConfig(app *gospa.Application) {
    watcher, _ := fsnotify.NewWatcher()
    watcher.Add(".env")
    
    go func() {
        for event := range watcher.Events {
            if event.Op&fsnotify.Write == fsnotify.Write {
                // Reload config
                loadConfig()
            }
        }
    }()
}
```

---

## Common Mistakes

### Mistake 1: Production Build in Dev Mode

```go
// BAD - Forces dev mode even in production
app := gospa.New(gospa.Config{
    DevMode: true,  // Never hardcode in committed code
})

// GOOD - Use environment variable
app := gospa.New(gospa.Config{
    DevMode: os.Getenv("ENV") == "development",
})
```

### Mistake 2: Watching Node Modules

```go
// BAD - Watches too much
app := gospa.New(gospa.Config{
    HMR: gospa.HMRConfig{
        WatchPaths: []string{"."},  // Watches everything!
    },
})

// GOOD - Be specific
app := gospa.New(gospa.Config{
    HMR: gospa.HMRConfig{
        WatchPaths:  []string{"./routes", "./static"},
        IgnorePaths: []string{"./node_modules", "./.git"},
    },
})
```

### Mistake 3: Not Handling HMR Errors

```javascript
// BAD - Silent failures
import { initHMR } from '@gospa/runtime';
initHMR();

// GOOD - Handle errors
import { initHMR } from '@gospa/runtime';
initHMR({
    onError: (error) => {
        console.error('[HMR] Error:', error);
        // Optionally show user notification
    },
    onConnect: () => console.log('[HMR] Ready'),
    onDisconnect: () => console.warn('[HMR] Lost connection'),
});
```
# Troubleshooting Island Hydration & Components

## Island Not Hydrating

### Problem
The island element appears in the DOM but never becomes interactive. No `gospa:hydrated` event fires.

### Causes & Solutions

#### 1. Missing `data-gospa-island` Attribute

```html
<!-- BAD - No island attribute -->
<div id="counter">
    <button>Count: 0</button>
</div>

<!-- GOOD - Properly marked island -->
<div data-gospa-island="Counter" id="counter">
    <button>Count: 0</button>
</div>
```

#### 2. Missing Island Module

Ensure the island module exists at the expected path:

```javascript
// Default: /islands/{name}.js
// For data-gospa-island="Counter", expects:
// GET /islands/Counter.js
```

Check browser Network tab for 404 errors on island module requests.

#### 3. Island Manager Not Initialized

```javascript
// Ensure islands are initialized
import { initIslands } from '@gospa/runtime';

// Auto-initialization with data-gospa-auto on <html>
// Or manual:
const manager = initIslands({
    moduleBasePath: '/islands',
    defaultTimeout: 30000,
});
```

#### 4. JavaScript Error in Island Module

Check console for errors in the island's `hydrate` function:

```javascript
// /islands/Counter.js
export default {
    hydrate(element, props, state) {
        console.log('Hydrating counter:', element, props, state);
        // Check for errors here
    }
};
```

---

## "Island Hydration Timeout"

### Problem
Island takes too long to hydrate and times out.

### Causes & Solutions

#### 1. Large Module Size

```javascript
// BAD - Heavy imports blocking hydration
import { HeavyLibrary } from 'heavy-lib';

export default {
    async hydrate(element, props, state) {
        const lib = new HeavyLibrary();  // Slow initialization
    }
};

// GOOD - Lazy load heavy code
export default {
    async hydrate(element, props, state) {
        const { HeavyLibrary } = await import('heavy-lib');
        const lib = new HeavyLibrary();
    }
};
```

#### 2. Slow Network

Increase timeout for slow connections:

```javascript
const manager = initIslands({
    defaultTimeout: 60000,  // 60 seconds
});
```

Or for specific islands:

```html
<div 
    data-gospa-island="Chart" 
    data-gospa-timeout="60000"
>
```

---

## Island Hydrates Multiple Times

### Problem
Island `hydrate` function runs more than once, causing duplicate event listeners or state issues.

### Solution
Check if already hydrated before running:

```javascript
export default {
    hydrate(element, props, state) {
        // Prevent double hydration
        if (element.dataset.hydrated) return;
        element.dataset.hydrated = 'true';
        
        // Your hydration logic
        const button = element.querySelector('button');
        let count = state.count || 0;
        
        button.addEventListener('click', () => {
            count++;
            button.textContent = `Count: ${count}`;
        });
    }
};
```

Or use GoSPA's built-in protection:

```javascript
import { hydrateIsland } from '@gospa/runtime';

// This is idempotent - only hydrates once
await hydrateIsland('counter');
```

---

## Island Not Found in Lazy Mode

### Problem
Calling `hydrateIsland()` returns `null` or throws "Island not found".

### Causes & Solutions

#### 1. Wrong ID or Name

```javascript
// HTML
<div data-gospa-island="Counter" id="my-counter">

// JavaScript - Can use either:
await hydrateIsland('my-counter');  // By ID
await hydrateIsland('Counter');      // By island name
```

#### 2. Island Not in DOM

Ensure island exists before calling:

```javascript
import { getIslandManager } from '@gospa/runtime';

const manager = getIslandManager();
const island = manager.getIsland('my-counter');

if (island) {
    await manager.hydrateIsland(island);
} else {
    console.error('Island not found');
}
```

#### 3. Discover Islands After DOM Changes

For dynamically added islands:

```javascript
// Add new island to DOM
container.innerHTML = `
    <div data-gospa-island="DynamicChart" id="chart-1">
        <canvas></canvas>
    </div>
`;

// Re-scan for new islands
const manager = getIslandManager();
manager.discoverIslands();

// Now hydrate
await hydrateIsland('chart-1');
```

---

## Hydration Mode Not Working

### Problem
`data-gospa-mode` doesn't behave as expected (e.g., `visible` hydrates immediately).

### Solutions

#### Visible Mode Not Triggering

```html
<!-- Ensure element is initially in viewport or has threshold -->
<div 
    data-gospa-island="LazyImage"
    data-gospa-mode="visible"
    data-gospa-threshold="100"  <!-- Trigger 100px before visible -->
>
```

Check if `IntersectionObserver` is supported (polyfill for older browsers).

#### Idle Mode Never Triggering

```html
<!-- Add delay if needed -->
<div 
    data-gospa-island="Analytics"
    data-gospa-mode="idle"
    data-gospa-defer="1000"  <!-- Wait 1s after idle -->
>
```

Note: `requestIdleCallback` may never fire on busy main thread. Use with caution for critical UI.

#### Interaction Mode Issues

```html
<!-- Works with: mouseenter, touchstart, focusin, click -->
<div 
    data-gospa-island="Tooltip"
    data-gospa-mode="interaction"
>
```

If not triggering, check CSS doesn't block events:

```css
/* BAD - Blocks interaction events */
[data-gospa-island] {
    pointer-events: none;
}

/* GOOD - Allow events through */
[data-gospa-island] {
    pointer-events: auto;
}
```

---

## Props Not Available in Hydrate

### Problem
`props` parameter is empty or undefined in `hydrate()`.

### Solutions

#### 1. Check JSON Format

```html
<!-- BAD - Invalid JSON (single quotes) -->
<div data-gospa-props="{'initial': 10}">

<!-- GOOD - Valid JSON -->
<div data-gospa-props='{"initial": 10}'>
```

#### 2. Escape Quotes Properly

```html
<!-- In templ templates, use proper escaping -->
<div data-gospa-props={ fmt.Sprintf(`{"message": "%s"}`, escapedMessage) }>
```

#### 3. Access Props Safely

```javascript
export default {
    hydrate(element, props = {}, state = {}) {
        const initial = props.initial ?? 0;
        const message = props.message ?? 'Default';
        
        console.log('Props:', props);
        console.log('State:', state);
    }
};
```

---

## State Not Persisting Across Navigation

### Problem
Island state resets when navigating between pages.

### Solutions

#### 1. Use State Prop

```html
<div 
    data-gospa-island="Counter"
    data-gospa-state='{"count": 5}'
>
```

#### 2. Store in URL or Session

```javascript
export default {
    hydrate(element, props, state) {
        // Restore from URL
        const params = new URLSearchParams(window.location.search);
        let count = parseInt(params.get('count')) || state.count || 0;
        
        const button = element.querySelector('button');
        button.addEventListener('click', () => {
            count++;
            updateURL();
            render();
        });
        
        function updateURL() {
            const url = new URL(window.location);
            url.searchParams.set('count', count);
            window.history.replaceState({}, '', url);
        }
        
        function render() {
            button.textContent = `Count: ${count}`;
        }
        
        render();
    }
};
```

#### 3. Use Global State

```javascript
import { getState } from '@gospa/runtime';

export default {
    hydrate(element, props, state) {
        // Access global state that persists across navigation
        const globalCount = getState('counter');
    }
};
```

---

## Server-Only Island Still Tries to Hydrate

### Problem
`data-gospa-server-only` island attempts hydration anyway.

### Solution
Verify attribute value:

```html
<!-- These all work -->
<div data-gospa-island="Static" data-gospa-server-only></div>
<div data-gospa-island="Static" data-gospa-server-only="true"></div>
<div data-gospa-island="Static" data-gospa-server-only="1"></div>

<!-- Check island manager config -->
<script>
    initIslands({
        respectServerOnly: true,  // Default
    });
</script>
```

---

## Priority Queue Not Working

### Problem
`data-gospa-priority` doesn't affect hydration order.

### Solution
Priority only affects `immediate` mode islands:

```html
<!-- High priority - hydrates first -->
<div 
    data-gospa-island="Navigation"
    data-gospa-priority="high"
    data-gospa-mode="immediate"
>

<!-- Low priority - hydrates last -->
<div 
    data-gospa-island="FooterWidget"
    data-gospa-priority="low"
    data-gospa-mode="immediate"
>
```

Non-immediate modes (`visible`, `idle`, `interaction`, `lazy`) are not affected by priority.

---

## Common Mistakes

### Mistake 1: Modifying Props During Hydration

```javascript
// BAD - Props should be immutable
export default {
    hydrate(element, props) {
        props.count = 10;  // Don't modify props!
    }
};

// GOOD - Use local state
export default {
    hydrate(element, props) {
        let count = props.count || 0;  // Copy to local variable
        count = 10;  // OK to modify local
    }
};
```

### Mistake 2: Not Cleaning Up Event Listeners

```javascript
// BAD - Memory leak on re-hydration
export default {
    hydrate(element) {
        element.addEventListener('click', handler);
    }
};

// GOOD - Track for cleanup
export default {
    hydrate(element) {
        if (element._cleanup) element._cleanup();
        
        const handler = () => console.log('clicked');
        element.addEventListener('click', handler);
        
        element._cleanup = () => {
            element.removeEventListener('click', handler);
        };
    }
};
```

### Mistake 3: Expecting Island Before DOM Ready

```javascript
// BAD - May run before island is registered
document.addEventListener('DOMContentLoaded', () => {
    hydrateIsland('counter');  // Might fail
});

// GOOD - Wait for manager ready
import { initIslands } from '@gospa/runtime';

const manager = initIslands();
manager.ready().then(() => {
    hydrateIsland('counter');
});
```
# Troubleshooting State Synchronization

## State Changes Not Syncing to Client

### Problem
Server state changes (Runes) don't appear on the client.

### Causes & Solutions

#### 1. WebSocket Not Enabled

```go
app := gospa.New(gospa.Config{
    WebSocket: true,  // Required for real-time sync
})
```

#### 2. Component Not Registered for Sync

```go
// Register component state for synchronization
stateMap := state.NewStateMap()
stateMap.Add("count", count)
stateMap.Add("user", userRune)

// In your handler, use SyncState
func HandlePage(c *routing.Context) error {
    return c.RenderWithState("page", data, stateMap)
}
```

#### 3. Not Broadcasting Changes

```go
// BAD - Set without broadcast
count.Set(42)

// GOOD - Broadcast to all clients
app.BroadcastState("count", count.Get())

// Or use automatic sync
count.Set(42)
app.SyncComponentState("counter-component", stateMap)
```

---

## Client Changes Not Reaching Server

### Problem
Client state updates don't propagate to the server.

### Solutions

#### 1. Use Synced Rune

```javascript
import { syncedRune, getWebSocket } from '@gospa/runtime';

const ws = getWebSocket();
const count = syncedRune('count', 0, ws);

// This automatically sends to server
count.set(5);
```

#### 2. Manual State Sync

```javascript
import { getWebSocket } from '@gospa/runtime';

const ws = getWebSocket();

function updateServerState(key, value) {
    ws.send({
        type: 'state:update',
        key: key,
        value: value,
        componentId: 'my-component',
    });
}

// Usage
updateServerState('count', 42);
```

#### 3. Check WebSocket Connection

```javascript
const ws = getWebSocket();

if (!ws.isConnected()) {
    console.error('WebSocket not connected, state will not sync');
    await ws.connect();
}
```

---

## "State Message Too Large" Error

### Problem
WebSocket closes with error about message size.

### Cause
State exceeds 64KB limit (WebSocket frame limit).

### Solutions

#### 1. Enable State Diffing

```go
app := gospa.New(gospa.Config{
    WebSocket:   true,
    StateDiffing: true,  // Only send changes, not full state
})
```

#### 2. Prune Large Fields

```go
type User struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Avatar   string `json:"avatar,omitempty"`
    // Exclude large fields from sync
    History  []Action `json:"-"`  // Don't sync
}

// Or use a DTO for sync
type UserSyncDTO struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}
```

#### 3. Compress State

```go
app := gospa.New(gospa.Config{
    WebSocket:     true,
    CompressState: true,  // Compress before sending
})
```

#### 4. Paginate Large Lists

```go
// BAD - Sync entire list
items := state.NewRune(allItems)  // 1000+ items

// GOOD - Sync paginated view
pageItems := state.NewRune(allItems[:20])
totalPages := state.NewRune(len(allItems) / 20)
```

---

## Circular Reference Errors

### Problem
State fails to serialize due to circular references.

### Error
```
json: unsupported value: encountered a cycle
```

### Solutions

#### 1. Use JSON Tags to Exclude

```go
type Node struct {
    ID       string  `json:"id"`
    Value    int     `json:"value"`
    Parent   *Node   `json:"-"`  // Exclude from JSON
    Children []*Node `json:"children"`
}
```

#### 2. Create DTOs for Sync

```go
// Domain model with circular refs
type Category struct {
    ID       string
    Parent   *Category
    Children []*Category
}

// DTO for synchronization
type CategoryDTO struct {
    ID       string   `json:"id"`
    ParentID string   `json:"parentId"`
    Children []string `json:"children"` // Just IDs
}

// Convert before setting state
categoryRune.Set(toDTO(category))
```

#### 3. Use Custom Marshal

```go
type SafeNode struct {
    *Node
}

func (n SafeNode) MarshalJSON() ([]byte, error) {
    // Custom serialization that avoids cycles
    type nodeJSON struct {
        ID       string      `json:"id"`
        Value    int         `json:"value"`
        Children []nodeJSON  `json:"children"`
    }
    
    var convert func(*Node) nodeJSON
    convert = func(n *Node) nodeJSON {
        if n == nil {
            return nodeJSON{}
        }
        children := make([]nodeJSON, len(n.Children))
        for i, c := range n.Children {
            children[i] = convert(c)
        }
        return nodeJSON{
            ID:       n.ID,
            Value:    n.Value,
            Children: children,
        }
    }
    
    return json.Marshal(convert(n.Node))
}
```

---

## Race Conditions in State Updates

### Problem
Multiple simultaneous updates cause inconsistent state.

### Solutions

#### 1. Use Server-Side Batching

```go
state.Batch(func() {
    count.Set(1)
    name.Set("updated")
    items.Set(newItems)
    // All sent as single atomic update
})
```

#### 2. Optimistic Updates with Rollback

```javascript
import { optimisticUpdate } from '@gospa/runtime';

// Apply immediately, rollback on error
optimisticUpdate(count, 5, {
    onConfirm: () => console.log('Server confirmed'),
    onRollback: (error) => {
        console.error('Update failed, rolled back:', error);
    },
});
```

#### 3. Version Your State

```go
type VersionedState struct {
    Version int         `json:"version"`
    Data    interface{} `json:"data"`
}

func (v *VersionedState) Update(fn func(interface{}) interface{}) error {
    newData := fn(v.Data)
    
    // Check for conflicts
    if v.Version != currentVersion {
        return errors.New("state conflict detected")
    }
    
    v.Version++
    v.Data = newData
    return nil
}
```

---

## State Out of Sync After Reconnection

### Problem
Client reconnects to WebSocket but state is stale.

### Solutions

#### 1. Request Full Sync on Connect

```javascript
const ws = initWebSocket({
    url: 'ws://localhost:3000/_gospa/ws',
    onConnect: () => {
        // Request current state from server
        ws.send({ type: 'state:sync:request' });
    },
});
```

Server-side handler:

```go
func HandleStateSyncRequest(client *Client) {
    state := getCurrentState()
    client.Send(StateMessage{
        Type:  "state:sync:full",
        State: state,
    })
}
```

#### 2. Use Version Vectors

```go
type SyncState struct {
    Data    interface{}       `json:"data"`
    Version int64             `json:"version"`
    Vector  map[string]int64  `json:"vector"`  // Per-client versions
}

// On reconnect, client sends its last known version
// Server sends only changes since that version
```

---

## Memory Leaks from Subscriptions

### Problem
Component unmounts but subscriptions remain active.

### Solutions

#### 1. Always Unsubscribe

```javascript
import { getState } from '@gospa/runtime';

const count = getState('count');

// Subscribe
const unsubscribe = count.subscribe((value) => {
    updateUI(value);
});

// Cleanup on unmount
window.addEventListener('beforeunload', unsubscribe);

// Or in component lifecycle
function destroy() {
    unsubscribe();
}
```

#### 2. Use Effect for Auto-Cleanup

```javascript
import { Effect } from '@gospa/runtime';

const effect = new Effect(() => {
    console.log('Count:', count.get());
    
    // Cleanup function
    return () => {
        console.log('Effect cleaned up');
    };
});

effect.dependOn(count);

// Later
effect.dispose();  // Cleans up subscription
```

#### 3. WeakMap for Component State

```javascript
// Use WeakMap so state is GC'd when component is
const componentStates = new WeakMap();

function mountComponent(element) {
    const state = createComponentState();
    componentStates.set(element, state);
}

function unmountComponent(element) {
    // No need to manually delete - GC handles it
    // when element is removed from DOM
}
```

---

## Type Errors in State Values

### Problem
State value type differs between server and client.

### Solutions

#### 1. Consistent Types

```go
// Server - Use specific types, not interface{}
count := state.NewRune(0)  // int, not float64
```

```javascript
// Client - Match the type
const count = new Rune(0);  // number, default 0
```

#### 2. Validate on Receive

```javascript
ws.onMessage = (msg) => {
    if (msg.type === 'state:update') {
        const schema = stateSchemas[msg.key];
        const validated = schema.validate(msg.value);
        
        if (!validated.valid) {
            console.error('Invalid state received:', validated.errors);
            return;
        }
        
        updateState(msg.key, validated.value);
    }
};
```

#### 3. Use TypeScript

```typescript
// Define shared types
interface AppState {
    count: number;
    user: {
        id: string;
        name: string;
    } | null;
}

// Type-safe state access
function getState<K extends keyof AppState>(key: K): AppState[K] {
    return stateMap.get(key) as AppState[K];
}
```

---

## Common Mistakes

### Mistake 1: Mutating State Directly

```javascript
// BAD - Mutates without notification
const items = getState('items');
items.push(newItem);  // No notification!

// GOOD - Immutable update
const items = getState('items');
setState('items', [...items, newItem]);
```

### Mistake 2: Not Handling Disconnection

```javascript
// BAD - Updates fail silently when disconnected
function increment() {
    count.set(count.get() + 1);
}

// GOOD - Queue or notify
function increment() {
    if (!ws.isConnected()) {
        // Queue for later
        pendingUpdates.push({ key: 'count', value: count.get() + 1 });
        showOfflineWarning();
        return;
    }
    count.set(count.get() + 1);
}
```

### Mistake 3: Syncing Everything

```go
// BAD - Syncs all state including ephemeral
type ComponentState struct {
    User      User      // Sync
    FormDraft FormData  // Don't sync - ephemeral
    UIState   UIConfig  // Don't sync - client-only
}

// GOOD - Selective sync
type ComponentState struct {
    User User `json:"user"`  // Only sync user
    // Other fields excluded with `json:"-"`
}
```

### Mistake 4: Ignoring Conflicts

```go
// BAD - Last write wins
func UpdateState(key string, value interface{}) {
    state.Set(key, value)  // Ignores conflicts
}

// GOOD - Check for conflicts
func UpdateState(key string, value interface{}, clientVersion int) error {
    currentVersion := getVersion(key)
    
    if clientVersion < currentVersion {
        return fmt.Errorf("conflict: server has newer version %d vs %d", 
            currentVersion, clientVersion)
    }
    
    state.Set(key, value)
    incrementVersion(key)
    return nil
}
```
# Troubleshooting Build & Deployment

## Build Fails with "templ not found"

### Problem

```
exec: "templ": executable file not found in $PATH
```

### Solution

Install the templ CLI:

```bash
go install github.com/a-h/templ/cmd/templ@latest

# Verify installation
templ version
```

Ensure `$GOPATH/bin` is in your PATH:

```bash
# Add to ~/.bashrc, ~/.zshrc, etc.
export PATH=$PATH:$(go env GOPATH)/bin
```

---

## "undefined: gospa.Config" or Import Errors

### Problem
Build fails with undefined symbols or import errors.

### Causes & Solutions

#### 1. Missing go.mod Entry

```bash
# Ensure gospa is in go.mod
go get github.com/aydenstechdungeon/gospa

# Tidy dependencies
go mod tidy
```

#### 2. Wrong Package Name

```go
// BAD - Wrong import path
import "github.com/aydenstechdungeon/gospa/pkg/gospa"

// GOOD - Correct import
import "github.com/aydenstechdungeon/gospa"
```

#### 3. Version Mismatch

```bash
# Check installed version
go list -m github.com/aydenstechdungeon/gospa

# Update to latest
go get -u github.com/aydenstechdungeon/gospa
```

---

## Binary Size Too Large

### Problem
Compiled binary is hundreds of megabytes.

### Solutions

#### 1. Strip Debug Info

```bash
go build -ldflags="-s -w" -o app .
```

#### 2. Disable DWARF

```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o app .
```

#### 3. Use UPX Compression (Optional)

```bash
# Install UPX
apt-get install upx  # Debian/Ubuntu
brew install upx     # macOS

# Compress binary
upx --best -o app-compressed app
```

#### 4. Remove Unused Features

```go
app := gospa.New(gospa.Config{
    // Disable unused features
    WebSocket:     false,
    EnableSSE:     false,
    EnableCSRF:    false,
    SimpleRuntime: true,  // Use lighter runtime
})
```

---

## Template Not Found in Production

### Problem
App works in dev but templates fail in production build.

### Causes & Solutions

#### 1. Templates Not Generated

```bash
# Generate templates before build
templ generate

# Or use go:generate
//go:generate templ generate
```

#### 2. Missing Embedded Files

```go
// Ensure embed directive includes templates
//go:embed routes/*.templ
var templates embed.FS
```

#### 3. Template Path Issues

```go
// Use relative paths that work in both dev and production
app := gospa.New(gospa.Config{
    TemplateDir: "./routes",  // Relative to working dir
})

// Or use embed.FS for production
app := gospa.New(gospa.Config{
    TemplateFS: templates,
})
```

---

## Environment Variables Not Loading

### Problem
`os.Getenv()` returns empty in production.

### Solutions

#### 1. Load .env File

```go
import "github.com/joho/godotenv"

func init() {
    // Load .env in development only
    if os.Getenv("ENV") != "production" {
        godotenv.Load()
    }
}
```

#### 2. Set Variables in Production

```bash
# Systemd service
[Service]
Environment="DATABASE_URL=postgres://..."
Environment="PORT=8080"

# Docker
ENV DATABASE_URL=postgres://...
ENV PORT=8080

# Kubernetes
env:
  - name: DATABASE_URL
    valueFrom:
      secretKeyRef:
        name: db-secret
        key: url
```

---

## Static Assets 404 in Production

### Problem
CSS/JS files return 404 after deployment.

### Solutions

#### 1. Correct Static File Path

```go
// BAD - Relative path may not work
app.Static("/static", "./static")

// GOOD - Absolute path or embed
app.Static("/static", "/app/static")

// Or use embed
//go:embed static/*
var staticFiles embed.FS

app.StaticFS("/static", staticFiles)
```

#### 2. Check File Permissions

```bash
# Ensure files are readable
chmod -R 755 ./static

# Verify ownership
chown -R www-data:www-data ./static
```

#### 3. Configure CDN (if applicable)

```go
app := gospa.New(gospa.Config{
    StaticURL: "https://cdn.example.com",  // CDN URL
})
```

---

## Port Already in Use (Production)

### Problem

```
bind: address already in use
```

### Solutions

#### 1. Use Environment Port

```go
port := os.Getenv("PORT")
if port == "" {
    port = "3000"
}

app := gospa.New(gospa.Config{
    Port: port,
})
```

#### 2. Graceful Shutdown

```go
// Handle SIGTERM properly
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

go func() {
    <-quit
    log.Println("Shutting down...")
    app.Shutdown()
}()
```

#### 3. Kill Existing Process

```bash
# Find process using port
lsof -i :8080

# Kill it
kill -9 <PID>
```

---

## Docker Build Issues

### Problem
Docker build fails or image is too large.

### Solutions

#### 1. Multi-Stage Build

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go generate ./...
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main .

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/static ./static
EXPOSE 8080
CMD ["./main"]
```

#### 2. BuildKit for Faster Builds

```bash
DOCKER_BUILDKIT=1 docker build -t myapp .
```

#### 3. Cache Dependencies

```dockerfile
# Copy only dependency files first
COPY go.mod go.sum ./
RUN go mod download

# Then copy source (cache layer reused if deps unchanged)
COPY . .
```

---

## WebSocket Fails Behind Load Balancer

### Problem
WebSocket connections fail in production behind Nginx/ALB.

### Solutions

#### 1. Nginx Configuration

```nginx
location /_gospa/ws {
    proxy_pass http://backend;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    
    # Timeout settings
    proxy_read_timeout 86400;
    proxy_send_timeout 86400;
}
```

#### 2. AWS ALB Configuration

```yaml
# Target group settings
ProtocolVersion: HTTP/1.1
HealthCheckProtocol: HTTP
Stickiness: 
  Enabled: true
  Type: lb_cookie
```

#### 3. Enable Proxy Trust

```go
app := gospa.New(gospa.Config{
    EnableProxy: true,  // Trust X-Forwarded-* headers
})
```

---

## SSL/TLS Certificate Issues

### Problem
HTTPS fails or shows certificate warnings.

### Solutions

#### 1. Auto HTTPS (Let's Encrypt)

```go
app := gospa.New(gospa.Config{
    AutoTLS: true,
    TLSDomains: []string{"example.com", "www.example.com"},
})
```

#### 2. Custom Certificates

```go
app := gospa.New(gospa.Config{
    TLSCert: "/path/to/cert.pem",
    TLSKey:  "/path/to/key.pem",
})
```

#### 3. Reverse Proxy SSL (Nginx)

```nginx
server {
    listen 443 ssl http2;
    server_name example.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

---

## Memory Leaks in Production

### Problem
Memory usage grows continuously.

### Solutions

#### 1. Enable Pprof

```go
import _ "net/http/pprof"

func init() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
}

# Analyze
# go tool pprof http://localhost:6060/debug/pprof/heap
```

#### 2. Limit Concurrent Connections

```go
app := gospa.New(gospa.Config{
    MaxConnections: 1000,
    ReadTimeout:    30 * time.Second,
    WriteTimeout:   30 * time.Second,
})
```

#### 3. Clean Up WebSocket Clients

```go
// Set ping interval to detect dead connections
app := gospa.New(gospa.Config{
    WSHeartbeat:       30 * time.Second,
    WSHeartbeatTimeout: 10 * time.Second,
})
```

---

## "too many open files" in Production

### Problem
Server crashes with file descriptor limit errors.

### Solution

#### 1. Increase System Limits

```bash
# /etc/security/limits.conf
* soft nofile 65536
* hard nofile 65536

# Or in systemd service
[Service]
LimitNOFILE=65536
```

#### 2. Reduce Keep-Alive

```go
app := gospa.New(gospa.Config{
    IdleTimeout: 60 * time.Second,
})
```

---

## Database Connection Issues

### Problem
Database connections fail or timeout in production.

### Solutions

#### 1. Connection Pooling

```go
db, err := sql.Open("postgres", dsn)
if err != nil {
    log.Fatal(err)
}

// Configure pool
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(5 * time.Minute)
```

#### 2. Health Checks

```go
app.Get("/health", func(c *routing.Context) error {
    if err := db.Ping(); err != nil {
        return c.Status(503).JSON(map[string]string{
            "status": "unhealthy",
            "error": err.Error(),
        })
    }
    return c.JSON(map[string]string{"status": "healthy"})
})
```

---

## Common Mistakes

### Mistake 1: Hardcoded Development Values

```go
// BAD
app := gospa.New(gospa.Config{
    DevMode: true,  // Never in production!
    Port:    "3000",
})

// GOOD
app := gospa.New(gospa.Config{
    DevMode: os.Getenv("ENV") == "development",
    Port:    getEnv("PORT", "8080"),
})
```

### Mistake 2: No Request Timeouts

```go
// BAD - No timeouts
app := gospa.New(gospa.Config{})

// GOOD - Always set timeouts
app := gospa.New(gospa.Config{
    ReadTimeout:  30 * time.Second,
    WriteTimeout: 30 * time.Second,
    IdleTimeout:  120 * time.Second,
})
```

### Mistake 3: Logging Sensitive Data

```go
// BAD
log.Printf("User login: %s, password: %s", email, password)

// GOOD
log.Printf("User login attempt: %s", email)
// Never log passwords, tokens, or PII
```

### Mistake 4: Not Handling Panics

```go
// BAD - Panic crashes server
app.Get("/api/data", func(c *routing.Context) error {
    result := riskyOperation()  // May panic
    return c.JSON(result)
})

// GOOD - Recover from panics
app.Use(func(c *routing.Context) error {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Panic recovered: %v", r)
            c.Status(500).JSON(map[string]string{
                "error": "Internal server error",
            })
        }
    }()
    return c.Next()
})
```


---

## Running Quality & Security Checks Before Deploy

Use the repository script to run Bun + Go checks in one command:

```bash
./scripts/quality-check.sh
```

Default checks:
- `bun check` (root script)
- `gofmt` check
- `go vet`
- `staticcheck`
- `golangci-lint`
- `govulncheck`
- `go build`
- `go test`

Examples:

```bash
# Include race detector
./scripts/quality-check.sh --with-race

# Skip tools not installed in local env
./scripts/quality-check.sh --skip-golangci --skip-vulncheck --skip-staticcheck
```
