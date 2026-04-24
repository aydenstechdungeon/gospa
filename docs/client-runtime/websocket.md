# WebSocket Client

Real-time state synchronization between server and client.

## WSClient

WebSocket client for real-time state synchronization with auto-reconnect and heartbeat support.

```typescript
import { initWebSocket, getWebSocketClient } from '@gospa/client';

// Initialize
const ws = initWebSocket({
  url: 'ws://localhost:3000/ws',
  reconnect: true,
  maxReconnectAttempts: 10,
  heartbeatInterval: 30000,
  telemetry: true
});

// Connect
await ws.connect();
```

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `url` | string | - | WebSocket server URL |
| `reconnect` | boolean | true | Auto-reconnect on disconnect |
| `reconnectInterval` | number | 1000 | Base reconnection interval (ms) |
| `maxReconnectAttempts` | number | 10 | Maximum reconnection attempts |
| `reconnectBackoffMultiplier` | number | 2 | Exponential backoff multiplier |
| `reconnectJitterRatio` | number | 0.2 | Random jitter ratio applied to reconnect delay |
| `reconnectMaxDelay` | number | 30000 | Max reconnect delay (ms) |
| `heartbeatInterval` | number | 30000 | Heartbeat ping interval (ms) |
| `staleStateGuard` | boolean | true | Drop stale server state messages |
| `staleReplayWindowMs` | number | 20000 | Replay tolerance window for out-of-order messages |
| `telemetry` | boolean | true | Emit websocket telemetry events |
| `onTelemetry` | function | `() => {}` | Callback for telemetry payloads |

## Built-in Telemetry

When telemetry is enabled, the runtime emits `window` events:

- `gospa:ws-telemetry` with payload `{ type, timestamp, detail }`

Event types:

- `connect`
- `disconnect`
- `reconnect-scheduled`
- `reconnect-attempt`
- `latency`
- `stale-message-dropped`
- `invalid-message`
- `patch-failure`
- `decompress-failure`

## Synced Rune

Create a rune that automatically syncs with the server.

```typescript
import { syncedRune } from '@gospa/client';

const count = syncedRune(0, {
  componentId: 'counter',
  key: 'count',
  debounce: 100
});

// Local update (optimistic)
count.set(5);
```

## Connection State

Monitor connection state changes.

```typescript
const ws = getWebSocketClient();

// State values: 'connecting' | 'connected' | 'disconnecting' | 'disconnected'
console.log('Current state:', ws.state);

// Listen for state changes
ws.onStateChange((newState) => {
  console.log('State changed to:', newState);
});
```
