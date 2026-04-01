package gospa

import (
	"strings"
	"testing"
	"time"

	gofiber "github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

func TestValidatePublicHost(t *testing.T) {
	app := New(Config{DevMode: true})
	defer func() { _ = app.Fiber.Shutdown() }()

	tests := []struct {
		name     string
		host     string
		expected string
		valid    bool
	}{
		{"empty", "", "", false},
		{"too long", strings.Repeat("a", 254), "", false},
		{"contains @", "user@host.com", "", false},
		{"contains ://", "http://host.com", "", false},
		{"invalid chars", "host!name.com", "", false},
		{"valid local", "localhost", "localhost", true},
		{"valid ip", "127.0.0.1", "127.0.0.1", true},
		{"valid with port", "localhost:8080", "localhost:8080", true},
		{"multiple colons", "localhost:80:80", "localhost:80:80", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, valid := app.validatePublicHost(tt.host)
			if valid != tt.valid {
				t.Errorf("expected valid=%v for host %q, got %v", tt.valid, tt.host, valid)
			}
			if valid && res != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, res)
			}
		})
	}
}

func TestValidatePublicHost_Production_NoOrigin(t *testing.T) {
	// In production (DevMode: false) and no PublicOrigin set, all hosts are rejected
	app := New(Config{DevMode: false, PublicOrigin: ""})
	defer func() { _ = app.Fiber.Shutdown() }()

	_, valid := app.validatePublicHost("example.com")
	if valid {
		t.Error("expected host to be invalid in prod without PublicOrigin")
	}
}

func TestValidatePublicHost_Production_WithOrigin(t *testing.T) {
	app := New(Config{DevMode: false, PublicOrigin: "https://example.com"})
	defer func() { _ = app.Fiber.Shutdown() }()

	res, valid := app.validatePublicHost("example.com:443")
	if !valid {
		t.Error("expected host example.com to be valid")
	}
	if res != "example.com:443" {
		t.Errorf("expected result to be example.com:443, got %s", res)
	}

	_, valid = app.validatePublicHost("malicious.com:80")
	if valid {
		t.Error("expected malicious host to be invalid")
	}
}

func TestNormalizeWSConfig(t *testing.T) {
	app := New(Config{
		WSReconnectDelay: 500 * time.Millisecond,
		WSMaxReconnect:   5,
		WSHeartbeat:      10 * time.Second,
	})
	defer func() { _ = app.Fiber.Shutdown() }()

	rd, mr, hb := app.normalizeWSConfig()
	if rd != 500 || mr != 5 || hb != 10000 {
		t.Errorf("expected 500/5/10000, got %d/%d/%d", rd, mr, hb)
	}

	// Test fallback defaults
	appDefaults := New(Config{
		WSReconnectDelay: -1,
		WSMaxReconnect:   -1,
		WSHeartbeat:      -1,
	})
	defer func() { _ = appDefaults.Fiber.Shutdown() }()

	rdd, mdd, hbd := appDefaults.normalizeWSConfig()
	if rdd != 1000 || mdd != 10 || hbd != 30000 {
		t.Errorf("expected 1000/10/30000 defaults, got %d/%d/%d", rdd, mdd, hbd)
	}
}

func TestToJS(t *testing.T) {
	js := toJS(map[string]interface{}{"key": "value"})
	if js != `{"key":"value"}` {
		t.Errorf("expected `{\"key\":\"value\"}`, got %s", js)
	}
}

func TestGetWSUrl(t *testing.T) {
	app := New(Config{
		PublicOrigin:  "https://example.com:8443",
		WebSocketPath: "/wsx",
	})
	defer func() { _ = app.Fiber.Shutdown() }()

	// create mock context
	f := gofiber.New()
	reqCtx := &fasthttp.RequestCtx{}
	c := f.AcquireCtx(reqCtx)

	ws := app.getWSUrl(c)
	if ws != "wss://example.com:8443/wsx" {
		t.Errorf("expected wss://example.com:8443/wsx, got %s", ws)
	}
}

func TestGetWSUrl_DevFallback(t *testing.T) {
	app := New(Config{
		DevMode:       true,
		WebSocketPath: "/wsx",
	})
	defer func() { _ = app.Fiber.Shutdown() }()

	f := gofiber.New()
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetHost("localhost:3000")
	c := f.AcquireCtx(reqCtx)

	ws := app.getWSUrl(c)
	if ws != "ws://localhost:3000/wsx" {
		t.Errorf("expected ws://localhost:3000/wsx, got %s", ws)
	}
}
