package gospa

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/a-h/templ"
	"github.com/aydenstechdungeon/gospa/routing"
	"github.com/aydenstechdungeon/gospa/routing/kit"
	fiber "github.com/gofiber/fiber/v3"
)

func TestResolveLoadChain_HelperSemantics(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()

	routePath := "/m1-helper-chain"
	layoutPath := "/m1-helper-chain"

	routing.RegisterLayoutLoad("", func(_ routing.LoadContext) (map[string]interface{}, error) {
		kit.Depends("dep:root")
		return map[string]interface{}{"root": "r"}, nil
	})
	routing.RegisterLayoutLoad(layoutPath, func(c routing.LoadContext) (map[string]interface{}, error) {
		parent, err := kit.Parent[map[string]interface{}](c)
		if err != nil {
			return nil, err
		}
		kit.Depends("dep:layout")
		return map[string]interface{}{
			"layout":       "l",
			"layoutParent": parent["root"],
		}, nil
	})
	routing.RegisterLoad(routePath, func(c routing.LoadContext) (map[string]interface{}, error) {
		parent, err := kit.Parent[map[string]interface{}](c)
		if err != nil {
			return nil, err
		}
		kit.Depends("dep:page")
		_ = kit.Untrack(func() error {
			kit.Depends("dep:ignored")
			return nil
		})
		return map[string]interface{}{
			"pageParent": parent["layout"],
		}, nil
	})
	defer routing.RegisterLayoutLoad("", nil)
	defer routing.RegisterLayoutLoad(layoutPath, nil)
	defer routing.RegisterLoad(routePath, nil)

	props, depKeys, err := app.resolveLoadChainWithContext(
		newStaticLoadContext(routePath, nil),
		&routing.Route{Path: routePath},
		[]*routing.Route{{Path: layoutPath}},
	)
	if err != nil {
		t.Fatalf("resolveLoadChainWithContext failed: %v", err)
	}

	if props["layoutParent"] != "r" {
		t.Fatalf("expected layout parent from root loader, got %v", props["layoutParent"])
	}
	if props["pageParent"] != "l" {
		t.Fatalf("expected page parent from nearest layout loader, got %v", props["pageParent"])
	}

	got := make(map[string]bool, len(depKeys))
	for _, key := range depKeys {
		got[key] = true
	}
	if !got["dep:root"] || !got["dep:layout"] || !got["dep:page"] {
		t.Fatalf("expected captured dependency keys, got %v", depKeys)
	}
	if got["dep:ignored"] {
		t.Fatalf("expected untracked dependency to be excluded, got %v", depKeys)
	}
}

func TestRenderRoute_LoadKitErrorDataAndSSR(t *testing.T) {
	app := New(Config{})
	defer func() { _ = app.Fiber.Shutdown() }()

	path := "/m1-load-error"
	route := &routing.Route{Path: path}
	routing.RegisterPage(path, func(_ map[string]interface{}) templ.Component {
		return templ.Raw("<div>ok</div>")
	})
	routing.RegisterLoad(path, func(_ routing.LoadContext) (map[string]interface{}, error) {
		return nil, kit.Error(http.StatusTeapot, map[string]interface{}{"reason": "x"})
	})
	defer routing.RegisterPage(path, nil)
	defer routing.RegisterLoad(path, nil)

	app.Get(path, func(c fiber.Ctx) error {
		return app.renderRoute(c, route, nil)
	})

	reqData := httptest.NewRequest(http.MethodGet, path+"?__data=1", nil)
	respData, err := app.Fiber.Test(reqData)
	if err != nil {
		t.Fatalf("data request failed: %v", err)
	}
	if respData.StatusCode != http.StatusTeapot {
		t.Fatalf("expected 418 on data request, got %d", respData.StatusCode)
	}

	reqSSR := httptest.NewRequest(http.MethodGet, path, nil)
	respSSR, err := app.Fiber.Test(reqSSR)
	if err != nil {
		t.Fatalf("ssr request failed: %v", err)
	}
	if respSSR.StatusCode != http.StatusTeapot {
		t.Fatalf("expected 418 on SSR request, got %d", respSSR.StatusCode)
	}
}
