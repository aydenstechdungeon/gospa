# Security Policy for GoSPA

## Supported Versions

Currently, only the latest version of GoSPA is supported with security updates.

| Version | Supported          |
| ------- | ------------------ |
| v0.x    | :white_check_mark: |

## Security Features

GoSPA incorporates several fundamental security practices by design:

- **Cross-Site Scripting (XSS) Protection**: Client-side states and UI reactivity use DOMPurify to effectively sanitize any injected HTML layout payloads unless explicitly opted out (`SimpleRuntimeSVGs` configuration).
- **Cross-Site Request Forgery (CSRF)**: Framework provides optional Double-Submit-Cookie strategies for authenticating mutable state requests.

## CSRF Protection Setup

GoSPA uses a **two-middleware pattern** for CSRF protection. You must configure both middleware correctly:

### 1. CSRFSetTokenMiddleware (GET/HEAD requests)
This middleware sets the CSRF cookie and token for safe (read-only) requests:

```go
app.Use(gospa.CSRFSetTokenMiddleware(gospa.CSRFConfig{
    CookieName:     "csrf_token",
    CookieHTTPOnly: true,
    CookieSecure:   true,  // Use true in production with HTTPS
    CookieSameSite: "Strict",
}))
```

### 2. CSRFTokenMiddleware (POST/PUT/DELETE/PATCH)
This middleware validates the CSRF token on mutating requests:

```go
app.Use(gospa.CSRFTokenMiddleware(gospa.CSRFConfig{
    TokenLookup:    "header:X-CSRF-Token",
    CookieName:     "csrf_token",
    CookieHTTPOnly: true,
    CookieSecure:   true,
}))
```

### Important: Middleware Order Matters

Place `CSRFSetTokenMiddleware` BEFORE `CSRFTokenMiddleware`:

```go
// CORRECT: Set token first, then validate
app.Use(gospa.CSRFSetTokenMiddleware(csrfConfig))  // Sets cookie on GET
app.Use(gospa.CSRFTokenMiddleware(csrfConfig))     // Validates on POST/PUT/DELETE

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

- **Data Encapsulation**: Server states synchronize differentially while validating scopes. Ensure proper RBAC authorization on all sensitive payload actions via custom Handlers or Plugins.

## Configuration Hardening Guidelines

For production environments, ensure you abide by these guidelines:
- Set `DevMode: false`. Development mode enables detailed stack traces to leak which is unsafe for public endpoints.
- Initialize explicitly locked `AllowedOrigins` for `CORSMiddleware`. Avoid wildcard `*` domains whenever sensitive cookies/authentication tokens are utilized.
- Never place session identifiers in URL query arguments or params.
- Rate-limit Action routes dynamically leveraging proxies or internal token buckets.

## Reporting a Vulnerability

If you have discovered a security vulnerability in this project, do not open a public issue. We handle vulnerability disclosures privately.

Please report any identified vulnerability via email directly to the maintainers or report it through our GitHub Security platform.
You will receive an acknowledgment within 48 hours with an estimation of when the problem will be resolved.

## Security Update Policy

Vulnerabilities with a "Critical" or "High" classification are usually prioritized immediately with an out-of-band hotfix release. Regular security updates are appended to the next minor version iteration.
