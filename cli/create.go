// Package cli provides command-line interface tools for GoSPA.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectConfig holds configuration for a new GoSPA project.
type ProjectConfig struct {
	Name       string
	Module     string
	OutputDir  string
	WithGit    bool
	WithDocker bool
}

// CreateProject creates a new GoSPA project with the given name.
func CreateProject(name string) {
	config := &ProjectConfig{
		Name:      name,
		Module:    fmt.Sprintf("github.com/%s/%s", getGitUsername(), name),
		OutputDir: name, // Create in current directory
		WithGit:   true,
	}

	if err := createProject(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating project: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Created GoSPA project '%s'\n", name)
	fmt.Println("\nNext steps:")
	fmt.Printf("  cd %s\n", config.OutputDir)
	fmt.Println("  go mod tidy")
	fmt.Println("  gospa dev")
}

// CreateProjectWithConfig creates a new GoSPA project with custom configuration.
func CreateProjectWithConfig(config *ProjectConfig) error {
	return createProject(config)
}

func createProject(config *ProjectConfig) error {
	// Create project directory
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Create subdirectories
	dirs := []string{
		"routes",
		"components",
		"lib",
		"static",
		"static/css",
		"static/js",
	}

	for _, dir := range dirs {
		path := filepath.Join(config.OutputDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create go.mod
	if err := createGoMod(config); err != nil {
		return err
	}

	// Create main.go
	if err := createMainGo(config); err != nil {
		return err
	}

	// Create routes/page.templ
	if err := createHomePage(config); err != nil {
		return err
	}

	// Create routes/layout.templ
	if err := createLayout(config); err != nil {
		return err
	}

	// Create components/counter.templ
	if err := createCounterComponent(config); err != nil {
		return err
	}

	// Create lib/state.go
	if err := createStateFile(config); err != nil {
		return err
	}

	// Create static/css/style.css
	if err := createCSSFile(config); err != nil {
		return err
	}

	// Create .gitignore
	if config.WithGit {
		if err := createGitignore(config); err != nil {
			return err
		}
	}

	return nil
}

func createGoMod(config *ProjectConfig) error {
	content := fmt.Sprintf(`module %s

go 1.24.0

require (
	github.com/a-h/templ v0.3.977
	github.com/aydenstechdungeon/gospa v0.1.4
)
`, config.Module)

	path := filepath.Join(config.OutputDir, "go.mod")
	return os.WriteFile(path, []byte(content), 0644)
}

func createMainGo(config *ProjectConfig) error {
	content := fmt.Sprintf(`package main

import (
	"log"

	"%s/lib"
	_ "%s/routes" // Import routes to trigger init()

	"github.com/aydenstechdungeon/gospa"
)

func main() {
	app := gospa.New(gospa.Config{
		RoutesDir: "./routes",
		DevMode:   true,
		AppName:   "%s",
		DefaultState: map[string]interface{}{
			"count": lib.GlobalCounter.Count,
		},
	})

	if err := app.Run(":3000"); err != nil {
		log.Fatal(err)
	}
}
`, config.Module, config.Module, config.Name)

	path := filepath.Join(config.OutputDir, "main.go")
	return os.WriteFile(path, []byte(content), 0644)
}

func createHomePage(config *ProjectConfig) error {
	content := `package main

templ Page() {
	<div class="container mx-auto px-4 py-8">
		<h1 class="text-4xl font-bold mb-4">Welcome to GoSPA</h1>
		<p class="text-lg text-gray-600 mb-8">
			A modern SPA framework for Go with Fiber and Templ.
		</p>
		
		<div class="bg-white rounded-lg shadow p-6">
			@Counter()
		</div>
		
		<div class="mt-8 grid grid-cols-1 md:grid-cols-3 gap-4">
			<div class="bg-blue-50 p-4 rounded-lg">
				<h3 class="font-semibold text-blue-800">Reactive State</h3>
				<p class="text-sm text-blue-600">Svelte-like runes for Go</p>
			</div>
			<div class="bg-green-50 p-4 rounded-lg">
				<h3 class="font-semibold text-green-800">File-based Routing</h3>
				<p class="text-sm text-green-600">SvelteKit-style routing</p>
			</div>
			<div class="bg-purple-50 p-4 rounded-lg">
				<h3 class="font-semibold text-purple-800">Real-time Sync</h3>
				<p class="text-sm text-purple-600">WebSocket state sync</p>
			</div>
		</div>
	</div>
}
`

	path := filepath.Join(config.OutputDir, "routes", "page.templ")
	return os.WriteFile(path, []byte(content), 0644)
}

func createLayout(config *ProjectConfig) error {
	content := `package main

import "github.com/aydenstechdungeon/gospa/templ"

templ Layout(title string) {
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8"/>
		<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
		<title>{ title }</title>
		<link rel="stylesheet" href="/static/css/style.css"/>
		@templ.RuntimeScript()
	</head>
	<body class="bg-gray-100 min-h-screen">
		<nav class="bg-white shadow-sm">
			<div class="container mx-auto px-4 py-3">
				<a href="/" class="text-xl font-bold text-gray-800">GoSPA</a>
			</div>
		</nav>
		<main>
			{ children... }
		</main>
	</body>
	</html>
}
`

	path := filepath.Join(config.OutputDir, "routes", "layout.templ")
	return os.WriteFile(path, []byte(content), 0644)
}

func createCounterComponent(config *ProjectConfig) error {
	content := `package main

templ Counter() {
	<div 
		class="flex flex-col items-center justify-center p-8"
		data-gospa-component="counter"
		data-gospa-state='{"count":0}'
	>
		<h2 class="text-2xl font-semibold mb-4">Counter Example</h2>
		<div class="flex items-center gap-4">
			<button 
				class="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600"
				data-on="click:decrement"
			>
				-
			</button>
			<span class="text-3xl font-mono" data-bind="count">
				0
			</span>
			<button 
				class="px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600"
				data-on="click:increment"
			>
				+
			</button>
		</div>
		<p class="mt-4 text-gray-600">
			Click the buttons to change the count. State syncs automatically!
		</p>
	</div>
}
`

	path := filepath.Join(config.OutputDir, "components", "counter.templ")
	return os.WriteFile(path, []byte(content), 0644)
}

func createStateFile(config *ProjectConfig) error {
	content := `package lib

// AppState holds application-wide state.
type AppState struct {
	// Add your application state here
}

// NewAppState creates a new application state.
func NewAppState() *AppState {
	return &AppState{}
}
`

	path := filepath.Join(config.OutputDir, "lib", "state.go")
	return os.WriteFile(path, []byte(content), 0644)
}

func createCSSFile(config *ProjectConfig) error {
	content := `/* GoSPA Default Styles */

* {
	box-sizing: border-box;
}

body {
	font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
	margin: 0;
	padding: 0;
}

.container {
	max-width: 1200px;
	margin: 0 auto;
}

/* Tailwind-like utilities */
.text-4xl { font-size: 2.25rem; line-height: 2.5rem; }
.text-2xl { font-size: 1.5rem; line-height: 2rem; }
.text-lg { font-size: 1.125rem; line-height: 1.75rem; }
.text-xl { font-size: 1.25rem; line-height: 1.75rem; }
.text-sm { font-size: 0.875rem; line-height: 1.25rem; }

.font-bold { font-weight: 700; }
.font-semibold { font-weight: 600; }
.font-mono { font-family: monospace; }

.text-gray-600 { color: #4b5563; }
.text-gray-800 { color: #1f2937; }
.text-blue-800 { color: #1e40af; }
.text-green-800 { color: #166534; }
.text-purple-800 { color: #6b21a8; }
.text-blue-600 { color: #2563eb; }
.text-green-600 { color: #16a34a; }
.text-purple-600 { color: #9333ea; }
.text-white { color: #ffffff; }

.bg-gray-100 { background-color: #f3f4f6; }
.bg-white { background-color: #ffffff; }
.bg-blue-50 { background-color: #eff6ff; }
.bg-green-50 { background-color: #f0fdf4; }
.bg-purple-50 { background-color: #faf5ff; }
.bg-red-500 { background-color: #ef4444; }
.bg-green-500 { background-color: #22c55e; }

.hover\:bg-red-600:hover { background-color: #dc2626; }
.hover\:bg-green-600:hover { background-color: #16a34a; }

.min-h-screen { min-height: 100vh; }

.shadow { box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px 0 rgba(0, 0, 0, 0.06); }
.shadow-sm { box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05); }

.rounded { border-radius: 0.25rem; }
.rounded-lg { border-radius: 0.5rem; }

.px-4 { padding-left: 1rem; padding-right: 1rem; }
.py-2 { padding-top: 0.5rem; padding-bottom: 0.5rem; }
.py-3 { padding-top: 0.75rem; padding-bottom: 0.75rem; }
.py-8 { padding-top: 2rem; padding-bottom: 2rem; }
.p-4 { padding: 1rem; }
.p-6 { padding: 1.5rem; }

.mb-4 { margin-bottom: 1rem; }
.mb-8 { margin-bottom: 2rem; }
.mt-4 { margin-top: 1rem; }
.mt-8 { margin-top: 2rem; }

.flex { display: flex; }
.items-center { align-items: center; }
.gap-4 { gap: 1rem; }

.grid { display: grid; }
.grid-cols-1 { grid-template-columns: repeat(1, minmax(0, 1fr)); }

@media (min-width: 768px) {
	.md\:grid-cols-3 { grid-template-columns: repeat(3, minmax(0, 1fr)); }
}
`

	path := filepath.Join(config.OutputDir, "static", "css", "style.css")
	return os.WriteFile(path, []byte(content), 0644)
}

func createGitignore(config *ProjectConfig) error {
	content := `# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary
*.test

# Output of the go coverage tool
*.out

# Dependency directories
vendor/

# Go workspace file
go.work

# Build output
dist/
build/

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Environment
.env
.env.local
.env.*.local

# Temp files
tmp/
temp/
`

	path := filepath.Join(config.OutputDir, ".gitignore")
	return os.WriteFile(path, []byte(content), 0644)
}

func getGitUsername() string {
	// Try to get git username from git config
	// For now, return a default
	return "yourusername"
}

// ValidateProjectName checks if a project name is valid.
func ValidateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	if strings.Contains(name, " ") {
		return fmt.Errorf("project name cannot contain spaces")
	}

	if strings.HasPrefix(name, "-") || strings.HasPrefix(name, "_") {
		return fmt.Errorf("project name cannot start with - or _")
	}

	return nil
}
