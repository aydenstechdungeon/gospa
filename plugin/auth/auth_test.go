package auth

import (
	"encoding/base32"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

func TestMain(m *testing.M) {
	// DefaultConfig() only auto-generates a dev secret when not in production; avoid CI/env
	// accidentally setting production without JWT_SECRET.
	_ = os.Setenv("ENV", "development")
	os.Exit(m.Run())
}

func TestEnableOTPHandler_UnauthorizedWithoutUserLocal(t *testing.T) {
	app := fiber.New()
	p := New(DefaultConfig())
	app.Post("/auth/otp/enable", p.EnableOTPHandler())

	req := httptest.NewRequest(http.MethodPost, "/auth/otp/enable", nil)
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if res.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", fiber.StatusUnauthorized, res.StatusCode)
	}
}

func TestDefaultConfig_PanicsWithoutJWTWhenProduction(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("GOSPA_ENV", "production")
	t.Setenv("ENV", "development")
	t.Setenv("APP_ENV", "")
	t.Setenv("GO_ENV", "")
	t.Setenv("GIN_MODE", "")

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when GOSPA_ENV=production and JWT_SECRET is empty")
		}
	}()
	_ = DefaultConfig()
}

func TestDefaultConfig_GOEnvProductionUsesJWTSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "unit-test-jwt-secret-at-least-32-characters")
	t.Setenv("GO_ENV", "production")
	t.Setenv("GOSPA_ENV", "")

	cfg := DefaultConfig()
	if cfg.JWTSecret != "unit-test-jwt-secret-at-least-32-characters" {
		t.Fatalf("expected JWT from env, got %q", cfg.JWTSecret)
	}
}

func TestValidateToken_RejectsMismatchedIssuer(t *testing.T) {
	base := DefaultConfig()
	base.JWTSecret = "shared-secret"
	base.Issuer = "service-a"
	issuerA := New(base)

	token, err := issuerA.CreateToken("u1", "u1@example.com", "user")
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	otherCfg := DefaultConfig()
	otherCfg.JWTSecret = "shared-secret"
	otherCfg.Issuer = "service-b"
	issuerB := New(otherCfg)

	_, err = issuerB.ValidateToken(token)
	if err == nil {
		t.Fatal("expected issuer validation error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid issuer") {
		t.Fatalf("expected invalid issuer error, got %q", err.Error())
	}
}

func TestVerifyOTPHandler_RequiresAuthenticatedPrincipal(t *testing.T) {
	app := fiber.New()
	p := New(DefaultConfig())
	app.Post("/auth/otp/verify", p.VerifyOTPHandler())

	body := []byte(`{"code":"123456"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/otp/verify", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if res.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", fiber.StatusUnauthorized, res.StatusCode)
	}
}

func TestVerifyOTPHandler_UsesServerSideSecretOnly(t *testing.T) {
	cfg := DefaultConfig()
	p := New(cfg)

	storedSecret, _, err := p.GenerateOTP("user1@example.com")
	if err != nil {
		t.Fatalf("failed to generate OTP secret: %v", err)
	}

	cfg.ResolveOTPSecret = func(userID string) (string, error) {
		if userID != "u1" {
			return "", fmt.Errorf("missing user")
		}
		return storedSecret, nil
	}

	validCode := otpCodeForSecret(t, p, storedSecret)

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", &User{ID: "u1", Email: "user1@example.com"})
		return c.Next()
	})
	app.Post("/auth/otp/verify", p.VerifyOTPHandler())

	attackerField := "sec" + "ret"
	body, err := json.Marshal(map[string]string{
		attackerField: "ATTACKER_SUPPLIED_SECRET",
		"code":        validCode,
	})
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/otp/verify", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status %d, got %d", fiber.StatusOK, res.StatusCode)
	}
}

func TestVerifyOTPHandler_RateLimitBoundToUserAndIP(t *testing.T) {
	cfg := DefaultConfig()
	p := New(cfg)

	storedSecret, _, err := p.GenerateOTP("user@example.com")
	if err != nil {
		t.Fatalf("failed to generate OTP secret: %v", err)
	}

	cfg.ResolveOTPSecret = func(userID string) (string, error) {
		switch userID {
		case "u1", "u2":
			return storedSecret, nil
		default:
			return "", fmt.Errorf("missing user")
		}
	}

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", &User{ID: c.Get("X-User-ID"), Email: "test@example.com"})
		return c.Next()
	})
	app.Post("/auth/otp/verify", p.VerifyOTPHandler())

	invalidBody := []byte(`{"code":"000000"}`)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/auth/otp/verify", strings.NewReader(string(invalidBody)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "u1")
		res, err := app.Test(req)
		if err != nil {
			t.Fatalf("request %d failed: %v", i+1, err)
		}
		if res.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("expected status %d for failed attempt %d, got %d", fiber.StatusUnauthorized, i+1, res.StatusCode)
		}
	}

	blockedReq := httptest.NewRequest(http.MethodPost, "/auth/otp/verify", strings.NewReader(string(invalidBody)))
	blockedReq.Header.Set("Content-Type", "application/json")
	blockedReq.Header.Set("X-User-ID", "u1")
	blockedRes, err := app.Test(blockedReq)
	if err != nil {
		t.Fatalf("blocked request failed: %v", err)
	}
	if blockedRes.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("expected status %d after lockout, got %d", fiber.StatusTooManyRequests, blockedRes.StatusCode)
	}

	otherUserReq := httptest.NewRequest(http.MethodPost, "/auth/otp/verify", strings.NewReader(string(invalidBody)))
	otherUserReq.Header.Set("Content-Type", "application/json")
	otherUserReq.Header.Set("X-User-ID", "u2")
	otherUserRes, err := app.Test(otherUserReq)
	if err != nil {
		t.Fatalf("other user request failed: %v", err)
	}
	if otherUserRes.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected status %d for other user, got %d", fiber.StatusUnauthorized, otherUserRes.StatusCode)
	}
}

func otpCodeForSecret(t *testing.T, p *AuthPlugin, secret string) string {
	t.Helper()

	normalized := strings.ToUpper(strings.ReplaceAll(secret, " ", ""))
	key, err := base32.StdEncoding.DecodeString(normalized)
	if err != nil {
		t.Fatalf("failed to decode secret: %v", err)
	}

	counter := time.Now().Unix() / int64(p.config.OTPPeriod)
	return p.generateOTP(key, counter)
}
