# Reactive Primitives (JavaScript/TypeScript)

Client-side reactive primitives for the GoSPA runtime.

## SFC Primitives ($state, $derived, $effect)

Ergonomic global functions for use in `.gospa` Single File Components.

```typescript
const count = $state(0);
count.value++;

const double = $derived(() => count.value * 2);

$effect(() => {
  console.log(`Count: ${count.value}`);
});
```

## Low-level API

### Rune

```typescript
import { Rune } from '/_gospa/runtime.js';

const count = new Rune(0);
count.set(1);
const val = count.get();
```

### Derived

```typescript
import { derived } from '/_gospa/runtime.js';

const double = derived(() => count.get() * 2);
```

### Effect

```typescript
import { effect } from '/_gospa/runtime.js';

effect(() => {
  console.log(count.get());
});
```

## Advanced Primitives

### Resource

Async data fetching with status tracking.

```typescript
const user = resource(async () => {
  const res = await fetch('/api/user');
  return res.json();
});
```

### StateMap

Collection of named runes.

```typescript
const states = new StateMap();
states.set('count', 0);
```
