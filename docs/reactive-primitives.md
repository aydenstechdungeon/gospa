# Reactive Primitives

GoSPA's reactivity system is built on top of high-performance signals, providing a consistent API across both Go (server) and TypeScript (client).

## Basic Primitives

### Rune
A `Rune` is the base atom of reactivity. It holds a single value and tracks its subscribers.

**TypeScript:**
```typescript
import { $state } from "/_gospa/runtime.js";

const count = $state(0);
console.log(count.get()); // 0
count.set(1);
```

**Go:**
```go
import "github.com/aydenstechdungeon/gospa/state"

count := state.NewRune(0)
value := count.Get().(int)
count.Set(1)
```

### Derived
A `Derived` value automatically re-calculates when its dependencies change. It is lazy and memoized.

**TypeScript:**
```typescript
const count = $state(1);
const double = $derived(() => count.get() * 2);
```

### Effect
An `Effect` runs a function and automatically re-runs it whenever any reactive values accessed inside it change.

```typescript
$effect(() => {
  console.log("Count is now:", count.get());
});
```

## Advanced Primitives

### EffectScope (New)
The `EffectScope` utility allows you to group effects together for bulk disposal. This is critical for preventing memory leaks in SPAs.

```typescript
const scope = new EffectScope();
scope.run(() => {
  $effect(() => { /* ... */ });
});

// Later, clean up all effects in the scope at once
scope.dispose();
```

> [!TIP]
> GoSPA automatically manages `EffectScope` for all Island components. You only need to use it manually if you are building custom advanced reactive logic outside of the component system.

## Performance Considerations

1.  **Batching**: Use `batch(() => { ... })` to group multiple state updates into a single re-render cycle.
2.  **Untracking**: Use `untrack(() => { ... })` to read a reactive value without creating a dependency.
3.  **Scoped Disposal**: Always ensure effects created inside long-lived objects are wrapped in an `EffectScope`.
