# GoSPA Unimplemented Tools — Implementation Plan

Generated: 2026-02-25  
Status: **FULLY COMPLETED 2026-02-25** — All 10 phases implemented. Zero lint errors. Zero format issues.

This plan covers every API surface that is referenced in documentation, code comments,
or the website but **not yet implemented**, plus documentation and lint corrections.

---

## Summary of Gaps Found

| Category | Item | Location | Priority |
|----------|------|----------|----------|
| Config fields | `CompressState`, `StateDiffing` | `gospa.go` | Low |
| Config fields | `WSReconnectDelay`, `WSMaxReconnect`, `WSHeartbeat` | `gospa.go` | Medium |
| Config fields | `StateSerializer`, `StateDeserializer` | `gospa.go` | Medium |
| Config fields | `SSR` (global SSR mode) | `gospa.go` | Low |
| Middleware | `RequestLoggerMiddleware()` (no-op) | `fiber/middleware.go` | Low |
| Doc sync | `DEV_TOOLS.md` still says ErrorOverlay/DevTools are "planned" | `docs/DEV_TOOLS.md` | High |
| Error overlay | Not wired to Fiber's error handler in DevMode | `gospa.go` | High |
| Website | DevTools page references unimplemented functions | `website/routes/docs/devtools/page.templ` | Medium |
| gofmt | Three generated files not formatted | `examples/*/routes/generated_routes.go`, `website/routes/generated_routes.go` | High |

---

## Phase 1 — Fix gofmt Issues ✅ DONE

### 1.1 Run gofmt on all files

The following files need formatting:
- `examples/counter/routes/generated_routes.go`
- `examples/counter-test-prefork/routes/generated_routes.go`
- `website/routes/generated_routes.go`

**Action:** Run `gofmt -w` on each, or add a `gofmt -w ./...` step.

```bash
gofmt -w examples/counter/routes/generated_routes.go
gofmt -w examples/counter-test-prefork/routes/generated_routes.go
gofmt -w website/routes/generated_routes.go
```

---

## Phase 2 — Fix Documentation Sync ✅ DONE

### 2.1 Update `docs/DEV_TOOLS.md`

**Problem:** Lines 84-92 say `ErrorOverlay` and `DevTools/StateInspectorMiddleware` are "planned but not yet implemented."  
**Reality:** Both are **fully implemented** in `fiber/error_overlay.go` and `fiber/dev.go`.

**Changes needed:**

**Error Overlay section (line 84-88):**  
Replace the "Planned" note with actual usage documentation covering:
- `fiber.ErrorOverlayConfig` struct fields
- `fiber.NewErrorOverlay(config)` constructor
- `overlay.RenderOverlay(err, req)` method
- How to wire it into Fiber's error handler

**State Inspector section (line 90-93):**  
Replace the "Planned" note with actual usage docs covering:
- `fiber.NewDevTools(config DevConfig)` — creates dev tools
- `devTools.Start()` / `devTools.Stop()` lifecycle
- `fiber.StateInspectorMiddleware(devTools, config)` — add before routes
- `devTools.DevPanelHandler()` → mount at `/_gospa/dev`
- `devTools.DevToolsHandler()` → mount WebSocket at `/_gospa/dev/ws`
- `devTools.LogStateChange(key, oldVal, newVal, source)` — manual logging
- `devTools.GetStateLog()` / `devTools.GetStateKeys()`

**RequestLoggerMiddleware section (line 96-104):**  
Update note to accurately say it's a no-op placeholder and document the Fiber logger alternative.

### 2.2 Update `docs/API.md`

Verify the `ErrorOverlay` and `DevTools` APIs are documented (they are mentioned as planned in lines 133-138). Add a proper API reference section.

---

## Phase 3 — Wire Error Overlay to DevMode ✅ DONE

### 3.1 Integrate `ErrorOverlay` with Fiber's error handler

**File:** `gospa.go` — `New()` function  
**Problem:** `fiber/error_overlay.go` is fully implemented but never wired to Fiber's error handler in `gospa.go`. In DevMode, 500 errors should display the overlay HTML instead of a JSON response.

**Implementation:**

```go
// In gospa.go New(), after creating fiberApp:
if config.DevMode {
    overlay := fiber.NewErrorOverlay(fiber.DefaultErrorOverlayConfig())
    fiberApp.Use(func(c *fiberpkg.Ctx) error {
        err := c.Next()
        if err != nil {
            // Render error overlay for HTML requests in dev mode
            if strings.Contains(string(c.Request().Header.Peek("Accept")), "text/html") {
                overlayHTML := overlay.RenderOverlay(err, nil)
                c.Status(500)
                c.Set("Content-Type", "text/html; charset=utf-8")
                return c.SendString(overlayHTML)
            }
        }
        return err
    })
}
```

**Files to modify:**
- `gospa.go`: Add overlay to `setupMiddleware()`, add `strings` import

---

## Phase 4 — Implement `RequestLoggerMiddleware()` ✅ DONE

### 4.1 Remove the no-op and add actual logging

**File:** `fiber/middleware.go` — `RequestLoggerMiddleware()`  
**Current state:** Function returns a handler that calls `c.Next()` and logs nothing.

**Implementation:**

```go
// RequestLoggerMiddleware logs requests with method, path, status, and duration.
func RequestLoggerMiddleware() gofiber.Handler {
    return func(c *gofiber.Ctx) error {
        start := time.Now()
        err := c.Next()
        log.Printf("[%s] %s %d %v",
            c.Method(),
            c.Path(),
            c.Response().StatusCode(),
            time.Since(start),
        )
        return err
    }
}
```

**Files to modify:**
- `fiber/middleware.go`: Add `log` and `time` imports, implement the body

---

## Phase 5 — Implement `WSReconnectDelay`, `WSMaxReconnect`, `WSHeartbeat` Config Fields ✅ DONE

### 5.1 Pass WS config to the client runtime

**Files:** `gospa.go`  
**Current state:** The three fields exist in `Config` but are never read. The client runtime manages reconnect using its own hardcoded defaults (1s delay, 10 attempts, 30s heartbeat).

**Implementation:** When rendering the init script in `renderRoute()`, pass these values if set:

```go
// In renderRoute(), in the runtime.init({...}) call, add:
reconnectDelay := int(a.Config.WSReconnectDelay.Milliseconds())
if reconnectDelay == 0 {
    reconnectDelay = 1000 // default 1s
}
maxReconnect := a.Config.WSMaxReconnect
if maxReconnect == 0 {
    maxReconnect = 10 // default 10 attempts
}
heartbeat := int(a.Config.WSHeartbeat.Milliseconds())
if heartbeat == 0 {
    heartbeat = 30000 // default 30s
}

// Inject into script:
// wsReconnectDelay: %d,
// wsMaxReconnect: %d,
// wsHeartbeat: %d,
```

**Client runtime:** The runtime already supports reconnect config; the Go side just needs to pass values into the `init()` call.

**Files to modify:**
- `gospa.go`: Read and inject WS config into the rendered script block

---

## Phase 6 — Implement `CompressState` ✅ DONE

### 6.1 Compress state JSON with gzip before sending

**File:** `fiber/websocket.go` — `SendState()` and `SendInitWithSession()`  
**Current state:** State is sent as plain JSON. `CompressState` config field exists but has no effect.

The difficulty: `gospa.Config` is not available inside the `fiber` package. Options:
1. Pass a `compress bool` flag to `SendState()`.
2. Add a `CompressState bool` field to `WebSocketConfig`.

**Implementation Plan:**
1. Add `CompressState bool` to `WebSocketConfig` in `fiber/websocket.go`
2. In `WebSocketHandler`, read `config.CompressState ` 
3. In `WSClient`, add a `compressState bool` field set at creation time
4. In `SendState()` / `SendInitWithSession()`: if `compressState`, gzip the JSON before sending, set a `"compressed": true` flag in the outer message
5. In `gospa.go` `setupRoutes()`: pass `CompressState: a.Config.CompressState` to `WebSocketConfig`
6. Client runtime: detect `"compressed": true` and decompress using `DecompressionStream` API

**Files to modify:**
- `fiber/websocket.go`: Add compress support to `WebSocketConfig`, `WSClient`, `SendState`, `SendInitWithSession`
- `gospa.go`: Wire `Config.CompressState` to `WebSocketConfig`
- `client/src/websocket.ts`: Add decompression support

---

## Phase 7 — Implement `StateDiffing` ✅ DONE

### 7.1 Send only changed state keys instead of full state

**Current state:** On every state sync, the full state JSON is sent. `StateDiffing` exists but is ignored.

**Implementation Plan:**
1. In `WSClient`, add a `lastSentState map[string]interface{}` field
2. In `SendState()`: if `stateDiffing`, compute the diff between `lastSentState` and current state, send only changed keys with a `"type": "patch"` message
3. Client runtime: handle `"type": "patch"` messages by merging partial state

**Files to modify:**
- `fiber/websocket.go`: Add `StateDiffing bool` to `WebSocketConfig`, implement diff logic
- `gospa.go`: Wire `Config.StateDiffing` to `WebSocketConfig`
- `client/src/websocket.ts`: Handle `"patch"` message type

---

## Phase 8 — Implement `StateSerializer` / `StateDeserializer` ✅ DONE

### 8.1 Allow custom serialization for state

**Current state:** State is always JSON-serialized. `StateSerializerFunc` and `StateDeserializerFunc` types exist in `gospa.go` but nothing uses them.

**Implementation Plan:**
1. Add `Serializer StateSerializerFunc` and `Deserializer StateDeserializerFunc` to `WebSocketConfig`
2. In `SendState()` / `SendInitWithSession()`: if `Serializer` is set, use it instead of `json.Marshal`
3. In `DefaultMessageHandler` on `"update"`: if `Deserializer` is set, use it instead of `json.Unmarshal`
4. In `gospa.go`: wire `Config.StateSerializer/Deserializer` to `WebSocketConfig`

**Files to modify:**
- `fiber/websocket.go`: Add `Serializer/Deserializer` to `WebSocketConfig`, use in send/receive paths
- `gospa.go`: Wire `Config.StateSerializer/StateDeserializer`

---

## Phase 9 — Update Website Docs ✅ DONE

### 9.1 Update `website/routes/docs/devtools/page.templ`

The State Inspector section references:
```typescript
import { enableStateInspector } from 'gospa/runtime';
enableStateInspector({ logChanges: true, showTimeline: true, highlightUpdates: true });
window.__GOSPA_STATE_INSPECTOR__.getState();
```

These functions don't exist. Replace with the real API:

```go
// Server-side (real API):
devTools := fiber.NewDevTools(fiber.DevConfig{...})
app.Use(fiber.StateInspectorMiddleware(devTools, config))
app.Get("/_gospa/dev", devTools.DevPanelHandler())
app.Get("/_gospa/dev/ws", devTools.DevToolsHandler())
```

And:
```js
// Client-side (real API):
window.__GOSPA__.globalState  // Access live state directly
```

Also fix the Performance Profiling section — `window.__GOSPA_PROFILER__` doesn't exist. Replace with a note that performance profiling is not yet implemented and direct users to browser DevTools.

### 9.2 Fix Error Overlay section in website

Update to show actual working usage with `fiber.NewErrorOverlay(config)` wired into Fiber's error handler. Remove the reference to `displayError()` from `@gospa/runtime` (doesn't exist).

---

## Phase 10 — Lint & Format ✅ DONE

### 10.1 Run gofmt

```bash
gofmt -w ./...
```

Specifically fixes:
- `examples/counter/routes/generated_routes.go`
- `examples/counter-test-prefork/routes/generated_routes.go`
- `website/routes/generated_routes.go`

### 10.2 Run golangci-lint

```bash
golangci-lint run --timeout 120s
```

Verify zero errors before and after all changes.

---

## Execution Order

1. **Phase 1** — gofmt (no code risk, trivial)
2. **Phase 2** — Fix `docs/DEV_TOOLS.md` (docs only)
3. **Phase 3** — Wire ErrorOverlay to DevMode (small wiring change)
4. **Phase 4** — Implement `RequestLoggerMiddleware()` (trivial)
5. **Phase 5** — Pass WS reconnect config to client (low risk)
6. **Phase 9** — Update website docs (templ/docs only)
7. **Phase 6** — `CompressState` (medium complexity)
8. **Phase 7** — `StateDiffing` (medium complexity)
9. **Phase 8** — `StateSerializer/Deserializer` (medium complexity)
10. **Phase 10** — Final lint + format pass

---

## Files to Modify

| File | Phases |
|------|--------|
| `gospa.go` | 3, 5, 6, 7, 8 |
| `fiber/middleware.go` | 4 |
| `fiber/websocket.go` | 6, 7, 8 |
| `docs/DEV_TOOLS.md` | 2 |
| `docs/API.md` | 2 |
| `website/routes/docs/devtools/page.templ` | 9 |
| `examples/counter/routes/generated_routes.go` | 1 |
| `examples/counter-test-prefork/routes/generated_routes.go` | 1 |
| `website/routes/generated_routes.go` | 1 |

---

## What Is NOT Changing

- The public Go API surface (no breaking changes to existing implemented methods)
- `CompressState`, `StateDiffing`, `StateSerializer`, `StateDeserializer` fields remain in `Config` — they just get wired up
- Plugin system untouched
- Client runtime TypeScript (except for new message type handling)
- No new required configuration
