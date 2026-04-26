package gospa

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ahtempl "github.com/a-h/templ"
	"github.com/aydenstechdungeon/gospa/plugin"
	"github.com/aydenstechdungeon/gospa/routing"
	templpkg "github.com/aydenstechdungeon/gospa/templ"
	fiberpkg "github.com/gofiber/fiber/v3"
)

func TestConfigOptionsApply(t *testing.T) {
	cfg := DefaultConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	opts := []ConfigOption{
		WithAppName("Test App"),
		WithDevMode(false),
		WithPort("9999"), // no-op by design
		WithWebSocket(false),
		WithWebSocketPath("/ws-custom"),
		WithRoutesDir("./custom-routes"),
		WithStaticDir("./custom-static"),
		WithStaticPrefix("/assets"),
		WithCacheTemplates(true),
		WithLogger(logger),
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.AppName != "Test App" || cfg.DevMode || cfg.EnableWebSocket || cfg.WebSocketPath != "/ws-custom" {
		t.Fatalf("functional options were not applied correctly: %+v", cfg)
	}
	if cfg.RoutesDir != "./custom-routes" || cfg.StaticDir != "./custom-static" || cfg.StaticPrefix != "/assets" {
		t.Fatalf("path options were not applied correctly: routes=%q static=%q prefix=%q", cfg.RoutesDir, cfg.StaticDir, cfg.StaticPrefix)
	}
	if !cfg.CacheTemplates || cfg.Logger != logger {
		t.Fatalf("cache/logger options were not applied correctly")
	}
}

func TestCacheTagAndKeyDefaults(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()

	tags := app.defaultCacheTags("   ", "  ")
	if len(tags) != 2 || tags[0] != "route:/" || tags[1] != "strategy:ssr" {
		t.Fatalf("unexpected default tags: %#v", tags)
	}

	keys := app.defaultCacheKeys("   ")
	if len(keys) != 2 || keys[0] != "path:/" || keys[1] != "/" {
		t.Fatalf("unexpected default keys: %#v", keys)
	}
}

func TestStorePprShellEvictionAndDedup(t *testing.T) {
	app := New(Config{SSGCacheMaxEntries: 3})
	app.Config.Storage = nil
	defer func() { _ = app.Fiber.Shutdown() }()

	app.storePprShell("/a", []byte("a"), []string{"t1"}, []string{"k1"})
	app.storePprShell("/b", []byte("b"), nil, nil)
	app.storePprShell("/c", []byte("c"), nil, nil)
	app.storePprShell("/d", []byte("d"), nil, nil)
	app.storePprShell("/c", []byte("c2"), nil, nil)

	app.pprShellMu.RLock()
	defer app.pprShellMu.RUnlock()

	if _, ok := app.pprShellCache["/a"]; ok {
		t.Fatalf("expected oldest shell to be evicted")
	}
	if len(app.pprShellKeys) != 2 {
		t.Fatalf("expected 2 shell keys after capped insert+dedup path, got %d", len(app.pprShellKeys))
	}
	if app.pprShellKeys[len(app.pprShellKeys)-1] != "/c" {
		t.Fatalf("expected deduped key /c to move to tail, got keys=%v", app.pprShellKeys)
	}
	if got := string(app.pprShellCache["/c"].html); got != "c2" {
		t.Fatalf("expected latest shell payload for /c, got %q", got)
	}
}

func TestApplyPPRSlotsReplacesKnownSlots(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()

	pagePath := "/ppr-slots-" + strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "")
	routing.RegisterSlot(pagePath, "sidebar", func(props map[string]interface{}) ahtempl.Component {
		return ahtempl.ComponentFunc(func(_ context.Context, w io.Writer) error {
			_, _ = io.WriteString(w, "slot-for:"+props["path"].(string))
			return nil
		})
	})
	defer routing.RegisterSlot(pagePath, "sidebar", nil)

	shell := []byte(`<main>` + templpkg.SlotPlaceholder("sidebar") + templpkg.SlotPlaceholder("missing") + `</main>`)
	out, err := app.applyPPRSlots(context.Background(), &routing.Route{Path: pagePath}, shell, pagePath, routing.RouteOptions{
		DynamicSlots: []string{"sidebar", "missing"},
	})
	if err != nil {
		t.Fatalf("applyPPRSlots returned error: %v", err)
	}

	html := string(out)
	if !strings.Contains(html, `data-gospa-slot="sidebar"`) || !strings.Contains(html, "slot-for:"+pagePath) {
		t.Fatalf("expected rendered sidebar slot, got: %s", html)
	}
	if !strings.Contains(html, templpkg.SlotPlaceholder("missing")) {
		t.Fatalf("expected unresolved placeholder for missing slot to remain in shell")
	}
}

func TestISRInitSemaphoreAndBackgroundRevalidate(t *testing.T) {
	app := New(Config{ISRSemaphoreLimit: 2, ISRTimeout: 2 * time.Second})
	app.Config.Storage = nil
	defer func() { _ = app.Fiber.Shutdown() }()

	app.initSemaphore()
	if app.isrSemaphore == nil || cap(app.isrSemaphore) != 2 {
		t.Fatalf("expected ISR semaphore capacity 2, got %d", cap(app.isrSemaphore))
	}

	path := "/isr-test-" + strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "")
	app.isrRevalidating.Store(path, struct{}{})
	app.backgroundRevalidate(path, &routing.Route{Path: path})

	if _, ok := app.isrRevalidating.Load(path); ok {
		t.Fatalf("expected in-flight ISR key to be removed after backgroundRevalidate")
	}

	app.ssgCacheMu.RLock()
	entry, ok := app.ssgCache[path]
	app.ssgCacheMu.RUnlock()
	if !ok {
		t.Fatalf("expected ISR background render to populate ssg cache")
	}
	if len(entry.html) == 0 {
		t.Fatalf("expected cached ISR HTML to be non-empty")
	}
}

func TestResolveLoadChainMergesInOrder(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()

	routePath := "/load-chain/:id"
	layoutPath := "/load-chain"

	routing.RegisterLayoutLoad("", func(_ routing.LoadContext) (map[string]interface{}, error) {
		return map[string]interface{}{"root": true, "shared": "root"}, nil
	})
	routing.RegisterLayoutLoad(layoutPath, func(lc routing.LoadContext) (map[string]interface{}, error) {
		return map[string]interface{}{"layoutParam": lc.Param("id"), "shared": "layout"}, nil
	})
	routing.RegisterLoad(routePath, func(lc routing.LoadContext) (map[string]interface{}, error) {
		return map[string]interface{}{"query": lc.Query("q"), "shared": "page"}, nil
	})
	defer routing.RegisterLayoutLoad("", nil)
	defer routing.RegisterLayoutLoad(layoutPath, nil)
	defer routing.RegisterLoad(routePath, nil)

	var got map[string]interface{}
	app.Get("/load-chain/:id", func(c fiberpkg.Ctx) error {
		props, _, err := app.resolveLoadChain(c, &routing.Route{Path: routePath}, []*routing.Route{{Path: layoutPath}})
		if err != nil {
			return c.Status(http.StatusInternalServerError).SendString(err.Error())
		}
		got = props
		return c.SendStatus(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/load-chain/42?q=abc", nil)
	resp, err := app.Fiber.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", resp.StatusCode)
	}

	if got["root"] != true || got["layoutParam"] != "42" || got["query"] != "abc" {
		t.Fatalf("unexpected merged load props: %#v", got)
	}
	if got["shared"] != "page" {
		t.Fatalf("expected page load data to win last-write merge, got shared=%v", got["shared"])
	}
}

func TestResolveTierHelpers(t *testing.T) {
	app := New(Config{RuntimeTier: RuntimeTierMicro})
	defer func() { _ = app.Fiber.Shutdown() }()

	layoutPath := "/tier-layout"
	routing.RegisterLayoutWithOptions(layoutPath, nil, "full")
	defer routing.RegisterLayoutWithOptions(layoutPath, nil, "")

	tier, reason := app.resolveTierWithReason(routing.RouteOptions{RuntimeTier: "core"}, []*routing.Route{{Path: layoutPath}})
	if tier != "full" {
		t.Fatalf("expected highest tier full, got %q", tier)
	}
	if !strings.Contains(reason, "config:micro") || !strings.Contains(reason, "page:core") || !strings.Contains(reason, "layout:"+layoutPath+"=full") {
		t.Fatalf("unexpected tier reason: %q", reason)
	}

	if tierToLevel("micro") != 1 || tierToLevel("core") != 2 || tierToLevel("full") != 3 || tierToLevel("unknown") != 0 {
		t.Fatalf("tierToLevel mapping mismatch")
	}
	if levelToTier(3) != "full" || levelToTier(2) != "core" || levelToTier(1) != "micro" || levelToTier(0) != "full" {
		t.Fatalf("levelToTier mapping mismatch")
	}
}

func TestRouteHelpersAndFiberLoadContextMethods(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()

	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "hello.txt"), []byte("world"), 0600); err != nil {
		t.Fatalf("failed to write static fixture: %v", err)
	}

	app.Static("/assets", tmp)
	api := app.Group("/api")
	api.Get("/group", func(c fiberpkg.Ctx) error { return c.SendString("group-ok") })

	app.Get("/ctx/:id", func(c fiberpkg.Ctx) error {
		lc := &fiberLoadContext{c: c}
		return c.JSON(map[string]string{
			"param":  lc.Param("id"),
			"query":  lc.Query("q", "def"),
			"header": lc.Header("X-Test"),
			"cookie": lc.Cookie("session"),
			"form":   lc.FormValue("f", "fallback"),
			"form2":  lc.FormValue("missing", "fallback"),
			"method": lc.Method(),
			"path":   lc.Path(),
		})
	})
	app.Post("/post", func(c fiberpkg.Ctx) error { return c.SendStatus(http.StatusCreated) })
	app.Put("/put", func(c fiberpkg.Ctx) error { return c.SendStatus(http.StatusAccepted) })
	app.Delete("/del", func(c fiberpkg.Ctx) error { return c.SendStatus(http.StatusNoContent) })

	cases := []struct {
		method string
		target string
		body   io.Reader
		want   int
	}{
		{http.MethodPost, "/post", nil, http.StatusCreated},
		{http.MethodPut, "/put", nil, http.StatusAccepted},
		{http.MethodDelete, "/del", nil, http.StatusNoContent},
		{http.MethodGet, "/api/group", nil, http.StatusOK},
		{http.MethodGet, "/assets/hello.txt", nil, http.StatusOK},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.target, tc.body)
		resp, err := app.Fiber.Test(req)
		if err != nil {
			t.Fatalf("%s %s failed: %v", tc.method, tc.target, err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != tc.want {
			t.Fatalf("%s %s expected %d got %d", tc.method, tc.target, tc.want, resp.StatusCode)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/ctx/77?q=ok", strings.NewReader("f=formv"))
	req.Header.Set("X-Test", "hdr")
	req.Header.Set("Cookie", "session=s123")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Fiber.Test(req)
	if err != nil {
		t.Fatalf("context request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for context handler, got %d", resp.StatusCode)
	}

	var payload map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode context payload: %v", err)
	}

	if payload["param"] != "77" || payload["query"] != "ok" || payload["header"] != "hdr" || payload["cookie"] != "s123" || payload["method"] != "GET" || payload["path"] != "/ctx/77" {
		t.Fatalf("unexpected context payload: %#v", payload)
	}
	if payload["form"] != "formv" {
		t.Fatalf("expected parsed form value from request body, got %q", payload["form"])
	}
	if payload["form2"] != "fallback" {
		t.Fatalf("expected fallback for missing form key, got %q", payload["form2"])
	}
}

type testRuntimePlugin struct {
	name string
}

func (p *testRuntimePlugin) Name() string { return p.name }
func (p *testRuntimePlugin) Init() error  { return nil }
func (p *testRuntimePlugin) Dependencies() []plugin.Dependency {
	return nil
}
func (p *testRuntimePlugin) Config() plugin.Config { return plugin.Config{} }
func (p *testRuntimePlugin) Middlewares() []interface{} {
	return []interface{}{fiberpkg.Handler(func(c fiberpkg.Ctx) error {
		c.Set("X-Plugin", "ok")
		return c.Next()
	})}
}
func (p *testRuntimePlugin) TemplateFuncs() map[string]interface{} {
	return map[string]interface{}{"pluginFn": func() string { return "ok" }}
}

func TestPluginHelpers(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()

	name := "plugin-test-" + strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "")
	p := &testRuntimePlugin{name: name}
	defer plugin.Unregister(name)

	if err := app.UsePlugins(p); err != nil {
		t.Fatalf("UsePlugins failed: %v", err)
	}

	got, ok := app.GetPlugin(name)
	if !ok || got == nil || got.Name() != name {
		t.Fatalf("GetPlugin failed, ok=%v plugin=%v", ok, got)
	}

	found := false
	for _, info := range app.ListPlugins() {
		if info.Name == name {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected plugin %q in ListPlugins output", name)
	}

	if _, ok := app.GetTemplateFuncs()["pluginFn"]; !ok {
		t.Fatalf("expected runtime plugin template function to be registered")
	}

	app.applyPluginMiddleware()
	app.Get("/plugin-check", func(c fiberpkg.Ctx) error { return c.SendStatus(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodGet, "/plugin-check", nil)
	resp, err := app.Fiber.Test(req)
	if err != nil {
		t.Fatalf("plugin middleware request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.Header.Get("X-Plugin") != "ok" {
		t.Fatalf("expected plugin middleware header, got %q", resp.Header.Get("X-Plugin"))
	}
}
