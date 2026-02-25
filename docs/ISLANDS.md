# Island Hydration & Streaming SSR

GoSPA implements a partial hydration (Islands Architecture) system, allowing you to ship static HTML by default and selectively hydrate interactive components based on viewport visibility, idle time, or user interaction.

---

## Islands Architecture

An **island** is an interactive component that is loaded and hydrated on the client. The server renders the HTML placeholder; the client loads the JavaScript module and calls its `hydrate` or `mount` function.

### DOM Attributes

Mark an element as an island using `data-gospa-island`:

```html
<div
  data-gospa-island="Counter"
  data-gospa-mode="visible"
  data-gospa-priority="high"
  data-gospa-props='{"initial": 10}'
  data-gospa-state='{"count": 0}'
  data-gospa-threshold="200"
  id="my-counter"
>
  <!-- Server-rendered placeholder content here -->
  <p>Loading...</p>
</div>
```

| Attribute | Type | Description |
|-----------|------|-------------|
| `data-gospa-island` | string | **Required.** Island name — maps to `/islands/{name}.js` |
| `data-gospa-mode` | string | Hydration mode (see below, default: `immediate`) |
| `data-gospa-priority` | string | Queue priority: `high`, `normal`, `low` (default: `normal`) |
| `data-gospa-props` | JSON | Props passed to `hydrate(element, props, state)` |
| `data-gospa-state` | JSON | State passed to `hydrate(element, props, state)` |
| `data-gospa-threshold` | number | Intersection margin in px for `visible` mode (default: 200) |
| `data-gospa-defer` | number | ms delay for `idle` mode (default: 2000) |
| `data-gospa-client-only` | boolean | Skip server rendering entirely |
| `data-gospa-server-only` | boolean | Never hydrate — render only |

### Hydration Modes

| Mode | Description |
|------|-------------|
| `immediate` | (Default) Hydrate when the island is discovered |
| `visible` | Hydrate when the element enters the viewport (uses `IntersectionObserver`) |
| `idle` | Hydrate when the browser is idle (uses `requestIdleCallback`, falls back to `setTimeout`) |
| `interaction` | Hydrate on first user event: `mouseenter`, `touchstart`, `focusin`, or `click` |
| `lazy` | Never auto-hydrate — call `hydrateIsland(id)` manually |

### Priority Queue

When multiple islands are scheduled for `immediate` hydration, they are processed in priority order:

| Priority | Use Case |
|----------|----------|
| `high` | Navigation, purchase buttons, search — hydrate first |
| `normal` | (Default) General interactive components |
| `low` | Below-the-fold, non-essential widgets |

---

## Island Module Convention

Island modules must be ES modules served at `/islands/{name}.js` (configurable via `moduleBasePath`).

```typescript
// /islands/Counter.ts
export default {
    hydrate(element: Element, props: Record<string, unknown>, state: Record<string, unknown>) {
        const btn = element.querySelector('button')!;
        let count = (state.count as number) ?? (props.initial as number) ?? 0;

        const render = () => btn.textContent = `Count: ${count}`;
        render();
        btn.addEventListener('click', () => { count++; render(); });
    }
}
```

Both `hydrate` and `mount` are accepted (either on the default export or as named exports). `hydrate` takes precedence.

---

## Client API

### `initIslands(config?) → IslandManager`

Initializes the global island manager. Auto-discovers all `[data-gospa-island]` elements in the DOM.

```typescript
import { initIslands } from './island.ts';

const manager = initIslands({
    moduleBasePath: '/islands',   // Where to load island JS from
    defaultTimeout: 30000,        // ms until hydration times out
    debug: true,                  // Enable console logging
    // Custom loader for bundlers that handle imports differently:
    moduleLoader: async (name) => import(`./islands/${name}`),
});
```

### `getIslandManager() → IslandManager | null`

Returns the global island manager.

### `hydrateIsland(idOrName) → Promise<IslandHydrationResult | null>`

Manually triggers hydration for a specific island (by DOM `id` or island name):

```typescript
import { hydrateIsland } from './island.ts';

// Hydrate the lazy island when user clicks "Show chart"
document.getElementById('show-chart')?.addEventListener('click', async () => {
    const result = await hydrateIsland('Chart');
    if (!result?.success) console.error('Chart hydration failed:', result?.error);
});
```

### `IslandManager` Methods

```typescript
manager.discoverIslands()               // Scan DOM for new islands
manager.hydrateIsland(data)             // Hydrate a specific island data object
manager.hydrate(idOrName)               // Hydrate by id or name string
manager.isHydrated(id)                  // Check if an island is already hydrated
manager.getIslands()                    // All discovered islands
manager.getIsland(id)                   // Get island data by id
manager.destroy()                       // Clean up observers and listeners
```

### Browser Globals

```js
// Available at runtime when island.ts is loaded
window.__GOSPA_ISLAND_MANAGER__.init(config?)   // == initIslands
window.__GOSPA_ISLAND_MANAGER__.get()           // == getIslandManager
window.__GOSPA_ISLAND_MANAGER__.hydrate(name)   // == hydrateIsland
```

### Events

```typescript
// Fired on the island element when hydration completes
element.addEventListener('gospa:hydrated', (ev: CustomEvent) => {
    console.log('hydrated:', ev.detail.island); // IslandElementData
});
```

---

## Streaming SSR

GoSPA's `StreamingManager` handles progressive hydration from server-streamed HTML chunks. It is usually initialized automatically; each chunk type triggers a different action:

| Chunk Type | Action |
|------------|--------|
| `html` | Updates a specific DOM element by `id` with new innerHTML |
| `island` | Registers and schedules the island for hydration per its mode |
| `script` | Dynamically injects and executes a `<script>` tag |
| `state` | Merges the payload into the global `__GOSPA_STATE__` |
| `error` | Logs a server-side rendering error to the console |

### Manual Setup

```typescript
import { initStreaming } from './streaming.ts';

const streamer = initStreaming({
    hydrationTimeout: 5000,  // ms before island hydration times out
    enableLogging: true,
});

// Manually hydrate an island that arrived via streaming
await streamer.hydrate('IslandId');

// Check hydration status
console.log(streamer.getHydratedIslands()); // Set<string>
console.log(streamer.isHydrated('IslandId')); // boolean
```

---

## Auto-Initialize

If a `<script data-gospa-islands>` tag is present, `initIslands()` is called automatically when the DOM is ready. Otherwise, call it manually before the page finishes loading.

---

## Notes

- Islands do **not** require a framework. They can be Svelte, Preact, Vanilla JS, or any library that runs in a browser `<script type="module">`.
- Islands discovered after `initIslands()` (e.g. via SPA navigation) can be re-scanned with `manager.discoverIslands()`.
- Setting `data-gospa-server-only="true"` skips hydration entirely — useful for purely decorative server-rendered components.
