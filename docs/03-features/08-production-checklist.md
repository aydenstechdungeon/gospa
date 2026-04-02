# Production Checklist

Use this page as the recommended baseline for shipping a GoSPA app to production.

## 1. Start from `gospa.ProductionConfig()`

```go
config := gospa.ProductionConfig()
config.AppName = "myapp"
config.AllowedOrigins = []string{"https://example.com"}

app := gospa.New(config)
```

Why this baseline works:
- `ProductionConfig()` keeps the standard runtime and route conventions intact.
- It disables development overlays and enables template caching by default.
- `AllowedOrigins` makes cross-origin behavior explicit.
- The secure CSRF default remains enabled.

## 2. Lock down origins and public entry points

- Set `AllowedOrigins` to the real application origins you serve.
- Set `PublicOrigin` when your app runs behind a proxy/CDN and request-derived origins are unreliable.
- Keep `RemoteActionMiddleware` enabled for authenticated actions.
- Do not set `AllowUnauthenticatedRemoteActions` unless the endpoint is intentionally public.

## 3. Use secure transport

- Serve the app over HTTPS.
- Ensure websocket connections upgrade over `wss://` in production.
- Tighten `ContentSecurityPolicy` beyond the built-in default when your deployment allows it (the default allows inline script/style for typical GoSPA output; see `fiber.DefaultContentSecurityPolicy`).

## 4. Scale prefork safely

If you enable `Prefork`, also configure shared backends:

```go
config.Prefork = true
config.Storage = redisStore
config.PubSub = redisPubSub
```

Without shared `Storage` and `PubSub`, worker processes diverge and realtime state becomes inconsistent.

## 5. Validate builds before release

Recommended repo checks:

```bash
bun check
go test ./...
govulncheck ./...
./scripts/validate-examples.sh
```

`./scripts/quality-check.sh` runs a fuller suite (fmt, vet, staticcheck, golangci-lint, govulncheck, examples). Go packages under `client/node_modules` are excluded from `go test` / `go build` patterns in that script. If your app includes a client package, also run its Bun test/typecheck pipeline before tagging a release.

## 6. Release checklist

- [ ] `DevMode` is off
- [ ] `AllowedOrigins` is explicit
- [ ] `EnableCSRF` remains on
- [ ] remote actions are behind middleware or intentionally public
- [ ] prefork deployments use shared `Storage` and `PubSub`
- [ ] Bun and Go validation pass cleanly
- [ ] example and scaffold drift has been checked for the release
