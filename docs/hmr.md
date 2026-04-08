# Hot Module Replacement (HMR)

GoSPA includes a built-in HMR system that allows you to update your application in real-time without losing state.

## How it Works

The HMR system consists of three parts:
1.  **File Watcher**: Monitors your source files (`.templ`, `.go`, `.ts`, `.js`, `.css`) for changes.
2.  **Server Hub**: Orchestrates updates and broadcasts change events to connected clients.
3.  **Client Runtime**: Receives update events and applies them dynamically to the DOM.

## Configuration

HMR is enabled by default in development mode. You can configure it in your `gospa.Config`:

```go
app := gospa.New(gospa.Config{
    DevMode: true,
    HMR: gospa.HMRConfig{
        Enabled:      true,
        WatchPaths:   []string{"./routes", "./islands", "./static"},
        DebounceTime: 500 * time.Millisecond,
    },
})
```

## State Preservation

GoSPA's HMR system is designed to preserve your application's reactive state across updates. When a component is updated, its state is serialized, stored, and then re-applied to the new version of the component.

### Registering for State Preservation

You can manually register state for preservation using the `window.__gospaHMR` API:

```javascript
if (import.meta.hot) {
    import.meta.hot.dispose((data) => {
        data.myState = currentLocalState;
    });
}
```

## Troubleshooting

- **Full Reloads**: If the HMR system cannot safely apply an update (e.g., changes to internal framework logic), it will trigger a full page reload to ensure consistency.
- **Connection Issues**: The HMR client will attempt to reconnect automatically if the WebSocket connection is lost.
