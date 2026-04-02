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
