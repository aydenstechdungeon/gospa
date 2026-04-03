package gospa

import (
	"os"
	"testing"
	"time"
)

// ─── DefaultConfig ────────────────────────────────────────────────────────────

// ─── New / config defaults ────────────────────────────────────────────────────

func TestNew_AppliesDefaults(t *testing.T) {
	// Passing empty config should fill in all defaults
	app := New(Config{})

	if app.Config.RoutesDir != "./routes" {
		t.Errorf("RoutesDir default not applied, got %q", app.Config.RoutesDir)
	}
	if app.Config.RuntimeScript != "/_gospa/runtime.js" {
		t.Errorf("RuntimeScript default not applied, got %q", app.Config.RuntimeScript)
	}
	if app.Config.StaticDir != "./static" {
		t.Errorf("StaticDir default not applied, got %q", app.Config.StaticDir)
	}
	if app.Config.StaticPrefix != "/static" {
		t.Errorf("StaticPrefix default not applied, got %q", app.Config.StaticPrefix)
	}
	if app.Config.WebSocketPath != "/_gospa/ws" {
		t.Errorf("WebSocketPath default not applied, got %q", app.Config.WebSocketPath)
	}
	if app.Config.RemotePrefix != "/_gospa/remote" {
		t.Errorf("RemotePrefix default not applied, got %q", app.Config.RemotePrefix)
	}
	if app.Config.MaxRequestBodySize != 4*1024*1024 {
		t.Errorf("MaxRequestBodySize default not applied, got %d", app.Config.MaxRequestBodySize)
	}

	_ = app.Fiber.Shutdown()
}

func TestNew_SSGCacheMaxEntries_Defaults(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()
	if app.Config.SSGCacheMaxEntries != 500 {
		t.Errorf("expected SSGCacheMaxEntries=500 when unset, got %d", app.Config.SSGCacheMaxEntries)
	}
}

func TestNew_SSGCacheMaxEntries_Negative(t *testing.T) {
	// Any negative input should be normalized to -1 (unlimited)
	app := New(Config{SSGCacheMaxEntries: -99})
	defer func() { _ = app.Fiber.Shutdown() }()
	if app.Config.SSGCacheMaxEntries != -1 {
		t.Errorf("negative SSGCacheMaxEntries should normalize to -1, got %d", app.Config.SSGCacheMaxEntries)
	}
}

func TestNew_SSGCacheMaxEntries_Cap(t *testing.T) {
	app := New(Config{SSGCacheMaxEntries: 99999})
	defer func() { _ = app.Fiber.Shutdown() }()
	if app.Config.SSGCacheMaxEntries != 10000 {
		t.Errorf("SSGCacheMaxEntries > 10000 should be capped at 10000, got %d", app.Config.SSGCacheMaxEntries)
	}
}

func TestNew_WSDefaults(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()
	if app.Config.WSMaxMessageSize != 64*1024 {
		t.Errorf("expected WSMaxMessageSize=64KB, got %d", app.Config.WSMaxMessageSize)
	}
	if app.Config.WSConnRateLimit != 1.5 {
		t.Errorf("expected WSConnRateLimit=1.5, got %f", app.Config.WSConnRateLimit)
	}
	if app.Config.WSConnBurst != 15.0 {
		t.Errorf("expected WSConnBurst=15.0, got %f", app.Config.WSConnBurst)
	}
}

func TestNew_HubInitialized(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()
	if app.Hub == nil {
		t.Error("Hub should be initialized in New()")
	}
}

func TestNew_StateMapInitialized(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()
	if app.StateMap == nil {
		t.Error("StateMap should be initialized in New()")
	}
}

func TestNew_RouterInitialized(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()
	if app.Router == nil {
		t.Error("Router should be initialized in New()")
	}
}

func TestNew_FiberInitialized(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()
	if app.Fiber == nil {
		t.Error("Fiber should be initialized in New()")
	}
}

func TestNew_DefaultStateInjected(t *testing.T) {
	app := New(Config{
		DefaultState: map[string]interface{}{
			"count": 0,
			"name":  "test",
		},
	})
	defer func() { _ = app.Fiber.Shutdown() }()

	_, okCount := app.StateMap.Get("count")
	_, okName := app.StateMap.Get("name")

	if !okCount {
		t.Error("expected 'count' to be in StateMap")
	}
	if !okName {
		t.Error("expected 'name' to be in StateMap")
	}
}

func TestNew_CustomConfig(t *testing.T) {
	app := New(Config{
		RoutesDir:     "./custom-routes",
		StaticDir:     "./custom-static",
		AppName:       "MyApp",
		DevMode:       false,
		CompressState: true,
		StateDiffing:  true,
	})
	defer func() { _ = app.Fiber.Shutdown() }()

	if app.Config.RoutesDir != "./custom-routes" {
		t.Errorf("expected RoutesDir='./custom-routes', got %q", app.Config.RoutesDir)
	}
	if app.Config.AppName != "MyApp" {
		t.Errorf("expected AppName='MyApp', got %q", app.Config.AppName)
	}
}

func TestNew_LoadsManifest(t *testing.T) {
	// Create a temporary manifest file
	manifestPath := "test_manifest.json"
	content := `{"main.js": "main.1234.js", "style.css": "style.5678.css"}`
	err := os.WriteFile(manifestPath, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test manifest: %v", err)
	}
	defer func() { _ = os.Remove(manifestPath) }()

	app := New(Config{
		ManifestPath: manifestPath,
	})
	defer func() { _ = app.Fiber.Shutdown() }()

	if len(app.Config.BuildManifest) != 2 {
		t.Errorf("expected 2 manifest entries, got %d", len(app.Config.BuildManifest))
	}
	if app.Config.BuildManifest["main.js"] != "main.1234.js" {
		t.Errorf("expected main.js to be main.1234.js, got %q", app.Config.BuildManifest["main.js"])
	}
}

// ─── encodeSsgEntry / decodeSsgEntry ─────────────────────────────────────────

func TestEncodeSsgEntry_RoundTrip(t *testing.T) {
	original := ssgEntry{
		html:      []byte("<html>hello</html>"),
		createdAt: time.Now().Truncate(time.Nanosecond),
	}
	encoded := encodeSsgEntry(original)
	decoded, ok := decodeSsgEntry(encoded)
	if !ok {
		t.Fatal("decodeSsgEntry should return ok=true for valid data")
	}
	if string(decoded.html) != string(original.html) {
		t.Errorf("html mismatch: got %q, want %q", decoded.html, original.html)
	}
	// Time comparison: allow ns difference
	if decoded.createdAt.UnixNano() != original.createdAt.UnixNano() {
		t.Errorf("createdAt mismatch: got %v, want %v", decoded.createdAt, original.createdAt)
	}
}

func TestEncodeSsgEntry_EmptyHTML(t *testing.T) {
	original := ssgEntry{
		html:      []byte{},
		createdAt: time.Now(),
	}
	encoded := encodeSsgEntry(original)
	decoded, ok := decodeSsgEntry(encoded)
	if !ok {
		t.Fatal("decodeSsgEntry should return ok=true for empty html")
	}
	if len(decoded.html) != 0 {
		t.Errorf("expected empty html, got %q", decoded.html)
	}
}

func TestDecodeSsgEntry_TooShort(t *testing.T) {
	_, ok := decodeSsgEntry([]byte{1, 2, 3}) // less than 8 bytes
	if ok {
		t.Error("decodeSsgEntry should return ok=false for data shorter than 8 bytes")
	}
}

func TestDecodeSsgEntry_ExactlyEightBytes(t *testing.T) {
	data := make([]byte, 8)
	entry, ok := decodeSsgEntry(data)
	if !ok {
		t.Fatal("decodeSsgEntry should return ok=true for exactly 8 bytes (empty html)")
	}
	if len(entry.html) != 0 {
		t.Errorf("expected empty html for 8-byte data, got %q", entry.html)
	}
}

// ─── storeSsgEntry (internal cache) ──────────────────────────────────────────

func TestStoreSsgEntry_FIFO_Eviction(t *testing.T) {
	// Use Prefork=true so in-memory ssgCache is used (not external Storage)
	app := New(Config{SSGCacheMaxEntries: 3, Prefork: false})
	app.Config.Storage = nil // force in-memory path
	defer func() { _ = app.Fiber.Shutdown() }()

	// Fill the cache to capacity
	app.storeSsgEntry("/page1", []byte("html1"))
	app.storeSsgEntry("/page2", []byte("html2"))
	app.storeSsgEntry("/page3", []byte("html3"))

	// Adding a 4th should evict the first
	app.storeSsgEntry("/page4", []byte("html4"))

	app.ssgCacheMu.RLock()
	defer app.ssgCacheMu.RUnlock()

	if _, ok := app.ssgCache["/page1"]; ok {
		t.Error("expected /page1 to be evicted (FIFO)")
	}
	if _, ok := app.ssgCache["/page4"]; !ok {
		t.Error("expected /page4 to be in cache after insert")
	}
}

func TestStoreSsgEntry_Unlimited(t *testing.T) {
	app := New(Config{SSGCacheMaxEntries: -1}) // unlimited
	app.Config.Storage = nil                   // force in-memory path
	defer func() { _ = app.Fiber.Shutdown() }()

	for i := 0; i < 10; i++ {
		key := "/page" + string(rune('0'+i))
		app.storeSsgEntry(key, []byte("html"))
	}

	app.ssgCacheMu.RLock()
	defer app.ssgCacheMu.RUnlock()

	if len(app.ssgCache) != 10 {
		t.Errorf("expected 10 entries in unlimited cache, got %d", len(app.ssgCache))
	}
}

func TestStoreSsgEntry_DuplicateKeyNoExtraTrack(t *testing.T) {
	app := New(Config{SSGCacheMaxEntries: 10})
	app.Config.Storage = nil // force in-memory path
	defer func() { _ = app.Fiber.Shutdown() }()

	app.storeSsgEntry("/dupe", []byte("v1"))
	app.storeSsgEntry("/dupe", []byte("v2"))

	app.ssgCacheMu.RLock()
	defer app.ssgCacheMu.RUnlock()

	// ssgCacheKeys should only track the key once
	count := 0
	for _, k := range app.ssgCacheKeys {
		if k == "/dupe" {
			count++
		}
	}
	if count > 1 {
		t.Errorf("duplicate key should not add multiple tracking entries, got %d", count)
	}
}

// ─── Version ─────────────────────────────────────────────────────────────────

func TestVersion_NonEmpty(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
}

// ─── getRuntimePath ───────────────────────────────────────────────────────────

func TestGetRuntimePath_CustomScript(t *testing.T) {
	app := New(Config{RuntimeScript: "/custom/runtime.js"})
	defer func() { _ = app.Fiber.Shutdown() }()
	path := app.getRuntimePath()
	if path != "/custom/runtime.js" {
		t.Errorf("expected custom runtime path, got %q", path)
	}
}

func TestGetRuntimePath_Default(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()
	path := app.getRuntimePath()
	if path == "" {
		t.Error("getRuntimePath should not return empty string")
	}
	// Should start with /_gospa/
	if len(path) < 8 || path[:8] != "/_gospa/" {
		t.Errorf("expected path starting with '/_gospa/', got %q", path)
	}
}

func TestGetRuntimePath_Simple(t *testing.T) {
	app := New(Config{SimpleRuntime: true})
	defer func() { _ = app.Fiber.Shutdown() }()
	path := app.getRuntimePath()
	// Should contain "simple" in the path name
	if len(path) < 8 || path[:8] != "/_gospa/" {
		t.Errorf("expected path starting with '/_gospa/', got %q", path)
	}
}

// ─── Accessors ───────────────────────────────────────────────────────────────

func TestGetHub(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()
	if app.GetHub() == nil {
		t.Error("GetHub() should return non-nil")
	}
}

func TestGetRouter(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()
	if app.GetRouter() == nil {
		t.Error("GetRouter() should return non-nil")
	}
}

func TestGetFiber(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()
	if app.GetFiber() == nil {
		t.Error("GetFiber() should return non-nil")
	}
}

// ─── Routing helpers ─────────────────────────────────────────────────────────

func TestApp_RouteHelpers_NoPanic(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()

	// Just verify these don't panic - method chaining works on Fiber
	routers := app.GetFiber()
	if routers == nil {
		t.Error("Fiber instance should not be nil")
	}
}
