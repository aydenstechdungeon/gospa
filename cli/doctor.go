package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DoctorConfig controls CLI doctor checks.
type DoctorConfig struct {
	RoutesDir string
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

	printer.Title("GoSPA Doctor")
	printer.Subtitle("Checking Go, Bun, project layout, and runtime entrypoints...")

	checks := []doctorCheck{
		checkBinary("go", true),
		checkBinary("bun", false),
		checkProjectFile("go.mod", true, "Go module"),
		checkProjectFile("main.go", false, "application entrypoint"),
		checkProjectDir(config.RoutesDir, false, "routes directory"),
		checkAnyFile("client", []string{"src/runtime.ts", "src/index.ts", "src/main.ts"}, false, "client runtime entrypoint"),
		checkAnyFile(".", []string{"package.json", "client/package.json"}, false, "Bun package manifest"),
		checkLibrary("libwebp", false),
		checkLibrary("libheif", false),
	}

	hasFailure := false
	for _, check := range checks {
		if check.Err != nil {
			if check.Required {
				printer.Error("%s: %v", check.Name, check.Err)
				hasFailure = true
				continue
			}
			printer.Warning("%s: %v", check.Name, check.Err)
			continue
		}

		if check.Detail != "" {
			printer.Success("%s: %s", check.Name, check.Detail)
		} else {
			printer.Success("%s", check.Name)
		}
	}

	if hasFailure {
		fmt.Fprintln(os.Stderr, "\nGoSPA Doctor found blocking setup issues.")
		os.Exit(1)
	}

	fmt.Println("\nGoSPA Doctor found no blocking setup issues.")
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
