package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckCSPNonceConfig(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.go")
	content := `package main
func main() {
	_ = SecurityHeadersMiddleware("script-src 'self' {nonce}")
}
`
	if err := os.WriteFile(mainPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	check := checkCSPNonceConfig()
	if check.Err != nil {
		t.Fatalf("expected nonce config check to pass, got err: %v", check.Err)
	}
}

func TestCheckPreforkStoragePubSubConfig(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.go")
	content := `package main
func main() {
	_ = struct{
		Prefork bool
		Storage any
		PubSub any
	}{
		Prefork: true,
		Storage: struct{}{},
		PubSub: struct{}{},
	}
}
`
	if err := os.WriteFile(mainPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	check := checkPreforkStoragePubSubConfig()
	if check.Err != nil {
		t.Fatalf("expected prefork consistency check to pass, got err: %v", check.Err)
	}
}

func TestCheckSFCStrict(t *testing.T) {
	dir := t.TempDir()
	routesDir := filepath.Join(dir, "routes")
	if err := os.MkdirAll(routesDir, 0750); err != nil {
		t.Fatalf("failed to create routes dir: %v", err)
	}
	content := `<script lang="go">
var count = $state(0)
</script>
<template><button on:click={func() { count++ }}>{count}</button></template>
`
	if err := os.WriteFile(filepath.Join(routesDir, "page.gospa"), []byte(content), 0600); err != nil {
		t.Fatalf("failed to write gospa file: %v", err)
	}

	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	check := checkSFCStrict("./routes")
	if check.Err != nil {
		t.Fatalf("expected SFC strict check to pass, got err: %v", check.Err)
	}
}

func TestCheckSFCModuleServerConflicts_PassesWithoutConflict(t *testing.T) {
	dir := t.TempDir()
	routesDir := filepath.Join(dir, "routes")
	if err := os.MkdirAll(routesDir, 0750); err != nil {
		t.Fatalf("failed to create routes dir: %v", err)
	}

	content := `<script context="module" lang="go">
func Load(c routing.LoadContext) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
</script>
<template><div>ok</div></template>
`
	if err := os.WriteFile(filepath.Join(routesDir, "+page.gospa"), []byte(content), 0600); err != nil {
		t.Fatalf("failed to write gospa file: %v", err)
	}

	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	check := checkSFCModuleServerConflicts("./routes")
	if check.Err != nil {
		t.Fatalf("expected conflict check to pass, got err: %v", check.Err)
	}
}

func TestCheckSFCModuleServerConflicts_FailsWithConflict(t *testing.T) {
	dir := t.TempDir()
	routesDir := filepath.Join(dir, "routes")
	if err := os.MkdirAll(routesDir, 0750); err != nil {
		t.Fatalf("failed to create routes dir: %v", err)
	}

	content := `<script context="module" lang="go">
func Load(c routing.LoadContext) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
</script>
<template><div>ok</div></template>
`
	if err := os.WriteFile(filepath.Join(routesDir, "+page.gospa"), []byte(content), 0600); err != nil {
		t.Fatalf("failed to write gospa file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(routesDir, "+page.server.go"), []byte("package routes"), 0600); err != nil {
		t.Fatalf("failed to write +page.server.go: %v", err)
	}

	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	check := checkSFCModuleServerConflicts("./routes")
	if check.Err == nil {
		t.Fatalf("expected conflict check to fail")
	}
}

func TestCheckSFCModuleScriptImports_PassesWhenUsed(t *testing.T) {
	dir := t.TempDir()
	routesDir := filepath.Join(dir, "routes")
	if err := os.MkdirAll(routesDir, 0750); err != nil {
		t.Fatalf("failed to create routes dir: %v", err)
	}

	content := `<script context="module" lang="go">
import "github.com/aydenstechdungeon/gospa/routing"

func Load(c routing.LoadContext) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
</script>
<template><div>ok</div></template>
`
	if err := os.WriteFile(filepath.Join(routesDir, "+page.gospa"), []byte(content), 0600); err != nil {
		t.Fatalf("failed to write gospa file: %v", err)
	}

	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	check := checkSFCModuleScriptImports("./routes")
	if check.Err != nil {
		t.Fatalf("expected import usage check to pass, got err: %v", check.Err)
	}
}

func TestCheckSFCModuleScriptImports_FailsWhenUnused(t *testing.T) {
	dir := t.TempDir()
	routesDir := filepath.Join(dir, "routes")
	if err := os.MkdirAll(routesDir, 0750); err != nil {
		t.Fatalf("failed to create routes dir: %v", err)
	}

	content := `<script context="module" lang="go">
import (
	"github.com/aydenstechdungeon/gospa/routing"
	"github.com/aydenstechdungeon/gospa/routing/kit"
)

func ActionDefault(c routing.LoadContext) (interface{}, error) {
	return map[string]interface{}{"ok": true}, nil
}
</script>
<template><div>ok</div></template>
`
	if err := os.WriteFile(filepath.Join(routesDir, "+page.gospa"), []byte(content), 0600); err != nil {
		t.Fatalf("failed to write gospa file: %v", err)
	}

	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	check := checkSFCModuleScriptImports("./routes")
	if check.Err == nil {
		t.Fatalf("expected import usage check to fail for unused import")
	}
}
