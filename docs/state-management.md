# State Management

GoSPA provides a unified state management system that synchronizes data between the server and all connected clients in real-time.

## Server-Side State

The `StateMap` is the core of server-side state. It contains a collection of named `Runes`.

```go
app := gospa.New(gospa.Config{
    DefaultState: map[string]interface{}{
        "user_name": "New User",
        "is_online": false,
    },
})

// Update state on the server
app.StateMap.Get("is_online").Set(true)
```

## Client-Side State

GoSPA automatically serializes and hydrates the initial state on the client.

```typescript
import { getState, setState } from "/_gospa/runtime.js";

const name = getState("user_name");
console.log(name); // "New User"

// Mutate state locally (re-renders only this client)
setState("user_name", "Updated User");
```

## Real-Time Synchronization

When the server state changes, GoSPA pushes the updates to all connected clients via WebSockets.

### Serialization
GoSPA supports both **JSON** and **MessagePack** serialization. MessagePack is recommended for high-performance applications with large state objects.

```go
app := gospa.New(gospa.Config{
    SerializationFormat: "msgpack",
})
```

### Batching and Efficiency
State updates are batched on the server to minimize network overhead. The client only receives "diffs" for the keys that have actually changed.

## Computed State

You can define computed (derived) state on the server that automatically updates based on other state variables.

```go
app.Computed("greeting", []string{"user_name"}, func(states map[string]interface{}) interface{} {
    return "Hello, " + states["user_name"].(string) + "!"
})
```

Whenever `user_name` changes, the `greeting` is re-calculated and pushed to all clients.

## Security and Privacy

GoSPA's state management includes several security measures:
1.  **Authorization**: Control who can read or write specific state keys via the `StateMiddleware`.
2.  **Redaction**: Sensitive headers and data are automatically filtered out before being sent to the dev error overlay.
3.  **Prototype Pollution Protection**: Client-side state hydration uses a safe JSON parsing utility that prevents property injection via `__proto__`.
