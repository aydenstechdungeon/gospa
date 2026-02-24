# GoSPA Documentation Gap Analysis

This document identifies all undocumented or partially documented public APIs, configuration options, and internal modules of the GoSPA framework.

---

## 1. Go Server-Side (Fiber/Templ) Gaps

### 1.1 `gospa` Core Package
*   **`Config`**: Missing documentation for `RoutesFS`, `StaticDir`, `StaticPrefix`, `RuntimeScript`, `WebSocketPath`, `WebSocketMiddleware`, `DevConfig`, `ErrorOverlayConfig`.
*   **`App`**: Missing `Scan()`, `RegisterRoutes()`, `Use()`, `Group()`, `HandleSSE()`, `HandleWS()`.

### 1.2 `state` Package
*   **`serialize.go`**: **ENTIRELY UNDOCUMENTED**.
    *   `StateMap.AddAny()`, `StateMap.ForEach()`, `StateMap.ToMap()`, `StateMap.MarshalJSON()`.
    *   `StateValidator`, `NewStateValidator()`, `AddValidator()`, `Validate()`, `ValidateAll()`.
    *   `SerializeState()`.
*   **`rune.go`**: Missing `ID()`, `MarshalJSON()`, `GetAny()`, `SubscribeAny()`, `SetAny()`.
*   **`derived.go`**: Missing `DependOn()`, `ID()`, `MarshalJSON()`.
*   **`effect.go`**: Missing `DependOn()`, `IsActive()`, `Pause()`, `Resume()`, `Dispose()`.

### 1.3 `routing` Package
*   **`params.go`**: **ENTIRELY UNDOCUMENTED**.
    *   `Params` type, `GetDefault()`, `Has()`, `Set()`.
*   **`auto.go`**: Internal logic for file-based routing is undocumented.
*   **`manual.go`**: Manual route registration API is undocumented.
*   **`registry.go`**: Route registry and lookup logic is undocumented.

### 1.4 `fiber` Package (Integrations)
*   **`sse.go`**: **ENTIRELY UNDOCUMENTED**.
    *   `SSEBroker`, `SSEEvent`, `SSEClient`, `SSEConfig`.
    *   `NewSSEBroker()`, `SetupSSE()`, `SSEHelper`.
    *   Notification API: `Notify()`, `NotifyAll()`, `NotifyTopic()`.
    *   Update API: `Update()`, `UpdateAll()`, `UpdateTopic()`.
*   **`hmr.go`**: **ENTIRELY UNDOCUMENTED**.
    *   `HMRManager`, `HMRConfig`, `HMRFileWatcher`.
    *   `HMRMessage`, `HMRUpdatePayload`.
    *   `RegisterClient()`, `Broadcast()`, `PreserveState()`.
*   **`error_overlay.go`**: **ENTIRELY UNDOCUMENTED**.
    *   `ErrorOverlay`, `ErrorInfo`, `StackFrame`, `RequestInfo`.
    *   `ErrorOverlayConfig`, `DefaultErrorOverlayConfig()`.
*   **`websocket.go`**: Partially documented in `API.md` but missing detailed server-side API for `WSHub` internals.
*   **`dev.go`**: **ENTIRELY UNDOCUMENTED**.
    *   `DevTools`, `DevConfig`, `FileWatcher`.
    *   `StateLogEntry`, `DebugMiddleware()`, `StateInspectorMiddleware()`.
*   **`middleware.go`**: **ENTIRELY UNDOCUMENTED**.
    *   All framework-specific Fiber middlewares.
*   **`errors.go`**: **ENTIRELY UNDOCUMENTED**.
    *   Custom error types and handling logic.

---

## 2. TypeScript Client Runtime Gaps

### 2.1 `state.ts` (Reactive Primitives)
*   **`Rune<T>`**: Missing documentation for `ID()`, `toJSON()`, `toString()`, `valueOf()`, `peek()`.
*   **`Derived<T>`**: Missing `toJSON()`.
*   **`Effect`**: Missing `isActive`, `pause()`, `resume()`, `dispose()`.
*   **`StateMap`**: Missing `clear()`, `size`, `keys()`, `values()`, `entries()`.

### 2.2 `dom.ts` (DOM Bindings)
*   **`registerBinding()`**, **`unregisterBinding()`**: Internal binding registry is undocumented.
*   **`setSanitizer()`**, **`getSanitizer()`**: Sanitization configuration.

### 2.3 `events.ts` (Event System)
*   **`offAll()`**: Undocumented.
*   **`transformers`**: Many event modifiers (stop, prevent, self, etc.) are undocumented.
*   **`keys`**: Keyboard mapping constants are undocumented.

### 2.4 `navigation.ts` (SPA)
*   **`back()`, `forward()`, `go()`**: Navigation history API is undocumented.
*   **`initNavigation()`, `destroyNavigation()`**: Manual navigation setup.
*   **`createNavigationState()`**: State handling for navigation.

### 2.5 `sse.ts` (SSE Client)
*   **ENTIRELY UNDOCUMENTED**.
    *   `SSEClient`, `SSEEvent`, `SSEConnectionState`, `SSEConfig`.
    *   `on()`, `off()`, `subscribe()`, `unsubscribe()`, `close()`.

### 2.6 `streaming.ts` (Streaming SSR)
*   **ENTIRELY UNDOCUMENTED**.
    *   `StreamChunk`, `IslandData`, `HydrationQueue`.
    *   `StreamingRuntime`, `initStreaming()`, `processChunk()`.

### 2.7 `island.ts` (Island Hydration)
*   **ENTIRELY UNDOCUMENTED**.
    *   `IslandElementData`, `IslandHydrationMode`, `IslandPriority`.
    *   `IslandManager`, `registerIsland()`, `hydrateIsland()`.

### 2.8 `priority.ts` (Priority Hydration)
*   **ENTIRELY UNDOCUMENTED**.
    *   `PriorityScheduler`, `PriorityLevel`, `PriorityIsland`, `HydrationPlan`.
    *   `registerPlan()`, `forceHydrate()`, `getStats()`.

### 2.9 `hmr.ts` (HMR Client)
*   **ENTIRELY UNDOCUMENTED**.
    *   `HMRClient`, `HMRMessage`, `ModuleRegistry`.
    *   `CSSHMR`, `TemplateHMR`.
    *   `initHMR()`, `registerHMRModule()`, `acceptHMR()`.

### 2.10 `error_overlay.ts` (Error UI)
*   **ENTIRELY UNDOCUMENTED**.
    *   `ErrorOverlayClient`, `displayError()`, `clearError()`, `reportError()`.

### 2.11 `runtime-simple.ts` (Minimal Runtime)
*   **ENTIRELY UNDOCUMENTED**.
    *   Features included vs excluded.
    *   Selection criteria.

---

## 3. Summary of Documentation Debt

- **Go Files**: 16 files analyzed
    - **Documented**: 6 (37.5%)
    - **Undocumented**: 10 (62.5%)
- **TypeScript Files**: 18 files analyzed
    - **Documented**: 8 (44.4%)
    - **Undocumented**: 10 (55.6%)
- **Overall Coverage**: ~41%
