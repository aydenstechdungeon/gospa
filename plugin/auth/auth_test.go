package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

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
