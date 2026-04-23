package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestFindConfigFile(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	if got := FindConfigFile(); got != "" {
		t.Fatalf("expected no config file, got %q", got)
	}

	if err := os.WriteFile("gospa.config.yml", []byte("version: \"1\"\n"), 0600); err != nil {
		t.Fatalf("write config failed: %v", err)
	}
	if got := FindConfigFile(); got != filepath.Join(".", "gospa.config.yml") {
		t.Fatalf("unexpected config path: %q", got)
	}

	if err := os.WriteFile("gospa.config.yaml", []byte("version: \"1\"\n"), 0600); err != nil {
		t.Fatalf("write config failed: %v", err)
	}
	if got := FindConfigFile(); got != filepath.Join(".", "gospa.config.yaml") {
		t.Fatalf("expected yaml to have higher priority, got %q", got)
	}
}

func TestLoadConfig_DefaultWhenMissing(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg.Version != configFileVersion {
		t.Fatalf("expected default version %q, got %q", configFileVersion, cfg.Version)
	}
}

func TestLoadConfig_ParsesYAMLAndSetsVersion(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "gospa.config.yaml")
	content := `
project:
  name: demo
dev:
  port: 4242
build:
  output: out
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg.Version != configFileVersion {
		t.Fatalf("expected version to be defaulted to %q, got %q", configFileVersion, cfg.Version)
	}
	if cfg.Project.Name != "demo" || cfg.Dev.Port != 4242 || cfg.Build.Output != "out" {
		t.Fatalf("config fields not parsed correctly: %+v", cfg)
	}
}

func TestLoadConfig_JSON(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "gospa.config.json")
	content := `{"version":"1","project":{"name":"json-demo"},"dev":{"port":3131}}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg.Project.Name != "json-demo" || cfg.Dev.Port != 3131 {
		t.Fatalf("json config fields not parsed correctly: %+v", cfg)
	}
}

func TestLoadConfig_TOMLUnsupported(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "gospa.config.toml")
	if err := os.WriteFile(path, []byte(`version = "1"`), 0600); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for TOML format")
	}
	if !strings.Contains(err.Error(), "TOML format not yet supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSaveConfigAndLoadRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "saved.yaml")
	in := &GoSPAConfig{
		Project: ProjectSection{Name: "roundtrip", Module: "example.com/roundtrip"},
		Dev:     DevSection{Port: 1234},
		Build:   BuildSection{Output: "dist-custom"},
	}

	if err := SaveConfig(in, path); err != nil {
		t.Fatalf("SaveConfig returned error: %v", err)
	}

	out, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if out.Version != configFileVersion {
		t.Fatalf("expected saved version %q, got %q", configFileVersion, out.Version)
	}
	if out.Project.Name != "roundtrip" || out.Dev.Port != 1234 || out.Build.Output != "dist-custom" {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}

func TestMergeWithEnv(t *testing.T) {
	cfg := DefaultConfig()
	t.Setenv("GOSPA_DEV_PORT", "9999")
	t.Setenv("GOSPA_DEV_HOST", "0.0.0.0")
	t.Setenv("GOSPA_DEV_ROUTES_DIR", "/tmp/routes")
	t.Setenv("GOSPA_DEV_PROXY", "http://localhost:4000")
	t.Setenv("GOSPA_DEV_TIMEOUT", "45s")
	t.Setenv("GOSPA_BUILD_OUTPUT", "build")
	t.Setenv("GOSPA_BUILD_MINIFY", "0")
	t.Setenv("GOSPA_BUILD_ENV", "staging")
	t.Setenv("GOSPA_GENERATE_OUTPUT", "gen")
	t.Setenv("GOSPA_SERVE_PORT", "9090")

	cfg.MergeWithEnv()

	if cfg.Dev.Port != 9999 || cfg.Dev.Host != "0.0.0.0" || cfg.Dev.RoutesDir != "/tmp/routes" || cfg.Dev.Proxy != "http://localhost:4000" || cfg.Dev.Timeout != 45*time.Second {
		t.Fatalf("dev env overrides not applied: %+v", cfg.Dev)
	}
	if cfg.Build.Output != "build" || cfg.Build.Minify || cfg.Build.Env != "staging" {
		t.Fatalf("build env overrides not applied: %+v", cfg.Build)
	}
	if cfg.Generate.Output != "gen" {
		t.Fatalf("generate env override not applied: %+v", cfg.Generate)
	}
	if cfg.Serve.Port != 9090 {
		t.Fatalf("serve env override not applied: %+v", cfg.Serve)
	}
}

func TestConfigConversions(t *testing.T) {
	cfg := &GoSPAConfig{
		Dev: DevSection{
			Port:       1234,
			Host:       "127.0.0.1",
			Open:       true,
			RoutesDir:  "routes",
			WatchPaths: []string{"routes", "components"},
			Proxy:      "http://localhost:5000",
			HMRPort:    4444,
			Debounce:   200 * time.Millisecond,
			Timeout:    1 * time.Minute,
		},
		Build: BuildSection{
			Output:    "dist",
			Minify:    true,
			Compress:  true,
			SourceMap: true,
			CGO:       true,
			Env:       "production",
			AssetsDir: "static",
			LDFlags:   "-s -w",
			Tags:      "prod",
		},
		Generate: GenerateSection{
			Output: "generated",
			Type:   "island",
			Strict: true,
		},
		Serve: ServeSection{
			Port:    8080,
			Gzip:    true,
			Brotli:  true,
			Cache:   true,
			Headers: map[string]string{"X-Test": "1"},
		},
		BuildAll: BuildAllSection{
			Targets:   []string{"linux/amd64"},
			OutputDir: "releases",
			Compress:  true,
			Manifest:  false,
			Parallel:  2,
		},
	}

	dev := cfg.ToDevConfig()
	if dev.Port != 1234 || dev.Host != "127.0.0.1" || dev.HMRPort != 4444 || dev.Timeout != time.Minute {
		t.Fatalf("ToDevConfig mismatch: %+v", dev)
	}

	build := cfg.ToBuildConfig()
	if build.OutputDir != "dist" || !build.Minify || !build.Compress || !build.SourceMap || !build.CGO || build.Env != "production" || build.Tags != "prod" {
		t.Fatalf("ToBuildConfig mismatch: %+v", build)
	}

	gen := cfg.ToGenerateConfig()
	if gen.OutputDir != "generated" || gen.ComponentType != "island" || !gen.Strict {
		t.Fatalf("ToGenerateConfig mismatch: %+v", gen)
	}

	serve := cfg.ToServeConfig()
	if serve.Port != 8080 || !serve.Gzip || !serve.Brotli || !serve.Cache || serve.Headers["X-Test"] != "1" {
		t.Fatalf("ToServeConfig mismatch: %+v", serve)
	}

	buildAll := cfg.ToBuildAllConfig()
	if !reflect.DeepEqual(buildAll.Targets, []string{"linux/amd64"}) || buildAll.OutputDir != "releases" || !buildAll.Compress || buildAll.Manifest || buildAll.Parallel != 2 {
		t.Fatalf("ToBuildAllConfig mismatch: %+v", buildAll)
	}
}

func TestToBuildAllConfig_DefaultTargets(t *testing.T) {
	cfg := &GoSPAConfig{BuildAll: BuildAllSection{}}
	got := cfg.ToBuildAllConfig()
	want := []string{
		"linux/amd64",
		"linux/arm64",
		"darwin/amd64",
		"darwin/arm64",
		"windows/amd64",
		"windows/arm64",
	}
	if !reflect.DeepEqual(got.Targets, want) {
		t.Fatalf("unexpected default targets: %#v", got.Targets)
	}
}

func TestToolingHelpers(t *testing.T) {
	if BunPM.String() != "bun" {
		t.Fatalf("unexpected PackageManager.String(): %q", BunPM.String())
	}

	if got := GetBundlerCommand(BunPM); got != "bun" {
		t.Fatalf("GetBundlerCommand(BunPM) = %q", got)
	}
	if got := GetBundlerCommand(PnpmPM); got != "pnpm" {
		t.Fatalf("GetBundlerCommand(PnpmPM) = %q", got)
	}
	if got := GetBundlerCommand(NonePM); got != "" {
		t.Fatalf("GetBundlerCommand(NonePM) = %q", got)
	}

	if got := GetExecuteCommand(BunPM); got != "bun x" {
		t.Fatalf("GetExecuteCommand(BunPM) = %q", got)
	}
	if got := GetExecuteCommand(PnpmPM); got != "pnpm dlx" {
		t.Fatalf("GetExecuteCommand(PnpmPM) = %q", got)
	}
	if got := GetExecuteCommand(NpmPM); got != "npx" {
		t.Fatalf("GetExecuteCommand(NpmPM) = %q", got)
	}
	if got := GetExecuteCommand(NonePM); got != "npx" {
		t.Fatalf("GetExecuteCommand(NonePM) = %q", got)
	}

	if got := GetRunCommand(BunPM); got != "bun" {
		t.Fatalf("GetRunCommand(BunPM) = %q", got)
	}
	if got := GetRunCommand(NonePM); got != "npm" {
		t.Fatalf("GetRunCommand(NonePM) = %q", got)
	}
}

func TestParseBuildTargets(t *testing.T) {
	in := []string{
		"linux/amd64",
		"LINUX / ARM64",
		"windows/386",
		"badformat",
		"plan9/amd64",
		"linux/mips",
	}
	got := parseBuildTargets(in)
	want := []targetSpec{
		{platform: "linux", arch: "amd64"},
		{platform: "linux", arch: "arm64"},
		{platform: "windows", arch: "386"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseBuildTargets mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestDisplayOrFallback(t *testing.T) {
	if got := displayOrFallback("", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}
	if got := displayOrFallback("value", "fallback"); got != "value" {
		t.Fatalf("expected value, got %q", got)
	}
}

func TestFormatFileSize(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "f.bin")
	if err := os.WriteFile(path, make([]byte, 1536), 0600); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	got := formatFileSize(path)
	if got != "1.5 KB" {
		t.Fatalf("expected 1.5 KB, got %q", got)
	}

	missing := formatFileSize(filepath.Join(tmp, "missing.bin"))
	if missing != "size unavailable" {
		t.Fatalf("expected size unavailable for missing file, got %q", missing)
	}
}

func TestParseExecuteCommand(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		wantCmd  string
		wantArgs []string
	}{
		{name: "npx", in: "npx", wantCmd: "npx", wantArgs: nil},
		{name: "pnpm dlx", in: "pnpm dlx", wantCmd: "pnpm", wantArgs: []string{"dlx"}},
		{name: "bun x", in: "bun x", wantCmd: "bun", wantArgs: []string{"x"}},
		{name: "multiple args", in: "tool arg1 arg2", wantCmd: "tool", wantArgs: []string{"arg1", "arg2"}},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cmd, args := parseExecuteCommand(tc.in)
			if cmd != tc.wantCmd || !reflect.DeepEqual(args, tc.wantArgs) {
				t.Fatalf("parseExecuteCommand(%q) = (%q, %#v), want (%q, %#v)", tc.in, cmd, args, tc.wantCmd, tc.wantArgs)
			}
		})
	}
}
