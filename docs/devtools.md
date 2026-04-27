# DevTools & Debugging

GoSPA includes an integrated suite of development tools to help you build, debug, and monitor your application's state, connectivity, and performance.

## Error Overlay

In development mode (`DevMode: true`), GoSPA will display a full-page error overlay if a server-side route or component crashes.

### Key Features
- **Stack Trace**: Directly see where in your Go or Templ code the error occurred.
- **Sensitive Header Redaction**: For security, headers like `Authorization` and `Cookie` are automatically redacted before being displayed.
- **Auto-Reconnection**: If the server restarts (e.g., during development), the overlay will automatically reload once the server is back online.

## HMR (Hot Module Replacement)

HMR is enabled by default in development mode. It allows you to update your application's code and see changes in real-time without losing state.

### State Persistence
GoSPA's HMR system is carefully designed to preserve your reactive runes. When a component is updated, its state is serialized and re-applied to the new version, ensuring a seamless editing experience.

## Performance Monitoring

The client-side runtime includes a performance monitoring utility to track hydration time, state update latency, and network performance.

```typescript
import { measure } from "/_gospa/runtime.js";

const result = await measure("heavyTask", async () => {
    // Perform complex logic...
});
```

## Debug Panel

You can toggle a built-in debug panel by pressing `Ctrl + Shift + D` (or your configured hotkey). This panel allows you to:
- **Inspect State**: View the current values of all global and component-level reactive runes.
- **Monitor WebSockets**: See real-time message traffic and connection status.
- **Hydration Stats**: Check how many island components were hydrated and their individual hydration times.
- **Memory Usage**: Monitor the number of active `Effect` and `EffectScope` instances.
