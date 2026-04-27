# Error Handling

GoSPA provides a robust error handling system designed to maintain application reliability across server and client boundaries.

## Server-Side Errors

### Error Boundaries
GoSPA uses a `withErrorBoundary` utility to wrap routes and components. If a component crashes during rendering, the framework will catch the error and display a fallback component.

```go
routing.RegisterError("/dashboard", MyDashboardError)
```

### Dev Mode Error Overlay
In development mode, a full-page overlay displays the error message, stack trace, and relevant request metadata. Sensitive information like `Authorization` and `Cookie` headers are automatically redacted for security.

## Client-Side Errors

### Boundary Management
The client-side runtime mirrors the server's error boundaries. If a client-side component crashes (e.g., in `onMount`), the runtime will catch the error and attempt to recover or display a fallback.

```typescript
import { onComponentError } from "/_gospa/runtime.js";

onComponentError((id, error) => {
    console.error(`Component ${id} failed:`, error);
});
```

### WebSocket Failures
GoSPA automatically handles WebSocket connection failures with exponential backoff and automatic reconnection. In-flight messages are queued and re-sent once the connection is restored.
