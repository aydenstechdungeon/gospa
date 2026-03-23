# GoSPA

<div align="center">
  <img src="https://github.com/user-attachments/assets/9e9d126d-8c91-465a-a5c0-968e41c095fb" width="128" height="128" alt="GoSPA Logo 1">
  <img src="https://github.com/user-attachments/assets/338c5be2-9ce1-4f7a-a389-bfe176c6a9d6" width="128" height="128" alt="GoSPA Logo 2">
</div>

GoSPA (Go Spa and Go S-P-A are the only valid pronunciations)  brings Svelte-like reactive primitives (`Runes`, `Effects`, `Derived`) to the Go ecosystem. It is a high-performance framework for building reactive SPAs with Templ, Fiber, file-based routing, and real-time state synchronization.

## Highlights

- **Native Reactivity** - `Rune`, `Derived`, `Effect` primitives that work exactly like Svelte 5.
- **WebSocket Sync** - Transparent client-server state synchronization with GZIP delta patching.
- **File-Based Routing** - SvelteKit-style directory structure for `.templ` files.
- **Hybrid Rendering** - Mix SSR, SSG, ISR, and PPR on a per-page basis.
- **Type-Safe RPC** - Call server functions directly from the client without boilerplate endpoints.
- **High Performance** - Integrated `go-json` and optional MessagePack for minimal overhead.

## Quick Start

### 0. Prerequisites
- **Go 1.23+** (see `go.mod`; use a current stable toolchain)
- **Bun**: Required for the SPA build process (CSS extraction, Vite optimization, JS bundling).
- **`JWT_SECRET`**: Ensure this environment variable is set for production authentication contexts.

### 1. Install CLI
```bash
go install github.com/aydenstechdungeon/gospa/cmd/gospa@latest
```

### 2. Scaffold & Run
```bash
gospa create myapp
cd myapp
go mod tidy
gospa doctor
gospa dev
```

> For local client/runtime tooling, use Bun. The GoSPA client package and repo JS/TS workflows are Bun-first.

### 3. A Simple Page
```templ
// routes/page.templ
package routes

templ Page() {
    <div data-gospa-component="counter" data-gospa-state='{"count":0}'>
        <h1 data-bind="text:count">0</h1>
        <button data-on="click:increment">+</button>
    </div>
}
```

## Comparison

| Feature | GoSPA | HTMX | Alpine | SvelteKit | MoonZoon |
| :-- | :--: | :--: | :--: | :--: | :--: |
| **Language** | Go | HTML | JS | JS/TS | Rust |
| **Runtime** | ~15KB | ~14KB | ~15KB | Varies | ~27KB |
| **App Speed** | Very High | High | High | Very High | Very High |
| **DX Speed** | High | Very High | Very High | High | Moderate |
| **Reactivity** | ✅ | ❌ | ✅ | ✅ | ✅ |
| **WS Sync** | ✅ | ❌ | ❌ | ✅ | ✅ |
| **File Routing** | ✅ | ❌ | ❌ | ✅ | ❌ |
| **Type Safety** | ✅ | ❌ | ❌ | ✅ | ✅ |

## Recommended Production Baseline

Start from `gospa.ProductionConfig()` and tighten only what your app needs:

```go
config := gospa.ProductionConfig()
config.AllowedOrigins = []string{"https://example.com"}
config.AppName = "myapp"
```

For prefork deployments, add external `Storage` and `PubSub` backends so state and realtime traffic stay consistent across workers.

## Security

- **Vulnerability scanning (Go):** run `govulncheck ./...` regularly; the repo’s GitHub Actions workflow runs tests and govulncheck. For a full local gate, use `./scripts/quality-check.sh`.
- **Auth plugin:** set `JWT_SECRET` in production. Production is inferred from `GOSPA_ENV`, `ENV` / `APP_ENV` / `GO_ENV`, or legacy `GIN_MODE`—see [Security](docs/03-features/04-security.md#auth-plugin-jwt-and-production-detection).
- **CSP:** the default policy (`fiber.DefaultContentSecurityPolicy`) allows inline scripts and styles for typical GoSPA output. Override `ContentSecurityPolicy` when you need a stricter policy.

## Documentation

- **Browse:** [gospa.onrender.com/docs](https://gospa.onrender.com/docs) (website)
- **Authoritative Markdown:** [`docs/README.md`](docs/README.md) (table of contents for the `docs/` tree)
- **Config & API:** [`docs/04-api-reference/02-configuration.md`](docs/04-api-reference/02-configuration.md), [`docs/04-api-reference/01-core-api.md`](docs/04-api-reference/01-core-api.md)
- **Production:** [Production checklist](docs/03-features/07-production-checklist.md), [Security](docs/03-features/04-security.md)

---

[Apache License 2.0](LICENSE)
