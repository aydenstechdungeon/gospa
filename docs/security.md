# Security Guide

GoSPA is designed with security as a first-class citizen. This guide outlines the built-in security features and your responsibilities as a developer to ensure a secure deployment.

## 1. Production Hardening

When deploying GoSPA to production, ensure that `DevMode` is set to `false` in your `gospa.Config`.

- **Insecure Settings Enforcement**: In production, GoSPA automatically ignores `AllowInsecureWS` (forcing `wss://`) and `AllowUnauthenticatedRemoteActions`.
- **Logger**: Use a structured logger (`slog.Default()` or similar) to track security events without leaking sensitive data in logs.

### Checklist
- [ ] Set `DevMode: false`.
- [ ] Provide a strong `JWT_SECRET` environment variable (for the Auth plugin).
- [ ] Configure `PublicOrigin` to match your production domain.
- [ ] Set `AllowedOrigins` to restrict CORS.

## 2. CSRF Protection

GoSPA includes robust CSRF (Cross-Site Request Forgery) protection for both AJAX/WebSocket requests and standard HTML form submissions.

### How it Works
1. `CSRFSetTokenMiddleware` sets an `HttpOnly`, `SameSite: Strict` cookie named `csrf_token`.
2. `CSRFTokenMiddleware` validates incoming mutating requests (POST, PUT, DELETE, PATCH).

### Usage
- **AJAX/Fetch**: Include the `X-CSRF-Token` header.
- **HTML Forms**: Include a hidden input named `_csrf`. GoSPA provides the token in the global state as `window.__GOSPA_CSRF_TOKEN__`.

```html
<form method="POST" action="/update">
  <input type="hidden" name="_csrf" value={csrfToken} />
  <!-- ... -->
</form>
```

## 3. Content Security Policy (CSP)

GoSPA supports strict CSPs by automatically generating per-request **Nonces**.

### Automated Nonce Injection
GoSPA's `SecurityHeadersMiddleware` generates a unique nonce for every request. This nonce is:
1. Added to the `Content-Security-Policy` header.
2. Injected into the framework-managed `<script>` tags for state hydration and runtime initialization.

### Configuring a Strict Policy
Instead of using `'unsafe-inline'`, use the `{nonce}` placeholder in your policy:

```go
config.ContentSecurityPolicy = "default-src 'self'; script-src 'self' 'nonce-{nonce}'; ..."
```

### Common Nonce Pitfall
If you use a custom root layout, CSP can still fail even with `SecurityHeadersMiddleware` enabled.

Typical failure mode:
- Browser logs `Refused to execute inline script` or `Refused to load the script`.
- Framework scripts load, but custom inline/module script blocks fail.

Fix:
1. Keep `'nonce-{nonce}'` in `script-src`.
2. Add the generated nonce to every custom inline/module script tag.

```templ
<script src="/static/js/app.js" type="module" nonce={ gospatempl.GetNonce(ctx) }></script>
<script type="module" nonce={ gospatempl.GetNonce(ctx) }>
  // custom bootstrap
</script>
```

## 4. Authentication (Auth Plugin)

The optional Auth plugin provides JWT-based session management.

- **Storage**: Sessions are stored in a `store.Storage` backend (Memory by default, Redis recommended for multi-node setups).
- **Cookies**: Session tokens are stored in `HttpOnly`, `Secure` cookies to mitigate XSS-based token theft.

## 5. SFC Trust Boundary

`.gospa` (Single File Components) are compiled into Go source code. 

> [!CAUTION]
> **Never compile untrusted SFCs.**
> Treat SFCs as part of your application source code. If you must compile SFCs from semi-trusted sources, enable `SafeMode` in common compiler options to restrict available Go primitives within the component script.

## 6. Real-time Security (WebSockets)

- **Rate Limiting**: GoSPA includes a built-in token-bucket rate limiter for WebSocket connections to prevent DoS.
- **Message Validation**: Inbound WebSocket messages are validated for JSON nesting depth and field lengths to prevent stack overflow and memory exhaustion attacks.

## 7. XSS Mitigation (New)

GoSPA enforces a "Secure by Default" posture for HTML rendering to mitigate Cross-Site Scripting (XSS) attacks.

### HTML Rendering Policy
GoSPA does not bundle a runtime sanitizer by default. Dynamic HTML bindings and stream HTML updates escape content unless you explicitly mark content as trusted with runtime policy helpers.

### Trust Boundary
If you absolutely must render raw HTML, only pass server-controlled content through trusted wrappers. Never pass user input directly into HTML bindings.

## 8. Prototype Pollution Protection

When hydrating component state from the server, GoSPA uses a `safeJSONParse` utility. This utility automatically strips dangerous keys like `__proto__`, `constructor`, and `prototype` from the incoming JSON payload, preventing attackers from hijacking the JavaScript prototype chain.

## 9. Sensitive Data Redaction

To prevent accidental data leakage during development, the GoSPA error overlay automatically redacts sensitive headers in its UI representation. Redacted headers include:
- `Authorization`
- `Cookie` / `Set-Cookie`
- `X-Api-Key`
- `X-Csrf-Token`

## 10. SafeMode (Compiler Sandboxing)

GoSPA includes a `SafeMode` option for the SFC compiler to prevent "sandbox escapes" when compiling `.gospa` files from semi-trusted sources (e.g. CMS-managed components).

### Protected Resources
When `SafeMode` is enabled, the compiler enforces the following restrictions:

1. **Blocked Packages**: Any `import` statement referencing dangerous packages is blocked. 
    - `os`, `os/exec` (Prevents RCE)
    - `unsafe`, `reflect` (Prevents memory manipulation)
    - `syscall` (Prevents direct OS interaction)
    - `net`, `http` (Prevents SSRF/Exfiltration)
    - **CGo**: `import "C"` is strictly forbidden.
    - **Dot Imports**: `import . "pkg"` is blocked to prevent namespace pollution and shadowing of built-ins.

2. **Dangerous Call Validation**: The compiler uses regex-based validation to block calls that could lead to resource exhaustion or bypasses, even if the package itself isn't blocked (e.g. certain `fmt` or `log` patterns).

3. **Restricted Variable Access**: Runes are limited to the component's internal scope.

### Enabling SafeMode
```go
opts := compiler.CompileOptions{
    SafeMode: true,
    // ...
}
```

> [!WARNING]
> While `SafeMode` provides a significant layer of defense, Go is a powerful language. **Never compile truly malicious code.** SafeMode is intended for *semi-trusted* environments where you want to prevent accidental or common malicious patterns from compromised configuration sources.
