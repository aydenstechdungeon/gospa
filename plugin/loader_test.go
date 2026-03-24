package plugin

import (
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
