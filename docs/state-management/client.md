# Client-side State Management

GoSPA's client-side state is powered by a high-performance signal-based system that mirrors the server's reactivity.

## Reactive Runes

The client runtime provides `$state`, `$derived`, and `$effect` for component-level state management.

```typescript
const count = $state(0);
const double = $derived(() => count.value * 2);
```

## Global Stores

Stores allow you to share reactive state across different islands and components without server round-trips.

```typescript
// Create or get a named global store
const auth = createStore('auth', { user: null, loading: true });

// Access in another island
const auth = getStore('auth');
```

## Auto-Batching

The client runtime automatically batches updates within microtasks to prevent layout thrashing and minimize WebSocket traffic.

## State Pruning

To keep the client-side memory footprint small, GoSPA supports automatic pruning of state that is no longer needed after a component is destroyed.
