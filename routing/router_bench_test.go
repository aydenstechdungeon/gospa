package routing

import (
	"fmt"
	"testing"
)

func BenchmarkRouterMatch_Static(b *testing.B) {
	r := NewRouter(makeFS(
		"page.templ",
		"about/page.templ",
		"docs/page.templ",
	))
	if err := r.Scan(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		route, params := r.Match("/about")
		if route == nil || route.Path != "/about" || len(params) != 0 {
			b.Fatalf("unexpected match result: route=%v params=%v", route, params)
		}
	}
}

func BenchmarkRouterMatch_Dynamic(b *testing.B) {
	paths := []string{"page.templ", "users/page.templ"}
	for i := 0; i < 256; i++ {
		paths = append(paths, fmt.Sprintf("users/[id]/orders/%d/page.templ", i))
	}
	r := NewRouter(makeFS(paths...))
	if err := r.Scan(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		route, params := r.Match("/users/42/orders/120")
		if route == nil || params["id"] != "42" {
			b.Fatalf("unexpected match result: route=%v params=%v", route, params)
		}
	}
}
