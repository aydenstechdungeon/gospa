package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

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
