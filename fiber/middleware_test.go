package fiber

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	gofiber "github.com/gofiber/fiber/v3"
	"github.com/aydenstechdungeon/gospa/state"
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
	defer func() { _ = resp1.Body.Close() }()

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
	defer func() { _ = resp2.Body.Close() }()

	if got := resp2.Header.Get("Set-Cookie"); strings.Contains(got, "csrf_token=") {
		t.Fatalf("expected csrf token not to rotate, got Set-Cookie=%q", got)
	}
}
func TestPreloadHeadersMiddleware(t *testing.T) {
	app := gofiber.New()
	config := DefaultPreloadConfig()
	config.CSSLinks = []string{"/style.css"}
	app.Use(PreloadHeadersMiddleware(config))

	app.Get("/", func(c gofiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		return c.SendString("<html></html>")
	})

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	link := resp.Header.Get("Link")
	if !strings.Contains(link, "</style.css>; rel=preload; as=style") {
		t.Errorf("expected CSS preload in Link header, got %q", link)
	}
	if !strings.Contains(link, "</_gospa/runtime.js>; rel=modulepreload") {
		t.Errorf("expected runtime preload in Link header, got %q", link)
	}

	// Count preloads - should be capped
	chunks := strings.Split(link, ",")
	if len(chunks) > 6 {
		t.Errorf("expected at most 6 preloads, got %d: %q", len(chunks), link)
	}
}

func TestFlashMessages(t *testing.T) {
	app := gofiber.New()
	// Mock session middleware
	app.Use(func(c gofiber.Ctx) error {
		c.Locals("gospa.session", "test-token")
		return c.Next()
	})

	app.Get("/set", func(c gofiber.Ctx) error {
		SetFlash(c, "success", "Hello")
		return c.SendStatus(gofiber.StatusOK)
	})

	app.Get("/get", func(c gofiber.Ctx) error {
		flashes := GetFlashes(c)
		if val, ok := flashes["success"].(string); ok && val == "Hello" {
			return c.SendStatus(gofiber.StatusOK)
		}
		return c.Status(gofiber.StatusBadRequest).SendString("Flash not found or incorrect")
	})

	app.Get("/get-again", func(c gofiber.Ctx) error {
		flashes := GetFlashes(c)
		if len(flashes) == 0 {
			return c.SendStatus(gofiber.StatusOK)
		}
		return c.Status(gofiber.StatusBadRequest).SendString("Flash was not cleared")
	})

	// 1. Set flash
	req1 := httptest.NewRequest("GET", "/set", nil)
	resp1, err := app.Test(req1)
	if err != nil || resp1.StatusCode != gofiber.StatusOK {
		t.Fatalf("failed to set flash: %v", err)
	}

	// 2. Get flash (should succeed and clear)
	req2 := httptest.NewRequest("GET", "/get", nil)
	resp2, err := app.Test(req2)
	if err != nil || resp2.StatusCode != gofiber.StatusOK {
		t.Fatalf("failed to get flash: %v", err)
	}

	// 3. Get flash again (should be empty)
	req3 := httptest.NewRequest("GET", "/get-again", nil)
	resp3, err := app.Test(req3)
	if err != nil || resp3.StatusCode != gofiber.StatusOK {
		t.Fatalf("flash was not cleared after reading: %v", err)
	}
}

func TestCSRFTokenMiddleware_FormSupport(t *testing.T) {
	app := gofiber.New()
	app.Post("/test", CSRFTokenMiddleware(), func(c gofiber.Ctx) error {
		return c.SendStatus(gofiber.StatusOK)
	})

	// 1. Missing token - should fail
	req1 := httptest.NewRequest("POST", "/test", nil)
	resp1, _ := app.Test(req1)
	if resp1.StatusCode != gofiber.StatusForbidden {
		t.Errorf("expected 403 for missing token, got %v", resp1.StatusCode)
	}

	// 2. Token in form body - should succeed
	csrfToken := "test-csrf-token" //nolint:gosec // this is a test token
	formData := "name=val&_csrf=" + csrfToken
	req2 := httptest.NewRequest("POST", "/test", strings.NewReader(formData))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req2.Header.Set("Cookie", "csrf_token="+csrfToken)
	resp2, _ := app.Test(req2)
	if resp2.StatusCode != gofiber.StatusOK {
		t.Errorf("expected 200 for valid form token, got %v", resp2.StatusCode)
	}
}

func TestSecurityHeadersMiddleware_Nonce(t *testing.T) {
	app := gofiber.New()
	app.Use(SecurityHeadersMiddleware("script-src 'self' {nonce}"))
	app.Get("/", func(c gofiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)

	csp := resp.Header.Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src 'self' ") || strings.Contains(csp, "{nonce}") {
		t.Errorf("expected nonce in CSP, but not found or placeholder still present: %q", csp)
	}
}

func TestStateMiddleware_NonceInjection(t *testing.T) {
	config := Config{
		DevMode:  true,
		StateKey: "gospa.state",
	}
	app := gofiber.New()
	// Mock nonce and state
	app.Use(func(c gofiber.Ctx) error {
		c.Locals("gospa.csp_nonce", "test-nonce-123")
		stateMap := state.NewStateMap()
		c.Locals(config.StateKey, stateMap)
		return c.Next()
	})
	app.Use(StateMiddleware(config))
	app.Get("/", func(c gofiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		return c.SendString("<body></body>")
	})

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)
	body, _ := io.ReadAll(resp.Body)

	if !strings.Contains(string(body), `nonce="test-nonce-123"`) {
		t.Errorf("expected nonce in injected script, but not found in body: %s", string(body))
	}
}
