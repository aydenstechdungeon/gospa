package gospa

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aydenstechdungeon/gospa/routing"
	"github.com/aydenstechdungeon/gospa/routing/kit"
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

func TestHandleFormAction_EnhancedStructuredResponse(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()

	routePath := fmt.Sprintf("/test-form-action-structured-%d", time.Now().UnixNano())
	route := &routing.Route{Path: routePath}

	routing.RegisterAction(routePath, "default", func(_ routing.LoadContext) (interface{}, error) {
		return routing.ActionResponse{
			Data: map[string]interface{}{"ok": true},
			Validation: &routing.ActionValidationError{
				FieldErrors: map[string]string{"email": "invalid"},
			},
			Revalidate:     []string{"/dashboard"},
			RevalidateTags: []string{"route:/dashboard"},
			RevalidateKeys: []string{"path:/dashboard"},
		}, nil
	})

	app.Post(routePath, func(c fiberpkg.Ctx) error {
		return app.handleFormAction(c, route)
	})

	req := httptest.NewRequest(http.MethodPost, routePath, nil)
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
		Data           map[string]interface{}         `json:"data"`
		Validation     *routing.ActionValidationError `json:"validation"`
		Revalidate     []string                       `json:"revalidate"`
		RevalidateTags []string                       `json:"revalidateTags"`
		RevalidateKeys []string                       `json:"revalidateKeys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.Data["ok"] != true {
		t.Fatalf("expected ok=true data payload, got %v", payload.Data["ok"])
	}
	if payload.Validation == nil || payload.Validation.FieldErrors["email"] != "invalid" {
		t.Fatalf("expected validation errors in response")
	}
	if len(payload.Revalidate) != 1 || payload.Revalidate[0] != "/dashboard" {
		t.Fatalf("expected revalidate path hints in response")
	}
}

func TestHandleFormAction_EnhancedKitRedirect(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()

	routePath := fmt.Sprintf("/test-form-action-kit-redirect-%d", time.Now().UnixNano())
	route := &routing.Route{Path: routePath}

	routing.RegisterAction(routePath, "default", func(_ routing.LoadContext) (interface{}, error) {
		return nil, kit.Redirect(http.StatusFound, "/next")
	})

	app.Post(routePath, func(c fiberpkg.Ctx) error {
		return app.handleFormAction(c, route)
	})

	req := httptest.NewRequest(http.MethodPost, routePath, nil)
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
		Code     string                  `json:"code"`
		Redirect *routing.ActionRedirect `json:"redirect"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != "REDIRECT" || payload.Redirect == nil || payload.Redirect.To != "/next" || payload.Redirect.Status != http.StatusFound {
		t.Fatalf("unexpected redirect payload: %+v", payload)
	}
}

func TestHandleFormAction_EnhancedKitFail(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()

	routePath := fmt.Sprintf("/test-form-action-kit-fail-%d", time.Now().UnixNano())
	route := &routing.Route{Path: routePath}

	routing.RegisterAction(routePath, "default", func(_ routing.LoadContext) (interface{}, error) {
		return nil, kit.Fail(http.StatusUnprocessableEntity, map[string]interface{}{"fieldErrors": map[string]string{"email": "invalid"}})
	})

	app.Post(routePath, func(c fiberpkg.Ctx) error {
		return app.handleFormAction(c, route)
	})

	req := httptest.NewRequest(http.MethodPost, routePath, nil)
	req.Header.Set("X-Gospa-Enhance", "1")
	resp, err := app.Fiber.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", resp.StatusCode)
	}

	var payload struct {
		Code       string                         `json:"code"`
		Data       map[string]interface{}         `json:"data"`
		Validation *routing.ActionValidationError `json:"validation"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != "FAIL" {
		t.Fatalf("expected FAIL code, got %s", payload.Code)
	}
	if payload.Validation == nil || payload.Validation.FieldErrors["email"] != "invalid" {
		t.Fatalf("expected validation mapping in response payload")
	}
}

func TestHandleFormAction_EnhancedUnexpectedErrorUsesCanonicalEnvelope(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()

	routePath := fmt.Sprintf("/test-form-action-unexpected-fail-%d", time.Now().UnixNano())
	route := &routing.Route{Path: routePath}

	routing.RegisterAction(routePath, "default", func(_ routing.LoadContext) (interface{}, error) {
		return nil, fmt.Errorf("db timeout")
	})

	app.Post(routePath, func(c fiberpkg.Ctx) error {
		return app.handleFormAction(c, route)
	})

	req := httptest.NewRequest(http.MethodPost, routePath, nil)
	req.Header.Set("X-Gospa-Enhance", "1")
	resp, err := app.Fiber.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", resp.StatusCode)
	}

	var payload struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != "FAIL" {
		t.Fatalf("expected FAIL code, got %s", payload.Code)
	}
	if payload.Error == "" {
		t.Fatalf("expected non-empty error message")
	}
}

func TestHandleFormAction_EnhancedKitError(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()

	routePath := fmt.Sprintf("/test-form-action-kit-error-%d", time.Now().UnixNano())
	route := &routing.Route{Path: routePath}

	routing.RegisterAction(routePath, "default", func(_ routing.LoadContext) (interface{}, error) {
		return nil, kit.Error(http.StatusConflict, map[string]interface{}{"reason": "duplicate"})
	})

	app.Post(routePath, func(c fiberpkg.Ctx) error {
		return app.handleFormAction(c, route)
	})

	req := httptest.NewRequest(http.MethodPost, routePath, nil)
	req.Header.Set("X-Gospa-Enhance", "1")
	resp, err := app.Fiber.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", resp.StatusCode)
	}

	var payload struct {
		Code  string                 `json:"code"`
		Error string                 `json:"error"`
		Data  map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != "FAIL" {
		t.Fatalf("expected FAIL code, got %s", payload.Code)
	}
	if payload.Data["reason"] != "duplicate" {
		t.Fatalf("expected error body payload, got %+v", payload.Data)
	}
}
