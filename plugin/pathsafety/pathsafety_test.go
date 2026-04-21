package pathsafety

import (
	"path/filepath"
	"testing"
)

func TestResolvePath_ContainedByDefault(t *testing.T) {
	base := t.TempDir()

	resolved, err := ResolvePath(base, "generated/auth", false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := filepath.Join(base, "generated", "auth")
	if resolved != expected {
		t.Fatalf("expected %q, got %q", expected, resolved)
	}
}

func TestResolvePath_RejectsTraversal(t *testing.T) {
	base := t.TempDir()

	if _, err := ResolvePath(base, "../outside", false); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
}

func TestResolvePath_RejectsAbsoluteOutsideByDefault(t *testing.T) {
	base := t.TempDir()
	outside := filepath.Join(filepath.Dir(base), "outside-target")

	if _, err := ResolvePath(base, outside, false); err == nil {
		t.Fatal("expected absolute outside path to be rejected")
	}
}

func TestResolvePath_AllowsEscapeWhenEnabled(t *testing.T) {
	base := t.TempDir()
	outside := filepath.Join(filepath.Dir(base), "outside-target")

	resolved, err := ResolvePath(base, outside, true)
	if err != nil {
		t.Fatalf("expected path escape to be allowed, got %v", err)
	}

	if resolved != filepath.Clean(outside) {
		t.Fatalf("expected %q, got %q", filepath.Clean(outside), resolved)
	}
}
