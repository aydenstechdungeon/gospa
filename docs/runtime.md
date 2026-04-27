# Client Runtime Lifecycle

GoSPA's client-side runtime is a high-performance, modular system that handles reactivity, hydration, and communication.

## Initialization

The GoSPA runtime is usually initialized automatically when a client lands on a GoSPA page.

```typescript
import * as GoSPA from "/_gospa/runtime.js";

GoSPA.init({
    wsUrl: "/_gospa/ws",
    debug: true,
    hydration: {
        mode: "idle", // 'immediate', 'idle', or 'visible'
        timeout: 5000,
    },
});
```

### Automatic Initialization
If you add the `data-gospa-auto` attribute to your `<html>` or `<body>` tag, GoSPA will automatically scan and initialize your application on DOM ready.

## Hydration

Hydration is the process of attaching reactive logic to server-rendered HTML. GoSPA supports three main hydration modes:

1.  **Immediate**: Hydrate all islands as soon as the runtime loads.
2.  **Idle**: Hydrate islands only when the browser is idle (using `requestIdleCallback`).
3.  **Visible**: Hydrate islands only when they enter the viewport (using `IntersectionObserver`).

### Strategic Hydration
For large applications, you can also manually trigger hydration:

```typescript
import { hydrateIsland } from "/_gospa/runtime.js";

// Hydrate a specific island by ID or Name
await hydrateIsland("shopping-cart");
```

## Lifecycle Hooks

GoSPA component islands provide several hooks to interact with the runtime's lifecycle:

- **`onMount`**: Runs once after the island has been hydrated and attached to the DOM.
- **`onDestroy`**: Runs before the island is removed from the DOM or the page is unnavigated.
- **`onUpdate`**: Runs whenever a reactive dependency inside the island's scope changes.

## Global Object

In the browser, GoSPA exposes a global `GoSPA` object that allows you to interact with the runtime from existing legacy scripts or the browser's developer console.

```javascript
window.GoSPA.inspect(); // Opens the debug panel
```
