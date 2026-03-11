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
