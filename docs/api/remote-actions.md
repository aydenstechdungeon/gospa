# Remote Actions

Remote Actions are a powerful way for your client-side code to invoke server-side Go functions. They are the primary way to perform mutations and complex logic in a GoSPA application.

## Registering Remote Actions

Remote Actions are registered on the server using `routing.RegisterRemoteAction`. All actions must have a name and a handler function.

```go
import "github.com/aydenstechdungeon/gospa/routing"

routing.RegisterRemoteAction("greet", func(ctx context.Context, rc routing.RemoteContext, input interface{}) (interface{}, error) {
    name := input.(string)
    return "Hello, " + name, nil
})
```

## Invoking From the Client

GoSPA provides a type-safe bridge for calling remote actions from your TypeScript code.

```typescript
import { remoteAction } from "/_gospa/runtime.js";

async function sayHello() {
    const result = await remoteAction("greet", "World");
    console.log(result); // "Hello, World"
}
```

## Security and Rate Limiting

### RemoteActionMiddleware (Production)
In production mode, GoSPA enforces the use of `RemoteActionMiddleware` to protect your actions. If an action is invoked without this middleware, the framework will block the request and return a `StatusUnauthorized` (401) error.

```go
app := gospa.New(gospa.Config{
    RemoteActionMiddleware: func(c fiber.Ctx) error {
        // Authenticate user before allowing the action
        if !isAuthenticated(c) {
            return c.Status(401).SendString("Unauthorized")
        }
        return c.Next()
    },
})
```

### Rate Limiting
All remote actions are automatically rate-limited per IP to prevent abuse and DoS attacks. The default limits (burst=50, refill=20/sec) can be customized:

```go
fiber.SetRemoteActionRateLimiter(100.0, 50.0)
```

## Type-Safe Bridge

By running `gospa generate`, the framework scans your Go code for `RegisterRemoteAction` calls and generates a TypeScript interface for all your action definitions. This ensures full type-safety for both inputs and outputs.

```typescript
// generated bridge
export interface RemoteActions {
    greet: (input: string) => Promise<string>;
}
```
