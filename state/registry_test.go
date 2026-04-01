package state

import (
	"context"
	"strings"
	"testing"
)

func TestRegistry_Registration(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}

	r.Register("island1", map[string]interface{}{"prop": 1}, map[string]interface{}{"stateVar": "hello"})

	data := r.GetData()
	if len(data) != 1 {
		t.Fatalf("expected 1 island data, got %d", len(data))
	}

	if data[0].ID != "island1" {
		t.Errorf("expected ID 'island1', got %s", data[0].ID)
	}

	if val, ok := data[0].Props["prop"]; !ok || val != 1 {
		t.Errorf("expected prop to equal 1, got %v", val)
	}

	if val, ok := data[0].State["stateVar"]; !ok || val != "hello" {
		t.Errorf("expected stateVar to equal 'hello', got %v", val)
	}
}

func TestRegistry_FromContext(t *testing.T) {
	ctx := context.Background()

	// Should return nil when not present
	if r := FromContext(ctx); r != nil {
		t.Errorf("expected nil registry initially, got %v", r)
	}

	r := NewRegistry()
	ctx = context.WithValue(ctx, RegistryContextKey, r)

	if got := FromContext(ctx); got != r {
		t.Errorf("expected to retrieve registry from ctx, got %v", got)
	}
}

func TestRegistry_GetDataJSON(t *testing.T) {
	r := NewRegistry()
	r.Register("island1", map[string]interface{}{"prop": 1}, nil)
	r.Register("island2", nil, map[string]interface{}{"state": 2})

	jsonStr := r.GetDataJSON()

	if !strings.Contains(jsonStr, `"id":"island1"`) {
		t.Errorf("expected island1 in JSON: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"id":"island2"`) {
		t.Errorf("expected island2 in JSON: %s", jsonStr)
	}
}

func TestRegistry_GetRegistryDataJSON_FromContext(t *testing.T) {
	ctx := context.Background()
	r := NewRegistry()
	r.Register("test", nil, nil)
	ctx = context.WithValue(ctx, RegistryContextKey, r)

	jsonStr := GetRegistryDataJSON(ctx)
	if !strings.Contains(jsonStr, `"id":"test"`) {
		t.Errorf("expected test id in context JSON, got %s", jsonStr)
	}

	// Test missing context fallback to "[]"
	emptyJSON := GetRegistryDataJSON(context.Background())
	if emptyJSON != "[]" {
		t.Errorf("expected '[]' for missing context registry, got %s", emptyJSON)
	}
}
