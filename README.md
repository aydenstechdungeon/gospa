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
- **Go 1.21+**
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

| Feature | GoSPA | HTMX | Alpine | SvelteKit |
| :-- | :--: | :--: | :--: | :--: |
| **Language** | Go | HTML | JS | JS/TS |
| **Runtime** | ~15KB | ~14KB | ~15KB | Varies |
| **Reactivity** | ✅ | ❌ | ✅ | ✅ |
| **WS Sync** | ✅ | ❌ | ❌ | ✅ |
| **File Routing** | ✅ | ❌ | ❌ | ✅ |
| **Type Safety** | ✅ | ❌ | ❌ | ✅ |

## Recommended Production Baseline

Start from `gospa.ProductionConfig()` and tighten only what your app needs:

```go
config := gospa.ProductionConfig()
config.AllowedOrigins = []string{"https://example.com"}
config.AppName = "myapp"
```

For prefork deployments, add external `Storage` and `PubSub` backends so state and realtime traffic stay consistent across workers.

## Documentation

Full guides and API reference are available at [gospa.onrender.com](https://gospa.onrender.com/docs), including the [production hardening checklist](docs/03-features/07-production-checklist.md), or in the `/docs` directory.

---

[Apache License 2.0](LICENSE)
