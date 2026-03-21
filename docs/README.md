# GoSPA Documentation

This directory contains the complete documentation for GoSPA. The documentation is organized to guide users from their first steps through advanced topics.

## Structure

```
docs/
├── 01-getting-started/     # Installation and first steps
├── 02-core-concepts/       # Essential concepts
├── 03-features/            # Runtime, security, deployment, and production guides
├── 04-api-reference/       # API documentation
├── 05-advanced/            # Advanced topics
├── 06-migration/           # Version migration guides
├── 07-troubleshooting/     # Common production and developer issues
└── 08-audits/              # Security and performance audit history
```

## Quick Navigation

### Getting Started
- [Quick Start](01-getting-started/01-quick-start.md) - Installation and first app
- [Tutorial](01-getting-started/02-tutorial.md) - Build a todo app

### Core Concepts
- [Rendering](02-core-concepts/02-rendering.md) - SSR, SPA, and islands
- [State](02-core-concepts/03-state.md) - Reactive state management
- [Components](02-core-concepts/04-components.md) - Component system
- [Islands](02-core-concepts/05-islands.md) - Partial hydration
- [Routing](02-core-concepts/06-routing.md) - Route parameters and navigation

### Features
- [Client Runtime](03-features/01-client-runtime.md) - Runtime variants and selection
- [Runtime API](03-features/02-runtime-api.md) - Client-side API reference
- [Realtime](03-features/03-realtime.md) - SSE and WebSocket
- [Security](03-features/04-security.md) - Security model and CSP
- [Dev Tools](03-features/05-dev-tools.md) - HMR and debugging
- [Production Checklist](03-features/07-production-checklist.md) - Recommended production baseline

### API Reference
- [Core API](04-api-reference/01-core-api.md) - Go API documentation
- [Configuration](04-api-reference/02-configuration.md) - Config options
- [CLI](04-api-reference/03-cli.md) - Command line interface
- [Plugins](04-api-reference/04-plugins.md) - Plugin system

### Advanced
- [Error Handling](05-advanced/01-error-handling.md)
- [State Pruning](05-advanced/02-state-pruning.md)

### Migration
- [v1 to v2](06-migration/01-v1-to-v2.md) - Migrating from v1.x to v2.0

### Troubleshooting
- [Remote Actions](07-troubleshooting/02-remote-actions.md)
- [WebSocket Connections](07-troubleshooting/03-websocket-connections.md)
- [Build & Deployment](07-troubleshooting/07-build-deployment.md)

### Audits
- [2026-03-12 Security & Performance Audit](08-audits/2026-03-12-security-performance-audit.md)
- [2026-03-21 Security, Performance & Docs Audit](08-audits/2026-03-21-security-performance-docs-audit.md)

## Website Integration

This documentation structure is designed to be consumed by the GoSPA website. Each folder represents a documentation section, and files are ordered by their numerical prefix.

To render these docs on the website:
1. Read `README.md` for structure
2. Parse each section folder
3. Render markdown files in numerical order
4. Use frontmatter (if present) for metadata
