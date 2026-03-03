# Security Policy for GoSPA

## Supported Versions

Currently, only the latest version of GoSPA is supported with security updates.

| Version | Supported          |
| ------- | ------------------ |
| v0.x    | :white_check_mark: |

## Security Features

GoSPA incorporates several security practices by design:

### Cross-Site Scripting (XSS) Protection

GoSPA provides multiple layers of XSS protection:

1. **DOMPurify Integration**: The full runtime (`runtime.ts`) includes DOMPurify 3.3.1 for comprehensive HTML sanitization.
   - Lazy-loaded to minimize initial bundle impact (~20KB only when needed)
   - Configured with strict allowlists for tags and attributes
   - Blocks all event handlers, dangerous URLs (javascript:, data:), and DOM Clobbering vectors

2. **Simple Sanitizer**: The lightweight runtime (`runtime-simple.ts`) includes a basic sanitizer for scenarios where bundle size is critical.
   - Removes script elements, event handlers, and dangerous elements
   - Blocks dangerous URL schemes
   - Strips `name` and `form` attributes to prevent DOM Clobbering
   - **Warning**: Not recommended for untrusted user content; use DOMPurify for production security

3. **Default Sanitization**: HTML content is automatically sanitized when using reactive bindings with `type: 'html'`.

### Runtime Variants and Security

Choose the appropriate runtime based on your security requirements:

| Runtime | Size | Sanitizer | Use Case |
|---------|------|-----------|----------|
| `runtime.js` | ~20KB | DOMPurify (full) | Production with untrusted content |
| `runtime-simple.js` | ~18KB | Simple (basic) | Trusted content, bundle-critical apps |
| `runtime-core.js` | ~15KB | None (passthrough) | Custom sanitizer implementation |
| `runtime-micro.js` | ~5KB | None | State-only, no DOM operations |

### DOM Clobbering Protection

GoSPA's DOMPurify configuration explicitly prevents DOM Clobbering attacks:

- `name` attributes are stripped from all elements
- `form` attributes are removed
- `id` attributes are sanitized to prevent property shadowing
- `SANITIZE_DOM: true` and `SANITIZE_NAMED_PROPS: true` are enabled

### Content Security Policy (CSP) Recommendations

For optimal security, configure CSP headers:

```http
Content-Security-Policy: 
  default-src 'self';
  script-src 'self' 'unsafe-inline';
  style-src 'self' 'unsafe-inline';
  img-src 'self' data: https:;
  connect-src 'self' wss:;
  require-trusted-types-for 'script';
```

**Note**: `'unsafe-inline'` is currently required for GoSPA's inline event handling. Future versions may support strict CSP with nonces.

## CSRF Protection Setup

GoSPA uses a **two-middleware pattern** for CSRF protection. You must configure both middleware correctly:

```go
app.Use(fiber.CSRFSetTokenMiddleware())
```

```go
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

## HTML Sanitization API

### Using the DOMPurify Sanitizer

```typescript
import { sanitize, sanitizeSync, isSanitizerReady, preloadSanitizer } from '@gospa/runtime';

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

```typescript
import { setSanitizer } from '@gospa/runtime';
import DOMPurify from 'dompurify';

// Set custom sanitizer
setSanitizer((html) => DOMPurify.sanitize(html, {
  ALLOWED_TAGS: ['b', 'i', 'em', 'strong', 'p', 'br'],
  ALLOWED_ATTR: []
}));
```

## Configuration Hardening Guidelines

For production environments, ensure you abide by these guidelines:

- Set `DevMode: false`. Development mode enables detailed stack traces to leak which is unsafe for public endpoints.
- Initialize explicitly locked `AllowedOrigins` for `CORSMiddleware`. Avoid wildcard `*` domains whenever sensitive cookies/authentication tokens are utilized.
- Never place session identifiers in URL query arguments or params.
- Rate-limit Action routes dynamically leveraging proxies or internal token buckets.
- Use HTTPS in production to prevent man-in-the-middle attacks.
- Enable HSTS headers for HTTPS enforcement.

## Security Update Policy

Vulnerabilities with a "Critical" or "High" classification are usually prioritized immediately with an out-of-band hotfix release. Regular security updates are appended to the next minor version iteration.

## Reporting a Vulnerability

If you have discovered a security vulnerability in this project, do not open a public issue. We handle vulnerability disclosures privately.

Please report any identified vulnerability via email directly to the maintainers or report it through our GitHub Security platform.
You will receive an acknowledgment within 48 hours with an estimation of when the problem will be resolved.

## Known Limitations

1. **Trusted Types**: While GoSPA supports Trusted Types API when available, not all browsers implement it. The sanitizer falls back to string-based sanitization in unsupported browsers.

2. **SVG Support**: SVG elements are disabled by default in both sanitizers due to XSS risks (animation events, foreignObject). Enable only if you understand and accept the risks.

3. **Style Attributes**: Inline styles are allowed by DOMPurify. While URLs in styles are sanitized, CSS-based data exfiltration is theoretically possible through clever use of selectors and external resources.

4. **WebSocket Security**: WebSocket connections should use WSS (WebSocket Secure) in production. The runtime does not enforce this; configure your server appropriately.
