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
  heartbeatInterval: 30000
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
| `heartbeatInterval` | number | 30000 | Heartbeat ping interval (ms) |

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
