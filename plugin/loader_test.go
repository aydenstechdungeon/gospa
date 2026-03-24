package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFromGitHubRejectsMutableRefsByDefault(t *testing.T) {
	loader := NewExternalPluginLoaderWithCache(t.TempDir())

	_, err := loader.LoadFromGitHub("owner/repo")
	if err == nil {
		t.Fatal("expected mutable ref rejection, got nil")
	}
	if !strings.Contains(err.Error(), "mutable plugin ref") {
		t.Fatalf("expected mutable ref error, got %v", err)
	}
}

func TestLoadFromGitHubAllowsMutableRefsWhenExplicitlyEnabled(t *testing.T) {
	loader := NewExternalPluginLoaderWithCache(t.TempDir()).AllowMutableRefs(true)

	_, err := loader.LoadFromGitHub("owner/repo")
	if err == nil {
		t.Fatal("expected download failure after mutable refs were allowed, got nil")
	}
	if strings.Contains(err.Error(), "mutable plugin ref") {
		t.Fatalf("mutable ref guard should be bypassed when explicitly enabled: %v", err)
	}
}

func TestExpectResolvedRefRejectsInvalidHash(t *testing.T) {
	loader := NewExternalPluginLoaderWithCache(t.TempDir()).ExpectResolvedRef("not-a-sha")
	_, err := loader.LoadFromGitHub("owner/repo@v1.0.0")
	if err == nil {
		t.Fatal("expected resolved ref validation error")
	}
	if !strings.Contains(err.Error(), "failed to validate expected resolved ref") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadFromPathRejectsResolvedRefMismatch(t *testing.T) {
	tmp := t.TempDir()
	pluginPath := filepath.Join(tmp, "plugin")
	if err := os.MkdirAll(pluginPath, 0o750); err != nil {
		t.Fatalf("failed creating plugin dir: %v", err)
	}

	metadata := `{"name":"example","version":"v1.0.0","source":"github.com/acme/example","resolvedRef":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`
	if err := os.WriteFile(filepath.Join(pluginPath, "plugin.json"), []byte(metadata), 0o600); err != nil {
		t.Fatalf("failed writing plugin metadata: %v", err)
	}

	loader := NewExternalPluginLoaderWithCache(tmp).ExpectResolvedRef("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	_, err := loader.loadFromPath(pluginPath, loader.expectedRef)
	if err == nil {
		t.Fatal("expected resolved ref mismatch")
	}
	if !strings.Contains(err.Error(), "resolved ref mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}
