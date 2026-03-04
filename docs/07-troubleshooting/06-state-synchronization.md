# Troubleshooting State Synchronization

## State Changes Not Syncing to Client

### Problem
Server state changes (Runes) don't appear on the client.

### Causes & Solutions

#### 1. WebSocket Not Enabled

```go
app := gospa.New(gospa.Config{
    WebSocket: true,  // Required for real-time sync
})
```

#### 2. Component Not Registered for Sync

```go
// Register component state for synchronization
stateMap := state.NewStateMap()
stateMap.Add("count", count)
stateMap.Add("user", userRune)

// In your handler, use SyncState
func HandlePage(c *routing.Context) error {
    return c.RenderWithState("page", data, stateMap)
}
```

#### 3. Not Broadcasting Changes

```go
// BAD - Set without broadcast
count.Set(42)

// GOOD - Broadcast to all clients
app.BroadcastState("count", count.Get())

// Or use automatic sync
count.Set(42)
app.SyncComponentState("counter-component", stateMap)
```

---

## Client Changes Not Reaching Server

### Problem
Client state updates don't propagate to the server.

### Solutions

#### 1. Use Synced Rune

```javascript
import { syncedRune, getWebSocket } from '@gospa/runtime';

const ws = getWebSocket();
const count = syncedRune('count', 0, ws);

// This automatically sends to server
count.set(5);
```

#### 2. Manual State Sync

```javascript
import { getWebSocket } from '@gospa/runtime';

const ws = getWebSocket();

function updateServerState(key, value) {
    ws.send({
        type: 'state:update',
        key: key,
        value: value,
        componentId: 'my-component',
    });
}

// Usage
updateServerState('count', 42);
```

#### 3. Check WebSocket Connection

```javascript
const ws = getWebSocket();

if (!ws.isConnected()) {
    console.error('WebSocket not connected, state will not sync');
    await ws.connect();
}
```

---

## "State Message Too Large" Error

### Problem
WebSocket closes with error about message size.

### Cause
State exceeds 64KB limit (WebSocket frame limit).

### Solutions

#### 1. Enable State Diffing

```go
app := gospa.New(gospa.Config{
    WebSocket:   true,
    StateDiffing: true,  // Only send changes, not full state
})
```

#### 2. Prune Large Fields

```go
type User struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Avatar   string `json:"avatar,omitempty"`
    // Exclude large fields from sync
    History  []Action `json:"-"`  // Don't sync
}

// Or use a DTO for sync
type UserSyncDTO struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}
```

#### 3. Compress State

```go
app := gospa.New(gospa.Config{
    WebSocket:     true,
    CompressState: true,  // Compress before sending
})
```

#### 4. Paginate Large Lists

```go
// BAD - Sync entire list
items := state.NewRune(allItems)  // 1000+ items

// GOOD - Sync paginated view
pageItems := state.NewRune(allItems[:20])
totalPages := state.NewRune(len(allItems) / 20)
```

---

## Circular Reference Errors

### Problem
State fails to serialize due to circular references.

### Error
```
json: unsupported value: encountered a cycle
```

### Solutions

#### 1. Use JSON Tags to Exclude

```go
type Node struct {
    ID       string  `json:"id"`
    Value    int     `json:"value"`
    Parent   *Node   `json:"-"`  // Exclude from JSON
    Children []*Node `json:"children"`
}
```

#### 2. Create DTOs for Sync

```go
// Domain model with circular refs
type Category struct {
    ID       string
    Parent   *Category
    Children []*Category
}

// DTO for synchronization
type CategoryDTO struct {
    ID       string   `json:"id"`
    ParentID string   `json:"parentId"`
    Children []string `json:"children"` // Just IDs
}

// Convert before setting state
categoryRune.Set(toDTO(category))
```

#### 3. Use Custom Marshal

```go
type SafeNode struct {
    *Node
}

func (n SafeNode) MarshalJSON() ([]byte, error) {
    // Custom serialization that avoids cycles
    type nodeJSON struct {
        ID       string      `json:"id"`
        Value    int         `json:"value"`
        Children []nodeJSON  `json:"children"`
    }
    
    var convert func(*Node) nodeJSON
    convert = func(n *Node) nodeJSON {
        if n == nil {
            return nodeJSON{}
        }
        children := make([]nodeJSON, len(n.Children))
        for i, c := range n.Children {
            children[i] = convert(c)
        }
        return nodeJSON{
            ID:       n.ID,
            Value:    n.Value,
            Children: children,
        }
    }
    
    return json.Marshal(convert(n.Node))
}
```

---

## Race Conditions in State Updates

### Problem
Multiple simultaneous updates cause inconsistent state.

### Solutions

#### 1. Use Server-Side Batching

```go
state.Batch(func() {
    count.Set(1)
    name.Set("updated")
    items.Set(newItems)
    // All sent as single atomic update
})
```

#### 2. Optimistic Updates with Rollback

```javascript
import { optimisticUpdate } from '@gospa/runtime';

// Apply immediately, rollback on error
optimisticUpdate(count, 5, {
    onConfirm: () => console.log('Server confirmed'),
    onRollback: (error) => {
        console.error('Update failed, rolled back:', error);
    },
});
```

#### 3. Version Your State

```go
type VersionedState struct {
    Version int         `json:"version"`
    Data    interface{} `json:"data"`
}

func (v *VersionedState) Update(fn func(interface{}) interface{}) error {
    newData := fn(v.Data)
    
    // Check for conflicts
    if v.Version != currentVersion {
        return errors.New("state conflict detected")
    }
    
    v.Version++
    v.Data = newData
    return nil
}
```

---

## State Out of Sync After Reconnection

### Problem
Client reconnects to WebSocket but state is stale.

### Solutions

#### 1. Request Full Sync on Connect

```javascript
const ws = initWebSocket({
    url: 'ws://localhost:3000/_gospa/ws',
    onConnect: () => {
        // Request current state from server
        ws.send({ type: 'state:sync:request' });
    },
});
```

Server-side handler:

```go
func HandleStateSyncRequest(client *Client) {
    state := getCurrentState()
    client.Send(StateMessage{
        Type:  "state:sync:full",
        State: state,
    })
}
```

#### 2. Use Version Vectors

```go
type SyncState struct {
    Data    interface{}       `json:"data"`
    Version int64             `json:"version"`
    Vector  map[string]int64  `json:"vector"`  // Per-client versions
}

// On reconnect, client sends its last known version
// Server sends only changes since that version
```

---

## Memory Leaks from Subscriptions

### Problem
Component unmounts but subscriptions remain active.

### Solutions

#### 1. Always Unsubscribe

```javascript
import { getState } from '@gospa/runtime';

const count = getState('count');

// Subscribe
const unsubscribe = count.subscribe((value) => {
    updateUI(value);
});

// Cleanup on unmount
window.addEventListener('beforeunload', unsubscribe);

// Or in component lifecycle
function destroy() {
    unsubscribe();
}
```

#### 2. Use Effect for Auto-Cleanup

```javascript
import { Effect } from '@gospa/runtime';

const effect = new Effect(() => {
    console.log('Count:', count.get());
    
    // Cleanup function
    return () => {
        console.log('Effect cleaned up');
    };
});

effect.dependOn(count);

// Later
effect.dispose();  // Cleans up subscription
```

#### 3. WeakMap for Component State

```javascript
// Use WeakMap so state is GC'd when component is
const componentStates = new WeakMap();

function mountComponent(element) {
    const state = createComponentState();
    componentStates.set(element, state);
}

function unmountComponent(element) {
    // No need to manually delete - GC handles it
    // when element is removed from DOM
}
```

---

## Type Errors in State Values

### Problem
State value type differs between server and client.

### Solutions

#### 1. Consistent Types

```go
// Server - Use specific types, not interface{}
count := state.NewRune(0)  // int, not float64
```

```javascript
// Client - Match the type
const count = new Rune(0);  // number, default 0
```

#### 2. Validate on Receive

```javascript
ws.onMessage = (msg) => {
    if (msg.type === 'state:update') {
        const schema = stateSchemas[msg.key];
        const validated = schema.validate(msg.value);
        
        if (!validated.valid) {
            console.error('Invalid state received:', validated.errors);
            return;
        }
        
        updateState(msg.key, validated.value);
    }
};
```

#### 3. Use TypeScript

```typescript
// Define shared types
interface AppState {
    count: number;
    user: {
        id: string;
        name: string;
    } | null;
}

// Type-safe state access
function getState<K extends keyof AppState>(key: K): AppState[K] {
    return stateMap.get(key) as AppState[K];
}
```

---

## Common Mistakes

### Mistake 1: Mutating State Directly

```javascript
// BAD - Mutates without notification
const items = getState('items');
items.push(newItem);  // No notification!

// GOOD - Immutable update
const items = getState('items');
setState('items', [...items, newItem]);
```

### Mistake 2: Not Handling Disconnection

```javascript
// BAD - Updates fail silently when disconnected
function increment() {
    count.set(count.get() + 1);
}

// GOOD - Queue or notify
function increment() {
    if (!ws.isConnected()) {
        // Queue for later
        pendingUpdates.push({ key: 'count', value: count.get() + 1 });
        showOfflineWarning();
        return;
    }
    count.set(count.get() + 1);
}
```

### Mistake 3: Syncing Everything

```go
// BAD - Syncs all state including ephemeral
type ComponentState struct {
    User      User      // Sync
    FormDraft FormData  // Don't sync - ephemeral
    UIState   UIConfig  // Don't sync - client-only
}

// GOOD - Selective sync
type ComponentState struct {
    User User `json:"user"`  // Only sync user
    // Other fields excluded with `json:"-"`
}
```

### Mistake 4: Ignoring Conflicts

```go
// BAD - Last write wins
func UpdateState(key string, value interface{}) {
    state.Set(key, value)  // Ignores conflicts
}

// GOOD - Check for conflicts
func UpdateState(key string, value interface{}, clientVersion int) error {
    currentVersion := getVersion(key)
    
    if clientVersion < currentVersion {
        return fmt.Errorf("conflict: server has newer version %d vs %d", 
            currentVersion, clientVersion)
    }
    
    state.Set(key, value)
    incrementVersion(key)
    return nil
}
```
