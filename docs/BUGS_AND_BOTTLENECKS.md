# Prefork and Redis Review - Findings

Reviewing the Redis and Prefork implementations in GoSPA reveals three severe issues: one massive performance bottleneck (with associated rare conditions) and two architectural flaws regarding Prefork isolation.

## 1. Massive Performance Bottleneck & Data Race in `client.State.OnChange`
**Location:** `fiber/websocket.go` (inside `WebSocketHandler`)

Whenever a websocket client triggers a state update, `client.State.OnChange` intercepts it to persist it to Redis and sync it across clients:

```go
client.State.OnChange = func(key string, value any) {
    // Save state to persistent store safely
    globalClientStateStore.Save(sessionID, client.State)
```

**The Bottleneck:**
`globalClientStateStore.Save` calls `sm.MarshalJSON()`, converting the **entire state map** (which can grow to megabytes for complex SPAs) into JSON and issuing a synchronous `SET` command to Redis over the network. Because this is triggered on every single state mutation (e.g., every keystroke in a bound text input), it crushes the CPU with allocations and blocks the network thread, hammering Redis with the full state payload.

**The Race Condition:**
If two clients sharing the same session ID (or two tabs) update different keys simultaneously in different processes, they both read and update their local full `StateMap` and write it to Redis. The last one to `SET` the full map to Redis clobbers and deletes the other's update (Lost Update).

## 2. Prefork Scope Isolation for WebSocket Session Sync
**Location:** `fiber/websocket.go` (inside `client.State.OnChange`)

When syncing a session update, the code manually loops over the websocket `Hub` clients:

```go
config.Hub.mu.RLock()
for _, hubClient := range config.Hub.Clients {
    if hubClient.SessionID == sessionID {
        _ = hubClient.SendJSON(...)
    }
}
config.Hub.mu.RUnlock()
```

**The Bug:**
This only loops over the clients connected locally to the current Prefork process! It does not utilize the `h.pubsub` mechanism. If a user is logged in on their phone (hit worker A) and their computer (hit worker B) under the same session, changes made on the phone will not reflect on the computer because the sync message is never broadcasted via Redis `pubsub`.

## 3. SSE Broker is Completely Isolated in Prefork
**Location:** `fiber/sse.go` 

The `SSEBroker` is initialized purely in memory:
```go
type SSEBroker struct {
clients map[string]*SSEClient
    // ... no pubsub adapter
}
```

When you call `broker.Broadcast` or `broker.BroadcastToTopic`, it only ranges over its local `b.clients` map. 

**The Bug:**
SSE does not support distributed environments or Prefork. If you run GoSPA in Prefork mode and emit a global notification via `SSEBroker.Broadcast()`, only the users randomly load-balanced to that specific worker process will receive the Server-Sent Event. The framework needs a `store.PubSub` integration for SSE similar to what was attempted for WebSockets.

---
## Summary Recommendations:
1. **Debounce / Diffing for Redis:** Stop fully marshaling and synchronously saving the state on every tick via `OnChange`. Either debounce Saves asynchronously, or write discrete state patch updates to Redis (`HSET` or JSON patch) instead of replacing the entire key. 
2. **Session Sync via PubSub:** Update `client.State.OnChange` to publish a targeted sync payload like `{"type": "targeted_sync", "sessionID": "<id>", "payload": ...}` to the Redis `gospa:broadcast` channel. The subscriber should then route that payload to matching sessionIDs across all processes.
3. **SSE Redis Integration:** Wire the `SSEBroker` to accept a `store.PubSub` backend instance during initialization and route its broadcasts through Redis so events span all prefork worker processes.
