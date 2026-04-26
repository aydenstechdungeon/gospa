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

func TestInvalidateAll(t *testing.T) {
	app := New(Config{SSGCacheMaxEntries: 10, Prefork: false})
	app.Config.Storage = nil
	defer func() { _ = app.Fiber.Shutdown() }()

	app.storeSsgEntry("/docs/a", []byte("a"), []string{"docs"}, []string{"path:/docs/a"})
	app.storeSsgEntry("/docs/b", []byte("b"), []string{"docs"}, []string{"path:/docs/b"})
	app.storePprShell("/ppr/a", []byte("shell"), nil, nil)

	if n := app.InvalidateAll(); n != 3 {
		t.Fatalf("expected 3 invalidations, got %d", n)
	}
	if len(app.ssgCache) != 0 {
		t.Fatalf("expected empty ssg cache after InvalidateAll")
	}
	if len(app.pprShellCache) != 0 {
		t.Fatalf("expected empty ppr shell cache after InvalidateAll")
	}
}

func TestInvalidateDependencyTagAndKey(t *testing.T) {
	app := New(Config{SSGCacheMaxEntries: 10, Prefork: false})
	app.Config.Storage = nil
	defer func() { _ = app.Fiber.Shutdown() }()

	deps := []string{"posts:list"}
	tags := append(app.defaultCacheTags("/deps", "ssr"), dependencyTags(deps)...)
	keys := append(app.defaultCacheKeys("/deps"), dependencyKeys(deps)...)
	app.storeSsgEntry("/deps", []byte("deps"), tags, keys)

	if n := app.InvalidateTag("dep:posts:list"); n != 1 {
		t.Fatalf("expected 1 invalidation by dep tag, got %d", n)
	}

	app.storeSsgEntry("/deps2", []byte("deps"), tags, keys)
	if n := app.InvalidateKey("dep:posts:list"); n != 1 {
		t.Fatalf("expected 1 invalidation by dep key, got %d", n)
	}
}
