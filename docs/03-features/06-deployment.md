# Deployment Guide

Deploying a GoSPA application to production is simple because GoSPA compiles your backend and HTML generators into a single lightweight Go executable.

## 1. Building the Application

GoSPA incorporates a CLI pipeline to handle both your frontend TypeScript compilation and your backend `templ`/Go files.

To prepare a production build:
```bash
gospa build
```

This compiles `templ` outputs, executes client build tasks, and constructs a standalone Go binary usually located at `bin/app`.

## 2. Environment Configuration

GoSPA uses environment variables to drive production behavior:

- `GOSPA_ENV=production`: Enables production optimizations (template caching, minification).
- `JWT_SECRET`: (If using Auth plugin) A secure, random string (min 32 chars).
- `PUBLIC_ORIGIN`: Your public URL (e.g., `https://myapp.com`). Required for secure WebSockets.

## 3. Security Hardening

- **HTTPS Only**: Always serve early over TLS.
- **CSRF Protection**: Enabled by default; the `csrf_token` cookie is `HttpOnly`.
- **Content Security Policy**: Use `fiber.StrictContentSecurityPolicy()`.
- **Allowed Origins**: Be explicit in production configs.

## 4. Containerization (Docker)

Optimal lightweight multi-stage `Dockerfile`:

```dockerfile
# Stage 1: Build
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache nodejs npm
RUN npm install -g bun

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN bun install
RUN gospa build
RUN go build -o main .

# Stage 2: Prod
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/static ./static
COPY --from=builder /app/routes ./routes

EXPOSE 3000
CMD ["./main"]
```

## 5. Scaling & Reverse Proxies

For multi-instance deployments (Kubernetes, Prefork), use external storage:

```go
config := gospa.ProductionConfig()
config.Storage = redisStore.New(...)
config.PubSub = redisPubSub.New(...)
```

### Nginx Configuration
Ensure you handle WebSocket upgrades:

```nginx
location /_gospa/ws {
    proxy_pass http://app:3000;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "Upgrade";
    proxy_set_header Host $host;
}
```
