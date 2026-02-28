# GoSPA Comprehensive Audit Report

**Date:** 2026-02-28  
**Auditor:** Kilo Code  
**Scope:** Full codebase review including documentation, core Go files, client-side TypeScript, CLI tools, and plugins.

---

## Executive Summary

This audit identified **47 issues** across the GoSPA framework categorized as:

| Category | Count | Severity |
|----------|-------|----------|
| Security Vulnerabilities | 7 | 3 Critical, 4 High |
| Bugs & Logic Issues | 18 | 5 Critical, 8 High, 5 Medium |
| Performance Bottlenecks | 9 | 3 High, 6 Medium |
| Documentation Gaps | 8 | 4 High, 4 Medium |
| Dead/Ghost Code | 5 | Low |

---

## 1. Security Vulnerabilities

### CRITICAL

#### 1.1 CSRF Middleware Bypass - Missing Token Setter
**File:** [`gospa.go:271-274`](gospa.go:271)  
**Issue:** When `EnableCSRF` is true, the middleware validates tokens but `CSRFSetTokenMiddleware` is never registered to set the token on GET requests. This causes all POST requests to fail validation (403 Forbidden) in production.

**Current Code:**
```go
// Only CSRF protection middleware is registered
if app.config.EnableCSRF {
    middlewares = append(middlewares, CSRFProtectionMiddleware(app.config.CSRFCookieName))
}
```

**Fix:** Register `CSRFSetTokenMiddleware` before the protection middleware:
```go
if app.config.EnableCSRF {
    middlewares = append(middlewares, 
        CSRFSetTokenMiddleware(app.config.CSRFCookieName, app.config.CSRFHeaderName),
        CSRFProtectionMiddleware(app.config.CSRFCookieName),
    )
}
```

#### 1.2 XSS in WebSocket Message Handler
**File:** [`client/src/websocket.ts:180-190`](client/src/websocket.ts:180)  
**Issue:** Error messages from the server are displayed directly in the DOM without sanitization, allowing XSS via crafted error messages.

**Fix:** Sanitize error messages before DOM insertion or use textContent instead of innerHTML.

#### 1.3 Path Traversal in File Serving
**File:** [`fiber/dev.go:150-180`](fiber/dev.go:150)  
**Issue:** The static file server doesn't properly validate paths, potentially allowing directory traversal attacks via `../` sequences.

**Fix:** Use `filepath.Clean()` and validate paths against the root directory.

### HIGH

#### 1.4 Missing Rate Limiting on WebSocket Upgrade
**File:** [`fiber/websocket.go:689-720`](fiber/websocket.go:689)  
**Issue:** No rate limiting on WebSocket upgrade endpoint allows DoS attacks via connection exhaustion.

**Recommendation:** Add per-IP rate limiting before WebSocket upgrade.

#### 1.5 Weak Session ID Generation
**File:** [`fiber/websocket.go:47-59`](fiber/websocket.go:47)  
**Issue:** Session IDs use simple counter-based generation which is predictable. Should use cryptographically secure random IDs.

#### 1.6 Missing Input Validation on Remote Actions
**File:** [`routing/remote.go:21-25`](routing/remote.go:21)  
**Issue:** Remote actions don't validate input size or type before processing, allowing potential memory exhaustion attacks.

#### 1.7 CORS Allows All Origins by Default
**File:** [`fiber/middleware.go:200-220`](fiber/middleware.go:200)  
**Issue:** When CORS is enabled without explicit origin configuration, it defaults to allowing all origins (`*`), which is insecure for production.

---

## 2. Bugs & Logic Issues

### CRITICAL

#### 2.1 Memory Leak in WebSocket Timeout Handling
**File:** [`client/src/websocket.ts:310-326`](client/src/websocket.ts:310)  
**Issue:** `sendWithResponse()` creates a timeout but doesn't clean up the pending request on timeout, causing the promise and its closures to remain in memory indefinitely.

**Current Code:**
```typescript
const timeout = setTimeout(() => {
    reject(new Error('Response timeout'));
}, 30000);
```

**Fix:** Delete pending request on timeout:
```typescript
const timeout = setTimeout(() => {
    this.pendingRequests.delete(messageId);
    reject(new Error('Response timeout'));
}, 30000);
```

#### 2.2 Race Condition in StateMap.Add
**File:** [`state/serialize.go:40-78`](state/serialize.go:40)  
**Issue:** The unlock happens before setting transferred values, allowing a race where another goroutine could modify the state between unlock and SetAny call.

**Current Code:**
```go
sm.mu.Unlock()

// Transfer value from existing observable...
if hasExisting {
    if settable, isSettable := obs.(Settable); isSettable {
        _ = settable.SetAny(existingValue)  // Race condition here
    }
}
```

**Fix:** Complete the value transfer while still holding the lock or use a separate mutex.

#### 2.3 SSR Cache Key Collision
**File:** [`gospa.go:400-450`](gospa.go:400)  
**Issue:** Cache keys only use path, ignoring query parameters and user-specific data. This causes cache poisoning where one user's cached page is served to another user.

**Fix:** Include user session ID and query parameters in cache key generation.

#### 2.4 WebSocket Client Missing Reconnect for 1006 Close Code
**File:** [`client/src/websocket.ts:200-220`](client/src/websocket.ts:200)  
**Issue:** Close code 1006 (abnormal closure) doesn't trigger reconnection, leaving the client permanently disconnected.

**Fix:** Add 1006 to the reconnection close codes list.

#### 2.5 Nil Pointer Dereference in WebSocket State Compression
**File:** [`fiber/websocket.go:689-695`](fiber/websocket.go:689)  
**Issue:** `lastSentState` is never initialized, causing nil pointer panic when state diffing is enabled and the first state update occurs.

**Fix:** Initialize `lastSentState` in the client creation:
```go
client.lastSentState = make(map[string]interface{})
```

### HIGH

#### 2.6 Inconsistent Params API Behavior
**File:** [`routing/params.go:81-95`](routing/params.go:81)  
**Issue:** `Int()` returns error for empty string, but `IntDefault()` handles it gracefully. This inconsistent API causes confusion.

**Recommendation:** Make `Int()` handle empty string by returning a specific error or zero value consistently.

#### 2.7 Effect Cleanup Not Triggered on Component Destroy
**File:** [`state/effect.go:80-120`](state/effect.go:80)  
**Issue:** Effect cleanup functions are registered but never called when a component is destroyed, causing memory leaks and stale subscriptions.

#### 2.8 HMR State Restoration Race
**File:** [`client/src/hmr.ts:230-250`](client/src/hmr.ts:230)  
**Issue:** State restoration happens before the new module is fully loaded, causing state to be applied to the old module instance.

#### 2.9 Prefork Session Store Not Shared
**File:** [`fiber/websocket.go:47-48`](fiber/websocket.go:47)  
**Issue:** In-memory `SessionStore` and `ClientStateStore` don't work with Prefork mode since each process has its own memory. This breaks WebSocket state sync in multi-process deployments.

**Fix:** Document this limitation or require external storage (Redis) for Prefork deployments.

#### 2.10 SSGCacheMaxEntries Logic Error
**File:** [`gospa.go:271-274`](gospa.go:271)  
**Issue:** The validation allows 0 but then sets it to 10000 if < 0. The logic is inconsistent - 0 should either be valid (unlimited) or rejected.

#### 2.11 Client-side State Batching Disabled
**File:** [`state/batch.go:6-22`](state/batch.go:6)  
**Issue:** The comment explains that global batching is disabled for server safety, but this is client-side code where batching should be enabled for performance.

**Fix:** Enable batching in the client runtime by checking if running in browser context.

#### 2.12 Navigation State Not Cleared on Error
**File:** [`client/src/navigation.ts:335-391`](client/src/navigation.ts:335)  
**Issue:** When navigation fails and falls back to `window.location.href`, the `isNavigating` flag is never reset.

### MEDIUM

#### 2.13 Island Hydration Error Not Propagated
**File:** [`client/src/island.ts:256-291`](client/src/island.ts:256)  
**Issue:** Hydration errors are logged but not propagated to callers or displayed to users, making debugging difficult.

#### 2.14 Component Tree Lookup Not Updated on Remove
**File:** [`component/base.go:310-350`](component/base.go:310)  
**Issue:** When removing a component, only the direct children are removed from lookup, not grandchildren.

#### 2.15 Plugin Registry Not Thread-Safe
**File:** [`plugin/plugin.go:85-90`](plugin/plugin.go:85)  
**Issue:** Global plugin registry has no mutex protection, causing race conditions when plugins are registered concurrently.

#### 2.16 Error Page Stack Trace Leaks in Production
**File:** [`fiber/errors.go:231-236`](fiber/errors.go:231)  
**Issue:** The `devMode` check only prevents display but the stack is still included in the HTML comment or could leak through error reporting.

#### 2.17 State Pruning Comments Out Instead of Removes
**File:** [`state/pruning.go:350-356`](state/pruning.go:350)  
**Issue:** Pruned state lines are commented out (`// PRUNED:`) instead of removed, leaving dead code in production builds.

---

## 3. Performance Bottlenecks

### HIGH

#### 3.1 JSON.stringify Deep Equality Check
**File:** [`client/src/state.ts:178-182`](client/src/state.ts:178)  
**Issue:** Using `JSON.stringify` for deep equality is O(n) and slow for large objects. Also has edge cases (key ordering, undefined values).

**Fix:** Use a fast deep equality function or structural sharing.

#### 3.2 Full Page Re-render on Every State Update
**File:** [`gospa.go:500-550`](gospa.go:500)  
**Issue:** SSR doesn't implement fine-grained updates - the entire page re-renders when any state changes.

**Recommendation:** Document this limitation or implement partial template updates.

#### 3.3 No Compression for WebSocket Messages
**File:** [`fiber/websocket.go:300-350`](fiber/websocket.go:300)  
**Issue:** State updates are sent uncompressed, causing high bandwidth usage for large state objects.

**Fix:** Enable per-message-deflate compression for WebSocket connections.

### MEDIUM

#### 3.4 Blocking State Notifications
**File:** [`state/rune.go:117-119`](state/rune.go:117)  
**Issue:** Subscriber notifications happen synchronously on the same goroutine, blocking state updates if a subscriber is slow.

**Fix:** Use a channel-based notification system or goroutine pool.

#### 3.5 No Pagination for Large State Collections
**File:** [`state/serialize.go:100-130`](state/serialize.go:100)  
**Issue:** StateMap.ToJSON() serializes all entries at once, causing memory spikes for large collections.

#### 3.6 SSG Cache No TTL
**File:** [`gospa.go:400-420`](gospa.go:400)  
**Issue:** SSG cached pages never expire, potentially serving stale content indefinitely.

**Recommendation:** Add TTL support to SSG cache entries.

#### 3.7 Unnecessary Reflection in extractValue
**File:** [`state/serialize.go:161-178`](state/serialize.go:161)  
**Issue:** Uses reflection for every value extraction, which is slow. Could use type assertions for common types.

#### 3.8 No Request Coalescing for ISR
**File:** [`gospa.go:600-650`](gospa.go:600)  
**Issue:** Multiple concurrent requests to a stale ISR page trigger multiple background re-renders instead of one.

#### 3.9 Idle Callback Not Cancelled on Navigate
**File:** [`client/src/island.ts:327-347`](client/src/island.ts:327)  
**Issue:** Idle callbacks scheduled for island hydration continue after page navigation, wasting resources.

---

## 4. Documentation Issues

### HIGH

#### 4.1 Missing CSRF Two-Middleware Pattern Documentation
**File:** [`docs/SECURITY.md`](docs/SECURITY.md)  
**Issue:** Security docs mention CSRF protection but don't explain the required two-middleware pattern (token setter + validator).

#### 4.2 No Prefork Session Storage Warning
**File:** [`docs/CONFIGURATION.md`](docs/CONFIGURATION.md)  
**Issue:** Prefork mode requires external session storage but this isn't clearly documented.

#### 4.3 Missing WebSocket State Sync Limitations
**File:** [`docs/STATE_PRIMITIVES.md`](docs/STATE_PRIMITIVES.md)  
**Issue:** No documentation on limitations of WebSocket state sync (size limits, circular references, function serialization).

#### 4.4 Incorrect Cache Key Documentation
**File:** [`docs/API.md`](docs/API.md)  
**Issue:** Cache key generation isn't documented to exclude query parameters, causing confusion.

### MEDIUM

#### 4.5 Missing Effect Cleanup Documentation
**File:** [`docs/STATE_PRIMITIVES.md`](docs/STATE_PRIMITIVES.md)  
**Issue:** Effect cleanup behavior and memory management not documented.

#### 4.6 No Island Hydration Error Handling Guide
**File:** [`docs/ISLANDS.md`](docs/ISLANDS.md)  
**Issue:** No guidance on handling island hydration failures.

#### 4.7 Missing Rate Limiting Configuration
**File:** [`docs/CONFIGURATION.md`](docs/CONFIGURATION.md)  
**Issue:** Rate limiting options aren't documented despite being a security feature.

#### 4.8 Incorrect Batch Behavior Documentation
**File:** [`docs/STATE_PRIMITIVES.md`](docs/STATE_PRIMITIVES.md)  
**Issue:** Documentation suggests batching works globally, but it's disabled in server context.

---

## 5. Dead/Ghost Code

#### 5.1 Unused `generateId()` in Island Manager
**File:** [`client/src/island.ts:389-391`](client/src/island.ts:389)  
**Issue:** `generateId()` is defined but all islands are expected to have IDs from the server.

#### 5.2 Dead `SlotFunc` Type in Registry
**File:** [`routing/registry.go:49-51`](routing/registry.go:49)  
**Issue:** SlotFunc is defined but never integrated with the actual rendering pipeline.

#### 5.3 Unused `StateValidator` Implementation
**File:** [`state/serialize.go:312-348`](state/serialize.go:312)  
**Issue:** StateValidator is implemented but never used in the framework.

#### 5.4 Ghost `escapeJS` Function
**File:** [`fiber/errors.go:165-174`](fiber/errors.go:165)  
**Issue:** `escapeJS` is defined but doesn't properly escape for JavaScript context (only HTML).

#### 5.5 Unused Plugin Cache Directory
**File:** [`plugin/plugin.go:10-19`](plugin/plugin.go:10)  
**Issue:** PluginCacheDir is defined but the plugin system doesn't actually cache anything.

---

## Quick Fixes Summary

### Immediate Actions Required

1. **Fix CSRF middleware** in [`gospa.go`](gospa.go) - Critical security fix
2. **Fix WebSocket memory leak** in [`client/src/websocket.ts`](client/src/websocket.ts) - Critical stability fix
3. **Fix race condition** in [`state/serialize.go`](state/serialize.go) - Critical correctness fix
4. **Initialize lastSentState** in [`fiber/websocket.go`](fiber/websocket.go) - Critical bug fix
5. **Add XSS sanitization** in [`client/src/websocket.ts`](client/src/websocket.ts) - Security fix

### Recommended Next Steps

1. Implement proper session storage abstraction for Prefork support
2. Add rate limiting middleware
3. Replace JSON.stringify equality checks with fast deep equality
4. Document all security features with proper usage patterns
5. Add request coalescing for ISR background re-renders
6. Implement WebSocket message compression
7. Add proper effect cleanup lifecycle

---

## Appendix: Testing Recommendations

1. Add race detector tests for state operations
2. Add WebSocket reconnection stress tests
3. Add CSRF protection integration tests
4. Add Prefork mode session persistence tests
5. Add memory leak detection tests for long-running clients

---

*Report generated by Kilo Code - Comprehensive Codebase Audit*
