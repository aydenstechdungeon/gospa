# GoSPA Comprehensive Security & Code Quality Audit

**Audit Date:** 2026-03-03  
**Auditor:** Kilo Code (AI Code Reviewer)  
**Framework Version:** v0.1.13  
**Scope:** Full codebase audit including security, performance, DX, and documentation

---

## Executive Summary

GoSPA is a sophisticated Go-based SPA framework with Svelte-like reactivity, file-based routing, WebSocket state synchronization, and a plugin ecosystem. The codebase demonstrates solid architectural decisions with reactive primitives, SSR/SSG/ISR/PPR rendering strategies, and multi-process support via Redis.

**Overall Assessment:** The framework is well-architected with good security practices in most areas. However, several critical and high-priority issues require attention before production use, particularly around error handling race conditions, WebSocket security edge cases, and documentation gaps.

---

## Critical Issues (Must Fix Before Production)

### 1. Race Condition in PPR Shell Building ([`gospa.go:741-796`](gospa.go:741))
**Severity:** CRITICAL  
**Confidence:** 95%  
**Category:** Concurrency Bug

**Problem:** The PPR (Partial Prerendering) shell building mechanism uses a `sync.Map` for deduplication but the actual synchronization primitive (`chan struct{}`) is stored in the same map. When another goroutine finds an existing entry, it waits on the channel, but there's a race condition where the channel could be closed before the waiting goroutine receives from it.

**Code:**
```go
// Line 741-746
done := make(chan struct{})
actual, loaded := a.pprShellBuilding.LoadOrStore(cacheKey, done)
if !loaded {
    defer func() {
        close(done)  // Race: another goroutine might be waiting
        a.pprShellBuilding.Delete(cacheKey)
    }()
```

**Impact:** Panic from closing an already-closed channel or deadlock in concurrent PPR requests.

**Fix:** Use a proper sync primitive like `sync.Cond` or a dedicated structure with reference counting:
```go
type pprBuild struct {
    done chan struct{}
    once sync.Once
}

// Use once.Do to ensure only one builder executes
build := &pprBuild{done: make(chan struct{})}
if actual, loaded := a.pprShellBuilding.LoadOrStore(cacheKey, build); loaded {
    <-actual.(*pprBuild).done
} else {
    defer close(build.done)
    // ... build shell
}
```

---

### 2. Unbounded State Growth in WebSocket Hub ([`fiber/websocket.go:364-442`](fiber/websocket.go:364))
**Severity:** CRITICAL  
**Confidence:** 92%  
**Category:** Memory Leak / DoS

**Problem:** The `WSHub.Broadcast` channel is created with a buffer of 256 but has no backpressure mechanism. If clients disconnect uncleanly or process messages slowly, the hub can accumulate goroutines trying to send to full client channels.

**Code:**
```go
// Line 382
Broadcast:  make(chan []byte, 256),  // Fixed buffer

// Lines 400-408 - blocking send with no timeout
for _, client := range h.Clients {
    if sessionID == "" || client.SessionID == sessionID {
        select {
        case client.Send <- message:  // Can block forever if client dead
        default:
            // Client buffer full - silently dropped
        }
    }
}
```

**Impact:** Memory exhaustion from stuck goroutines, potential DoS.

**Fix:** Add timeouts and proper client cleanup:
```go
select {
case client.Send <- message:
case <-time.After(100 * time.Millisecond):
    // Client too slow, consider disconnecting
    go client.Close()
}
```

---

### 3. Missing Input Validation in Remote Actions ([`gospa.go:477-525`](gospa.go:477))
**Severity:** CRITICAL  
**Confidence:** 90%  
**Category:** Security / Injection

**Problem:** Remote actions receive JSON input that is parsed into `interface{}` but there's no validation of the action name format or input size before passing to handlers. The `MaxRequestBodySize` check happens after partial parsing.

**Code:**
```go
// Lines 487-508 - body parsed before size check
var input interface{}
if len(c.Body()) > 0 {  // Size check happens here
    // ... but already potentially parsed large JSON
    if len(c.Body()) > a.Config.MaxRequestBodySize {  // Too late
```

**Impact:** Potential for memory exhaustion from deeply nested JSON or large payloads.

**Fix:** Validate body size before any parsing:
```go
body := c.Body()
if len(body) > a.Config.MaxRequestBodySize {
    return c.Status(413).JSON(fiberpkg.Map{
        "error": "Request too large",
        "code": "REQUEST_TOO_LARGE",
    })
}
// Then parse
```

---

## High Priority Issues

### 4. WebSocket Session Fixation Vulnerability ([`fiber/websocket.go:962-986`](fiber/websocket.go:962))
**Severity:** HIGH  
**Confidence:** 88%  
**Category:** Security

**Problem:** When a session token is provided via URL query parameter (legacy fallback), it's vulnerable to session fixation attacks and logging exposure. The token can appear in server logs, browser history, and referrer headers.

**Code:**
```go
// Lines 975-985 - URL parameter fallback
sessionParam := c.Query("session")
if sessionParam != "" {
    if prevSessionID, ok := globalSessionStore.ValidateSession(sessionParam); ok {
```

**Impact:** Session tokens exposed in logs, potential session hijacking.

**Fix:** Remove URL parameter support entirely or deprecate with strong warnings. The message-based token exchange is already implemented and should be the only method.

---

### 5. Missing CORS Preflight Handling ([`fiber/middleware.go`](fiber/middleware.go))
**Severity:** HIGH  
**Confidence:** 85%  
**Category:** Security / CORS

**Problem:** The `CORSMiddleware` doesn't properly handle OPTIONS preflight requests for WebSocket upgrade endpoints, which can cause CORS failures in browser clients.

**Impact:** WebSocket connections failing in cross-origin scenarios.

**Fix:** Add explicit OPTIONS handler:
```go
if c.Method() == "OPTIONS" {
    c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
    c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
    return c.SendStatus(204)
}
```

---

### 6. Unbounded Cache Growth with TTL ([`gospa.go:1003-1042`](gospa.go:1003))
**Severity:** HIGH  
**Confidence:** 87%  
**Category:** Performance / Memory

**Problem:** The SSG cache eviction with TTL uses `time.AfterFunc` which creates a goroutine per entry. With high cache churn, this creates many goroutines.

**Code:**
```go
// Line 1029 - one goroutine per cached entry
if a.Config.SSGCacheTTL > 0 {
    time.AfterFunc(a.Config.SSGCacheTTL, func() {
        // ... cleanup
    })
}
```

**Impact:** Goroutine explosion with high-traffic sites using TTL.

**Fix:** Use a single cleanup goroutine with a priority queue or periodic sweep.

---

## Medium Priority Issues

### 7. Type Assertion Panic Risk ([`fiber/errors.go:139-143`](fiber/errors.go:139))
**Severity:** MEDIUM  
**Confidence:** 82%  
**Category:** Stability

**Problem:** State recovery assumes `*state.StateMap` type without safe checking:

```go
if stateMap, ok := c.Locals(config.StateKey).(*state.StateMap); ok && stateMap != nil {
```

If a different type is stored with the same key, this could panic in other code paths.

---

### 8. Missing WebSocket Close Frame Handling ([`fiber/websocket.go:514-547`](fiber/websocket.go:514))
**Severity:** MEDIUM  
**Confidence:** 80%  
**Category:** Protocol Compliance

**Problem:** The ReadPump doesn't distinguish between abnormal and normal WebSocket closures, treating all errors the same.

**Impact:** Clients can't cleanly disconnect, error logs polluted with normal close events.

---

### 9. Dev Server Process Leak ([`cli/dev.go:103-121`](cli/dev.go:103))
**Severity:** MEDIUM  
**Confidence:** 85%  
**Category:** Resource Leak

**Problem:** When restarting the dev server, the old process might not be fully terminated before starting a new one, especially on Windows.

```go
if currentCmd != nil && currentCmd.Process != nil {
    _ = currentCmd.Process.Signal(os.Interrupt)
    _ = currentCmd.Wait()  // Might hang
}
```

---

### 10. State Diffing Race Condition ([`fiber/websocket.go:624-667`](fiber/websocket.go:624))
**Severity:** MEDIUM  
**Confidence:** 78%  
**Category:** Concurrency

**Problem:** `lastSentState` is accessed without proper synchronization between `SendState()` and the broadcaster:

```go
c.lastSentStateMu.Lock()
prev := c.lastSentState  // Read
c.lastSentStateMu.Unlock()
// ... compute diff ...
c.lastSentStateMu.Lock()
c.lastSentState = stateMap  // Write
c.lastSentStateMu.Unlock()
```

A concurrent send could interleave between these operations.

---

## Low Priority Issues

### 11. Documentation Inconsistencies

**Severity:** LOW  
**Confidence:** 95%

- `docs/API.md` mentions features not yet implemented (SSR global mode)
- `docs/SECURITY.md` CSRF example uses wrong function names (`gospa.CSRFSetTokenMiddleware` vs `fiber.CSRFSetTokenMiddleware`)
- `docs/CLIENT_RUNTIME.md` documents `GoSPA.navigate()` which doesn't exist in the runtime

### 12. CLI Version Mismatch

**Severity:** LOW  
**Confidence:** 90%

`cli/create.go` generates projects with `go 1.23` but the framework requires `go 1.26` per `go.mod`.

### 13. Error Message Information Disclosure

**Severity:** LOW  
**Confidence:** 85%

`fiber/websocket.go:536` returns detailed unmarshaling errors to clients:

```go
c.SendError("Invalid message format")  // Good
// vs
client.SendError(fmt.Sprintf("parse error: %v", err))  // Information leak
```

---

## Documentation Gaps

| Gap | Impact | Recommendation |
|-----|--------|----------------|
| No Rate Limiting Guide | DoS vulnerability | Document how to configure WS rate limits |
| Missing Prefork Best Practices | Data inconsistency | Add Redis/external storage requirement docs |
| No Security Checklist | Misconfiguration | Create production deployment checklist |
| Missing TypeScript Types | DX friction | Add @types/gospa or bundled types |
| No Migration Guide | Adoption barrier | Document version upgrade steps |

---

## Performance Observations

### Positive
- State diffing reduces payload size significantly
- Fast deep equality in client runtime (~10x vs JSON.stringify)
- Regex compilation cached per route
- ISR semaphore prevents thundering herd

### Concerns
- `fiber/websocket.go:752-850` deepEqual uses reflection which is slow for large objects
- `routing/auto.go:147` extracts params using regex instead of manual parsing for static routes
- `gospa.go:573-608` SSG cache check holds read lock during external storage call

---

## Security Positives

✅ DOMPurify included by default for XSS protection  
✅ Session tokens sent in messages, not URLs (primary method)  
✅ CSRF protection available with double-submit cookie  
✅ WebSocket connection rate limiting implemented  
✅ Input size limits on WebSocket messages  
✅ Constant-time comparison for Telegram auth  
✅ Secure random token generation  
✅ JWT secrets not hardcoded (generated at runtime)  

---

## Recommendations Summary

### Immediate Actions (Critical/High)
1. Fix PPR shell building race condition
2. Add WebSocket hub backpressure
3. Validate remote action body size before parsing
4. Remove or deprecate URL-based session tokens
5. Add CORS preflight handling

### Short Term (Medium Priority)
6. Replace per-entry TTL goroutines with sweep-based cleanup
7. Add type-safe state recovery
8. Improve WebSocket close handling
9. Add process termination timeout in dev mode

### Long Term (Low Priority)
10. Update documentation for accuracy
11. Add TypeScript declaration files
12. Create security hardening guide

---

## Appendix: Code Quality Metrics

| Metric | Score | Notes |
|--------|-------|-------|
| Error Handling | B+ | Good use of structured errors, some panic risks |
| Concurrency | B | Several race conditions need addressing |
| Security | A- | Good practices overall, minor issues |
| Documentation | B | Comprehensive but some inaccuracies |
| Test Coverage | C | Limited test files visible |
| Performance | B+ | Good optimizations, some bottlenecks |

---

*End of Audit Report*
