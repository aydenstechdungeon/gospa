# Advanced Reactive Patterns

Advanced usage of the GoSPA reactive system.

## Performance Optimization

### Batching (Client)

Batch multiple updates to minimize DOM reflows and WebSocket messages.

```typescript
import { batch } from '@gospa/client';

batch(() => {
  count.set(1);
  name.set('Updated');
});
```

### Untracking

Execute logic without creating a reactive dependency.

```typescript
import { untrack } from '@gospa/client';

$effect(() => {
  const c = count.value; // Tracked
  const t = untrack(() => other.value); // Not tracked
});
```

## Special Primitives

### RuneRaw

Shallow reactive state without deep proxying. Pros: faster for large objects. Cons: requires reassignment to trigger updates.

```typescript
const large = runeRaw({ data: [...] });
large.value = { data: [...] }; // Triggers update
```

### PreEffect

An effect that runs *before* the DOM is updated. Useful for reading current DOM state before it changes.

```typescript
new PreEffect(() => {
  const scroll = window.scrollY; // Read old scroll
});
```

## Internal Synchronization

GoSPA uses a bounded worker queue for server-side notifications to prevent bottlenecks. Notifications fall back to synchronous execution under heavy load (backpressure).
