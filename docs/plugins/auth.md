# Authentication Plugin

Complete authentication solution with OAuth2, JWT, and OTP support.

## Installation

```bash
gospa add auth
```

## Configuration

```yaml
plugins:
  auth:
    jwt_secret: ${JWT_SECRET}
    jwt_expiry: 24
    oauth_providers: [google, github]
    otp_enabled: true
```

> [!IMPORTANT]
> **Security Requirement:** In production mode (`GOSPA_ENV=production`), the `jwt_secret` **must** be set, or the application will panic on startup. Use a secure 32-character hex string.

## Usage

```go
import "github.com/aydenstechdungeon/gospa/plugin/auth"

authPlugin := auth.New(&auth.Config{
    JWTSecret:  "your-secret",
    OTPEnabled: true,
})

token, _ := authPlugin.CreateToken(userID, userEmail, role)
```

## OTP Rate Limiting
The plugin includes a built-in rate limiter for OTP attempts (max 5 attempts per 5 minutes). When using the in-memory limiter, a background cleanup routine prunes expired entries every 15 minutes to ensure memory safety.
