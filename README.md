# GoSPA

<div align="center">
  <img src="https://github.com/user-attachments/assets/9e9d126d-8c91-465a-a5c0-968e41c095fb" width="128" height="128" alt="GoSPA Logo 1">
  <img src="https://github.com/user-attachments/assets/338c5be2-9ce1-4f7a-a389-bfe176c6a9d6" width="128" height="128" alt="GoSPA Logo 2">
</div>

GoSPA brings Svelte-like reactive primitives (`Runes`, `Effects`, `Derived`) to the Go ecosystem. It is a high-performance framework for building reactive SPAs with Templ, Fiber, file-based routing, and real-time state synchronization.

## Highlights

- **Native Reactivity** - `Rune`, `Derived`, `Effect` primitives that work exactly like Svelte 5.
- **WebSocket Sync** - Transparent client-server state synchronization with GZIP delta patching.
- **File-Based Routing** - SvelteKit-style directory structure for `.templ` files.
- **Hybrid Rendering** - Mix SSR, SSG, ISR, and PPR on a per-page basis.
- **Type-Safe RPC** - Call server functions directly from the client without boilerplate endpoints.
- **High Performance** - Integrated `go-json` and optional MessagePack for minimal overhead.

## Quick Start

### 1. Install CLI
```bash
go install github.com/aydenstechdungeon/gospa/cmd/gospa@latest
```

### 2. Scaffold & Run
```bash
gospa create myapp
cd myapp
go mod tidy
gospa dev
```

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

## Documentation

Full guides and API reference are available at [gospa.onrender.com](https://gospa.onrender.com/docs) or in the `/docs` directory.

---

[Apache License 2.0](LICENSE)
