# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.12] - 2026-03-04

### Added
- Created complete documentation suite and structure inside `docs/`.
- HSTS (`Strict-Transport-Security`) implemented securely in `SecurityHeadersMiddleware`.
- Deployment and production build instructions safely recorded.

### Fixed
- **Security**: Prevented raw remote action error exposure (`err.Error()`) to clients, now returns "Internal server error" string to mitigate credential and internal log leakage.
- **Security**: Fixed CSRF token fixation vulnerabilities by enforcing token issuance per GET/HEAD request. Scoped token path inherently to `/`.
- **Security**: Removed Cryptographic Modulo Bias from `randomString` when generating component IDs using boundary-rejection.
- **Security**: Upgraded `X-XSS-Protection` header from `1; mode=block` to `0` to prevent XS-Search attacks.
- **Security**: Fixed an unbounded query-parameter cache busting vulnerability in `SSG` cache mapping models.
- **Features**: Disabled `.Query()` search for `StateSync()`. Sessions are exclusively maintained over `Headers` to limit Referer metadata leakage.
- **Performance**: Improved array slicing limits for unbounded cache insertions, allowing for 10% batch slice truncation to prevent O(N) thrashing.
- **Performance**: Mutex bottleneck in state map initialization completely rewritten, optimizing the generation via atomic counters (`atomic.AddUint64`).
- **Performance**: Removed useless and slow background Goroutine loops from `.notify()` dependencies on `Derived` states. 
- **Stability**: Fixed memory leaks around untested Background Goroutines by tracking Channels within `MemoryStorage`, `ConnectionRateLimiter`, and `WSHub` loops. Cleanly tears down when `.Close()` or `App.Shutdown()` is invoked.
- **Stability**: Fixed RLock deadlocks when two goroutines initiate `StateMap.Diff()` operations concurrently against each other by buffering via an immediate `ToMap()` state clone execution.
- **Stability**: Fixed circular read deadlocks where a `Derived` value holding its RLock would invoke a mutable nested observer.
