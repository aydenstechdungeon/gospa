package cli

import "testing"

func TestValidateProjectName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid simple", input: "myapp"},
		{name: "valid dotted", input: "my.app-1"},
		{name: "empty", input: "", wantErr: true},
		{name: "contains spaces", input: "my app", wantErr: true},
		{name: "path traversal", input: "../myapp", wantErr: true},
		{name: "path separator", input: "my/app", wantErr: true},
		{name: "starts with dash", input: "-myapp", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateProjectName(tc.input)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for %q, got nil", tc.input)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("did not expect error for %q, got %v", tc.input, err)
			}
		})
	}
}

func TestCreateProjectRejectsEscapingOutputDir(t *testing.T) {
	t.Parallel()

	cfg := &ProjectConfig{
		Name:      "safe-name",
		Module:    "github.com/example/safe-name",
		OutputDir: "../escape",
	}

	if err := createProject(cfg); err == nil {
		t.Fatal("expected error for escaping output directory, got nil")
	}
}
