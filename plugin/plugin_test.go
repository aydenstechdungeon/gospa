package plugin

import "testing"

type testPlugin struct{ name string }

func (p testPlugin) Name() string               { return p.name }
func (p testPlugin) Init() error                { return nil }
func (p testPlugin) Dependencies() []Dependency { return nil }

func TestRegisterRejectsDuplicatePluginNames(t *testing.T) {
	Unregister("duplicate-test")
	if err := Register(testPlugin{name: "duplicate-test"}); err != nil {
		t.Fatalf("first registration failed: %v", err)
	}
	defer Unregister("duplicate-test")

	if err := Register(testPlugin{name: "duplicate-test"}); err == nil {
		t.Fatal("expected duplicate registration error, got nil")
	}
}
