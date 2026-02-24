# Island Hydration & Streaming SSR

GoSPA implements a "Partial Hydration" or "Islands Architecture" similar to Astro or Fresh, allowing you to ship minimal JavaScript by defaults and selectively hydrate interactive components.

## Islands Architecture

In GoSPA, an "Island" is an interactive component that is hydrated on the client. Everything else is rendered as static HTML.

### Hydration Modes

You can control *when* an island hydrates using the `data-gospa-mode` attribute:

| Mode | Description |
|------|-------------|
| `immediate` | (Default) Hydrate as soon as the script loads. |
| `visible` | Hydrate when the component enters the viewport (using Intersection Observer). |
| `idle` | Hydrate when the browser is idle (using `requestIdleCallback`). |
| `interaction` | Hydrate on first user interaction (click, focus, hover). |
| `lazy` | Hydrate only when manually triggered via API. |

### Configuration via DOM

```html
<div data-gospa-island="Counter" 
     data-gospa-mode="visible"
     data-gospa-priority="high"
     data-gospa-props='{"initial": 10}'>
    <!-- Server-rendered content goes here -->
</div>
```

---

## Priority Hydration

When multiple islands are scheduled for hydration, GoSPA uses a `PriorityScheduler` to ensure critical components load first.

### Priority Levels

| Level | Value | Use Case |
|-------|-------|----------|
| `critical` | 100 | Navigation, search, purchase buttons. |
| `high` | 75 | Above-the-fold interactive elements. |
| `normal` | 50 | (Default) General interactive components. |
| `low` | 25 | Below-the-fold or non-essential widgets. |

### Manual Scheduling

```typescript
import { getPriorityScheduler } from '@gospa/runtime';

const scheduler = getPriorityScheduler();
scheduler.forceHydrate('gospa-island-123');

console.log(scheduler.getStats()); // { total: 10, pending: 2, hydrated: 8, ... }
```

---

## Streaming SSR

GoSPA supports streaming responses, allowing the server to send the HTML skeleton immediately and "stream in" islands and state updates as they are ready.

### Features

- **Progressive Hydration**: Islands are hydrated as soon as their HTML chunk arrives.
- **Out-of-Order Updates**: The server can stream content to specific placeholders even after the main page has rendered.
- **Automatic State Sync**: The `__GOSPA_STATE__` is updated automatically as chunks arrive.

### Client-Side Management

The `StreamingManager` handles incoming chunks automatically:

```typescript
import { initStreaming } from '@gospa/runtime';

// Usually auto-initialized, but can be configured manually
const streamer = initStreaming({
  enableLogging: true,
  hydrationTimeout: 5000
});

document.addEventListener('gospa:hydrated', (ev) => {
  console.log('Island arrived and hydrated:', ev.detail.island.name);
});
```

### Chunk Types

GoSPA streams chunks in the following format:

| Type | Action |
|------|--------|
| `html` | Updates a specific DOM element by ID. |
| `island` | Registers and schedules an island for hydration. |
| `script` | Dynamically injects and executes a script. |
| `state` | Updates the global reactive state. |
| `error` | Reports a server-side rendering error. |

---

## Island Module Convention

Islands should export an object with `hydrate` or `mount` functions:

```typescript
// islands/Counter.ts
export default {
    hydrate(element: Element, props: any, state: any) {
        // Initialize your framework (Svelte, Preact, Vanilla) here
        console.log('Hydrating Counter with props:', props);
    }
}
```

By default, the `IslandManager` expects modules to be located at `/islands/{name}.js`. This can be customized in `initIslands()`.
