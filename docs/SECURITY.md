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
