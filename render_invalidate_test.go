package gospa

import "testing"

func TestInvalidateTagAndKey(t *testing.T) {
	app := New(Config{SSGCacheMaxEntries: 10, Prefork: false})
	app.Config.Storage = nil
	defer func() { _ = app.Fiber.Shutdown() }()

	app.storeSsgEntry("/docs/a", []byte("a"), []string{"route:/docs/a", "docs"}, []string{"path:/docs/a", "docs:a"})
	app.storeSsgEntry("/docs/b", []byte("b"), []string{"route:/docs/b", "docs"}, []string{"path:/docs/b", "docs:b"})

	if n := app.InvalidateTag("docs"); n != 2 {
		t.Fatalf("expected 2 invalidations by tag, got %d", n)
	}

	app.storeSsgEntry("/docs/c", []byte("c"), []string{"route:/docs/c", "docs"}, []string{"path:/docs/c", "docs:c"})
	if n := app.InvalidateKey("path:/docs/c"); n != 1 {
		t.Fatalf("expected 1 invalidation by key, got %d", n)
	}
}
