# Deployment Guide

Deploying a GoSPA application to production is very simple because GoSPA compiles your backend and HTML generators directly into a single lightweight Go executable. 

This guide details best practices to run securely at scale.

## 1. Building the Application

GoSPA incorporates a CLI pipeline to handle both your frontend typescript compilation and your backend templ/go files.

To prepare a production build, run:
```bash
gospa build
```

For the recommended baseline, start from `gospa.DefaultConfig()`, set `DevMode = false`, make `AllowedOrigins` explicit, and keep CSRF protection enabled.

This compiles `templ` outputs, executes client build tasks, outputs it inside your binary (if opted using `go:embed`), and constructs a standalone Go binary usually ending up at `bin/app`.

## 2. Environment Considerations

Always ensure `/bin/app` executes with explicit Production variables. A standard production context limits detailed error overlays. 

Ensure you enable HSTS flags and run your application exclusively over TLS / HTTPS.

## 3. Production Deployment with Docker

Most apps running GoSPA thrive via Docker. Below is an optimal lightweight multi-stage `Dockerfile`:

```dockerfile
# Stage 1: Build
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# We assume gospa build has run, or you can integrate it locally
RUN go build -o main .

# Stage 2: Prod 
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/main . 
EXPOSE 3000

CMD ["./main"]
```

## 4. Production WebSockets and CDN Handling

When hooking GoSPA behind Nginx, standard cloud balancing setups (AWS/GCP), or an edge CDN (Cloudflare), remember to proxy WebSocket upgrades continuously.

Additionally, handle rate limits. GoSPA will natively enforce per-IP WebSocket and action limits, but large ingress networks spoof IPs unless you actively forward using `X-Forwarded-For`.


## 5. Pre-Release Validation

Before tagging a release, run:

```bash
bun check
go test ./...
./scripts/validate-examples.sh
```

This catches runtime drift, stale examples, and Go/Bun integration issues before deployment.
