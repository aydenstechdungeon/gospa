// Package tailwind provides a Tailwind CSS v4 plugin for GoSPA.
// It supports configurable paths, content scanning, and both dev/watch and build modes.
package tailwind

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/aydenstechdungeon/gospa/plugin"
)

// Config holds Tailwind plugin configuration.
type Config struct {
	// Input is the source CSS file (default: static/css/app.css).
	Input string `yaml:"input" json:"input"`
	// Output is the compiled CSS file (default: static/dist/app.css).
	Output string `yaml:"output" json:"output"`
	// Content paths to scan for class names.
	Content []string `yaml:"content" json:"content"`
	// Minify enables CSS minification in production.
	Minify bool `yaml:"minify" json:"minify"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Input:  "static/css/app.css",
		Output: "static/dist/app.css",
		Content: []string{
			"./routes/**/*.templ",
			"./routes/**/*.go",
			"./components/**/*.templ",
			"./components/**/*.go",
			"./islands/**/*.ts",
			"./islands/**/*.js",
		},
		Minify: true,
	}
}

// TailwindPlugin provides Tailwind CSS v4 processing.
type TailwindPlugin struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	stopped bool
	config  *Config
}

// New creates a new Tailwind plugin with default configuration.
func New() *TailwindPlugin {
	return NewWithConfig(nil)
}

// NewWithConfig creates a new Tailwind plugin with the given configuration.
func NewWithConfig(cfg *Config) *TailwindPlugin {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &TailwindPlugin{config: cfg}
}

// Name returns the plugin name.
func (p *TailwindPlugin) Name() string {
	return "tailwind"
}

// Init initializes the plugin.
func (p *TailwindPlugin) Init() error {
	// Ensure output directory exists
	outputDir := filepath.Dir(p.config.Output)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	return nil
}

// Dependencies returns required Bun packages.
func (p *TailwindPlugin) Dependencies() []plugin.Dependency {
	return []plugin.Dependency{
		{Type: plugin.DepBun, Name: "tailwindcss", Version: "latest"},
		{Type: plugin.DepBun, Name: "@tailwindcss/cli", Version: "latest"},
	}
}

// OnHook handles lifecycle hooks.
func (p *TailwindPlugin) OnHook(hook plugin.Hook, ctx map[string]interface{}) error {
	switch hook {
	case plugin.BeforeDev:
		if p.isConfigured() {
			go p.watchWithContext()
		}
	case plugin.AfterDev:
		p.Stop()
	case plugin.BeforeBuild:
		if p.isConfigured() {
			return p.compile()
		}
	}
	return nil
}

// Stop gracefully stops the Tailwind watcher.
func (p *TailwindPlugin) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stopped {
		return
	}
	p.stopped = true

	if p.cancel != nil {
		p.cancel()
	}
	if p.cmd != nil && p.cmd.Process != nil {
		// Try graceful shutdown first with SIGTERM, then SIGKILL
		if err := p.cmd.Process.Signal(os.Interrupt); err != nil {
			_ = p.cmd.Process.Kill()
		}
	}
	fmt.Println("Tailwind: watcher stopped")
}

// Commands returns CLI commands.
func (p *TailwindPlugin) Commands() []plugin.Command {
	return []plugin.Command{
		{
			Name:        "add:tailwind",
			Description: "Install and configure Tailwind CSS v4",
			Action:      p.install,
		},
		{
			Name:        "tailwind:build",
			Alias:       "tw:build",
			Description: "Build Tailwind CSS for production",
			Action:      p.buildCommand,
		},
		{
			Name:        "tailwind:watch",
			Alias:       "tw:watch",
			Description: "Watch and rebuild Tailwind CSS on changes",
			Action:      p.watchCommand,
		},
	}
}

// GetConfig returns the current configuration.
func (p *TailwindPlugin) GetConfig() *Config {
	return p.config
}

// SetConfig updates the configuration.
func (p *TailwindPlugin) SetConfig(cfg *Config) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config = cfg
}

// isConfigured checks if Tailwind is properly configured.
func (p *TailwindPlugin) isConfigured() bool {
	// Check if input file exists
	if _, err := os.Stat(p.config.Input); os.IsNotExist(err) {
		return false
	}
	// Check if tailwindcss is installed
	cmd := exec.Command("bun", "pm", "ls", "tailwindcss")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// install installs and configures Tailwind CSS v4.
func (p *TailwindPlugin) install(args []string) error {
	fmt.Println("Installing Tailwind CSS v4...")

	// 1. Install dependencies with bun
	fmt.Println("Running: bun add -d tailwindcss @tailwindcss/cli")
	cmd := exec.Command("bun", "add", "-d", "tailwindcss", "@tailwindcss/cli")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install tailwind: %w", err)
	}

	// 2. Create input directory
	inputDir := filepath.Dir(p.config.Input)
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		return fmt.Errorf("failed to create input directory: %w", err)
	}

	// 3. Create output directory
	outputDir := filepath.Dir(p.config.Output)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 4. Create input CSS file if it doesn't exist
	if _, err := os.Stat(p.config.Input); os.IsNotExist(err) {
		appCSS := `@import "tailwindcss";

/* Custom theme configuration */
@theme {
  /* Add your custom theme values here */
  /* Example:
  --font-display: 'Inter', sans-serif;
  --color-primary-500: oklch(0.7 0.2 200);
  */
}

/* Your custom styles below */
`
		if err := os.WriteFile(p.config.Input, []byte(appCSS), 0644); err != nil {
			return fmt.Errorf("failed to create input CSS: %w", err)
		}
		fmt.Printf("Created %s\n", p.config.Input)
	}

	// 5. Create tailwind.config.ts if it doesn't exist
	if _, err := os.Stat("tailwind.config.ts"); os.IsNotExist(err) {
		configContent := fmt.Sprintf(`import type { Config } from 'tailwindcss';

export default {
  // Content paths to scan for class names
  content: %v,
} satisfies Config;
`, formatContentArray(p.config.Content))
		if err := os.WriteFile("tailwind.config.ts", []byte(configContent), 0644); err != nil {
			return fmt.Errorf("failed to create tailwind.config.ts: %w", err)
		}
		fmt.Println("Created tailwind.config.ts")
	}

	fmt.Println("\n✓ Tailwind CSS v4 installed!")
	fmt.Printf("  Input:  %s\n", p.config.Input)
	fmt.Printf("  Output: %s\n", p.config.Output)
	fmt.Println("\nUsage:")
	fmt.Println("  gospa dev          # Starts Tailwind watcher in dev mode")
	fmt.Println("  gospa build        # Builds minified CSS for production")
	fmt.Println("  gospa tw:watch     # Manual watch mode")
	fmt.Println("  gospa tw:build     # Manual build")
	return nil
}

// buildCommand is the CLI command for building.
func (p *TailwindPlugin) buildCommand(args []string) error {
	return p.compile()
}

// watchCommand is the CLI command for watching.
func (p *TailwindPlugin) watchCommand(args []string) error {
	p.watchWithContext()
	// Block forever
	select {}
}

// watchWithContext starts the Tailwind watcher.
func (p *TailwindPlugin) watchWithContext() {
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.mu.Unlock()

	fmt.Println("Tailwind: starting watcher...")
	fmt.Printf("  Input:  %s\n", p.config.Input)
	fmt.Printf("  Output: %s\n", p.config.Output)

	args := []string{"@tailwindcss/cli", "-i", p.config.Input, "-o", p.config.Output, "--watch"}

	// Add content paths
	for _, path := range p.config.Content {
		args = append(args, "--content", path)
	}

	cmd := exec.CommandContext(ctx, "bunx", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	p.mu.Lock()
	p.cmd = cmd
	p.mu.Unlock()

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.Canceled {
			fmt.Println("Tailwind: watcher stopped gracefully")
		} else {
			fmt.Fprintf(os.Stderr, "Tailwind watcher failed: %v\n", err)
		}
	}
}

// compile runs a single Tailwind build.
func (p *TailwindPlugin) compile() error {
	fmt.Println("Tailwind: compiling for production...")
	fmt.Printf("  Input:  %s\n", p.config.Input)
	fmt.Printf("  Output: %s\n", p.config.Output)

	args := []string{"@tailwindcss/cli", "-i", p.config.Input, "-o", p.config.Output}

	// Add content paths
	for _, path := range p.config.Content {
		args = append(args, "--content", path)
	}

	if p.config.Minify {
		args = append(args, "--minify")
	}

	cmd := exec.Command("bunx", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tailwind build failed: %w", err)
	}

	fmt.Println("Tailwind: build complete!")
	return nil
}

// formatContentArray formats content paths as a TypeScript array string.
func formatContentArray(paths []string) string {
	result := "[\n"
	for _, path := range paths {
		result += fmt.Sprintf("    '%s',\n", path)
	}
	result += "  ]"
	return result
}

// Ensure TailwindPlugin implements CLIPlugin interface.
var _ plugin.CLIPlugin = (*TailwindPlugin)(nil)
