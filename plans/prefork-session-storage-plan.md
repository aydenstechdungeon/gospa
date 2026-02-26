# GoSPA Prefork & External Session Storage Plan

## Overview
Currently, GoSPA's state synchronization (`ClientStateStore`), session management (`SessionStore`), and WebSocket broadcast hub (`WSHub`) are strictly **in-memory**. If Fiber's `Prefork` mode is enabled (which spawns multiple processes), state and WebSocket connections become isolated per process.

To achieve horizontal scalability and fully leverage `Prefork`, we need to decouple state and pub/sub mechanisms from local memory and allow backing them with external data stores like **Redis** or **Valkey**.

---

## Architectural Changes

### 1. Storage Abstraction
We need a unified Key-Value interface that can be backed by Memory (default) or External Storage (Redis/Valkey).

```go
type Storage interface {
    Get(key string) ([]byte, error)
    Set(key string, val []byte, exp time.Duration) error
    Delete(key string) error
}
```

- **SessionStore**: Move away from `map[string]sessionEntry` to using the `Storage` interface.
- **ClientStateStore**: Serialize `StateMap` to JSON and save it in `Storage`. On connection, load the JSON and deserialize it back into a `StateMap`.

### 2. Pub/Sub Abstraction for WebSocket Hub
`WSHub` needs to broadcast state changes across *all* connected clients in *all* processes.

```go
type PubSub interface {
    Publish(channel string, message []byte) error
    Subscribe(channel string, handler func(message []byte)) error
}
```

- When `BroadcastState` is called, it should `Publish` to a Redis channel instead of (or in addition to) the local `broadcast` channel.
- Each GoSPA process's `WSHub` will `Subscribe` to the channel. When a message is received from Redis, the `WSHub` forwards it to its locally connected `Clients`.

### 3. Configuration Updates
Update the `gospa.Config` struct to accept `Prefork` and external dependencies.

```go
type Config struct {
    // ... existing fields ...
    
    // Enable Fiber Prefork
    Prefork bool
    
    // External storage for Sessions and State (defaults to Memory)
    Storage Storage
    
    // PubSub for multi-process WebSocket broadcasting (defaults to local memory)
    PubSub PubSub
}
```

---

## Implementation Phases

### Phase 1: Abstraction & In-Memory Defaults
1. Introduce `store/storage.go` with the `Storage` interface.
2. Introduce `store/pubsub.go` with the `PubSub` interface.
3. Implement `memory` adapters for both to serve as the default so the framework runs out-of-the-box without Redis.
4. Refactor `fiber/websocket.go` (specifically `SessionStore` and `ClientStateStore`) to use the `Storage` interface instead of maps.
5. Refactor `fiber/websocket.go` (`WSHub`) to use the `PubSub` interface for broadcasting. 

### Phase 2: Configuration & Fiber Integration
1. Add `Prefork`, `Storage`, and `PubSub` to `gospa.Config` in `gospa.go`.
2. Map `Config.Prefork` to `fiberConfig.Prefork`.
3. Validate config on startup: Output a `WARNING` if `Prefork: true` is combined with memory-based Storage/PubSub, notifying the user that state will be isolated.

### Phase 3: Redis/Valkey Plugin
1. Create a `plugin/store/redis` or `store/redis` package.
2. Implement the `Storage` and `PubSub` interfaces using the `github.com/redis/go-redis/v9` client.
3. Create an example app (`examples/prefork`) demonstrating Prefork scaling with Redis.

### Phase 4: State Serialization Improvements
- In `state/serialize.go`, ensure `StateMap` can be fully serialized and fully hydrated from raw JSON bytes, as external storage only stores bytes.
- Handle state versioning or diffing carefully if multiple processes write to the same state concurrently (consider optimistic locking or last-write-wins).

---

## Security & Concurrency Considerations
- **Race Conditions**: With multiple processes accessing the same session/state, we may encounter race conditions. Ensure atomic updates where possible.
- **Payload Size**: `ClientStateStore` could grow large for complex apps. Minimize what is saved to the store by leveraging state diffing.
