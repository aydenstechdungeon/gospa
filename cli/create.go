// Package cli provides command-line interface tools for GoSPA.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ProjectConfig holds configuration for a new GoSPA project.
type ProjectConfig struct {
	Name           string
	Module         string
	OutputDir      string
	WithGit        bool
	WithDocker     bool
	Template       string
	NonInteractive bool
	PackageManager string // Package manager to use (bun, pnpm, npm, auto)
}

var (
	projectNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)
	modulePathPattern  = regexp.MustCompile(`^[a-zA-Z0-9._~/-]+$`)
)

// Valid templates
var validTemplates = map[string]bool{
	"default":  true,
	"minimal":  true,
	"api":      true,
	"realtime": true,
}

// CreateProject creates a new GoSPA project with the given name.
func CreateProject(name string) {
	CreateProjectWithOptions(name, "", false)
}

// CreateProjectWithTemplate creates a new GoSPA project with the specified template.
func CreateProjectWithTemplate(name string, template string) {
	CreateProjectWithOptions(name, template, false)
}

// CreateProjectWithOptions creates a new GoSPA project with custom options.
func CreateProjectWithOptions(name string, template string, nonInteractive bool) {
	if err := ValidateProjectName(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid project name %q: %v\n", name, err)
		os.Exit(1)
	}

	// Validate template
	if template == "" {
		template = "default"
	}
	if !validTemplates[template] {
		fmt.Fprintf(os.Stderr, "Error: invalid template %q. Valid templates: default, minimal, api, realtime\n", template)
		os.Exit(1)
	}

	// Prompt for module path if not provided via env or interactive
	module := askForModule(name, nonInteractive)

	config := &ProjectConfig{
		Name:           name,
		Module:         module,
		OutputDir:      name,
		WithGit:        true,
		Template:       template,
		NonInteractive: nonInteractive,
	}

	if err := createProject(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating project: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Created GoSPA project '%s' (template: %s)\n", name, template)
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
	if config == nil {
		return errors.New("project config is required")
	}
	if err := ValidateProjectName(config.Name); err != nil {
		return fmt.Errorf("invalid project name %q: %w", config.Name, err)
	}
	if config.Module == "" || !modulePathPattern.MatchString(config.Module) {
		return fmt.Errorf("invalid module path %q", config.Module)
	}

	cleanOutputDir := filepath.Clean(config.OutputDir)
	if cleanOutputDir == "." || cleanOutputDir == string(filepath.Separator) {
		return fmt.Errorf("invalid output directory %q", config.OutputDir)
	}
	if filepath.IsAbs(cleanOutputDir) {
		return fmt.Errorf("absolute output directory is not allowed: %q", config.OutputDir)
	}
	if strings.HasPrefix(cleanOutputDir, ".."+string(filepath.Separator)) || cleanOutputDir == ".." {
		return fmt.Errorf("output directory escapes current directory: %q", config.OutputDir)
	}
	config.OutputDir = cleanOutputDir

	// Create project directory
	if err := os.MkdirAll(config.OutputDir, 0750); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Create subdirectories — only routes/ is required. static/ and its
	// subdirectories are created so the default CSS works, but users may
	// delete them once they replace the starter template.
	dirs := []string{
		"routes",
		"static",
		"static/css",
	}

	for _, dir := range dirs {
		path := filepath.Join(config.OutputDir, dir)
		if err := os.MkdirAll(path, 0750); err != nil {
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

	// Create routes/root_layout.templ
	if err := createRootLayout(config); err != nil {
		return err
	}

	// Create routes/_error.templ
	if err := createErrorPage(config); err != nil {
		return err
	}

	// Create routes/_middleware.go
	if err := createMiddleware(config); err != nil {
		return err
	}

	// Create package.json
	if err := createPackageJSON(config); err != nil {
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
	// Get the current gospa version dynamically
	gospaVersion := getGoSPAGlobalVersion()

	content := fmt.Sprintf(`module %s

go 1.23

require (
	github.com/a-h/templ v0.3.1001
	github.com/aydenstechdungeon/gospa %s
)
`, config.Module, gospaVersion)

	path := filepath.Join(config.OutputDir, "go.mod")
	return os.WriteFile(path, []byte(content), 0600)
}

// getGoSPAGlobalVersion returns the current gospa version from go.mod
func getGoSPAGlobalVersion() string {
	// Try to get the version from the current module
	cmd := exec.Command("go", "list", "-m", "-json", "github.com/aydenstechdungeon/gospa")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to a known recent version if we can't determine
		return "v0.1.36"
	}

	var mod struct {
		Version string `json:"Version"`
	}
	if err := json.Unmarshal(output, &mod); err != nil || mod.Version == "" {
		return "v0.1.36"
	}
	return mod.Version
}

func createMainGo(config *ProjectConfig) error {
	content := fmt.Sprintf(`package main

import (
	"log"
	"os"

	_ "%s/routes" // Import routes to trigger init()

	"github.com/aydenstechdungeon/gospa"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	config := gospa.DefaultConfig()
	config.RoutesDir = "./routes"
	config.DevMode = true
	config.AppName = "%s"

	app := gospa.New(config)

	if err := app.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
`, config.Module, config.Name)

	path := filepath.Join(config.OutputDir, "main.go")
	return os.WriteFile(path, []byte(content), 0600)
}

func createHomePage(config *ProjectConfig) error {
	content := `package routes

templ Page() {
	<div class="welcome-container">
		<div class="welcome-content">
			<div class="logo">
				<svg width="80" height="80" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" xmlns="http://www.w3.org/2000/svg">
					<circle cx="12" cy="12" r="10"/>
					<path d="M12 6v6l4 2" stroke-linecap="round"/>
				</svg>
			</div>
			<h1 class="title">Welcome to GoSPA</h1>
			<p class="subtitle">
				A modern reactive framework for building single-page applications with Go.
			</p>
			<div class="actions">
				<a href="https://gospa.onrender.com/docs" class="btn btn-primary" target="_blank" rel="noopener">
					Read Documentation →
				</a>
			</div>
			<div class="features">
				<div class="feature">
					<span class="feature-icon">⚡</span>
					<span class="feature-text">Reactive State</span>
				</div>
				<div class="feature">
					<span class="feature-icon">🗂️</span>
					<span class="feature-text">File-Based Routing</span>
				</div>
				<div class="feature">
					<span class="feature-icon">🔄</span>
					<span class="feature-text">WebSocket Sync</span>
				</div>
			</div>
		</div>
	</div>
}
`

	path := filepath.Join(config.OutputDir, "routes", "page.templ")
	return os.WriteFile(path, []byte(content), 0600)
}

func createLayout(config *ProjectConfig) error {
	content := `package routes

templ Layout(title string) {
	<div class="layout-wrapper">
		<header>
			<nav>
				<a href="/">Home</a>
			</nav>
		</header>
		<div class="content">
			{ children... }
		</div>
	</div>
}
`

	path := filepath.Join(config.OutputDir, "routes", "layout.templ")
	return os.WriteFile(path, []byte(content), 0600)
}

func createRootLayout(config *ProjectConfig) error {
	content := `package routes

templ RootLayout(title string) {
	<!DOCTYPE html>
	<html lang="en" data-gospa-auto>
	<head>
		<meta charset="UTF-8"/>
		<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
		<meta name="description" content="A GoSPA application"/>
		<title>{ title }</title>
		
		<link rel="preconnect" href="/"/>
		<link rel="stylesheet" href="/static/css/style.css"/>
	</head>
	<body>
		{ children... }
		<script src="/_gospa/runtime.js" type="module"></script>
		<script data-gospa-islands></script>
	</body>
	</html>
}
`

	path := filepath.Join(config.OutputDir, "routes", "root_layout.templ")
	return os.WriteFile(path, []byte(content), 0600)
}

func createErrorPage(config *ProjectConfig) error {
	content := `package routes

templ Error(err string, code string) {
	<div class="error-container" style="text-align: center; padding: 50px;">
		<h1>Error { code }</h1>
		<p>{ err }</p>
		<a href="/" style="display: inline-block; margin-top: 20px;">Return Home</a>
	</div>
}
`

	path := filepath.Join(config.OutputDir, "routes", "_error.templ")
	return os.WriteFile(path, []byte(content), 0600)
}

func createMiddleware(config *ProjectConfig) error {
	content := `package routes

import (
	"github.com/gofiber/fiber/v3"
)

// Middleware applies to all routes in this directory and below
func Middleware(c fiber.Ctx) error {
	// Add your custom middleware logic here (e.g., Auth checking)
	return c.Next()
}
`

	path := filepath.Join(config.OutputDir, "routes", "_middleware.go")
	return os.WriteFile(path, []byte(content), 0600)
}

func createPackageJSON(config *ProjectConfig) error {
	pm := GetPackageManager()
	runCmd := GetRunCommand(pm)

	content := `{
	"name": "` + config.Name + `",
	"type": "module",
	"scripts": {
		"build": "` + string(runCmd) + ` run build:css",
		"build:css": "tailwindcss -i ./static/css/style.css -o ./static/css/main.css"
	},
	"devDependencies": {
		"tailwindcss": "^4.0.0",
		"@tailwindcss/cli": "^4.0.0"
	}
}
`

	path := filepath.Join(config.OutputDir, "package.json")
	return os.WriteFile(path, []byte(content), 0600)
}

func createCSSFile(config *ProjectConfig) error {
	content := `/* GoSPA Welcome Page Styles */

* {
	box-sizing: border-box;
	margin: 0;
	padding: 0;
}

body {
	font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	min-height: 100vh;
	display: flex;
	align-items: center;
	justify-content: center;
	color: #333;
}

.welcome-container {
	width: 100%;
	max-width: 600px;
	padding: 2rem;
}

.welcome-content {
	background: white;
	border-radius: 24px;
	padding: 3rem;
	text-align: center;
	box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
}

.logo {
	color: #667eea;
	margin-bottom: 1.5rem;
	display: flex;
	justify-content: center;
}

.title {
	font-size: 2.5rem;
	font-weight: 700;
	margin-bottom: 1rem;
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	-webkit-background-clip: text;
	-webkit-text-fill-color: transparent;
	background-clip: text;
}

.subtitle {
	font-size: 1.125rem;
	color: #6b7280;
	margin-bottom: 2rem;
	line-height: 1.6;
}

.actions {
	display: flex;
	gap: 1rem;
	justify-content: center;
	margin-bottom: 2.5rem;
	flex-wrap: wrap;
}

.btn {
	padding: 0.875rem 1.5rem;
	border-radius: 12px;
	font-size: 1rem;
	font-weight: 500;
	text-decoration: none;
	border: none;
	cursor: pointer;
	transition: all 0.2s ease;
}

.btn-primary {
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	color: white;
}

.btn-primary:hover {
	transform: translateY(-2px);
	box-shadow: 0 10px 20px -5px rgba(102, 126, 234, 0.4);
}

.btn-secondary {
	background: #f3f4f6;
	color: #374151;
}

.btn-secondary:hover {
	background: #e5e7eb;
	transform: translateY(-2px);
}

.features {
	display: flex;
	justify-content: center;
	gap: 2rem;
	flex-wrap: wrap;
}

.feature {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	font-size: 0.875rem;
	color: #6b7280;
}

.feature-icon {
	font-size: 1.25rem;
}

@media (max-width: 480px) {
	.welcome-content {
		padding: 2rem;
	}

	.title {
		font-size: 1.875rem;
	}

	.subtitle {
		font-size: 1rem;
	}

	.actions {
		flex-direction: column;
	}

	.btn {
		width: 100%;
	}
}
`

	path := filepath.Join(config.OutputDir, "static", "css", "style.css")
	return os.WriteFile(path, []byte(content), 0600)
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
tmp/

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
`

	path := filepath.Join(config.OutputDir, ".gitignore")
	return os.WriteFile(path, []byte(content), 0600)
}

func getGitUsername() string {
	// Try to get git username from git config
	cmd := exec.Command("git", "config", "user.name")
	if output, err := cmd.Output(); err == nil {
		username := strings.TrimSpace(string(output))
		if username != "" {
			return username
		}
	}

	// Fallback to default (we deliberately skip using user.email to avoid generating invalid missing github prefixes like "admin/myapp")

	// Fallback to default
	return "yourusername"
}

func askForModule(projectName string, nonInteractive bool) string {
	// First, try to get git username properly
	username := getGitUsername()
	if username == "yourusername" && !nonInteractive {
		// Prompt user for their GitHub username
		fmt.Print("Enter your GitHub username or organization: ")
		_, _ = fmt.Scanln(&username)
		if username == "" {
			username = "yourusername"
		}
	}

	return fmt.Sprintf("github.com/%s/%s", username, projectName)
}

// ValidateProjectName checks if a project name is valid.
func ValidateProjectName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	if strings.Contains(name, "..") {
		return fmt.Errorf("project name cannot contain '..'")
	}

	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("project name cannot contain path separators")
	}

	if strings.HasPrefix(name, "-") || strings.HasPrefix(name, "_") {
		return fmt.Errorf("project name cannot start with - or _")
	}

	if !projectNamePattern.MatchString(name) {
		return fmt.Errorf("project name can only include letters, numbers, '.', '_' or '-'")
	}

	return nil
}
