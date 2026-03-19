// Package cli provides command-line interface tools for GoSPA.
package cli

import (
	"fmt"
	"os"
	"os/exec"
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
	if err := os.MkdirAll(config.OutputDir, 0750); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Create subdirectories
	dirs := []string{
		"routes",
		"components",
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

go 1.26.0

require (
	github.com/a-h/templ v0.3.1001
	github.com/aydenstechdungeon/gospa v0.1.29
)
`, config.Module)

	path := filepath.Join(config.OutputDir, "go.mod")
	return os.WriteFile(path, []byte(content), 0600)
}

func createMainGo(config *ProjectConfig) error {
	content := fmt.Sprintf(`package main

import (
	"log"

	_ "%s/routes" // Import routes to trigger init()

	"github.com/aydenstechdungeon/gospa"
)

func main() {
	app := gospa.New(gospa.Config{
		RoutesDir: "./routes",
		DevMode:   true,
		AppName:   "%s",
	})

	if err := app.Run(":3000"); err != nil {
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
				<a href="https://gospa.dev/docs" class="btn btn-primary" target="_blank" rel="noopener">
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
	<!DOCTYPE html>
	<html lang="en" data-gospa-auto>
	<head>
		<meta charset="UTF-8"/>
		<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
		<meta name="description" content="A GoSPA application"/>
		<meta name="theme-color" content="#667eea"/>
		<title>{ title }</title>
		
		<!-- Preconnect to improve performance on navigation -->
		<link rel="preconnect" href="/"/>

		<link rel="stylesheet" href="/static/css/style.css"/>
	</head>
	<body>
		<main>
			{ children... }
		</main>
	</body>
	</html>
}
`

	path := filepath.Join(config.OutputDir, "routes", "layout.templ")
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

	// Try git config user.email as fallback
	cmd = exec.Command("git", "config", "user.email")
	if output, err := cmd.Output(); err == nil {
		email := strings.TrimSpace(string(output))
		if email != "" {
			// Extract username from email (part before @)
			if atIndex := strings.Index(email, "@"); atIndex > 0 {
				return email[:atIndex]
			}
		}
	}

	// Fallback to default
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
