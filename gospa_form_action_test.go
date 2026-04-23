package gospa

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aydenstechdungeon/gospa/routing"
	fiberpkg "github.com/gofiber/fiber/v3"
)

func TestHandleFormAction_UsesActionQueryParam(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()

	routePath := fmt.Sprintf("/test-form-action-%d", time.Now().UnixNano())
	route := &routing.Route{Path: routePath}

	routing.RegisterAction(routePath, "default", func(_ routing.LoadContext) (interface{}, error) {
		return map[string]interface{}{"which": "default"}, nil
	})
	routing.RegisterAction(routePath, "delete", func(_ routing.LoadContext) (interface{}, error) {
		return map[string]interface{}{"which": "delete"}, nil
	})

	app.Post(routePath, func(c fiberpkg.Ctx) error {
		return app.handleFormAction(c, route)
	})

	req := httptest.NewRequest(http.MethodPost, routePath+"?_action=delete", nil)
	req.Header.Set("X-Gospa-Enhance", "1")

	resp, err := app.Fiber.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var payload struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.Data["which"] != "delete" {
		t.Fatalf("expected delete action to run, got %v", payload.Data["which"])
	}
}

func TestHandleFormAction_FallsBackToDefaultForUnknownAction(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()

	routePath := fmt.Sprintf("/test-form-action-fallback-%d", time.Now().UnixNano())
	route := &routing.Route{Path: routePath}

	routing.RegisterAction(routePath, "default", func(_ routing.LoadContext) (interface{}, error) {
		return map[string]interface{}{"which": "default"}, nil
	})

	app.Post(routePath, func(c fiberpkg.Ctx) error {
		return app.handleFormAction(c, route)
	})

	req := httptest.NewRequest(http.MethodPost, routePath+"?_action=missing", nil)
	req.Header.Set("X-Gospa-Enhance", "1")

	resp, err := app.Fiber.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var payload struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.Data["which"] != "default" {
		t.Fatalf("expected default action to run, got %v", payload.Data["which"])
	}
}
