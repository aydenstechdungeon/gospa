# Configuration Overview

GoSPA is configured primarily through the `gospa.Config` struct passed to `gospa.New()`.

## Core Options

| Property | Type | Description |
| :--- | :--- | :--- |
| `AppName` | `string` | The display name of your application. |
| `DevMode` | `bool` | Enables verbose logging, HMR support, and relaxed security constraints. Set to `false` in production. |
| `RoutesDir` | `string` | Path to the directory containing `.templ` or `.gospa` route files. |
| `StaticDir` | `string` | Path to the directory served for static assets. |

## Security Settings

| Property | Type | Description |
| :--- | :--- | :--- |
| `EnableCSRF` | `bool` | Enables built-in CSRF protection for forms and AJAX. |
| `ContentSecurityPolicy` | `string` | Custom CSP header. Use `{nonce}` as a placeholder for automatically generated nonces. |
| `AllowedOrigins` | `[]string` | Sets the `Access-Control-Allow-Origin` header for CORS. |
| `PublicOrigin` | `string` | The base URL of your site (e.g., `https://example.com`). Required for secure WebSocket generation. |

## Performance & Optimization

| Property | Type | Description |
| :--- | :--- | :--- |
| `CompressState` | `bool` | Enables GZIP compression for outgoing WebSocket state payloads. |
| `StateDiffing` | `bool` | Only sends changed state keys (deltas) over WebSockets instead of full snapshots. |
| `SSGCacheMaxEntries` | `int` | Maximum number of pre-rendered pages to hold in the in-memory LRU cache. |
| `Prefork` | `bool` | Enables Fiber's prefork mode to utilize multiple CPU cores. Requires external `Storage` and `PubSub`. |

## Rendering Strategies

GoSPA supports multiple rendering strategies per route (configured via `+page` options):
- **SSR (Server-Side Rendering)**: Default. Fresh render on every request.
- **SSG (Static Site Generation)**: Rendered once and cached.
- **ISR (Incremental Static Regeneration)**: Cached with background revalidation.
- **PPR (Partial Prerendering)**: Static shell with dynamic slots.

For more details, see the [Rendering Strategy Guide](rendering.md).
