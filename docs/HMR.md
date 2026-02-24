# Hot Module Replacement (HMR)

GoSPA features a built-in Hot Module Replacement (HMR) system that allows you to update code and styles in real-time without losing the application state.

## How it Works

The HMR system consists of three parts:
1.  **Server Manager**: Watches for file changes and broadcasts updates to clients.
2.  **Client Runtime**: Receives updates and applies them to the current page.
3.  **State Preservation**: Saves and restores reactive state across updates.

---

## Server-Side (Go)

The `HMRManager` orchestrates the update cycle.

### Configuration

```go
config := fiber.HMRConfig{
    Enabled:      true,
    WatchPaths:   []string{"./routes", "./islands", "./static"},
    IgnorePaths:  []string{"node_modules", ".git"},
    DebounceTime: 100 * time.Millisecond,
}

hmr := fiber.InitHMR(config)
hmr.Start()
```

### HMR Endpoint

To enable HMR, you must register the WebSocket endpoint:

```go
app.Get("/__hmr", hmr.HMREndpoint())
```

---

## Client-Side (TypeScript)

The `HMRClient` manages the WebSocket connection and applies updates.

### Manual Initialization

```typescript
import { initHMR } from '@gospa/runtime';

const hmr = initHMR({
    wsUrl: `ws://${window.location.host}/__hmr`,
    reconnectInterval: 1000
});

hmr.onUpdate((msg) => {
    console.log('Module updated:', msg.moduleId);
});
```

### Module Registration

For a module to be hot-swappable, it must be registered:

```typescript
import { registerHMRModule, acceptHMR } from '@gospa/runtime';

registerHMRModule('my-module', exports);
acceptHMR('my-module');
```

---

## State Preservation

GoSPA can preserve your reactive state (`Rune`, `Derived`, etc.) across updates.

### Automatic Preservation
If you use the standard `StateMap` and components, GoSPA automatically serializes the state before an update and restores it after the new module is loaded.

### Manual Preservation
You can hook into the state preservation cycle:

```typescript
window.__gospaGetState = (id) => {
    return { scrollY: window.scrollY };
};

window.__gospaSetState = (id, state) => {
    window.scrollTo(0, state.scrollY);
};
```

---

## CSS Hot Reloading

When a `.css` file is modified, the `CSSHMR` system identifies the corresponding `<link>` tag and forces a reload by appending a timestamp, avoiding a full page refresh.

```typescript
import { CSSHMR } from '@gospa/runtime';

// Automatically handled by HMRClient for registered styles
CSSHMR.updateStyle('/static/css/main.css');
```

---

## Troubleshooting

- **Full Page Reload**: If a module cannot be hot-swapped (e.g., changes to internal Go logic or complex dependency graphs), the HMR system will fall back to a full page reload.
- **Connection Lost**: The client will attempt to reconnect with exponential backoff. If the server is restarted, the client refreshes the page to ensure synchronization.
- **Console Logs**: Enable `debug: true` in `HMRClientConfig` to see the internal HMR event log in the browser console.
