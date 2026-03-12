# GoSPA Comprehensive Audit (Security, Performance, Reliability, Docs)

Date: 2026-03-12  
Scope: `/workspace/gospa` (core framework, client runtime, plugins, docs)

## Executive Summary (Top 5)

| Rank | Severity | Category | Issue | Location |
|---|---|---|---|---|
| 1 | Medium | Broken Access Control | Public remote actions still possible if `AllowUnauthenticatedRemoteActions` is explicitly enabled. | `gospa.go` |
| 2 | Low | CSRF | CSRF can be disabled by config (`EnableCSRF: false`) for trusted environments. | `gospa.go` |
| 3 | Low | Reliability/Perf | Redis PubSub now supports context-aware cancellation; handlers still need bounded work. | `store/redis/redis.go` |
| 4 | Low | Performance/Scalability | Rate limiter lock scope reduced for storage backends; remaining optimization is per-key atomic updates. | `fiber/websocket.go` |
| 5 | Medium | Security (XSS footgun) | `SafeHTML`/`SafeAttr` intentionally bypass escaping and can be misused by app code. | `templ/bind.go` |

## Security Findings

### 1) Remote action authorization guard now secure-by-default (Fixed)
- **Status:** Fixed. In production, GoSPA now rejects remote action calls when `RemoteActionMiddleware` is not configured unless `AllowUnauthenticatedRemoteActions` is explicitly set.
- **Residual risk:** teams can still opt out of protection.
- **Safe PoC:**

```bash
curl -i -X POST http://localhost:3000/_gospa/remote/deleteAccount \
  -H 'content-type: application/json' \
  -d '{"userId":"victim"}'
```

- **Mitigation:** keep `AllowUnauthenticatedRemoteActions` at default `false` and always define `RemoteActionMiddleware`.

**Suggested patch (conceptual):**
```diff
@@ func (a *App) setupRoutes() {
- remoteHandlers := []fiberpkg.Handler{fiber.RemoteActionRateLimitMiddleware()}
+ remoteHandlers := []fiberpkg.Handler{fiber.RemoteActionRateLimitMiddleware()}
+ if !a.Config.DevMode && a.Config.RemoteActionMiddleware == nil {
+   remoteHandlers = append(remoteHandlers, func(c fiberpkg.Ctx) error {
+     return c.Status(fiberpkg.StatusUnauthorized).JSON(fiberpkg.Map{"error": "remote actions require auth middleware"})
+   })
+ }
```

### 2) CSRF default now enabled (Fixed)
- **Status:** Fixed. `EnableCSRF` default is now `true` in `DefaultConfig()`.
- **Residual risk:** setting `EnableCSRF: false` re-introduces exposure.
- **Safe PoC (if cookie auth exists):**

```html
<form action="https://target.example/_gospa/remote/transferFunds" method="POST">
  <input name='{"to":"attacker","amount":1}' />
</form>
<script>document.forms[0].submit()</script>
```

- **Mitigation:** keep CSRF enabled, and only disable for fully trusted internal traffic.

### 3) Unsafe HTML helper footgun (Medium)
- **Risk (OWASP A03 Injection / XSS):** `SafeHTML` and `SafeAttr` disable escaping intentionally.
- **Impact:** if app developers pass untrusted input to these helpers, stored/reflected XSS becomes trivial.
- **Safe PoC:**

```go
templ.Raw(templ.SafeHTML("<img src=x onerror=alert(1)>") )
```

- **Mitigation:** retain API but add strong warning docs + linter hook + optional runtime guard in dev mode.

### 4) Potential sensitive header overexposure to handlers (Low)
- **Risk:** all request headers are copied into `RemoteContext.Headers`; this can propagate secrets to logs/telemetry in downstream code.
- **Mitigation:** pass allowlisted headers only (`X-Request-Id`, tracing ids, etc.).

## Dependency/CVE Posture

- `govulncheck` was not available in the current environment.
- `bun` has no built-in `audit` subcommand in this environment (`bun pm audit` unsupported).
- Manual review indicates mostly modern pinned versions (`fiber/v3`, `jwt/v5`, `oauth2`) but no machine-verified CVE report was possible in-session.
- Recommended CI additions:
  - `go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...`
  - `osv-scanner --lockfile=client/bun.lock --lockfile=website/bun.lock --lockfile=go.sum`

## Performance Findings

| Issue | Impact | Fix | Expected Gain |
|---|---|---|---|
| Storage I/O under lock in rate limiter | **Improved**: storage I/O moved out of global lock in storage-backed path | optional per-key atomic update/Lua script for strict distributed consistency | 10-20% additional p95 gain possible |
| Header map copies per remote request | Extra allocs and GC churn | allowlist headers + pre-size map | 5-15% fewer allocs for hot remote APIs |
| Redis pubsub goroutines never cancellable | **Fixed**: `SubscribeWithContext` now supports cancellation lifecycle | adopt context-managed subscriptions in app lifecycles | Prevents memory growth over uptime |

## Reliability & Logic Findings

1. **Double-close panic risk** in rate limiter `Close()` if called multiple times.  
   - Fix with `sync.Once`.
2. **Silent failure path**: `CreateSession` returns empty string on RNG/storage errors without structured error propagation.  
   - Fix by returning `(string, error)`.
3. **Input-edge cases to add tests for:**
   - Remote action body at exact `MaxRequestBodySize` boundary.
   - Missing/invalid `Content-Type` with non-empty body.
   - Long action names (`>256`) and Unicode action names.

## Documentation Audit

### README.md
- Missing security hardening checklist (CSP, CSRF, auth middleware defaults).
- Informal statement in intro (`pushing to main/master ...`) reduces production readiness signal.
- Missing quick troubleshooting matrix and compatibility table (Go/Bun versions).

### /docs
- Good breadth, but no consolidated “production security baseline” page with copy/paste config.
- No dedicated dependency scanning section with CI examples.

### /website docs parity
- Needs explicit mapping from runtime variants (`default` vs `secure`) to threat model and sanitizer guidance.

**Documentation completeness score:** **7/10**.

## Exploit Chain (Mermaid)

```mermaid
flowchart TD
    A[Victim logged in via cookie auth] --> B[Attacker website submits POST]
    B --> C[/_gospa/remote/:name endpoint]
    C --> D{EnableCSRF?}
    D -- No --> E[Action executes]
    D -- Yes --> F[Blocked without token]
    E --> G[State-changing operation]
```

## Prioritized Recommendations

1. **Secure defaults first**: require auth middleware for remote actions in production mode.
2. **Enable CSRF by default** for mutating endpoints when cookies are used.
3. **Refactor rate limiter locking** to avoid I/O while holding global mutex.
4. **Add cancellable subscription APIs** in Redis pubsub layer.
5. **Add CI security scanning** (govulncheck + OSV lockfile scan).
6. **Expand docs** with a production hardening checklist and secure deployment examples.
