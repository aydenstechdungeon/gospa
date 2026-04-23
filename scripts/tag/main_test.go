package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTagArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantSkip  bool
		wantTag   string
		expectErr bool
	}{
		{
			name:     "flag before tag",
			args:     []string{"-skip-tag", "v1.2.3"},
			wantSkip: true,
			wantTag:  "v1.2.3",
		},
		{
			name:     "flag after tag",
			args:     []string{"v1.2.3", "-skip-tag"},
			wantSkip: true,
			wantTag:  "v1.2.3",
		},
		{
			name:    "tag only",
			args:    []string{"v1.2.3"},
			wantTag: "v1.2.3",
		},
		{
			name:      "missing tag",
			args:      []string{"-skip-tag"},
			expectErr: true,
		},
		{
			name:      "too many args",
			args:      []string{"v1.2.3", "v1.2.4"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSkip, gotTag, err := parseTagArgs(tt.args)
			if tt.expectErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseTagArgs returned error: %v", err)
			}
			if gotSkip != tt.wantSkip {
				t.Fatalf("skip mismatch: got %v want %v", gotSkip, tt.wantSkip)
			}
			if gotTag != tt.wantTag {
				t.Fatalf("tag mismatch: got %q want %q", gotTag, tt.wantTag)
			}
		})
	}
}

func TestUpdateVersionFile(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	if err := os.WriteFile("config.go", []byte(`package gospa
const Version = "0.1.0"
`), 0600); err != nil {
		t.Fatalf("write config.go failed: %v", err)
	}

	updateVersionFile("0.1.0", "0.2.0")

	data, err := os.ReadFile("config.go")
	if err != nil {
		t.Fatalf("read config.go failed: %v", err)
	}
	if !strings.Contains(string(data), `const Version = "0.2.0"`) {
		t.Fatalf("version line not updated, content:\n%s", string(data))
	}
}

func TestUpdateModFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "go.mod")
	content := `module example.com/app

require github.com/aydenstechdungeon/gospa v0.1.0
replace github.com/aydenstechdungeon/gospa v0.1.0 => ../gospa
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write go.mod failed: %v", err)
	}

	changed := updateModFile(path, "github.com/aydenstechdungeon/gospa", "0.1.0", "0.2.0", "v0.1.0", "v0.2.0")
	if !changed {
		t.Fatal("expected updateModFile to report changed=true")
	}

	//nolint:gosec // path is created from t.TempDir() in this test
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read go.mod failed: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "github.com/aydenstechdungeon/gospa v0.2.0") {
		t.Fatalf("module version not updated:\n%s", got)
	}
}

func TestUpdateOtherFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "README.md")
	content := `
import "github.com/aydenstechdungeon/gospa v0.1.0"
see github.com/aydenstechdungeon/gospa/plugin/image@v0.1.0
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write README failed: %v", err)
	}

	changed := updateOtherFile(path, "github.com/aydenstechdungeon/gospa", "0.1.0", "0.2.0", "v0.1.0", "v0.2.0")
	if !changed {
		t.Fatal("expected updateOtherFile to report changed=true")
	}

	//nolint:gosec // path is created from t.TempDir() in this test
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read README failed: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "v0.2.0") || strings.Contains(got, "v0.1.0") {
		t.Fatalf("expected all versions updated, content:\n%s", got)
	}
}
