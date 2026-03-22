package gospa

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/aydenstechdungeon/gospa/routing"
	"github.com/gofiber/fiber/v3"
)

func TestRemoteAction_ProductionBlocksWithoutMiddleware(t *testing.T) {
	name := strings.ReplaceAll(t.Name(), "/", "_")
	var called atomic.Bool
	routing.RegisterRemoteAction(name, func(_ context.Context, _ routing.RemoteContext, _ interface{}) (interface{}, error) {
		called.Store(true)
		return map[string]string{"ok": "true"}, nil
	})

	app := New(Config{DevMode: false})
	defer func() { _ = app.Fiber.Shutdown() }()

	req := httptest.NewRequest(http.MethodPost, "/_gospa/remote/"+name, strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Fiber.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if res.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", fiber.StatusUnauthorized, res.StatusCode)
	}
	if called.Load() {
		t.Fatal("remote action must not run without RemoteActionMiddleware in production")
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["code"] != "REMOTE_AUTH_REQUIRED" {
		t.Fatalf("expected REMOTE_AUTH_REQUIRED, got %#v", body)
	}
}

func TestRemoteAction_JSONTooDeep(t *testing.T) {
	name := strings.ReplaceAll(t.Name(), "/", "_")
	routing.RegisterRemoteAction(name, func(_ context.Context, _ routing.RemoteContext, _ interface{}) (interface{}, error) {
		t.Fatal("handler must not run when JSON exceeds max nesting")
		return nil, nil
	})

	app := New(Config{DevMode: true})
	defer func() { _ = app.Fiber.Shutdown() }()

	n := remoteJSONMaxNesting + 1
	payload := strings.Repeat(`{"a":`, n) + `0` + strings.Repeat(`}`, n)
	req := httptest.NewRequest(http.MethodPost, "/_gospa/remote/"+name, strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Fiber.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", fiber.StatusBadRequest, res.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["code"] != "JSON_TOO_DEEP" {
		t.Fatalf("expected JSON_TOO_DEEP, got %#v", body)
	}
}

func TestRemoteActionMiddleware_BlocksRequestBeforeHandler(t *testing.T) {
	var called atomic.Bool
	actionName := "test_remote_middleware_block"
	routing.RegisterRemoteAction(actionName, func(_ context.Context, _ routing.RemoteContext, _ interface{}) (interface{}, error) {
		called.Store(true)
		return map[string]string{"ok": "true"}, nil
	})

	app := New(Config{
		RemoteActionMiddleware: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "blocked"})
		},
	})
	defer func() { _ = app.Fiber.Shutdown() }()

	req := httptest.NewRequest(http.MethodPost, "/_gospa/remote/"+actionName, strings.NewReader(`{"k":"v"}`))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Fiber.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if res.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", fiber.StatusUnauthorized, res.StatusCode)
	}
	if called.Load() {
		t.Fatal("remote action should not have been invoked when middleware blocks request")
	}

	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["error"] != "blocked" {
		t.Fatalf("expected blocked error response, got %#v", body)
	}
}
