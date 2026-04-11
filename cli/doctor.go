package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DoctorConfig controls CLI doctor checks.
type DoctorConfig struct {
	RoutesDir    string // Routes directory
	Fix          bool   // Auto-fix detected issues
	JSONOutput   bool   // JSON output
	Quiet        bool   // Only show errors
	CheckUpdates bool   // Check for package updates
}

// Doctor inspects the current project for common setup issues.
func Doctor(config *DoctorConfig) {
	printer := NewColorPrinter()

	if config == nil {
		config = &DoctorConfig{RoutesDir: "./routes"}
	}
	if config.RoutesDir == "" {
		config.RoutesDir = "./routes"
	}

	if !config.Quiet && !config.JSONOutput {
		printer.Title("GoSPA Doctor")
		printer.Subtitle("Checking Go, Node.js tooling, project layout, and runtime entrypoints...")
	}

	checks := []doctorCheck{
		checkBinary("go", true),
		checkNodeTooling(),
		checkProjectFile("go.mod", true, "Go module"),
		checkProjectFile("main.go", false, "application entrypoint"),
		checkProjectDir(config.RoutesDir, false, "routes directory"),
		checkAnyFile("client", []string{"src/runtime.ts", "src/index.ts", "src/main.ts"}, false, "client runtime entrypoint"),
		checkAnyFile(".", []string{"package.json", "client/package.json"}, false, "Node.js package manifest"),
		checkLibrary("libwebp", false),
		checkLibrary("libheif", false),
		checkTemplVersion(),
		checkNodeToolingVersion(),
		checkIslandsBundle(),
	}

	// Add update checks if requested
	if config.CheckUpdates {
		checks = append(checks, checkForUpdates()...)
	}

	if config.JSONOutput {
		outputDoctorJSON(checks)
		return
	}

	hasFailure := false
	for _, check := range checks {
		if check.Err != nil {
			if check.Required {
				printer.Error("%s: %v", check.Name, check.Err)
				hasFailure = true
				continue
			}
			if !config.Quiet {
				printer.Warning("%s: %v", check.Name, check.Err)
			}
			continue
		}

		if !config.Quiet {
			if check.Detail != "" {
				printer.Success("%s: %s", check.Name, check.Detail)
			} else {
				printer.Success("%s", check.Name)
			}
		}
	}

	// Auto-fix if requested
	if config.Fix {
		hasFailure = doctorFix(config) || hasFailure
	}

	if hasFailure {
		fmt.Fprintln(os.Stderr, "\nGoSPA Doctor found blocking setup issues.")
		os.Exit(1)
	}

	if !config.Quiet {
		fmt.Println("\nGoSPA Doctor found no blocking setup issues.")
	}
}

// outputDoctorJSON outputs check results as JSON
func outputDoctorJSON(checks []doctorCheck) {
	type jsonCheck struct {
		Name     string `json:"name"`
		Detail   string `json:"detail,omitempty"`
		Required bool   `json:"required"`
		Passed   bool   `json:"passed"`
		Error    string `json:"error,omitempty"`
	}

	var results []jsonCheck
	for _, c := range checks {
		jc := jsonCheck{
			Name:     c.Name,
			Detail:   c.Detail,
			Required: c.Required,
			Passed:   c.Err == nil,
		}
		if c.Err != nil {
			jc.Error = c.Err.Error()
		}
		results = append(results, jc)
	}

	data, _ := json.MarshalIndent(results, "", "  ")
	fmt.Println(string(data))
}

// doctorFix attempts to auto-fix detected issues
func doctorFix(config *DoctorConfig) bool {
	hasFailure := false
	fixed := false

	// Fix missing directories
	if _, err := os.Stat(config.RoutesDir); os.IsNotExist(err) {
		fmt.Printf("Creating missing routes directory: %s\n", config.RoutesDir)
		if err := os.MkdirAll(config.RoutesDir, 0750); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create %s: %v\n", config.RoutesDir, err)
			hasFailure = true
		} else {
			fixed = true
		}
	}

	// Fix missing package.json
	if _, err := os.Stat("package.json"); os.IsNotExist(err) {
		fmt.Println("Creating missing package.json...")
		defaultPkgJSON := `{
	"name": "gospa-project",
	"type": "module",
	"scripts": {
		"dev": "gospa dev",
		"build": "gospa build"
	}
}
`
		if err := os.WriteFile("package.json", []byte(defaultPkgJSON), 0600); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create package.json: %v\n", err)
			hasFailure = true
		} else {
			fixed = true
		}
	}

	if fixed {
		fmt.Println("✓ Auto-fix complete")
	}

	return hasFailure
}

// checkForUpdates returns checks for package updates
func checkForUpdates() []doctorCheck {
	return []doctorCheck{
		checkGospaVersion(),
	}
}

func checkGospaVersion() doctorCheck {
	cmd := exec.Command("go", "list", "-m", "-json", "github.com/aydenstechdungeon/gospa")
	output, err := cmd.Output()
	if err != nil {
		return doctorCheck{
			Name:     "GoSPA version",
			Required: false,
			Err:      fmt.Errorf("cannot check GoSPA version: %v", err),
		}
	}

	var mod struct {
		Version string `json:"Version"`
	}
	if err := json.Unmarshal(output, &mod); err != nil {
		return doctorCheck{
			Name:     "GoSPA version",
			Required: false,
			Err:      fmt.Errorf("cannot parse GoSPA version: %v", err),
		}
	}

	return doctorCheck{
		Name:     "GoSPA version",
		Required: false,
		Detail:   mod.Version,
	}
}

type doctorCheck struct {
	Name     string
	Detail   string
	Required bool
	Err      error
}

func checkBinary(name string, required bool) doctorCheck {
	path, err := exec.LookPath(name)
	return doctorCheck{
		Name:     fmt.Sprintf("%s available", commandLabel(name)),
		Detail:   path,
		Required: required,
		Err:      err,
	}
}

func checkLibrary(name string, required bool) doctorCheck {
	_, err := exec.LookPath("pkg-config")
	if err != nil {
		return doctorCheck{
			Name:     fmt.Sprintf("Library %s", name),
			Required: required,
			Err:      fmt.Errorf("pkg-config not found (cannot check for %s)", name),
		}
	}

	// #nosec G204
	cmd := exec.Command("pkg-config", "--exists", name)
	if err := cmd.Run(); err != nil {
		return doctorCheck{
			Name:     fmt.Sprintf("Library %s", name),
			Required: required,
			Err:      fmt.Errorf("%s missing (arch: sudo pacman -S %s, ubuntu: sudo apt-get install %s-dev)", name, name, name),
		}
	}

	return doctorCheck{
		Name:     fmt.Sprintf("Library %s", name),
		Detail:   fmt.Sprintf("%s found", name),
		Required: required,
	}
}

func checkProjectFile(path string, required bool, label string) doctorCheck {
	info, err := os.Stat(path)
	if err != nil {
		return doctorCheck{Name: label, Required: required, Err: err}
	}
	if info.IsDir() {
		return doctorCheck{Name: label, Required: required, Err: fmt.Errorf("%s is a directory", path)}
	}
	return doctorCheck{Name: label, Detail: path, Required: required}
}

func checkProjectDir(path string, required bool, label string) doctorCheck {
	info, err := os.Stat(path)
	if err != nil {
		return doctorCheck{Name: label, Required: required, Err: err}
	}
	if !info.IsDir() {
		return doctorCheck{Name: label, Required: required, Err: fmt.Errorf("%s is not a directory", path)}
	}
	return doctorCheck{Name: label, Detail: path, Required: required}
}

func checkAnyFile(baseDir string, candidates []string, required bool, label string) doctorCheck {
	for _, candidate := range candidates {
		path := candidate
		if baseDir != "." && baseDir != "" {
			path = filepath.Join(baseDir, candidate)
		}
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return doctorCheck{Name: label, Detail: path, Required: required}
		}
	}

	return doctorCheck{
		Name:     label,
		Required: required,
		Err:      fmt.Errorf("none of %s found", strings.Join(candidates, ", ")),
	}
}

func commandLabel(name string) string {
	if name == "" {
		return "Command"
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

func checkTemplVersion() doctorCheck {
	cmd := exec.Command("go", "list", "-m", "-json", "github.com/a-h/templ")
	output, err := cmd.Output()
	if err != nil {
		return doctorCheck{
			Name:     "templ version",
			Required: false,
			Err:      fmt.Errorf("cannot check templ version: %v", err),
		}
	}

	var mod struct {
		Version string `json:"Version"`
	}
	if err := json.Unmarshal(output, &mod); err != nil {
		return doctorCheck{
			Name:     "templ version",
			Required: false,
			Err:      fmt.Errorf("cannot parse templ version: %v", err),
		}
	}

	// Accept templ v0.3.x (any v0.3 release is compatible)
	// The pinned version in dev.go should be updated periodically
	if strings.HasPrefix(mod.Version, "v0.3.") {
		return doctorCheck{
			Name:     "templ version",
			Required: false,
			Detail:   mod.Version,
		}
	}

	return doctorCheck{
		Name:     "templ version",
		Required: false,
		Detail:   fmt.Sprintf("found %s (recommended: v0.3.x)", mod.Version),
	}
}

func checkNodeToolingVersion() doctorCheck {
	pm := GetPackageManager()
	if pm == NonePM {
		return doctorCheck{
			Name:     "Node tooling version",
			Required: false,
			Err:      fmt.Errorf("bun, pnpm, and npm not found"),
		}
	}

	// #nosec G204
	cmd := exec.Command(string(pm), "--version")
	output, err := cmd.Output()
	if err != nil {
		return doctorCheck{
			Name:     fmt.Sprintf("%s version", string(pm)),
			Required: false,
			Err:      fmt.Errorf("cannot check %s version: %v", string(pm), err),
		}
	}

	version := strings.TrimSpace(string(output))
	path, _ := exec.LookPath(string(pm))
	return doctorCheck{
		Name:     fmt.Sprintf("%s version", strings.ToUpper(string(pm[:1]))+string(pm[1:])),
		Required: false,
		Detail:   fmt.Sprintf("%s at %s", version, path),
	}
}

func checkNodeTooling() doctorCheck {
	if path, err := exec.LookPath("bun"); err == nil {
		return doctorCheck{Name: "Bun found", Detail: path, Required: false}
	}
	if path, err := exec.LookPath("pnpm"); err == nil {
		return doctorCheck{Name: "Bun(preferred) not found; Pnpm found", Detail: path, Required: false}
	}
	if path, err := exec.LookPath("npm"); err == nil {
		return doctorCheck{Name: "Bun(preferred), Pnpm not found; Npm found", Detail: path, Required: false}
	}
	return doctorCheck{
		Name:     "Node Tooling",
		Required: false,
		Err:      fmt.Errorf("Bun(preferred), pnpm or npm weren't found"),
	}
}

func checkIslandsBundle() doctorCheck {
	islandsEntry := "generated/islands.ts"
	islandsOutput := "static/js/islands.js"

	if _, err := os.Stat(islandsEntry); os.IsNotExist(err) {
		return doctorCheck{
			Name:     "islands bundle",
			Required: false,
			Detail:   "no islands entry (optional)",
		}
	}

	if _, err := os.Stat(islandsOutput); os.IsNotExist(err) {
		return doctorCheck{
			Name:     "islands bundle",
			Required: false,
			Err:      fmt.Errorf("islands entry exists but %s not built", islandsOutput),
		}
	}

	return doctorCheck{
		Name:     "islands bundle",
		Required: false,
		Detail:   islandsOutput,
	}
}
