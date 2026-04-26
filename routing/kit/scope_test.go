package kit

import (
	"errors"
	"testing"

	"github.com/aydenstechdungeon/gospa/routing"
)

type testLoadContext struct {
	parent map[string]interface{}
}

func (t *testLoadContext) Param(string) string              { return "" }
func (t *testLoadContext) Params() map[string]string        { return map[string]string{} }
func (t *testLoadContext) Query(string, ...string) string   { return "" }
func (t *testLoadContext) QueryValues() map[string][]string { return map[string][]string{} }
func (t *testLoadContext) Header(string) string             { return "" }
func (t *testLoadContext) Headers() map[string]string       { return map[string]string{} }
func (t *testLoadContext) SetHeader(string, string)         {}
func (t *testLoadContext) Cookie(string) string             { return "" }
func (t *testLoadContext) SetCookie(string, string, int, string, bool, bool) {
}
func (t *testLoadContext) FormValue(string, ...string) string      { return "" }
func (t *testLoadContext) Method() string                          { return "GET" }
func (t *testLoadContext) Path() string                            { return "/" }
func (t *testLoadContext) Local(string) interface{}                { return nil }
func (t *testLoadContext) GospaParentData() map[string]interface{} { return t.parent }
func asLoadContext(v *testLoadContext) routing.LoadContext         { return v }

func TestExecutionScopeDependsAndUntrack(t *testing.T) {
	scope := NewExecutionScope()
	err := scope.Run(func() error {
		Depends("posts:list", "  ", "posts:list")
		return Untrack(func() error {
			Depends("ignored:key")
			return nil
		})
	})
	if err != nil {
		t.Fatalf("scope run failed: %v", err)
	}

	keys := scope.DependsKeys()
	if len(keys) != 1 || keys[0] != "posts:list" {
		t.Fatalf("unexpected dependency keys: %#v", keys)
	}
}

func TestParent(t *testing.T) {
	ctx := &testLoadContext{
		parent: map[string]interface{}{
			"title": "Dashboard",
			"count": float64(2),
		},
	}
	parent, err := Parent[struct {
		Title string `json:"title"`
		Count int    `json:"count"`
	}](asLoadContext(ctx))
	if err != nil {
		t.Fatalf("parent should decode, got err=%v", err)
	}
	if parent.Title != "Dashboard" || parent.Count != 2 {
		t.Fatalf("unexpected parent payload: %#v", parent)
	}
}

func TestParentMissing(t *testing.T) {
	_, err := Parent[map[string]interface{}](asLoadContext(&testLoadContext{}))
	if err == nil {
		t.Fatal("expected missing parent error")
	}
}

func TestHTTPErrorHelpers(t *testing.T) {
	base := Error(422, map[string]string{"reason": "invalid"})
	httpErr, ok := AsError(base)
	if !ok {
		t.Fatal("expected AsError to extract HTTPError")
	}
	if httpErr.Status != 422 {
		t.Fatalf("unexpected status: %d", httpErr.Status)
	}

	wrapped := errors.New("x: " + base.Error())
	if _, ok := AsError(wrapped); ok {
		t.Fatal("expected non-wrapped plain error not to match")
	}
}
