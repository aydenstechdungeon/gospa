package routing

import (
	"reflect"
	"testing"
)

func TestQueryParamsExtended(t *testing.T) {
	qp := NewQueryParams("a=1&b=2&b=3")

	if !qp.Has("a") {
		t.Error("expected to have key 'a'")
	}
	if qp.Get("a") != "1" {
		t.Errorf("expected a=1, got %s", qp.Get("a"))
	}

	vals := qp.GetAll("b")
	if !reflect.DeepEqual(vals, []string{"2", "3"}) {
		t.Errorf("expected b=[2, 3], got %v", vals)
	}

	qp.Set("c", "true")
	if qp.Get("c") != "true" {
		t.Errorf("expected c=true, got %s", qp.Get("c"))
	}

	qp.Add("c", "false")
	if len(qp.GetAll("c")) != 2 {
		t.Errorf("expected len(c)=2, got %d", len(qp.GetAll("c")))
	}

	qp.Del("c")
	if qp.Has("c") {
		t.Error("expected c to be deleted")
	}

	// Int testing
	if val, ok, err := qp.IntOk("a"); !ok || err != nil || val != 1 {
		t.Errorf("expected IntOk a=1, got %d, %v, %v", val, ok, err)
	}
	if val := qp.IntDefault("missing", 99); val != 99 {
		t.Errorf("expected IntDefault missing=99, got %d", val)
	}
	if val, ok, _ := qp.IntOk("missing"); ok {
		t.Errorf("expected missing to not be ok, got %v", val)
	}
	qp.Set("badInt", "abc")
	if _, ok, err := qp.IntOk("badInt"); !ok || err == nil {
		t.Errorf("expected badInt to return err, got ok=%v err=%v", ok, err)
	}
	if val := qp.IntDefault("badInt", 99); val != 99 {
		t.Errorf("expected IntDefault badInt=99, got %d", val)
	}

	// Bool testing
	qp.Set("boolTrue", "true")
	if val, ok, err := qp.BoolOk("boolTrue"); !ok || err != nil || val != true {
		t.Errorf("expected BoolOk boolTrue=true, got %v, %v, %v", val, ok, err)
	}
	if val := qp.BoolDefault("missingBool", false); val != false {
		t.Errorf("expected BoolDefault missingBool=false, got %v", val)
	}
	if val, ok, _ := qp.BoolOk("missingBool"); ok {
		t.Errorf("expected missingBool to not be ok, got %v", val)
	}
	qp.Set("badBool", "notabool")
	if _, ok, err := qp.BoolOk("badBool"); !ok || err == nil {
		t.Errorf("expected badBool to return err, got ok=%v err=%v", ok, err)
	}
	if val := qp.BoolDefault("badBool", true); val != true {
		t.Errorf("expected BoolDefault badBool=true, got %v", val)
	}

	m := qp.ToMap()
	if len(m) == 0 {
		t.Errorf("expected non-empty map")
	}
}

func TestBuildURLAndPathBuilder(t *testing.T) {
	// Using BuildURL manually
	params := Params{"id": "123", "slug": "test"}
	qp := NewQueryParams("sort=desc")
	url1 := BuildURL("/post/:id/:slug", params, qp)
	if url1 != "/post/123/test?sort=desc" {
		t.Errorf("expected BuildURL to be /post/123/test?sort=desc, got %s", url1)
	}

	// Using PathBuilder
	tb := NewPathBuilder("/post/:id/*slug").
		Param("id", "123").
		Param("slug", "category/test").
		Query("sort", "desc").
		QueryAdd("sort", "asc")

	url2 := tb.Build()
	if url2 != "/post/123/category/test?sort=desc&sort=asc" {
		t.Errorf("expected PathBuilder to return /post/123/category/test?sort=desc&sort=asc, got %s", url2)
	}
}
