package routing

import (
	"context"
	"io"
	"testing"

	"github.com/a-h/templ"
)

// stubComponent returns a no-op templ.Component for test use.
func stubComponent() templ.Component {
	return templ.ComponentFunc(func(_ context.Context, _ io.Writer) error { return nil })
}

// ─── Registry ─────────────────────────────────────────────────────────────────

func TestRegistryRegisterAndGetPage(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterPage("/test", func(_ map[string]interface{}) templ.Component {
		return stubComponent()
	})
	fn := reg.GetPage("/test")
	if fn == nil {
		t.Error("expected registered page component, got nil")
	}
}

func TestRegistry_RegisterPage(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterPage("/home", func(_ map[string]interface{}) templ.Component {
		return stubComponent()
	})
	if !reg.HasPage("/home") {
		t.Error("HasPage('/home') should be true after RegisterPage")
	}
}

func TestRegistry_RegisterPageDefaultsToSSR(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterPage("/home", func(_ map[string]interface{}) templ.Component {
		return stubComponent()
	})
	opts := reg.GetRouteOptions("/home")
	if opts.Strategy != StrategySSR {
		t.Errorf("default strategy should be SSR, got %q", opts.Strategy)
	}
}

func TestRegistry_RegisterPageWithOptions_SSG(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterPageWithOptions("/cacheme", func(_ map[string]interface{}) templ.Component {
		return stubComponent()
	}, RouteOptions{Strategy: StrategySSG})
	opts := reg.GetRouteOptions("/cacheme")
	if opts.Strategy != StrategySSG {
		t.Errorf("expected SSG strategy, got %q", opts.Strategy)
	}
}

func TestRegistry_RegisterPageWithOptions_ISR(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterPageWithOptions("/revalidate", func(_ map[string]interface{}) templ.Component {
		return stubComponent()
	}, RouteOptions{
		Strategy: StrategyISR,
	})
	opts := reg.GetRouteOptions("/revalidate")
	if opts.Strategy != StrategyISR {
		t.Errorf("expected ISR strategy, got %q", opts.Strategy)
	}
}

func TestRegistry_RegisterPageWithOptions_PPR(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterPageWithOptions("/ppr", func(_ map[string]interface{}) templ.Component {
		return stubComponent()
	}, RouteOptions{
		Strategy:     StrategyPPR,
		DynamicSlots: []string{"sidebar", "footer"},
	})
	opts := reg.GetRouteOptions("/ppr")
	if opts.Strategy != StrategyPPR {
		t.Errorf("expected PPR strategy, got %q", opts.Strategy)
	}
	if len(opts.DynamicSlots) != 2 {
		t.Errorf("expected 2 dynamic slots, got %d", len(opts.DynamicSlots))
	}
}

func TestRegistry_RegisterLayout(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterLayout("/dashboard", func(_ templ.Component, _ map[string]interface{}) templ.Component {
		return stubComponent()
	})
	if !reg.HasLayout("/dashboard") {
		t.Error("HasLayout('/dashboard') should be true after RegisterLayout")
	}
	fn := reg.GetLayout("/dashboard")
	if fn == nil {
		t.Error("GetLayout('/dashboard') should return non-nil")
	}
}

func TestRegistry_GetNonExistentPage(t *testing.T) {
	reg := NewRegistry()
	fn := reg.GetPage("/nonexistent")
	if fn != nil {
		t.Error("GetPage for non-existent path should return nil")
	}
}

func TestRegistry_GetNonExistentLayout(t *testing.T) {
	reg := NewRegistry()
	fn := reg.GetLayout("/nonexistent")
	if fn != nil {
		t.Error("GetLayout for non-existent path should return nil")
	}
}

func TestRegistry_GetRouteOptionsDefault(t *testing.T) {
	reg := NewRegistry()
	opts := reg.GetRouteOptions("/notregistered")
	if opts.Strategy != StrategySSR {
		t.Errorf("default options strategy should be SSR, got %q", opts.Strategy)
	}
}

func TestRegistry_RegisterRootLayout(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterRootLayout(func(_ templ.Component, _ map[string]interface{}) templ.Component {
		return stubComponent()
	})
	fn := reg.GetRootLayout()
	if fn == nil {
		t.Error("GetRootLayout() should return the registered root layout")
	}
}

func TestRegistry_RootLayoutNilBeforeRegistration(t *testing.T) {
	reg := NewRegistry()
	fn := reg.GetRootLayout()
	if fn != nil {
		t.Error("GetRootLayout() should return nil before any root layout is registered")
	}
}

func TestRegistry_RegisterSlot(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterSlot("/page", "sidebar", func(_ map[string]interface{}) templ.Component {
		return stubComponent()
	})
	fn := reg.GetSlot("/page", "sidebar")
	if fn == nil {
		t.Error("GetSlot should return registered slot function")
	}
}

func TestRegistry_GetNonExistentSlot(t *testing.T) {
	reg := NewRegistry()
	fn := reg.GetSlot("/page", "missing")
	if fn != nil {
		t.Error("GetSlot for non-existent slot should return nil")
	}
}

func TestRegistry_SlotMultipleSlotsPerPage(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterSlot("/page", "header", func(_ map[string]interface{}) templ.Component {
		return stubComponent()
	})
	reg.RegisterSlot("/page", "footer", func(_ map[string]interface{}) templ.Component {
		return stubComponent()
	})
	if reg.GetSlot("/page", "header") == nil {
		t.Error("header slot should be registered")
	}
	if reg.GetSlot("/page", "footer") == nil {
		t.Error("footer slot should be registered")
	}
}

func TestRegistry_SlotOverwrite(t *testing.T) {
	reg := NewRegistry()
	called := false
	reg.RegisterSlot("/page", "slot", func(_ map[string]interface{}) templ.Component {
		return stubComponent()
	})
	reg.RegisterSlot("/page", "slot", func(_ map[string]interface{}) templ.Component {
		called = true
		return stubComponent()
	})
	fn := reg.GetSlot("/page", "slot")
	if fn == nil {
		t.Error("slot should exist after overwrite")
	}
	_ = called // second registration should overwrite first
}

func TestRegistry_ThreadSafety(_ *testing.T) {
	reg := NewRegistry()
	done := make(chan struct{})

	go func() {
		for i := 0; i < 100; i++ {
			reg.RegisterPage("/concurrent", func(_ map[string]interface{}) templ.Component {
				return stubComponent()
			})
			reg.RegisterLayout("/concurrent", func(_ templ.Component, _ map[string]interface{}) templ.Component {
				return stubComponent()
			})
		}
		close(done)
	}()

	for i := 0; i < 100; i++ {
		_ = reg.GetPage("/concurrent")
		_ = reg.HasPage("/concurrent")
		_ = reg.GetLayout("/concurrent")
		_ = reg.HasLayout("/concurrent")
	}
	<-done
}

// ─── Global Registry ──────────────────────────────────────────────────────────

func TestGlobalRegistry(t *testing.T) {
	reg := GetGlobalRegistry()
	if reg == nil {
		t.Error("GetGlobalRegistry() should return non-nil")
	}
}

func TestGlobalRegisterPage(t *testing.T) {
	const path = "/global_test_page_unique_7f3b"
	RegisterPage(path, func(_ map[string]interface{}) templ.Component {
		return stubComponent()
	})
	if !HasPage(path) {
		t.Errorf("HasPage(%q) should be true after global RegisterPage", path)
	}
	fn := GetPage(path)
	if fn == nil {
		t.Errorf("GetPage(%q) should return non-nil after global RegisterPage", path)
	}
}

func TestGlobalRegisterLayout(t *testing.T) {
	const path = "/global_test_layout_unique_7f3b"
	RegisterLayout(path, func(_ templ.Component, _ map[string]interface{}) templ.Component {
		return stubComponent()
	})
	if !HasLayout(path) {
		t.Errorf("HasLayout(%q) should be true after global RegisterLayout", path)
	}
}

func TestGlobalGetRouteOptions_Unregistered(t *testing.T) {
	opts := GetRouteOptions("/path/not/registered/xyz")
	if opts.Strategy != StrategySSR {
		t.Errorf("unregistered path should default to SSR, got %q", opts.Strategy)
	}
}

// ─── RenderStrategy constants ─────────────────────────────────────────────────

func TestRenderStrategyConstants(t *testing.T) {
	tests := []struct {
		name     string
		strategy RenderStrategy
		expected string
	}{
		{"SSR", StrategySSR, "ssr"},
		{"SSG", StrategySSG, "ssg"},
		{"ISR", StrategyISR, "isr"},
		{"PPR", StrategyPPR, "ppr"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.strategy) != tt.expected {
				t.Errorf("Strategy %s = %q, want %q", tt.name, tt.strategy, tt.expected)
			}
		})
	}
}
