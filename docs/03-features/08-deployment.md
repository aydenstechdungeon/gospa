# Deployment Guide

This guide covers the best practices for deploying GoSPA applications to production environments.

## 1. Environment Configuration

GoSPA uses environment variables to drive production behavior. Ensure these are set in your production environment:

- `GOSPA_ENV=production`: Enables production optimizations (template caching, minification).
- `JWT_SECRET`: (If using Auth plugin) A long, random string.
- `PUBLIC_ORIGIN`: Your public URL (e.g., `https://myapp.com`). Required for secure WebSockets.

## 2. Security Hardening

Before shipping, verify the following:

- [ ] **HTTPS Only**: Always serve your app over HTTPS.
- [ ] **CSRF Protection**: GoSPA enables CSRF by default. The `csrf_token` cookie is `HttpOnly` for safety.
- [ ] **Content Security Policy**: Start with `fiber.StrictContentSecurityPolicy` and loosen only as needed.
- [ ] **Remote Actions**: Ensure `RemoteActionMiddleware` is configured to protect sensitive server functions.

## 3. Containerization (Docker)

Recommended `Dockerfile` for a GoSPA app:

```dockerfile
# Build stage
FROM golang:1.25.0-alpine AS builder
RUN apk add --no-cache nodejs npm
RUN npm install -g bun

WORKDIR /app
COPY . .
RUN bun install
RUN gospa build
RUN go build -o main .

# Final stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/static ./static
COPY --from=builder /app/routes ./routes

EXPOSE 3000
CMD ["./main"]
```

## 4. Scaling (Redis)

For deployments with multiple instances (e.g., Kubernetes, Prefork), you must use external storage for session and state consistency.

```go
config := gospa.ProductionConfig()
config.Storage = redisStore.New(...)
config.PubSub = redisPubSub.New(...)
```

## 5. Reverse Proxies

If running behind Nginx or Caddy, ensure you forward the correct headers:

- `X-Forwarded-For`
- `X-Forwarded-Proto`
- `Upgrade` and `Connection` (for WebSockets)

Example Nginx config:

```nginx
location /_gospa/ws {
    proxy_pass http://app:3000;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "Upgrade";
    proxy_set_header Host $host;
}
```
