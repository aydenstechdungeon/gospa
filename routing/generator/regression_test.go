package generator

import (
	"strings"
	"testing"
)

func TestGeneratePageCallWithPackage_ParsesStringTypedScalars(t *testing.T) {
	route := RouteInfo{
		ComponentFn: "UserPage",
		PackageName: "routes",
		Params: []FuncParam{
			{Name: "id", Type: "int"},
			{Name: "enabled", Type: "bool"},
			{Name: "score", Type: "float64"},
		},
	}

	generated := generatePageCallWithPackage(route)

	assertContains := func(needle string) {
		t.Helper()
		if !strings.Contains(generated, needle) {
			t.Fatalf("generated call missing %q\n%s", needle, generated)
		}
	}

	assertContains(`props["id"].(string)`)
	assertContains(`strconv.ParseInt(v, 10, 64)`)
	assertContains(`props["enabled"].(string)`)
	assertContains(`strconv.ParseBool(v)`)
	assertContains(`props["score"].(string)`)
	assertContains(`strconv.ParseFloat(v, 64)`)
}

func TestGenerateCode_AddsStrconvImportWhenScalarParsingNeeded(t *testing.T) {
	code, err := generateCode([]RouteInfo{
		{
			URLPath:     "/users/:id",
			ComponentFn: "UserPage",
			PackageName: "routes",
			RouteParams: []string{"id"},
			Params: []FuncParam{
				{Name: "id", Type: "int"},
			},
		},
	}, "routes", false)
	if err != nil {
		t.Fatalf("generateCode failed: %v", err)
	}

	if !strings.Contains(code, `"strconv"`) {
		t.Fatalf("expected generated code to import strconv\n%s", code)
	}
}

func TestRouteTypeScriptGenerator_UsesBoundedMatchCache(t *testing.T) {
	g := NewRouteTypeScriptGenerator(nil, "example.com/project")
	var sb strings.Builder

	g.generateRouteHelpers(&sb)
	output := sb.String()

	if !strings.Contains(output, `const ROUTE_MATCH_CACHE_MAX = 1000;`) {
		t.Fatalf("expected route cache max constant in helpers\n%s", output)
	}
	if !strings.Contains(output, `function setRouteMatchCache(cacheKey: string, value: boolean): void`) {
		t.Fatalf("expected bounded cache setter in helpers\n%s", output)
	}
	if strings.Contains(output, `routeMatchCache.set(cacheKey, true);`) {
		t.Fatalf("expected direct map writes to be replaced by bounded setter\n%s", output)
	}
}

func TestRouteTypeScriptGenerator_GetLinkPropsQueryDetection(t *testing.T) {
	g := NewRouteTypeScriptGenerator(nil, "example.com/project")
	var sb strings.Builder

	g.generateNavigationHelpers(&sb)
	output := sb.String()

	if !strings.Contains(output, `const hasParams = path.includes(':');`) {
		t.Fatalf("expected hasParams-based detection in getLinkProps\n%s", output)
	}
	if strings.Contains(output, `!('toString' in a)`) {
		t.Fatalf("legacy toString-based query detection should not exist\n%s", output)
	}
}
