# API Reference

GoSPA provides a unified API for building real-time, interactive web applications on top of Fiber.

## Fiber Integration (Go)
The core `gospa.App` handles both HTTP requests and WebSocket connections.

### App Lifecycle
- `gospa.New(config gospa.Config) *gospa.App`: Create a new GoSPA application.
- `app.Run(addr string) error`: Start the application on the specified address.
- `app.Shutdown() error`: Gracefully shutdown the application and all its components.

### Client Runtime & HMR
- `fiber.RuntimeMiddleware(tier string) fiber.Handler`: Serves the client-side GoSPA runtime.
- `fiber.HMRMiddleware(hub *fiber.WSHub) fiber.Handler`: Injects HMR scripts in development mode.

### Real-Time & WebSockets
- `fiber.NewWSHub(pubsub store.PubSub) *fiber.WSHub`: Initialize a WebSocket Hub for state sync.
- `app.Broadcast(message []byte)`: Sends a raw message to all connected clients.
- `app.BroadcastState(key string, value interface{})`: Synchronizes a state change across all clients globally.

## Client Runtime (TypeScript)
The GoSPA client runtime manages reactivity, hydration, and communication.

### Reactive State
- `$state(initial: T): T`: Create a reactive proxy for objects or values.
- `$derived(fn: () => T): T`: Define a derived value that automatically re-calculates.
- `$effect(fn: () => void)`: Define a reactive effect with automatic dependency tracking.

### Remote Communication
- `remoteAction(name: string, input?: any): Promise<any>`: Invoke a server-side Go function.
- `initWebSocket(config: RuntimeConfig): Promise<WebSocketClient>`: Setup the real-time sync connection.

## Routing Registry
Registry for manually or automatically registered route components.
- `routing.RegisterPage(path string, fn routing.ComponentFunc)`: Register a static or SSR route.
- `routing.RegisterAction(pagePath, actionName string, fn routing.ActionFunc)`: Add form action logic to a route.
- `routing.RegisterMiddleware(path string, fn routing.MiddlewareFunc)`: Add Fiber middleware to specific routes.
- `routing.RegisterSlot(pagePath, slotName string, fn routing.SlotFunc)`: Register a dynamic slot for Partial Prerendering (PPR).
