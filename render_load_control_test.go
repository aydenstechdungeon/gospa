package gospa

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/a-h/templ"
	"github.com/aydenstechdungeon/gospa/routing"
	"github.com/aydenstechdungeon/gospa/routing/kit"
	fiberpkg "github.com/gofiber/fiber/v3"
)

func TestRenderRoute_DataEndpointHandlesKitFail(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()

	routePath := fmt.Sprintf("/test-load-kit-fail-%d", time.Now().UnixNano())
	route := &routing.Route{Path: routePath}

	routing.RegisterPage(routePath, func(_ map[string]interface{}) templ.Component {
		return templ.ComponentFunc(func(_ context.Context, _ io.Writer) error {
			return nil
		})
	})
	routing.RegisterLoad(routePath, func(_ routing.LoadContext) (map[string]interface{}, error) {
		return nil, kit.Fail(http.StatusBadRequest, map[string]interface{}{"reason": "missing"})
	})

	app.Get(routePath, func(c fiberpkg.Ctx) error {
		return app.renderRoute(c, route, map[string]interface{}{})
	})

	req := httptest.NewRequest(http.MethodGet, routePath+"?__data=1", nil)
	resp, err := app.Fiber.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	var payload struct {
		Kind   string                 `json:"kind"`
		Status int                    `json:"status"`
		Data   map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if payload.Kind != "fail" || payload.Status != http.StatusBadRequest || payload.Data["reason"] != "missing" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}
