package routing

import "testing"

func TestLayoutResolverResolveChain_WithRootAndNestedLayouts(t *testing.T) {
	resolver := NewLayoutResolver()

	resolver.RegisterLayout(&Route{Path: "/", Type: RouteTypeLayout})
	resolver.RegisterLayout(&Route{Path: "/dashboard", Type: RouteTypeLayout})
	resolver.RegisterLayout(&Route{Path: "/dashboard/settings", Type: RouteTypeLayout})

	page := &Route{Path: "/dashboard/settings/profile", Type: RouteTypePage}
	chain := resolver.ResolveChain(page)

	if chain.Page != page {
		t.Fatalf("expected page pointer to be preserved")
	}

	want := []string{"/", "/dashboard", "/dashboard/settings"}
	if len(chain.Layouts) != len(want) {
		t.Fatalf("expected %d layouts, got %d", len(want), len(chain.Layouts))
	}

	for i, expected := range want {
		if chain.Layouts[i].Path != expected {
			t.Fatalf("layout[%d] = %q, want %q", i, chain.Layouts[i].Path, expected)
		}
	}
}

func TestLayoutResolverResolveChain_ErrorFallbackNearestParentThenRoot(t *testing.T) {
	resolver := NewLayoutResolver()

	resolver.RegisterError(&Route{Path: "/", Type: RouteTypeError})
	resolver.RegisterError(&Route{Path: "/dashboard", Type: RouteTypeError})

	chain := resolver.ResolveChain(&Route{Path: "/dashboard/settings/profile", Type: RouteTypePage})
	if chain.Error == nil || chain.Error.Path != "/dashboard" {
		t.Fatalf("expected nearest error route '/dashboard', got %#v", chain.Error)
	}

	chain = resolver.ResolveChain(&Route{Path: "/docs/intro", Type: RouteTypePage})
	if chain.Error == nil || chain.Error.Path != "/" {
		t.Fatalf("expected root error route '/', got %#v", chain.Error)
	}
}

func TestLayoutHelpers_PathAndSegmentUtilities(t *testing.T) {
	tests := []struct {
		name string
		fn   func() bool
	}{
		{
			name: "parentPath trims trailing slash",
			fn:   func() bool { return parentPath("/a/b/") == "/a" },
		},
		{
			name: "parentPath root and empty stay root",
			fn:   func() bool { return parentPath("/") == "/" && parentPath("") == "/" },
		},
		{
			name: "isDirectChild works for root and nested paths",
			fn: func() bool {
				return isDirectChild("/", "/blog") &&
					!isDirectChild("/", "/blog/posts") &&
					isDirectChild("/blog", "/blog/posts") &&
					!isDirectChild("/blog", "/blog/posts/2026")
			},
		},
		{
			name: "isParentPath checks parent relationship",
			fn:   func() bool { return isParentPath("/", "/any/path") && isParentPath("/a", "/a/b") && !isParentPath("/a", "/ab") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.fn() {
				t.Fatalf("assertion failed: %s", tt.name)
			}
		})
	}
}

func TestLayoutTree_NewAndFindLayoutChain(t *testing.T) {
	routes := []*Route{
		{Path: "/blog", Type: RouteTypeLayout},
		{Path: "/blog/admin", Type: RouteTypeLayout},
		{Path: "/docs", Type: RouteTypeLayout},
		{Path: "/docs/v2", Type: RouteTypeLayout},
		{Path: "/not-a-layout", Type: RouteTypePage},
	}

	tree := NewLayoutTree(routes)
	if tree == nil || tree.Root == nil || tree.Root.Route == nil || tree.Root.Route.Path != "/" {
		t.Fatalf("expected root layout node to be initialized")
	}

	if !hasChildPath(tree.Root, "/blog") || !hasChildPath(tree.Root, "/docs") {
		t.Fatalf("expected root to include both /blog and /docs as children")
	}

	chain := tree.FindLayoutChain("/blog/admin/users")
	want := []string{"/blog", "/blog/admin"}
	if len(chain) != len(want) {
		t.Fatalf("expected chain length %d, got %d", len(want), len(chain))
	}
	for i, expected := range want {
		if chain[i].Path != expected {
			t.Fatalf("chain[%d] = %q, want %q", i, chain[i].Path, expected)
		}
	}
}

func TestLayoutDataAndContext_PushPopMergeAndCurrent(t *testing.T) {
	ctx := NewLayoutContext("/dashboard", map[string]string{"id": "123"})
	if ctx.Path != "/dashboard" || ctx.Params["id"] != "123" {
		t.Fatalf("unexpected context initialization: %#v", ctx)
	}

	if ctx.CurrentData() != nil {
		t.Fatalf("expected nil current data for empty context")
	}

	base := NewLayoutData()
	base.Set("theme", "dark")

	override := NewLayoutData()
	override.Set("theme", "light")
	override.Set("title", "Settings")

	base.Merge(override)
	if theme, ok := base.Get("theme"); !ok || theme != "light" {
		t.Fatalf("expected merged theme to be 'light', got %v (ok=%v)", theme, ok)
	}

	ctx.PushData(base)
	if ctx.Depth != 1 || ctx.CurrentData() != base {
		t.Fatalf("expected depth=1 and current data to be base")
	}

	popped := ctx.PopData()
	if popped != base || ctx.Depth != 0 {
		t.Fatalf("expected pop to return base and depth to reset to 0")
	}

	if ctx.PopData() != nil {
		t.Fatalf("expected pop on empty context to return nil")
	}
}

func TestLayoutChainString_IncludesLayoutAndPagePaths(t *testing.T) {
	chain := &LayoutChain{
		Layouts: []*Route{{Path: "/"}, {Path: "/blog"}},
		Page:    &Route{Path: "/blog/post"},
	}

	s := chain.String()
	if s == "" {
		t.Fatalf("expected non-empty string output")
	}
	if !contains(s, "/blog/post") || !contains(s, "/blog") {
		t.Fatalf("expected string to contain layout and page paths, got %q", s)
	}
}

func hasChildPath(node *LayoutNode, path string) bool {
	for _, child := range node.Children {
		if child.Route != nil && child.Route.Path == path {
			return true
		}
	}
	return false
}
