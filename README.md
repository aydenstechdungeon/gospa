# GoSPA (Alpha)

<div align="center">
  <img src="https://github.com/user-attachments/assets/9e9d126d-8c91-465a-a5c0-968e41c095fb" width="128" height="128" alt="GoSPA Logo 1">
  <img src="https://github.com/user-attachments/assets/338c5be2-9ce1-4f7a-a389-bfe176c6a9d6" width="128" height="128" alt="GoSPA Logo 2">
</div>

GoSPA (Go Spa and Go S-P-A are the only valid pronunciations)  brings Svelte-like reactive primitives (`Runes`, `Effects`, `Derived`) to the Go ecosystem. It is a high-performance framework for building reactive SPAs with Templ, Fiber, file-based routing, and real-time state synchronization.

## Highlights

- **Native Reactivity** - `Rune`, `Derived`, `Effect` primitives that work exactly like Svelte 5.
- **WebSocket Sync** - Transparent client-server state synchronization with GZIP delta patching.
- **SFC System** - Single File Components (`.gospa`) with scoped CSS and Go-based logic.
- **File-Based Routing** - SvelteKit-style directory structure for `.templ` and `.gospa` files.
- **Hybrid Rendering** - Mix SSR, SSG, ISR, and PPR on a per-page basis.
- **Type-Safe RPC** - Call server functions directly from the client without boilerplate endpoints.
- **High Performance** - Integrated `go-json` and optional MessagePack for minimal overhead.

## Quick Start

### 0. Prerequisites
- **Go 1.26.0+** (matches `go.mod`; use a current stable toolchain)
- **Node.js Tooling**: **Bun** is preferred for the client-side build process (zero-config JS bundling, CSS extraction). **pnpm** and **npm** are supported as fallbacks using `esbuild`, but Bun remains the recommended choice for maximum performance.
- **`JWT_SECRET`**: Ensure this environment variable is set for production authentication contexts (when using the Auth plugin).

### 1. Install CLI
```bash
go install github.com/aydenstechdungeon/gospa/cmd/gospa@latest
```
or
```bash
go run github.com/aydenstechdungeon/gospa/cmd/gospa@latest
```

### 2. Scaffold & Run
```bash
gospa create myapp
cd myapp
go mod tidy
gospa doctor
gospa dev
```
or
```bash
go run github.com/aydenstechdungeon/gospa/cmd/gospa@latest create myapp
cd myapp
go mod tidy
go run github.com/aydenstechdungeon/gospa/cmd/gospa@latest doctor
go run github.com/aydenstechdungeon/gospa/cmd/gospa@latest dev
```

> For local client/runtime tooling, **Bun is strongly preferred**. The GoSPA CLI provides fallbacks for `pnpm` and `npm` using `esbuild`, but Bun's integrated bundler is the authoritative development target.

### 3. A Simple SFC
```svelte
// islands/Counter.gospa
<script lang="go">
  var count = $state(0)
  func increment() { count++ }
</script>

<template>
  <button on:click={increment}>
    Count is {count}
  </button>
</template>

<style>
  button { padding: 1rem; border-radius: 8px; }
</style>
```

GoSPA automatically compiles this to a reactive Templ component and a TypeScript hydration island.

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

## Documentation

Explore the full GoSPA documentation:

- [**Reactive Primitives**](docs/reactive-primitives.md) - `Rune`, `Derived`, `Effect`, and `EffectScope`.
- [**State Management**](docs/state-management.md) - Server-to-client state synchronization.
- [**File-Based Routing**](docs/routing.md) - Layouts, pages, and rendering strategies.
- [**Route Parameters**](docs/params.md) - Dynamic route segments.
- [**Remote Actions**](docs/api/remote-actions.md) - Type-safe RPC between client and server.
- [**WebSocket & Real-time**](docs/api/websocket.md) - High-performance state sync.
- [**Server-Sent Events**](docs/api/sse.md) - Lightweight real-time notifications.
- [**Plugin Architecture**](docs/plugins.md) - Extending the framework.
- [**DevTools & Debugging**](docs/devtools.md) - Error overlays and HMR.
- [**Client Runtime**](docs/internals/runtime.md) - Tiered runtime internals.
- [**API Reference**](docs/api.md) - Fiber and Client API details.

## Security & Performance

GoSPA is built with a "security-first" and "performance-by-default" philosophy. A comprehensive audit of the embedded asset pipeline was conducted in April 2026.

### Security Highlights
- **State Injection Safety**: All global state injected via `__GOSPA_STATE__` is HTML-escaped using `SetEscapeHTML(true)` to prevent `<script>` breakout XSS.
- **Content Security Policy**: Supports nonce-based CSP for all injected scripts. `ProductionConfig()` defaults to a `StrictContentSecurityPolicy` that disallows unsafe-inline scripts.
- **Trust-the-Server Model**: GoSPA adopts a server-trust security model. The runtime assumes the server renders safe HTML (Templ auto-escapes dynamic content), avoiding the overhead of heavy client-side sanitizers.
- **Client-side Baseline**: Streamed HTML chunks use a lightweight, whitelist-based manual sanitizer to mitigate XSS in dynamic fragments without large dependencies.
- **Auth & CSRF**: Built-in CSRF protection with `X-CSRF-Token` headers and secure session management.

### Performance Highlights
- **O(1) Routing**: Path matching uses optimized static lookups for zero latency at scale.
- **Tiered Runtime**: Choose between `Micro` (~1KB), `Core` (~13KB), or `Full` (~15KB) runtimes.
- **Delta Patching**: GZIP-compressed binary diffs for real-time state updates via WebSocket.
- **Pre-compression**: Automatic `.gz` and `.br` asset generation for static files.

For more details, see the [Comprehensive Security & Performance Audit](docs/security.md).

## Accessibility (A11y)

Building accessible SPAs is a first-class citizen in GoSPA:

- **Live Announcer**: Use `GoSPA.announce("Message")` to trigger screened reader notifications.
- **Focus Management**: Built-in utilities for focus trapping and restoration during navigation.
- **ARIA Helpers**: Lightweight helpers for managing ARIA attributes and roles reactively.

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) and [Code of Conduct](CODE_OF_CONDUCT.md).

## License

GoSPA is licensed under the [MIT License](LICENSE).
