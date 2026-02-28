package routing

import (
	"encoding/json"
	"testing"
)

func TestParamsGet(t *testing.T) {
	p := Params{"id": "123", "name": "test"}

	if v := p.Get("id"); v != "123" {
		t.Errorf("Expected '123', got %s", v)
	}

	if v := p.Get("name"); v != "test" {
		t.Errorf("Expected 'test', got %s", v)
	}

	if v := p.Get("nonexistent"); v != "" {
		t.Errorf("Expected empty string for nonexistent key, got %s", v)
	}
}

func TestParamsGetDefault(t *testing.T) {
	p := Params{"id": "123"}

	if v := p.GetDefault("id", "default"); v != "123" {
		t.Errorf("Expected '123', got %s", v)
	}

	if v := p.GetDefault("nonexistent", "default"); v != "default" {
		t.Errorf("Expected 'default', got %s", v)
	}
}

func TestParamsHas(t *testing.T) {
	p := Params{"id": "123"}

	if !p.Has("id") {
		t.Error("Expected Has('id') to be true")
	}

	if p.Has("nonexistent") {
		t.Error("Expected Has('nonexistent') to be false")
	}
}

func TestParamsSet(t *testing.T) {
	p := make(Params)
	p.Set("id", "123")

	if v := p.Get("id"); v != "123" {
		t.Errorf("Expected '123', got %s", v)
	}
}

func TestParamsDelete(t *testing.T) {
	p := Params{"id": "123"}
	p.Delete("id")

	if p.Has("id") {
		t.Error("Expected key 'id' to be deleted")
	}
}

func TestParamsClone(t *testing.T) {
	p := Params{"id": "123"}
	clone := p.Clone()

	// Modify original
	p.Set("id", "456")

	if clone.Get("id") != "123" {
		t.Errorf("Expected clone to have original value '123', got %s", clone.Get("id"))
	}
}

func TestParamsMerge(t *testing.T) {
	p1 := Params{"id": "123"}
	p2 := Params{"name": "test"}

	p1.Merge(p2)

	if !p1.Has("id") || !p1.Has("name") {
		t.Error("Expected merged params to have both keys")
	}

	if p1.Get("name") != "test" {
		t.Errorf("Expected 'test', got %s", p1.Get("name"))
	}
}

func TestParamsInt(t *testing.T) {
	p := Params{"count": "42", "invalid": "notanumber"}

	if v, err := p.Int("count"); err != nil || v != 42 {
		t.Errorf("Expected 42, got %d (err: %v)", v, err)
	}

	if _, err := p.Int("nonexistent"); err == nil {
		t.Error("Expected error for nonexistent key")
	}

	if _, err := p.Int("invalid"); err == nil {
		t.Error("Expected error for invalid number")
	}
}

func TestParamsIntOk(t *testing.T) {
	p := Params{"count": "42"}

	v, ok, err := p.IntOk("count")
	if !ok || err != nil || v != 42 {
		t.Errorf("Expected (42, true, nil), got (%d, %v, %v)", v, ok, err)
	}

	v, ok, err = p.IntOk("nonexistent")
	if ok || err != nil || v != 0 {
		t.Errorf("Expected (0, false, nil), got (%d, %v, %v)", v, ok, err)
	}
}

func TestParamsIntDefault(t *testing.T) {
	p := Params{"count": "42", "invalid": "notanumber"}

	if v := p.IntDefault("count", 0); v != 42 {
		t.Errorf("Expected 42, got %d", v)
	}

	if v := p.IntDefault("nonexistent", 100); v != 100 {
		t.Errorf("Expected default 100, got %d", v)
	}

	if v := p.IntDefault("invalid", 200); v != 200 {
		t.Errorf("Expected default 200 for invalid, got %d", v)
	}
}

func TestParamsFloat64(t *testing.T) {
	p := Params{"price": "19.99"}

	if v, err := p.Float64("price"); err != nil || v != 19.99 {
		t.Errorf("Expected 19.99, got %f (err: %v)", v, err)
	}
}

func TestParamsBool(t *testing.T) {
	p := Params{"active": "true", "disabled": "false"}

	if v, err := p.Bool("active"); err != nil || !v {
		t.Errorf("Expected true, got %v (err: %v)", v, err)
	}

	if v, err := p.Bool("disabled"); err != nil || v {
		t.Errorf("Expected false, got %v (err: %v)", v, err)
	}
}

func TestParamsSlice(t *testing.T) {
	p := Params{"path": "a/b/c"}

	slice := p.Slice("path")
	expected := []string{"a", "b", "c"}

	if len(slice) != len(expected) {
		t.Errorf("Expected slice length %d, got %d", len(expected), len(slice))
	}

	for i, v := range expected {
		if slice[i] != v {
			t.Errorf("Expected slice[%d] = %s, got %s", i, v, slice[i])
		}
	}
}

func TestParamsToJSON(t *testing.T) {
	p := Params{"id": "123", "name": "test"}

	data, err := p.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result["id"] != "123" || result["name"] != "test" {
		t.Errorf("Unexpected JSON result: %v", result)
	}
}

func TestParamsFromJSON(t *testing.T) {
	json := `{"id":"123","name":"test"}`

	p, err := ParamsFromJSON([]byte(json))
	if err != nil {
		t.Fatalf("ParamsFromJSON failed: %v", err)
	}

	if p.Get("id") != "123" {
		t.Errorf("Expected id '123', got %s", p.Get("id"))
	}

	if p.Get("name") != "test" {
		t.Errorf("Expected name 'test', got %s", p.Get("name"))
	}
}

func TestNewParamExtractor(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		match   bool
		params  Params
	}{
		{
			pattern: "/users/:id",
			path:    "/users/123",
			match:   true,
			params:  Params{"id": "123"},
		},
		{
			pattern: "/users/:id/posts/:postId",
			path:    "/users/123/posts/456",
			match:   true,
			params:  Params{"id": "123", "postId": "456"},
		},
		{
			pattern: "/files/*path",
			path:    "/files/a/b/c",
			match:   true,
			params:  Params{"path": "a/b/c"},
		},
		{
			pattern: "/users/:id",
			path:    "/posts/123",
			match:   false,
		},
	}

	for _, tt := range tests {
		pe := NewParamExtractor(tt.pattern)

		params, matched := pe.Extract(tt.path)
		if matched != tt.match {
			t.Errorf("Pattern %s, path %s: expected match=%v, got %v", tt.pattern, tt.path, tt.match, matched)
			continue
		}

		if !tt.match {
			continue
		}

		for k, v := range tt.params {
			if params.Get(k) != v {
				t.Errorf("Pattern %s, path %s: expected param %s=%s, got %s", tt.pattern, tt.path, k, v, params.Get(k))
			}
		}
	}
}

func TestParamExtractorMatch(t *testing.T) {
	pe := NewParamExtractor("/users/:id")

	if !pe.Match("/users/123") {
		t.Error("Expected Match to return true for /users/123")
	}

	if pe.Match("/posts/123") {
		t.Error("Expected Match to return false for /posts/123")
	}
}

func TestParamExtractorCustomRegex(t *testing.T) {
	pe := NewParamExtractor("/users/:id<[0-9]+>")

	params, matched := pe.Extract("/users/123")
	if !matched {
		t.Error("Expected match for numeric ID")
	}
	if params.Get("id") != "123" {
		t.Errorf("Expected id '123', got %s", params.Get("id"))
	}

	_, matched = pe.Extract("/users/abc")
	if matched {
		t.Error("Expected no match for non-numeric ID with custom regex")
	}
}

func TestQueryParams(t *testing.T) {
	qp := NewQueryParams("page=1&limit=10&sort=name")

	if v := qp.Get("page"); v != "1" {
		t.Errorf("Expected page '1', got %s", v)
	}

	if v := qp.Get("limit"); v != "10" {
		t.Errorf("Expected limit '10', got %s", v)
	}

	if v := qp.Get("nonexistent"); v != "" {
		t.Errorf("Expected empty string for nonexistent key, got %s", v)
	}
}

func TestQueryParamsInt(t *testing.T) {
	qp := NewQueryParams("page=1&invalid=abc")

	if v, err := qp.Int("page"); err != nil || v != 1 {
		t.Errorf("Expected 1, got %d (err: %v)", v, err)
	}

	if _, err := qp.Int("invalid"); err == nil {
		t.Error("Expected error for invalid number")
	}
}

func TestQueryParamsBool(t *testing.T) {
	qp := NewQueryParams("active=true&disabled=false")

	if v, err := qp.Bool("active"); err != nil || !v {
		t.Errorf("Expected true, got %v (err: %v)", v, err)
	}

	if v, err := qp.Bool("disabled"); err != nil || v {
		t.Errorf("Expected false, got %v (err: %v)", v, err)
	}
}

func TestPathBuilder(t *testing.T) {
	url := NewPathBuilder("/users/:id/posts/:postId").
		Param("id", "123").
		Param("postId", "456").
		Query("page", "1").
		Build()

	expected := "/users/123/posts/456?page=1"
	if url != expected {
		t.Errorf("Expected %s, got %s", expected, url)
	}
}

func TestBuildURL(t *testing.T) {
	params := Params{"id": "123", "postId": "456"}
	query := NewQueryParams("page=1")

	url := BuildURL("/users/:id/posts/:postId", params, query)

	expected := "/users/123/posts/456?page=1"
	if url != expected {
		t.Errorf("Expected %s, got %s", expected, url)
	}
}

func TestValidateParams(t *testing.T) {
	route := &Route{
		Path:   "/users/:id/posts/:postId",
		Params: []string{"id", "postId"},
	}

	// Valid params
	validParams := Params{"id": "123", "postId": "456"}
	if err := ValidateParams(route, validParams); err != nil {
		t.Errorf("Expected no error for valid params, got %v", err)
	}

	// Missing params
	invalidParams := Params{"id": "123"}
	if err := ValidateParams(route, invalidParams); err == nil {
		t.Error("Expected error for missing params")
	}
}
