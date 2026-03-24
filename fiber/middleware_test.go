package fiber

import (
	"net/http/httptest"
	"strings"
	"testing"

	gofiber "github.com/gofiber/fiber/v3"
)

func TestCSRFSetTokenMiddleware_DoesNotRotateExistingToken(t *testing.T) {
	app := gofiber.New()
	app.Use(CSRFSetTokenMiddleware())
	app.Get("/", func(c gofiber.Ctx) error {
		return c.SendStatus(gofiber.StatusOK)
	})

	req1 := httptest.NewRequest("GET", "/", nil)
	resp1, err := app.Test(req1)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	defer resp1.Body.Close()

	setCookie := resp1.Header.Get("Set-Cookie")
	if setCookie == "" {
		t.Fatal("expected csrf Set-Cookie header on first request")
	}

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.Header.Set("Cookie", setCookie)
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	defer resp2.Body.Close()

	if got := resp2.Header.Get("Set-Cookie"); strings.Contains(got, "csrf_token=") {
		t.Fatalf("expected csrf token not to rotate, got Set-Cookie=%q", got)
	}
}
