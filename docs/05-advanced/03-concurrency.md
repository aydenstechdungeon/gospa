# WebSocket Concurrency and State Sync

GoSPA's WebSocket state synchronization is powerful, but it introduces specific concurrency challenges when a user has multiple tabs open to the same application.

## The Multi-Tab Race Condition

When multiple tabs are connected to the same session via WebSockets, they all receive the same state updates from the server. However, if two tabs attempt to modify the same state key simultaneously, a race condition can occur:

1. **Tab A** reads `count = 5`.
2. **Tab B** reads `count = 5`.
3. **Tab A** sends `count = 6`.
4. **Tab B** sends `count = 6`.
5. The server receives both, but the final state is `6` instead of `7`.

## Solution: `WSTabSync`

GoSPA provides a built-in `WSTabSync` utility to coordinate state changes across tabs using the `BroadcastChannel` API.

### 1. Enabling Tab Sync

In your client-side entry point (e.g., `app.ts`), initialize the sync manager:

```typescript
import { initTabSync } from 'gospa/runtime';

// Initialize with a unique channel name for your app
const sync = initTabSync('my-app-state');
```

### 2. Synced Runes

When using `syncedRune`, the runtime automatically handles inter-tab coordination if `WSTabSync` is active.

```typescript
import { syncedRune } from 'gospa/runtime';

const count = syncedRune('count', 0, ws);
// count.set(v) will now notify other tabs via BroadcastChannel
// to prevent conflicting updates.
```

## Best Practices

### Avoid Frequent Global Overwrites
Instead of replacing large objects, use granular keys. GoSPA's delta-patching works best when updates are targeted.

### Use Server-Side Authority
For critical operations (like financial transactions or inventory), do not rely on client-side state incrementing. Instead, use a **Remote Action**:

```go
// Insecure (Client-side increment)
// onclick="state.set('count', state.get('count') + 1)"

// Secure (Server-side increment via Remote Action)
// onclick="gospa.call('incrementCount')"
```

The server-side action can safely lock the state or use atomic database operations, then broadcast the new value to all tabs via `app.BroadcastState("count", newValue)`.

## Handling Offline States

When a connection is lost, GoSPA enters "Buffered Mode". Changes made while offline are queued and replayed upon reconnection. 

> [!IMPORTANT]
> Replayed changes use a **Last-Write-Wins** strategy. If complex merging is required, implement a custom conflict resolution strategy in your `WSHub` handler on the server.
