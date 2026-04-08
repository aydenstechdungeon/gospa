# GoSPA Documentation

This directory is the **authoritative** Markdown documentation for GoSPA.

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
├── remote-actions.md    # Type-safe RPC / Remote Actions
├── websocket.md         # High-performance real-time sync
├── sse.md               # Server-Sent Events guide
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
- [Remote Actions](remote-actions.md)
- [Single File Components (.gospa)](gospasfc.md)

### Advanced Features
- [Security & Hardening](security.md)
- [Realtime (WebSockets)](websocket.md)
- [Server-Sent Events (SSE)](sse.md)
- [Plugin Architecture](plugins.md)
- [Dev Tools & HMR](devtools.md)
- [Runtime Lifecycle](runtime.md)

### Reference
- [Full API Reference](api.md)
- [Configuration Reference](configuration.md)
- [CLI Reference](cli.md)

### Support
- [General Troubleshooting](troubleshooting.md)
- [FAQ](faq.md)
- [Error Handling](errors.md)
