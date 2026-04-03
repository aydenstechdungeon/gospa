package fiber

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gofiber "github.com/gofiber/fiber/v3"
)

func TestBrotliGzipMiddlewareSkipsLargeBufferedResponses(t *testing.T) {
	app := gofiber.New()
	cfg := DefaultCompressionConfig()
	cfg.EnableBrotli = false
	cfg.EnableGzip = true
	cfg.MinSize = 1
	cfg.MaxBufferedSize = 32
	app.Use(BrotliGzipMiddleware(cfg))
	app.Get("/", func(c gofiber.Ctx) error {
		return c.SendString(strings.Repeat("a", 128))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if got := res.Header.Get("Content-Encoding"); got != "" {
		t.Fatalf("expected no compression for oversized buffered response, got %q", got)
	}
}

func TestBrotliGzipMiddlewareCompressesEligibleResponses(t *testing.T) {
	app := gofiber.New()
	cfg := DefaultCompressionConfig()
	cfg.EnableBrotli = false
	cfg.EnableGzip = true
	cfg.MinSize = 1
	cfg.MaxBufferedSize = 1024
	app.Use(BrotliGzipMiddleware(cfg))
	app.Get("/", func(c gofiber.Ctx) error {
		return c.SendString(strings.Repeat("compress-me-", 32))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if got := res.Header.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("expected gzip compression, got %q", got)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if len(body) == 0 || bytes.Equal(body, []byte(strings.Repeat("compress-me-", 32))) {
		t.Fatal("expected compressed response body")
	}
}

func TestBrotliGzipMiddlewareCompressesBrotli(t *testing.T) {
	app := gofiber.New()
	cfg := DefaultCompressionConfig()
	cfg.EnableBrotli = true
	cfg.MinSize = 1
	app.Use(BrotliGzipMiddleware(cfg))
	app.Get("/", func(c gofiber.Ctx) error {
		return c.SendString(strings.Repeat("brotli-me-", 32))
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "br")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if got := res.Header.Get("Content-Encoding"); got != "br" {
		t.Fatalf("expected br compression, got %q", got)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if len(body) == 0 || bytes.Equal(body, []byte(strings.Repeat("brotli-me-", 32))) {
		t.Fatal("expected compressed response body")
	}
}
