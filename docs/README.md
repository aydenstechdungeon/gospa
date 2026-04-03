# GoSPA Documentation

This directory is the **authoritative** Markdown documentation for GoSPA. The [documentation website](https://gospa.onrender.com/docs) renders curated pages from the same topics; when in doubt, **edit files here** and keep examples aligned with the current `gospa` module API.

## How to use these docs

1. **Start:** [Quick start](getstarted/quickstart) → [Tutorial](getstarted/tutorial)
2. **Configure:** [Configuration reference](configuration) (all `gospa.Config` fields)
3. **API surface:** [Core API](api/core) (packages) + [CLI](cli) + [Plugins](plugins)
4. **Ship:** [Production checklist](troubleshooting), [Deployment](configuration/scaling), [Security](configuration/scaling)
5. **Debug:** [Troubleshooting](troubleshooting) (runtime, WebSocket, remote actions, build)

## Structure

```
docs/
├── getstarted/          # Install and first app
├── gospasfc/           # Single File Components
├── routing/            # File-based routing
├── state-management/    # Server and client state
├── client-runtime/     # Client engine details
├── configuration/      # Configuration reference
├── plugins/            # Ecosystem and extensions
├── api/                # Core packages reference
├── reactive-primitives/ # Primitives reference
└── troubleshooting.md   # Operational fixes
```

## Quick navigation

### Getting started
- [Installation](getstarted/installation)
- [Quick start](getstarted/quickstart)
- [Tutorial](getstarted/tutorial)

### Core concepts
- [Rendering](rendering)
- [State](state-management/server)
- [Components](components)
- [Islands](islands)
- [Routing](routing)
- [Single File Components (.gospa)](gospasfc)

### Features
- [Client runtime](client-runtime/overview)
- [Runtime API (TypeScript)](reactive-primitives/js)
- [Realtime](websocket)
- [Security](configuration/scaling)
- [Dev tools](devtools)
- [Deployment](configuration/scaling)
- [Production checklist](troubleshooting)

### API reference
- [Core API (Go packages)](api/core)
- [Configuration (`gospa.Config`)](configuration)
- [CLI](cli)
- [Plugins](plugins)

### Advanced & migration
- [Error handling](errors)
- [State pruning](state-management/patterns)
- [v1 → v2 migration](faq)

### Troubleshooting
- [Runtime initialization](troubleshooting)
- [Remote actions](remote-actions)
- [WebSocket](websocket)
- [HMR / dev server](hmr)
- [Island hydration](troubleshooting)
- [State sync](troubleshooting)
- [Build & deployment](troubleshooting)

## Website (`/website`)

The site under `website/` serves a browsable docs UI. Topic pages are hand-authored in `website/routes/docs/**` (Templ). **Keep them consistent** with this folder: when you change defaults (e.g. `gospa.Config`, security behavior), update both Markdown and the relevant Templ page.

- Full narrative reference: **this `docs/` tree**
- Guided navigation & SEO: **`website/`** routes
