package tailwind

import (
	"strings"
	"testing"

	"github.com/aydenstechdungeon/gospa/plugin"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	if cfg.Input != "static/css/app.css" || cfg.Output != "static/dist/app.css" || !cfg.Minify {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if len(cfg.Content) == 0 {
		t.Fatal("expected default content globs")
	}
}

func TestNewAndConfigAccessors(t *testing.T) {
	p := New()
	if p == nil {
		t.Fatal("New returned nil")
	}
	if p.Name() != "tailwind" {
		t.Fatalf("unexpected plugin name: %q", p.Name())
	}

	custom := &Config{
		Input:   "in.css",
		Output:  "out.css",
		Content: []string{"./routes/**/*.templ"},
		Minify:  false,
	}
	p.SetConfig(custom)
	got := p.GetConfig()
	if got != custom {
		t.Fatalf("SetConfig/GetConfig did not preserve pointer; got %+v want %+v", got, custom)
	}

	p2 := NewWithConfig(custom)
	if p2.GetConfig() != custom {
		t.Fatal("NewWithConfig did not apply provided config")
	}
}

func TestDependenciesAndCommands(t *testing.T) {
	p := New()

	deps := p.Dependencies()
	if len(deps) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(deps))
	}
	if deps[0].Type != plugin.DepBun || deps[0].Name != "tailwindcss" {
		t.Fatalf("unexpected first dependency: %+v", deps[0])
	}
	if deps[1].Type != plugin.DepBun || deps[1].Name != "@tailwindcss/cli" {
		t.Fatalf("unexpected second dependency: %+v", deps[1])
	}

	cmds := p.Commands()
	if len(cmds) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(cmds))
	}
	if cmds[0].Name != "add:tailwind" || cmds[1].Name != "tailwind:build" || cmds[2].Name != "tailwind:watch" {
		t.Fatalf("unexpected command names: %+v", cmds)
	}
}

func TestInitCreatesOutputDirectory(t *testing.T) {
	tmp := t.TempDir()
	p := NewWithConfig(&Config{
		Input:   "input.css",
		Output:  tmp + "/nested/output.css",
		Content: []string{"./routes/**/*.templ"},
		Minify:  true,
	})

	if err := p.Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
}

func TestCompileRejectsUnsafePath(t *testing.T) {
	p := NewWithConfig(&Config{
		Input:   "safe.css",
		Output:  "unsafe|path.css",
		Content: []string{"./routes/**/*.templ"},
		Minify:  true,
	})

	err := p.compile()
	if err == nil {
		t.Fatal("expected error for unsafe path")
	}
	if !strings.Contains(err.Error(), "invalid characters") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFormatContentArray(t *testing.T) {
	got := formatContentArray([]string{"./a.templ", "./b.go"})
	if !strings.Contains(got, "'./a.templ'") || !strings.Contains(got, "'./b.go'") {
		t.Fatalf("formatted content missing expected entries: %q", got)
	}
	if !strings.HasPrefix(got, "[\n") || !strings.HasSuffix(got, "  ]") {
		t.Fatalf("formatted content has unexpected shape: %q", got)
	}
}

func TestIsPathSafe(t *testing.T) {
	cases := []struct {
		path string
		safe bool
	}{
		{path: "static/css/app.css", safe: true},
		{path: "generated/output.css", safe: true},
		{path: "bad|path.css", safe: false},
		{path: "bad;rm -rf /.css", safe: false},
		{path: "bad`cmd`.css", safe: false},
	}
	for _, tc := range cases {
		got := isPathSafe(tc.path)
		if got != tc.safe {
			t.Fatalf("isPathSafe(%q) = %v, want %v", tc.path, got, tc.safe)
		}
	}
}
