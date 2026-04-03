# Scaling and Security Configuration

Distributed deployment, horizontal scaling, and security configuration.

## Distributed and Scaling Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `Prefork` | `bool` | `false` | Enables Fiber Prefork for multi-process performance |
| `Storage` | `store.Storage` | `memory` | External Key-Value store (e.g., Redis) for shared state |
| `PubSub` | `store.PubSub` | `memory` | External messaging broker (e.g., Redis PubSub) for broadcasts |
| `SSGCacheMaxEntries` | `int` | `500` | FIFO eviction limit for page caches |
| `SSGCacheTTL` | `time.Duration` | `0` | Expiration time for cache entries |

> [!CAUTION]
> **Prefork requires external storage.** When `Prefork: true` is enabled, you MUST provide external `Storage` and `PubSub` implementations to ensure state consistency across worker processes.

## ISR (Incremental Static Regeneration) Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `DefaultRevalidateAfter` | `time.Duration` | `0` | Global ISR TTL fallback |
| `ISRSemaphoreLimit` | `int` | `10` | Limits concurrent background ISR revalidations |
| `ISRTimeout` | `time.Duration` | `60s` | Maximum time allowed for a single background revalidation |

## Security Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `AllowedOrigins` | `[]string` | `[]` | Allowed CORS origins |
| `EnableCSRF` | `bool` | `true` | Enable automatic CSRF protection |
| `ContentSecurityPolicy` | `string` | built-in | Optional CSP header value |
| `PublicOrigin` | `string` | `""` | Public base URL for stable WebSocket URLs |
| `AllowInsecureWS` | `bool` | `false` | Allow `ws://` even on `https://` pages |

## Example (High Performance Cluster)

```go
import "github.com/aydenstechdungeon/gospa/store/redis"

app := gospa.New(gospa.Config{
    Prefork: true,
    Storage: redis.NewStore(rdb),
    PubSub:  redis.NewPubSub(rdb),
    SimpleRuntime:   true,
    CompressState:   true,
    CacheTemplates:  true,
    HydrationMode:   "lazy",
})
```
