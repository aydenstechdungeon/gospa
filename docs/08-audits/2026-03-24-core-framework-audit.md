# Core Framework Audit (App Init, Config, Middleware, Route Registration, Plugin Integration)

Date: 2026-03-24
Scope: `gospa.go`, `config.go`, `fiber/*`, `routing/*`, `plugin/*`, top-level dependency manifests.

## Executive Summary (Top 5)

| Rank | Severity | Area | Issue | Why it matters |
|---|---|---|---|---|
| 1 | **High** | Route registration/scanning | `Run()` can scan routes twice, and `Router.Scan()` appends without reset | Repeated route scans grow `r.routes`, increase memory/CPU, and risk duplicate route registration behavior. |
| 2 | **High** | Supply chain / plugin integration | External plugin loader clones arbitrary GitHub repos without signature/checksum verification | A compromised repo/tag can become trusted runtime code in plugin workflows. |
| 3 | **Medium** | Reliability / middleware ordering | CSRF token is rotated on every `GET/HEAD`, creating token churn across tabs/concurrent requests | Legitimate mutating requests can fail with 403 due to stale token race. |
| 4 | **Medium** | Observability / logic bug | Remote action context reads `RequestID` from **response** headers instead of request headers | Breaks tracing/correlation for remote actions and hampers incident response. |
| 5 | **Low** | Dev-mode perf/reliability | File watcher polls filesystem every 100ms using full directory walks | High CPU/IO in larger trees and can degrade developer UX. |

---

## Security Findings

### 1) External plugin loader lacks integrity verification (Supply chain) — **High**

**Code evidence**

- Loader clones directly from GitHub and trusts source tree contents to produce plugin metadata (`rev-parse HEAD`), but does not verify commit signature, release signatures, or checksums. It also supports optional mutable refs via `AllowMutableRefs(true)`. (`plugin/loader.go`)  
- This is a supply-chain risk even though command-injection protections are present (owner/repo/version validation).

**Safe PoC (non-destructive)**

```bash
# Demonstrates mutable ref acceptance when explicitly enabled.
# If a maintainer force-moves a tag, the fetched code can change without detection.
# (Do not run in production.)
loader := plugin.NewExternalPluginLoader().AllowMutableRefs(true)
_, err := loader.LoadFromGitHub("trusted-owner/trusted-repo@v1")
```

**OWASP mapping**: A06:2021 Vulnerable and Outdated Components / Software & Data Integrity Failures.

**Mitigation patch (conceptual diff)**

```diff
--- a/plugin/loader.go
+++ b/plugin/loader.go
@@
 type ExternalPluginLoader struct {
     cacheDir         string
     allowMutableRefs bool
+    requireSignedTag bool
+    expectedSHA256   string
 }
@@
 func (l *ExternalPluginLoader) download(owner, repo, version string) error {
@@
+    // Verify immutable integrity pin if provided
+    if l.expectedSHA256 != "" {
+        digest, err := sha256Dir(pluginPath)
+        if err != nil { return err }
+        if digest != l.expectedSHA256 {
+            return fmt.Errorf("plugin integrity mismatch")
+        }
+    }
+
+    // Optionally enforce signed refs/tags
+    if l.requireSignedTag {
+        verifyCmd := exec.Command("git", "-C", pluginPath, "verify-tag", version)
+        if err := verifyCmd.Run(); err != nil {
+            return fmt.Errorf("unsigned or untrusted tag: %w", err)
+        }
+    }
 }
```

---

### 2) CORS wildcard mode can still broaden attack surface when paired with token-based auth — **Medium**

**Code evidence**

- If `AllowedOrigins` contains `"*"`, middleware reflects `Access-Control-Allow-Origin: *` and allows auth-related headers (`Authorization`, `X-CSRF-Token`). (`fiber/middleware.go`)

**Risk note**

- With cookie auth, browser credential rules block wildcard credentials, but token-based frontends embedded on untrusted origins can still perform cross-origin API calls if application-level auth permits it.

**Safe PoC**

```bash
curl -i -X OPTIONS https://app.example.com/_gospa/remote/doThing \
  -H 'Origin: https://evil.example' \
  -H 'Access-Control-Request-Method: POST' \
  -H 'Access-Control-Request-Headers: authorization,content-type'
```

**Mitigation**

- Disallow `*` in production for state-changing endpoints; require explicit allowlist.
- Split public/static CORS policy from private API policy.

---

### 3) Remote action endpoint auth fail-safe is good, but enablement flags can disable protection — **Low/Contextual**

**Code evidence**

- In production, remote actions are blocked if no `RemoteActionMiddleware` is configured, **unless** `AllowUnauthenticatedRemoteActions` is set true. (`gospa.go`)

**Risk**

- Misconfiguration (setting allow-unauthenticated) can expose callable server actions.

**Mitigation**

- Add startup hard-fail in production unless an explicit per-action allowlist exists.

---

## Performance Findings

| Issue | Impact | Fix | Expected Gain |
|---|---|---|---|
| Route scan accumulation (`Router.Scan()` append-only) + double-scan path in `Run()`/`RegisterRoutes()` | Repeated scans inflate route slice and sorting cost over time; potential duplicate route processing | Clear `r.routes` before walk; avoid redundant `Scan()` in `Run()` when `RegisterRoutes()` already scans or make `RegisterRoutes()` idempotent | 20–60% less startup/registration overhead in iterative dev or repeated setup paths (estimate) |
| File watcher polling with recursive walk every 100ms | Elevated CPU/IO in medium/large trees | Use `fsnotify`-based watch tree with debounce fallback | 30–80% lower idle CPU in dev (estimate) |
| Per-route rate limiter instance allocated in registration loop | More alloc/state than necessary for large route sets | Reuse limiter pools or instantiate only where rate limit is configured and route count is high | Minor-to-moderate memory reduction |

### Key performance code-path evidence

- `Run()` conditionally scans when route list empty, then calls `RegisterRoutes()`, which scans again. (`gospa.go`)  
- `Router.Scan()` appends into `r.routes` without resetting. (`routing/auto.go`)

**Mitigation patch (minimal concrete diff)**

```diff
--- a/routing/auto.go
+++ b/routing/auto.go
@@
 func (r *Router) Scan() error {
+    // Reset previous scan results to make Scan idempotent.
+    r.routes = r.routes[:0]
+
     err := fs.WalkDir(r.fs, ".", func(path string, d fs.DirEntry, err error) error {
@@
 }
```

```diff
--- a/gospa.go
+++ b/gospa.go
@@
 func (a *App) Run(addr string) error {
@@
-    if len(a.Router.GetRoutes()) == 0 {
-        if err := a.Scan(); err != nil {
-            return err
-        }
-    }
     if err := a.RegisterRoutes(); err != nil {
         return err
     }
@@
 }
```

---

## Bugs & Logic Errors

### 1) Double-scan + non-idempotent scanner — **High**

**Repro**

1. Start app via `Run()`.
2. Call `RegisterRoutes()` again (or repeat flows that invoke scans).
3. Observe growing route registry length / repeated route work.

**Likely effect**: memory growth and route duplication hazards.

---

### 2) Request ID extraction bug in remote context — **Medium**

**Code evidence**

- `RemoteContext.RequestID` reads `c.GetRespHeader("X-Request-Id")`, i.e., response header, while trace headers are read from request. (`gospa.go`)

**Safe PoC**

```bash
curl -i -X POST http://localhost:3000/_gospa/remote/example \
  -H 'Content-Type: application/json' \
  -H 'X-Request-Id: req-123' \
  -d '{}'
# Server-side remote context may not receive req-123 due to wrong accessor.
```

**Fix patch**

```diff
--- a/gospa.go
+++ b/gospa.go
@@
-    RequestID: c.GetRespHeader("X-Request-Id"),
+    RequestID: c.Get("X-Request-Id"),
```

---

### 3) CSRF token churn/race across tabs — **Medium**

**Code evidence**

- Token is regenerated for every `GET/HEAD` in `CSRFSetTokenMiddleware()`. (`fiber/middleware.go`)

**Repro**

1. Open app in two tabs.
2. Tab A fetches page (token A).
3. Tab B fetches page (token rotates to B cookie).
4. Tab A submits mutating request with stale token A ⇒ 403 mismatch.

**Fix pattern**

- Only generate token if cookie absent or nearing expiry; keep stable session-bound token.

---

## Reliability & Edge Cases

- **Input validation**: Remote action name length/body size/depth checks are present (positive finding). (`gospa.go`)  
- **Error handling**: Recovery middleware exists, but full stack traces are always enabled in recover config; review production logging/privacy tradeoffs. (`gospa.go`)  
- **Plugin hooks**: `TriggerHook` executes plugin hooks concurrently; absence of per-plugin timeout/circuit-breaker can stall startup lifecycle if plugin blocks. (`plugin/plugin.go`)

### Fuzzing suggestions

- Remote action JSON parser fuzz corpus: deeply nested arrays/objects, large numbers, unicode edge cases.
- Route parser fuzz corpus: malformed segment tokens (`[[...`, nested brackets, extremely long segments).
- Header fuzz for middleware: malformed origin/csrf/request-id values.

---

## Dependency / CVE Scan Notes

### Attempted checks

- `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` → blocked (403 from `proxy.golang.org` in this environment).
- `bunx --yes osv-scanner --lockfile=bun.lock` (client and website) → blocked (403 from npm registry in this environment).

Because remote registries were inaccessible, CVE verification could not be completed in this run.

### Manual dependency review highlights

- Go modules are mostly current and specific versions are pinned in `go.mod`.
- JS deps use Bun lockfiles (`client/bun.lock`, `website/bun.lock`), which is good for reproducibility.

---

## Documentation Audit (Core framework docs quality)

### README.md score: **8/10**

Strengths:
- Clear quick-start and production baseline.
- Security section explicitly references scanning and CSP hardening.

Gaps:
- Add a dedicated “Core framework threat model” subsection (App init/middleware/route/plugin boundaries).
- Add explicit warning block for `AllowUnauthenticatedRemoteActions`.
- Add troubleshooting entry for CSRF token mismatch caused by multi-tab token rotation.

### `docs/` tree score: **8.5/10**

Strengths:
- Good API/reference segmentation and troubleshooting structure.

Gaps:
- Missing a focused “plugin supply chain hardening” guide (signed tags/checksum pinning).
- Missing operational runbook for route-scan idempotency and route registration lifecycle.

---

## Mermaid: Exploit/Failure Chain (Core Framework)

```mermaid
flowchart TD
    A[App Run()] --> B{Routes already scanned?}
    B -- No --> C[Scan routes]
    C --> D[RegisterRoutes()]
    B -- Yes --> D
    D --> E[RegisterRoutes calls Scan again]
    E --> F[Route slice grows if Scan append-only]
    F --> G[Higher CPU/memory + duplicate processing risk]

    H[External Plugin Ref] --> I[Git clone from GitHub]
    I --> J{Integrity verified?}
    J -- No --> K[Trusts fetched code/metadata]
    K --> L[Supply-chain compromise risk]
```

---

## Prioritized Recommendations

1. **Fix route-scan idempotency immediately** (`routing/auto.go`, `gospa.go`).
2. **Add plugin integrity controls** (checksum pinning + optional signed tag enforcement).
3. **Stabilize CSRF token issuance** (session-bound token instead of per-GET rotation).
4. **Fix `RequestID` extraction accessor** in remote context.
5. **Replace polling file watcher with fsnotify-based implementation** for dev performance.
6. **Complete CVE scan in CI/networked environment** using `govulncheck` + OSV/Snyk.
