# GoSPA Documentation

This directory is the **authoritative** Markdown documentation for GoSPA.

## Source-of-truth policy

- `docs/**` is canonical for technical content, APIs, and version requirements.
- `website/routes/docs/**` is a rendered presentation layer and must not diverge semantically from `docs/**`.
- Any docs change that affects URLs, taxonomy, or prerequisites must update both markdown links and website routes/search index in the same change.

## How to use these docs

1. **Start:** [Quick start](getstarted/quickstart.md)
2. **Configure:** [Configuration reference](configuration.md)
3. **API surface:** [Core API](api.md) + [CLI](cli.md) + [Plugins](plugins.md)
4. **Ship:** [Security Guide](security.md), [Troubleshooting](troubleshooting.md)

## Structure

```
docs/
├── api.md               # Fiber & Client API Reference
├── routing.md           # File-based routing & rendering
├── state-management.md  # Reactive state synchronization
├── reactive-primitives.md # Rune, Derived, Effect, EffectScope
├── api/remote-actions.md # Type-safe RPC / Remote Actions
├── api/websocket.md      # High-performance real-time sync
├── api/sse.md            # Server-Sent Events guide
├── plugins.md           # Framework extensions & lifecycle
├── devtools.md          # Debugging, Error Overlay, HMR
├── runtime.md           # Client runtime lifecycle & hydration
├── security.md          # Security hardening & best practices
├── errors.md            # Error handling & boundaries
├── gospasfc.md          # GoSPA Single File Components
├── params.md            # Route parameters & query strings
├── root.md              # Root layout & nesting
└── faq.md               # Frequently asked questions
```

## Quick navigation

### Getting started
- [Installation](getstarted/installation.md)
- [Quick start](getstarted/quickstart.md)

### Core concepts
- [Reactive Primitives](reactive-primitives.md)
- [State Management](state-management.md)
- [Components & Islands](islands.md)
- [File-based Routing](routing.md)
- [Remote Actions](api/remote-actions.md)
- [Single File Components (.gospa, alpha)](gospasfc.md)

### Advanced Features
- [Security & Hardening](security.md)
- [Realtime (WebSockets)](api/websocket.md)
- [Server-Sent Events (SSE)](api/sse.md)
- [Plugin Architecture](plugins.md)
- [Dev Tools & HMR](devtools.md)
- [Runtime Lifecycle](runtime.md)
- [Fiber Migration Checklist](migration/fiber-to-gospa.md)

### Reference
- [Full API Reference](api.md)
- [Configuration Reference](configuration.md)
- [CLI Reference](cli.md)

### Support
- [General Troubleshooting](troubleshooting.md)
- [FAQ](faq.md)
- [Error Handling](errors.md)
