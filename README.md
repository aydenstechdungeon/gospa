# GoSPA (Alpha)

<div align="center">
  <img src="https://github.com/user-attachments/assets/9e9d126d-8c91-465a-a5c0-968e41c095fb" width="128" height="128" alt="GoSPA Logo 1">
  <img src="https://github.com/user-attachments/assets/338c5be2-9ce1-4f7a-a389-bfe176c6a9d6" width="128" height="128" alt="GoSPA Logo 2">
</div>

GoSPA (Go Spa and Go S-P-A are the only valid pronunciations)  brings Svelte-like reactive primitives (`Runes`, `Effects`, `Derived`) to the Go ecosystem. It is a high-performance framework for building reactive SPAs with Templ, Fiber, file-based routing, and real-time state synchronization.

## Table of Contents

- [GoSPA (Alpha)](#gospa-alpha)
  - [Table of Contents](#table-of-contents)
  - [Highlights](#highlights)
  - [Quick Start](#quick-start)
    - [0. Prerequisites](#0-prerequisites)
    - [1. Install CLI](#1-install-cli)
    - [2. Scaffold \& Run](#2-scaffold--run)
    - [3. A Simple SFC](#3-a-simple-sfc)
  - [Comparison](#comparison)
  - [Recommended Production Baseline](#recommended-production-baseline)
  - [Documentation](#documentation)
  - [Known Issues](#known-issues)
  - [Accessibility (A11y)](#accessibility-a11y)
  - [Contributing](#contributing)
  - [License](#license)

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

GoSPA Docs page (gospa.onrender.com - free hosting)
<img width="693" height="29" alt="Screenshot_20260415_184459-1" src="https://github.com/user-attachments/assets/ae973cae-26fc-4ebf-9290-7be3b966d286" />

SvelteKit Docs page (svelte.dev/docs/kit/introduction)
<img width="687" height="25" alt="Screenshot_20260415_184329" src="https://github.com/user-attachments/assets/8cc0b88d-3b61-49ae-92cf-aa0e289e6f19" />


## Recommended Production Baseline

Start from `gospa.ProductionConfig()` and tighten only what your app needs:

```go
config := gospa.ProductionConfig()
config.AllowedOrigins = []string{"https://example.com"}
config.AppName = "myapp"
```

For prefork deployments, add external `Storage` and `PubSub` backends so state and realtime traffic stay consistent across workers.

Dynamic HTML (`data-bind="html:*"` and stream HTML chunks) is escaped by default in the runtime. If you need to render raw HTML, only use trusted server-controlled content.

## Documentation

Explore the full GoSPA documentation:

- Source of truth policy: `docs/**` is canonical; website docs pages (`website/routes/docs/**`) must mirror this content and route taxonomy.
- When in doubt, update `docs/README.md` first, then sync website routes and search index.

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

## Known Issues

- **CSP nonce mismatch can block runtime scripts**
  - Symptom: Browser console shows CSP violations such as `Refused to execute inline script` or `Refused to load the script`.
  - Cause: custom CSP policy does not include `'nonce-{nonce}'`, or custom inline/module scripts are missing the per-request nonce.
  - Manual Developer Fix:

```go
cspPolicy := "default-src 'self'; script-src 'self' 'nonce-{nonce}'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self' wss: https:; frame-ancestors 'none'; base-uri 'self'; form-action 'self';"
app.Fiber.Use(gospafiber.SecurityHeadersMiddleware(cspPolicy))
```

```templ
<script type="module" nonce={ gospatempl.GetNonce(ctx) }>
  // custom client bootstrap code
</script>
```

  Use the same nonce source (`gospatempl.GetNonce(ctx)`) for every custom inline or module script in your layout.

## Accessibility (A11y)

Building accessible SPAs is a first-class citizen in GoSPA:

- **Live Announcer**: Use `GoSPA.announce("Message")` to trigger screened reader notifications.
- **Focus Management**: Built-in utilities for focus trapping and restoration during navigation.
- **ARIA Helpers**: Lightweight helpers for managing ARIA attributes and roles reactively.

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md).

## License

GoSPA is licensed under the [Apache-2.0 license](LICENSE).
