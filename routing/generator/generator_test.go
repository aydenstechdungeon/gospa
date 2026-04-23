package generator

import (
	"os"
	"path/filepath"
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
