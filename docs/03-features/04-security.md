# Security Policy for GoSPA

## Supported Versions

Currently, only the latest version of GoSPA is supported with security updates.

| Version | Supported          |
| ------- | ------------------ |
| v0.x    | :white_check_mark: |

## Security Philosophy: Trust the Server

GoSPA follows a **"trust the server"** security model, similar to SvelteKit and other modern frameworks. This means:

1. **Templ auto-escapes all content** during server-side rendering
2. **Client-side sanitization is opt-in**, not required by default
3. **Content Security Policy (CSP)** is the primary XSS defense
4. **User-generated content** is the only scenario requiring DOMPurify

### Why No Default Sanitization?

Traditional SPAs need client-side sanitization because they render untrusted HTML directly in the browser. GoSPA uses Templ, which:

- Auto-escapes all dynamic content at the server
- Prevents XSS at the source, not as an afterthought
- Renders clean HTML that doesn't need re-sanitization on the client

## XSS Protection Layers

### Layer 1: Templ Auto-Escaping (Always Active)

Templ automatically escapes all dynamic content:

```go
// In your Templ component
@templ.Component {
    // This is safe - userInput is auto-escaped
    <div>{ userInput }</div>
}
```

Output: `<div><script>alert(1)</script></div>`

### Layer 2: Content Security Policy (Recommended)

Configure strict CSP headers as your primary defense:

```http
Content-Security-Policy: 
  default-src 'self';
  script-src 'self';
  style-src 'self' 'unsafe-inline';
  img-src 'self' data: https:;
  connect-src 'self' wss:;
```

### Layer 3: DOMPurify (User-Generated Content Only)

Use DOMPurify only when displaying **untrusted user-generated content**:

```typescript
// Use the secure runtime for UGC scenarios
import { sanitize } from '@gospa/runtime-secure';

const clean = await sanitize(untrustedHtml);
```

## Runtime Variants and Security

Choose the appropriate runtime based on your needs:

| Runtime | Size | Sanitizer | Use Case |
|---------|------|-----------|----------|
| `@gospa/client` (default) | ~15KB | None (trust server) | **Recommended**: Server-rendered apps with CSP |
| `@gospa/client/runtime-secure` | ~35KB | DOMPurify (full) | Apps with user-generated content (comments, wikis) |

Only these two entrypoints are exported by the current package manifest. Internal runtime bundles such as `runtime-simple`, `runtime-core`, and `runtime-micro` exist in the source tree but are not public import paths.

### Default Runtime (`gospa`)

```typescript
import { init } from '@gospa/client';

// No sanitization - trusts server-rendered HTML
// Bundle size: ~15KB
init();
```

**Security model:**
- Trusts all HTML from your server
- Relies on Templ's auto-escaping for XSS prevention
- CSP is your primary defense
- Perfect for apps without user-generated HTML content

### Secure Runtime (`gospa/runtime-secure`)

```typescript
import { init, sanitize } from '@gospa/client/runtime-secure';

// Includes DOMPurify for user-generated content
// Bundle size: ~35KB
init();

// Sanitize untrusted HTML
const clean = await sanitize(userComment);
```

**Security model:**
- Same as default, plus DOMPurify for UGC
- Use only when displaying HTML from untrusted sources
- Social media apps, comment systems, wikis, forums

## When Do You Need DOMPurify?

| Scenario | Needs DOMPurify? | Reason |
|----------|-----------------|--------|
| Server-rendered Templ components | No | Templ auto-escapes |
| Client-side HTML bindings with server data | No | Data already escaped by Templ |
| Displaying user comments with HTML | **Yes** | Users can submit malicious HTML |
| Rich text editors (WYSIWYG) | **Yes** | Users control HTML content |
| Markdown rendering | Maybe | Depends on your markdown parser |
| Embedding third-party widgets | **Yes** | External content is untrusted |

## HTML Sanitization API

### Using DOMPurify (Secure Runtime)

```typescript
import { sanitize, sanitizeSync, isSanitizerReady, preloadSanitizer } from '@gospa/client/runtime-secure';

// Async sanitization (recommended)
const clean = await sanitize(untrustedHtml);

// Preload during idle time for faster first use
if (typeof window !== 'undefined') {
  requestIdleCallback(() => preloadSanitizer());
}

// Sync sanitization (only if already loaded)
if (isSanitizerReady()) {
  const clean = sanitizeSync(untrustedHtml);
}
```

### Custom Sanitizer Configuration

The current public runtime API does not export a `setSanitizer()` hook. If you need custom sanitization rules today, sanitize HTML before passing it into GoSPA or use [`@gospa/client/runtime-secure`](client/package.json) with your own wrapper module around DOMPurify.

## DOM Clobbering Protection

When using DOMPurify (via `runtime-secure`), GoSPA prevents DOM Clobbering attacks:

- `name` attributes are stripped from all elements
- `form` attributes are removed
- `id` attributes are sanitized to prevent property shadowing
- `SANITIZE_DOM: true` and `SANITIZE_NAMED_PROPS: true` are enabled

## CSRF Protection Setup

When `EnableCSRF` is enabled in your `gospa.Config`, GoSPA installs both CSRF middlewares automatically.

If you need to wire them manually in a custom Fiber stack, GoSPA uses a **two-middleware pattern** for CSRF protection:

```go
app.Use(fiber.CSRFSetTokenMiddleware())
app.Use(fiber.CSRFTokenMiddleware())
```

### Important: Middleware Order Matters

Place `CSRFSetTokenMiddleware` BEFORE `CSRFTokenMiddleware`:

```go
// CORRECT: Set token first, then validate
app.Use(fiber.CSRFSetTokenMiddleware())  // Sets cookie on GET
app.Use(fiber.CSRFTokenMiddleware())     // Validates on POST/PUT/DELETE

// INCORRECT: Don't reverse the order!
// app.Use(gospa.CSRFTokenMiddleware(csrfConfig))   // Validation will fail
// app.Use(gospa.CSRFSetTokenMiddleware(csrfConfig))
```

### Prefork Mode Warning

When using Prefork mode (`Prefork: true`), each worker process has **isolated memory**. CSRF tokens stored in memory will not be shared across workers, causing validation failures. You must use an external session store (Redis, database) for CSRF state in Prefork deployments.

### Client-Side Integration

Fetch the CSRF token from the cookie and include it in requests:

```javascript
// Get token from cookie
const csrfToken = document.cookie
  .split('; ')
  .find(row => row.startsWith('csrf_token='))
  ?.split('=')[1];

// Include in request headers
fetch('/api/action', {
  method: 'POST',
  headers: {
    'X-CSRF-Token': csrfToken,
    'Content-Type': 'application/json',
  },
  body: JSON.stringify(data)
});
```

## Configuration Hardening Guidelines

For production environments:

- Set `DevMode: false`. Development mode leaks stack traces.
- Configure `AllowedOrigins` for CORS. Avoid wildcard `*` domains.
- Never place session identifiers in URL query parameters.
- Rate-limit Action routes.
- Use HTTPS in production with HSTS headers.
- Implement a strict CSP (see examples above).

## Migration from v1.x

If you're upgrading from GoSPA v1.x:

1. **Default runtime no longer includes DOMPurify**
   - Most apps don't need to change anything (Templ already protects you)
   - If you display user-generated HTML, switch to `@gospa/client/runtime-secure`

2. **Legacy internal runtime bundles are not public imports**
   - Use `@gospa/client` for the default trust-the-server runtime
   - Use `@gospa/client/runtime-secure` if you need sanitization

3. **Remove `DisableSanitization: true` from server config**
   - It's no longer needed (and no longer exists)
   - The default runtime now trusts the server by default

See the [Migration Guide](/docs/migration-v2) for detailed instructions.

## Security Update Policy

Critical and High severity vulnerabilities are patched immediately with out-of-band hotfix releases. Regular security updates are included in minor version releases.

## Reporting a Vulnerability

If you discover a security vulnerability, please report it privately:

- Email: security@gospa.dev
- GitHub Security Advisories

You'll receive an acknowledgment within 48 hours with an estimated resolution timeline.

## Known Limitations

1. **Trusted Types**: GoSPA supports Trusted Types API when available, but falls back to string-based sanitization in unsupported browsers.

2. **SVG Support**: SVG elements are disabled by default in DOMPurify due to XSS risks. Enable only if you understand the risks.

3. **WebSocket Security**: Use WSS (WebSocket Secure) in production. The runtime does not enforce this.
