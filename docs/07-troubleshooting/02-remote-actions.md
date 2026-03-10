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

import "github.com/aydenstechdungeon/gospa/routing"

func init() {
    routing.RegisterRemoteAction("saveData", func(ctx context.Context, input any) (any, error) {
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
The request body isn't valid JSON.

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
```json
{
    "error": "unauthorized"
}
```

### Cause
A `RemoteActionMiddleware` blocked the request before the action handler ran.

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
routing.RegisterRemoteAction("processData", func(ctx context.Context, input any) (any, error) {
    log.Printf("Received input: %+v", input)
    
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
routing.RegisterRemoteAction("add", func(ctx context.Context, input any) (any, error) {
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

routing.RegisterRemoteAction("add", func(ctx context.Context, input any) (any, error) {
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
    return func(ctx context.Context, input any) (any, error) {
        log.Printf("[RemoteAction:%s] Input: %+v", name, input)
        start := time.Now()
        
        result, err := fn(ctx, input)
        
        log.Printf("[RemoteAction:%s] Duration: %v, Error: %v", 
            name, time.Since(start), err)
        return result, err
    }
}

routing.RegisterRemoteAction("process", loggedAction("process", func(ctx context.Context, input any) (any, error) {
    // Your logic here
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
routing.RegisterRemoteAction("slow", func(ctx context.Context, input any) (any, error) {
    result := slowOperation()  // No timeout handling
    return result, nil
})

// GOOD - Respect context
routing.RegisterRemoteAction("slow", func(ctx context.Context, input any) (any, error) {
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
routing.RegisterRemoteAction("deleteUser", func(ctx context.Context, input any) (any, error) {
    return deleteUser(input.(string)), nil
})

// GOOD - Check permissions
routing.RegisterRemoteAction("deleteUser", func(ctx context.Context, input any) (any, error) {
    user := auth.GetUser(ctx)  // Extract from context
    if !user.IsAdmin {
        return nil, errors.New("unauthorized")
    }
    return deleteUser(input.(string)), nil
})
```
