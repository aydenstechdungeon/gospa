# State Synchronization

GoSPA automatically synchronizes state between the server and the client using WebSockets.

## Snapshots and Diffs

1.  **Initial Snapshot**: When an island is first hydrated, the server sends a full snapshot of the `StateMap`.
2.  **Incremental Diffs**: When state changes on the server, a `StateDiff` message is sent with only the changed keys.
3.  **Automatic Patching**: The client runtime receives the diff and updates the corresponding reactive runes, triggering DOM updates.

## Message Format

```go
type StateMessage struct {
    Type        string      `json:"type"` // "init", "update", "sync", "error"
    ComponentID string      `json:"componentId,omitempty"`
    Key         string      `json:"key,omitempty"`
    Value       interface{} `json:"value,omitempty"`
    Timestamp   int64       `json:"timestamp"`
}
```

## Limitations

- **Max Message Size**: Large states should be optimized to fit within the `WSMaxMessageSize`.
- **Serialization**: Ensure your state objects are JSON-serializable and free of circular references.
- **Backpressure**: The server-side notification system uses a worker queue. Under extreme load, it falls back to synchronous delivery.
