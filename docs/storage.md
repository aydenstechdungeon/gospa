# Storage & PubSub

GoSPA provides a flexible storage and publish-subscribe system to handle session state, rate limiting, and real-time broadcasting across single or multiple processes.

## Storage Interface

The `store.Storage` interface defines a simple key-value store optimized for high-performance state management.

```go
type Storage interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, val []byte, exp time.Duration) error
    Delete(ctx context.Context, key string) error
}
```

### Implementations

- **MemoryStorage**: Default in-memory implementation. Features $O(1)$ scaling and LRU eviction for zero-TTL entries. Best for single-process development.
- **Redis Store**: Production-grade implementation using Redis. Required for horizontal scaling and `prefork` mode to ensure state consistency across worker processes.

## PubSub Interface

The `store.PubSub` interface enables message broadcasting across the application.

```go
type PubSub interface {
    Publish(ctx context.Context, channel string, message []byte) error
    Subscribe(ctx context.Context, channel string, handler func(message []byte)) (Unsubscribe, error)
}
```

### Implementations

- **MemoryPubSub**: Local in-memory broadcasting. Handlers are invoked asynchronously with panic recovery.
- **Redis PubSub**: Distributed broadcasting using Redis. Required for real-time features (WebSocket, SSE) in multi-process/prefork environments.

## Multi-Process (Prefork) Configuration

When using `Config.Prefork: true`, you **MUST** provide external Redis backends to maintain consistency.

```go
import (
    "github.com/aydenstechdungeon/gospa"
    "github.com/aydenstechdungeon/gospa/store/redis"
    goredis "github.com/redis/go-redis/v9"
)

func main() {
    rdb := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
    
    app := gospa.New(gospa.Config{
        Prefork: true,
        Storage: redis.NewStore(rdb),
        PubSub:  redis.NewPubSub(rdb),
    })
    
    app.Run(":3000")
}
```

## Security & Reliability

- **Context Awareness**: All operations support `context.Context` for proper timeout and cancellation propagation.
- **Panic Recovery**: PubSub handlers are wrapped in `recover()` blocks to prevent consumer errors from crashing the application.
- **LRU Eviction**: In-memory storage automatically prunes zero-TTL entries using an LRU policy when reaching `maxEntries` (default: 10,000).
