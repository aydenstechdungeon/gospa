# GoSPA Documentation

This directory is the **authoritative** Markdown documentation for GoSPA. The [documentation website](https://gospa.onrender.com/docs) renders curated pages from the same topics; when in doubt, **edit files here** and keep examples aligned with the current `gospa` module API.

## How to use these docs

1. **Start:** [Quick start](01-getting-started/01-quick-start.md) → [Tutorial](01-getting-started/02-tutorial.md)
2. **Configure:** [Configuration reference](04-api-reference/02-configuration.md) (all `gospa.Config` fields)
3. **API surface:** [Core API](04-api-reference/01-core-api.md) (packages) + [CLI](04-api-reference/03-cli.md) + [Plugins](04-api-reference/04-plugins.md)
10. **Ship:** [Production checklist](03-features/08-production-checklist.md), [Deployment](03-features/06-deployment.md), [Security](03-features/04-security.md)
5. **Debug:** [Troubleshooting](07-troubleshooting/) (runtime, WebSocket, remote actions, build)

## Structure

```
docs/
├── 01-getting-started/     # Install and first app
├── 02-core-concepts/       # Rendering, state, components, islands, routing
├── 03-features/            # Client runtime, APIs, realtime, security, deployment
├── 04-api-reference/       # Core API, configuration, CLI, plugins
├── 05-advanced/            # Error handling, pruning
├── 06-migration/           # Version migrations
├── 07-troubleshooting/     # Operational fixes
├── 08-audits/              # Audit history (not introductory reading)
└── llms/                   # LLM-oriented exports (optional)
```

## Quick navigation

### Getting started
- [Quick start](01-getting-started/01-quick-start.md)
- [Tutorial](01-getting-started/02-tutorial.md)

### Core concepts
- [Rendering](02-core-concepts/02-rendering.md)
- [State](02-core-concepts/03-state.md)
- [Components](02-core-concepts/04-components.md)
- [Islands](02-core-concepts/05-islands.md)
- [Routing](02-core-concepts/06-routing.md)
- [Single File Components (.gospa)](03-features/07-gospa-sfc.md)

### Features
- [Client runtime](03-features/01-client-runtime.md)
- [Runtime API (TypeScript)](03-features/02-runtime-api.md)
- [Realtime](03-features/03-realtime.md)
- [Security](03-features/04-security.md)
- [Dev tools](03-features/05-dev-tools.md)
- [Deployment](03-features/06-deployment.md)
- [Production checklist](03-features/08-production-checklist.md)

### API reference
- [Core API (Go packages)](04-api-reference/01-core-api.md)
- [Configuration (`gospa.Config`)](04-api-reference/02-configuration.md)
- [CLI](04-api-reference/03-cli.md)
- [Plugins](04-api-reference/04-plugins.md)

### Advanced & migration
- [Error handling](05-advanced/01-error-handling.md)
- [State pruning](05-advanced/02-state-pruning.md)
- [v1 → v2 migration](06-migration/01-v1-to-v2.md)

### Troubleshooting
- [Runtime initialization](07-troubleshooting/01-runtime-initialization.md)
- [Remote actions](07-troubleshooting/02-remote-actions.md)
- [WebSocket](07-troubleshooting/03-websocket-connections.md)
- [HMR / dev server](07-troubleshooting/04-hmr-dev-server.md)
- [Island hydration](07-troubleshooting/05-island-hydration.md)
- [State sync](07-troubleshooting/06-state-synchronization.md)
- [Build & deployment](07-troubleshooting/07-build-deployment.md)

### Audits
- [Security & performance audits](08-audits/)

## Website (`/website`)

The site under `website/` serves a browsable docs UI. Topic pages are hand-authored in `website/routes/docs/**` (Templ). **Keep them consistent** with this folder: when you change defaults (e.g. `gospa.Config`, security behavior), update both Markdown and the relevant Templ page.

- Full narrative reference: **this `docs/` tree**
- Guided navigation & SEO: **`website/`** routes

## Contributing to documentation

- Prefer **working code** in fenced blocks (match current import paths and `fiber/v3` APIs).
- Link to **[Configuration](04-api-reference/02-configuration.md)** for `Config` fields instead of duplicating large structs in multiple files.
- Run **`./scripts/quality-check.sh`** (or at least `go test ./...` and `bun check` in `client/`) before merging doc-only PRs that claim behavior.
