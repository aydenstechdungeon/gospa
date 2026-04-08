# Client Runtime Internals

GoSPA uses a tiered client-side runtime to balance performance and functionality. Depending on your page's complexity, you can choose between different "tiers" to minimize the JavaScript payload.

## Runtime Tiers

| Tier | Size (minified) | Description | Features |
| :--- | :--- | :--- | :--- |
| **Micro** | ~1KB | Extremely minimal runtime for static pages. | Core SPA navigation, simple event delegation. |
| **Core** | ~13KB | Standard runtime for reactive applications. | Runes, Effects, Derived, and basic DOM synchronization. |
| **Full** | ~52KB | Full-featured runtime for complex SPAs. | WebSockets, state sync, HMR, and advanced UI components. |

## Tier Selection

The runtime tier is determined automatically based on the requirements of the page and its layouts. You can also manually specify a tier in your `.templ` or `.gospa` files using the `@gospa:tier` directive:

```svelte
// page.templ
// @gospa:tier core

<template>
    <h1>My Reactive Page</h1>
</template>
```

### Hierarchy Rules
1. If any component in the layout chain (Root -> Layouts -> Page) requires a higher tier, the higher tier is used.
2. If `WebSocket` or `Remote Actions` are used, the runtime automatically upgrades to **Full**.
3. If no reactive primitives are used, it defaults to **Micro**.

## Hydration Mode

GoSPA supports different hydration modes to control when and how components become interactive:

- **Lazy**: Hydrates only when the component enters the viewport (using `IntersectionObserver`).
- **Immediate**: Hydrates as soon as the runtime loads (default).
- **Manual**: Hydrates only when explicitly triggered via `GoSPA.hydrate(id)`.

Configured in `gospa.Config`:
```go
config.HydrationMode = gospa.HydrationLazy
```

## Internal Architecture

The runtime is built as a set of ES modules. Critical chunks are preloaded using `Link: rel=modulepreload` headers to reduce Time-to-Interactive (TTI).

- **Reactivity Engine**: Based on a signal-based dependency tracker.
- **Micro-task Scheduler**: Batches DOM updates to prevent layout thrashing.
- **GZIP Delta Patching**: When using WebSockets, only the diffs of state changes are sent to the client.
