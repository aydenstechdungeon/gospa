# Troubleshooting Build & Deployment

## Build Fails with "templ not found"

### Problem

```
exec: "templ": executable file not found in $PATH
```

### Solution

Install the templ CLI:

```bash
go install github.com/a-h/templ/cmd/templ@latest

# Verify installation
templ version
```

Ensure `$GOPATH/bin` is in your PATH:

```bash
# Add to ~/.bashrc, ~/.zshrc, etc.
export PATH=$PATH:$(go env GOPATH)/bin
```

---

## "undefined: gospa.Config" or Import Errors

### Problem
Build fails with undefined symbols or import errors.

### Causes & Solutions

#### 1. Missing go.mod Entry

```bash
# Ensure gospa is in go.mod
go get github.com/aydenstechdungeon/gospa

# Tidy dependencies
go mod tidy
```

#### 2. Wrong Package Name

```go
// BAD - Wrong import path
import "github.com/aydenstechdungeon/gospa/pkg/gospa"

// GOOD - Correct import
import "github.com/aydenstechdungeon/gospa"
```

#### 3. Version Mismatch

```bash
# Check installed version
go list -m github.com/aydenstechdungeon/gospa

# Update to latest
go get -u github.com/aydenstechdungeon/gospa
```

---

## Binary Size Too Large

### Problem
Compiled binary is hundreds of megabytes.

### Solutions

#### 1. Strip Debug Info

```bash
go build -ldflags="-s -w" -o app .
```

#### 2. Disable DWARF

```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o app .
```

#### 3. Use UPX Compression (Optional)

```bash
# Install UPX
apt-get install upx  # Debian/Ubuntu
brew install upx     # macOS

# Compress binary
upx --best -o app-compressed app
```

#### 4. Remove Unused Features

```go
app := gospa.New(gospa.Config{
    // Disable unused features
    WebSocket:     false,
    EnableSSE:     false,
    EnableCSRF:    false,
    SimpleRuntime: true,  // Use lighter runtime
})
```

---

## Template Not Found in Production

### Problem
App works in dev but templates fail in production build.

### Causes & Solutions

#### 1. Templates Not Generated

```bash
# Generate templates before build
templ generate

# Or use go:generate
//go:generate templ generate
```

#### 2. Missing Embedded Files

```go
// Ensure embed directive includes templates
//go:embed routes/*.templ
var templates embed.FS
```

#### 3. Template Path Issues

```go
// Use relative paths that work in both dev and production
app := gospa.New(gospa.Config{
    TemplateDir: "./routes",  // Relative to working dir
})

// Or use embed.FS for production
app := gospa.New(gospa.Config{
    TemplateFS: templates,
})
```

---

## Environment Variables Not Loading

### Problem
`os.Getenv()` returns empty in production.

### Solutions

#### 1. Load .env File

```go
import "github.com/joho/godotenv"

func init() {
    // Load .env in development only
    if os.Getenv("ENV") != "production" {
        godotenv.Load()
    }
}
```

#### 2. Set Variables in Production

```bash
# Systemd service
[Service]
Environment="DATABASE_URL=postgres://..."
Environment="PORT=8080"

# Docker
ENV DATABASE_URL=postgres://...
ENV PORT=8080

# Kubernetes
env:
  - name: DATABASE_URL
    valueFrom:
      secretKeyRef:
        name: db-secret
        key: url
```

---

## Static Assets 404 in Production

### Problem
CSS/JS files return 404 after deployment.

### Solutions

#### 1. Correct Static File Path

```go
// BAD - Relative path may not work
app.Static("/static", "./static")

// GOOD - Absolute path or embed
app.Static("/static", "/app/static")

// Or use embed
//go:embed static/*
var staticFiles embed.FS

app.StaticFS("/static", staticFiles)
```

#### 2. Check File Permissions

```bash
# Ensure files are readable
chmod -R 755 ./static

# Verify ownership
chown -R www-data:www-data ./static
```

#### 3. Configure CDN (if applicable)

```go
app := gospa.New(gospa.Config{
    StaticURL: "https://cdn.example.com",  // CDN URL
})
```

---

## Port Already in Use (Production)

### Problem

```
bind: address already in use
```

### Solutions

#### 1. Use Environment Port

```go
port := os.Getenv("PORT")
if port == "" {
    port = "3000"
}

app := gospa.New(gospa.Config{
    Port: port,
})
```

#### 2. Graceful Shutdown

```go
// Handle SIGTERM properly
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

go func() {
    <-quit
    log.Println("Shutting down...")
    app.Shutdown()
}()
```

#### 3. Kill Existing Process

```bash
# Find process using port
lsof -i :8080

# Kill it
kill -9 <PID>
```

---

## Docker Build Issues

### Problem
Docker build fails or image is too large.

### Solutions

#### 1. Multi-Stage Build

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go generate ./...
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main .

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/static ./static
EXPOSE 8080
CMD ["./main"]
```

#### 2. BuildKit for Faster Builds

```bash
DOCKER_BUILDKIT=1 docker build -t myapp .
```

#### 3. Cache Dependencies

```dockerfile
# Copy only dependency files first
COPY go.mod go.sum ./
RUN go mod download

# Then copy source (cache layer reused if deps unchanged)
COPY . .
```

---

## WebSocket Fails Behind Load Balancer

### Problem
WebSocket connections fail in production behind Nginx/ALB.

### Solutions

#### 1. Nginx Configuration

```nginx
location /_gospa/ws {
    proxy_pass http://backend;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    
    # Timeout settings
    proxy_read_timeout 86400;
    proxy_send_timeout 86400;
}
```

#### 2. AWS ALB Configuration

```yaml
# Target group settings
ProtocolVersion: HTTP/1.1
HealthCheckProtocol: HTTP
Stickiness: 
  Enabled: true
  Type: lb_cookie
```

#### 3. Enable Proxy Trust

```go
app := gospa.New(gospa.Config{
    EnableProxy: true,  // Trust X-Forwarded-* headers
})
```

---

## SSL/TLS Certificate Issues

### Problem
HTTPS fails or shows certificate warnings.

### Solutions

#### 1. Auto HTTPS (Let's Encrypt)

```go
app := gospa.New(gospa.Config{
    AutoTLS: true,
    TLSDomains: []string{"example.com", "www.example.com"},
})
```

#### 2. Custom Certificates

```go
app := gospa.New(gospa.Config{
    TLSCert: "/path/to/cert.pem",
    TLSKey:  "/path/to/key.pem",
})
```

#### 3. Reverse Proxy SSL (Nginx)

```nginx
server {
    listen 443 ssl http2;
    server_name example.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

---

## Memory Leaks in Production

### Problem
Memory usage grows continuously.

### Solutions

#### 1. Enable Pprof

```go
import _ "net/http/pprof"

func init() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
}

# Analyze
# go tool pprof http://localhost:6060/debug/pprof/heap
```

#### 2. Limit Concurrent Connections

```go
app := gospa.New(gospa.Config{
    MaxConnections: 1000,
    ReadTimeout:    30 * time.Second,
    WriteTimeout:   30 * time.Second,
})
```

#### 3. Clean Up WebSocket Clients

```go
// Set ping interval to detect dead connections
app := gospa.New(gospa.Config{
    WSHeartbeat:       30 * time.Second,
    WSHeartbeatTimeout: 10 * time.Second,
})
```

---

## "too many open files" in Production

### Problem
Server crashes with file descriptor limit errors.

### Solution

#### 1. Increase System Limits

```bash
# /etc/security/limits.conf
* soft nofile 65536
* hard nofile 65536

# Or in systemd service
[Service]
LimitNOFILE=65536
```

#### 2. Reduce Keep-Alive

```go
app := gospa.New(gospa.Config{
    IdleTimeout: 60 * time.Second,
})
```

---

## Database Connection Issues

### Problem
Database connections fail or timeout in production.

### Solutions

#### 1. Connection Pooling

```go
db, err := sql.Open("postgres", dsn)
if err != nil {
    log.Fatal(err)
}

// Configure pool
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(5 * time.Minute)
```

#### 2. Health Checks

```go
app.Get("/health", func(c *routing.Context) error {
    if err := db.Ping(); err != nil {
        return c.Status(503).JSON(map[string]string{
            "status": "unhealthy",
            "error": err.Error(),
        })
    }
    return c.JSON(map[string]string{"status": "healthy"})
})
```

---

## Common Mistakes

### Mistake 1: Hardcoded Development Values

```go
// BAD
app := gospa.New(gospa.Config{
    DevMode: true,  // Never in production!
    Port:    "3000",
})

// GOOD
app := gospa.New(gospa.Config{
    DevMode: os.Getenv("ENV") == "development",
    Port:    getEnv("PORT", "8080"),
})
```

### Mistake 2: No Request Timeouts

```go
// BAD - No timeouts
app := gospa.New(gospa.Config{})

// GOOD - Always set timeouts
app := gospa.New(gospa.Config{
    ReadTimeout:  30 * time.Second,
    WriteTimeout: 30 * time.Second,
    IdleTimeout:  120 * time.Second,
})
```

### Mistake 3: Logging Sensitive Data

```go
// BAD
log.Printf("User login: %s, password: %s", email, password)

// GOOD
log.Printf("User login attempt: %s", email)
// Never log passwords, tokens, or PII
```

### Mistake 4: Not Handling Panics

```go
// BAD - Panic crashes server
app.Get("/api/data", func(c *routing.Context) error {
    result := riskyOperation()  // May panic
    return c.JSON(result)
})

// GOOD - Recover from panics
app.Use(func(c *routing.Context) error {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Panic recovered: %v", r)
            c.Status(500).JSON(map[string]string{
                "error": "Internal server error",
            })
        }
    }()
    return c.Next()
})
```


---

## Running Quality & Security Checks Before Deploy

Use the repository script to run Bun + Go checks in one command:

```bash
./scripts/quality-check.sh
```

Default checks:
- `bun check` (root script)
- `gofmt` check
- `go vet`
- `staticcheck`
- `golangci-lint`
- `govulncheck`
- `go build`
- `go test`

Examples:

```bash
# Include race detector
./scripts/quality-check.sh --with-race

# Skip tools not installed in local env
./scripts/quality-check.sh --skip-golangci --skip-vulncheck --skip-staticcheck
```
