package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"page", "Page"},
		{"layout", "Layout"},
		{"blog_post", "BlogPost"},
		{"user-profile", "UserProfile"},
		{"test.route", "TestRoute"},
	}

	for _, tt := range tests {
		if got := toPascalCase(tt.input); got != tt.expected {
			t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestFilePathToURLPath(t *testing.T) {
	tests := []struct {
		dir      string
		filename string
		expected string
	}{
		{".", "page.templ", "/"},
		{".", "+page.templ", "/"},
		{"blog", "page.templ", "/blog"},
		{"blog", "+page.templ", "/blog"},
		{"blog", "+layout.templ", "/blog"},
		{"blog", "+error.templ", "/blog"},
		{"blog", "_error.templ", "/blog"},
		{"blog", "page.gospa", "/blog"},
		{"blog/_id", "page.templ", "/blog/:id"},
		{"(auth)/login", "page.templ", "/login"},
		{"blog", "post.templ", "/blog/post"},
		{"users/_userId/posts/_postId", "page.templ", "/users/:userId/posts/:postId"},
	}

	for _, tt := range tests {
		if got := filePathToURLPath(tt.dir, tt.filename); got != tt.expected {
			t.Errorf("filePathToURLPath(%q, %q) = %q, want %q", tt.dir, tt.filename, got, tt.expected)
		}
	}
}

func TestParseRoute(t *testing.T) {
	// Note: parseRoute tries to read _templ.go files, so we test the basic structure here
	// without relying on existing files for now, or we could mock OS but let's test the URL part.
	route := parseRoute("blog/_id/page.templ", ".")
	if route.URLPath != "/blog/:id" {
		t.Errorf("expected URLPath /blog/:id, got %s", route.URLPath)
	}
	if len(route.RouteParams) != 1 || route.RouteParams[0] != "id" {
		t.Errorf("expected RouteParams [id], got %v", route.RouteParams)
	}
}

func TestParseRoute_ErrorBoundary(t *testing.T) {
	route := parseRoute("+error.templ", ".")
	if !route.IsError {
		t.Fatal("expected +error.templ to be classified as IsError=true")
	}
	if route.URLPath != "/" {
		t.Fatalf("expected root +error.templ URLPath '/', got %q", route.URLPath)
	}
}

func TestParseRoute_UnderscoreErrorBoundary(t *testing.T) {
	route := parseRoute("_error.gospa", ".")
	if !route.IsError {
		t.Fatal("expected _error.gospa to be classified as IsError=true")
	}
	if route.URLPath != "/" {
		t.Fatalf("expected root _error.gospa URLPath '/', got %q", route.URLPath)
	}
}

func TestScanRoutes_SeparatesPageAndErrorBoundaries(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "+page.templ"), []byte("package routes"), 0600); err != nil {
		t.Fatalf("write +page.templ: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "+error.templ"), []byte("package routes"), 0600); err != nil {
		t.Fatalf("write +error.templ: %v", err)
	}

	routes, err := scanRoutes(tmpDir)
	if err != nil {
		t.Fatalf("scanRoutes error: %v", err)
	}

	var pageCount, errorCount int
	for _, rt := range routes {
		if rt.IsError {
			errorCount++
		} else if !rt.IsLayout {
			pageCount++
		}
	}
	if pageCount != 1 {
		t.Fatalf("expected 1 page route, got %d", pageCount)
	}
	if errorCount != 1 {
		t.Fatalf("expected 1 error boundary route, got %d", errorCount)
	}
}

func TestScanRoutes_IncludesGospaAndUnderscoreError(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "page.gospa"), []byte("dummy"), 0600); err != nil {
		t.Fatalf("write page.gospa: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "_error.gospa"), []byte("dummy"), 0600); err != nil {
		t.Fatalf("write _error.gospa: %v", err)
	}

	routes, err := scanRoutes(tmpDir)
	if err != nil {
		t.Fatalf("scanRoutes error: %v", err)
	}

	var hasPage, hasError bool
	for _, rt := range routes {
		if rt.URLPath == "/" && !rt.IsError {
			hasPage = true
		}
		if rt.URLPath == "/" && rt.IsError {
			hasError = true
		}
	}
	if !hasPage {
		t.Fatal("expected page.gospa route to be discovered")
	}
	if !hasError {
		t.Fatal("expected _error.gospa error boundary to be discovered")
	}
}

func TestGetActions_DetectsNamedActionExports(t *testing.T) {
	tmpDir := t.TempDir()
	serverFile := filepath.Join(tmpDir, "+page.server.go")
	content := `package routes
import "github.com/aydenstechdungeon/gospa/routing"

func ActionDefault(c routing.LoadContext) (interface{}, error) { return nil, nil }
func ActionPublish(c routing.LoadContext) (interface{}, error) { return nil, nil }
`
	if err := os.WriteFile(serverFile, []byte(content), 0600); err != nil {
		t.Fatalf("write server file: %v", err)
	}

	actions, funcs := getActions(serverFile)
	if len(actions) != 2 {
		t.Fatalf("expected two actions, got %v", actions)
	}
	if funcs["default"] != "ActionDefault" {
		t.Fatalf("expected default action func mapping, got %v", funcs)
	}
	if funcs["publish"] != "ActionPublish" {
		t.Fatalf("expected publish action func mapping, got %v", funcs)
	}
}

func TestScanRoutes_ConflictsWhenModuleAndServerFileBothExist(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "posts"), 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	pagePath := filepath.Join(tmpDir, "posts", "+page.gospa")
	pageContent := `<script context="module" lang="go">
func Load(c routing.LoadContext) (map[string]interface{}, error) { return map[string]interface{}{}, nil }
</script>
<template><div>ok</div></template>`
	if err := os.WriteFile(pagePath, []byte(pageContent), 0600); err != nil {
		t.Fatalf("write page.gospa: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "posts", "+page.server.go"), []byte("package routes"), 0600); err != nil {
		t.Fatalf("write +page.server.go: %v", err)
	}

	_, err := scanRoutes(tmpDir)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !strings.Contains(err.Error(), "both") {
		t.Fatalf("expected conflict message, got %v", err)
	}
}

func TestScanRoutes_PrefersSourcePageGospaOverGeneratedTempl(t *testing.T) {
	tmpDir := t.TempDir()
	routeDir := filepath.Join(tmpDir, "docs", "gospasfc", "test")
	if err := os.MkdirAll(routeDir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(routeDir, "generated_page.templ"), []byte("package test"), 0600); err != nil {
		t.Fatalf("write generated_page.templ: %v", err)
	}

	pageContent := `<script context="module" lang="go">
func ActionDefault(c routing.LoadContext) (interface{}, error) { return nil, nil }
</script>
<template><div>ok</div></template>`
	if err := os.WriteFile(filepath.Join(routeDir, "page.gospa"), []byte(pageContent), 0600); err != nil {
		t.Fatalf("write page.gospa: %v", err)
	}

	routes, err := scanRoutes(tmpDir)
	if err != nil {
		t.Fatalf("scanRoutes error: %v", err)
	}

	var target *RouteInfo
	for i := range routes {
		if routes[i].URLPath == "/docs/gospasfc/test" && !routes[i].IsLayout && !routes[i].IsError {
			target = &routes[i]
			break
		}
	}
	if target == nil {
		t.Fatal("target route not found")
	}

	if filepath.Base(target.FilePath) != "page.gospa" {
		t.Fatalf("expected page.gospa to win, got %s", target.FilePath)
	}
	if !target.HasActions {
		t.Fatalf("expected actions discovered from page.gospa module script")
	}
}
