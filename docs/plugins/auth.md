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

## Usage

```go
import "github.com/aydenstechdungeon/gospa/plugin/auth"

authPlugin := auth.New(&auth.Config{
    JWTSecret:  "your-secret",
    OTPEnabled: true,
})

token, _ := authPlugin.CreateToken(userID, userEmail, role)
```
