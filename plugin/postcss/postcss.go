// Package postcss provides a PostCSS plugin for GoSPA with Tailwind CSS v4 support.
// It processes CSS through PostCSS with configurable plugins.
package postcss

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/aydenstechdungeon/gospa/plugin"
)

// PostCSSPlugin provides PostCSS processing with Tailwind CSS v4 support.
type PostCSSPlugin struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	stopped bool
	config  *Config
}

// Config holds PostCSS plugin configuration.
type Config struct {
	// Input is the source CSS file (default: styles/main.css).
	Input string `yaml:"input" json:"input"`
	// Output is the processed CSS file (default: static/css/main.css).
	Output string `yaml:"output" json:"output"`
	// Watch enables watch mode for development.
	Watch bool `yaml:"watch" json:"watch"`
	// Minify enables CSS minification in production.
	Minify bool `yaml:"minify" json:"minify"`
	// SourceMap enables source map generation.
	SourceMap bool `yaml:"sourceMap" json:"sourceMap"`
	// Plugins configures which PostCSS plugins to enable.
	Plugins PluginConfig `yaml:"plugins" json:"plugins"`
}

// PluginConfig configures individual PostCSS plugins.
// Note: ContainerQueries and LineClamp are built into Tailwind v4.
type PluginConfig struct {
	// Typography enables @tailwindcss/typography (prose classes).
	Typography bool `yaml:"typography" json:"typography"`
	// Forms enables @tailwindcss/forms (form styling).
	Forms bool `yaml:"forms" json:"forms"`
	// AspectRatio enables @tailwindcss/aspect-ratio.
	AspectRatio bool `yaml:"aspectRatio" json:"aspectRatio"`
	// Autoprefixer enables autoprefixer for vendor prefixes.
	Autoprefixer bool `yaml:"autoprefixer" json:"autoprefixer"`
	// CSSNano enables cssnano for minification.
	CSSNano bool `yaml:"cssnano" json:"cssnano"`
	// PostCSSNested enables postcss-nested for nested CSS.
	PostCSSNested bool `yaml:"postcssNested" json:"postcssNested"`
}

// DefaultConfig returns the default PostCSS plugin configuration.
func DefaultConfig() *Config {
	return &Config{
		Input:     "styles/main.css",
		Output:    "static/css/main.css",
		Watch:     true,
		Minify:    true,
		SourceMap: true,
		Plugins: PluginConfig{
			Typography:    true,
			Forms:         true,
			AspectRatio:   true,
			Autoprefixer:  true,
			CSSNano:       false, // Use Tailwind's minify instead
			PostCSSNested: true,
		},
	}
}

// New creates a new PostCSS plugin with default configuration.
func New() *PostCSSPlugin {
	return NewWithConfig(nil)
}

// NewWithConfig creates a new PostCSS plugin with the given configuration.
func NewWithConfig(cfg *Config) *PostCSSPlugin {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &PostCSSPlugin{config: cfg}
}

// Name returns the plugin name.
func (p *PostCSSPlugin) Name() string {
	return "postcss"
}

// Init initializes the PostCSS plugin.
func (p *PostCSSPlugin) Init() error {
	// Create output directory
	outputDir := filepath.Dir(p.config.Output)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	// Create input directory
	inputDir := filepath.Dir(p.config.Input)
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		return fmt.Errorf("failed to create input directory: %w", err)
	}
	return nil
}

// Dependencies returns the required Bun packages for PostCSS.
func (p *PostCSSPlugin) Dependencies() []plugin.Dependency {
	deps := []plugin.Dependency{
		{Type: plugin.DepBun, Name: "postcss", Version: "latest"},
		{Type: plugin.DepBun, Name: "postcss-cli", Version: "latest"},
		{Type: plugin.DepBun, Name: "@tailwindcss/postcss", Version: "latest"},
	}

	if p.config.Plugins.Typography {
		deps = append(deps, plugin.Dependency{
			Type: plugin.DepBun, Name: "@tailwindcss/typography", Version: "latest",
		})
	}
	if p.config.Plugins.Forms {
		deps = append(deps, plugin.Dependency{
			Type: plugin.DepBun, Name: "@tailwindcss/forms", Version: "latest",
		})
	}
	if p.config.Plugins.AspectRatio {
		deps = append(deps, plugin.Dependency{
			Type: plugin.DepBun, Name: "@tailwindcss/aspect-ratio", Version: "latest",
		})
	}
	if p.config.Plugins.Autoprefixer {
		deps = append(deps, plugin.Dependency{
			Type: plugin.DepBun, Name: "autoprefixer", Version: "latest",
		})
	}
	if p.config.Plugins.CSSNano {
		deps = append(deps, plugin.Dependency{
			Type: plugin.DepBun, Name: "cssnano", Version: "latest",
		})
	}
	if p.config.Plugins.PostCSSNested {
		deps = append(deps, plugin.Dependency{
			Type: plugin.DepBun, Name: "postcss-nested", Version: "latest",
		})
	}

	return deps
}

// OnHook handles lifecycle hooks.
func (p *PostCSSPlugin) OnHook(hook plugin.Hook, ctx map[string]interface{}) error {
	projectDir, _ := ctx["project_dir"].(string)
	if projectDir == "" {
		projectDir = "."
	}

	switch hook {
	case plugin.BeforeDev:
		// Generate config
		if err := p.generatePostCSSConfig(projectDir); err != nil {
			return fmt.Errorf("failed to generate PostCSS config: %w", err)
		}
		// Create input CSS if needed
		cssPath := filepath.Join(projectDir, p.config.Input)
		if _, err := os.Stat(cssPath); os.IsNotExist(err) {
			if err := p.generateMainCSS(cssPath); err != nil {
				return fmt.Errorf("failed to generate main CSS: %w", err)
			}
		}
		// Start watcher
		if p.config.Watch {
			go p.watchWithContext(projectDir)
		}

	case plugin.BeforeBuild:
		// Generate config
		if err := p.generatePostCSSConfig(projectDir); err != nil {
			return fmt.Errorf("failed to generate PostCSS config: %w", err)
		}
		// Build once
		return p.compile(projectDir)

	case plugin.AfterDev:
		p.Stop()
	}

	return nil
}

// Stop gracefully stops the PostCSS watcher.
func (p *PostCSSPlugin) Stop() {
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
		// Try graceful shutdown first with SIGINT, then force kill
		if err := p.cmd.Process.Signal(os.Interrupt); err != nil {
			_ = p.cmd.Process.Kill()
		}
	}
	fmt.Println("PostCSS: watcher stopped")
}

// Commands returns custom CLI commands.
func (p *PostCSSPlugin) Commands() []plugin.Command {
	return []plugin.Command{
		{
			Name:        "add:postcss",
			Description: "Install and configure PostCSS with Tailwind CSS v4",
			Action:      p.install,
		},
		{
			Name:        "postcss:build",
			Alias:       "pc:build",
			Description: "Build CSS with PostCSS",
			Action:      p.buildCommand,
		},
		{
			Name:        "postcss:watch",
			Alias:       "pc:watch",
			Description: "Watch and rebuild CSS on changes",
			Action:      p.watchCommand,
		},
		{
			Name:        "postcss:config",
			Alias:       "pc:config",
			Description: "Generate PostCSS configuration file",
			Action:      p.configCommand,
		},
	}
}

// GetConfig returns the current configuration.
func (p *PostCSSPlugin) GetConfig() *Config {
	return p.config
}

// SetConfig updates the configuration.
func (p *PostCSSPlugin) SetConfig(cfg *Config) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config = cfg
}

// install installs and configures PostCSS.
func (p *PostCSSPlugin) install(args []string) error {
	fmt.Println("Installing PostCSS with Tailwind CSS v4...")

	// Install dependencies
	fmt.Println("Running: bun add -d postcss postcss-cli @tailwindcss/postcss")
	cmd := exec.Command("bun", "add", "-d", "postcss", "postcss-cli", "@tailwindcss/postcss")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install postcss: %w", err)
	}

	// Install optional plugins
	if p.config.Plugins.Typography {
		fmt.Println("Installing @tailwindcss/typography...")
		cmd = exec.Command("bun", "add", "-d", "@tailwindcss/typography")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install typography: %w", err)
		}
	}
	if p.config.Plugins.Forms {
		fmt.Println("Installing @tailwindcss/forms...")
		cmd = exec.Command("bun", "add", "-d", "@tailwindcss/forms")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install forms: %w", err)
		}
	}
	if p.config.Plugins.AspectRatio {
		fmt.Println("Installing @tailwindcss/aspect-ratio...")
		cmd = exec.Command("bun", "add", "-d", "@tailwindcss/aspect-ratio")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install aspect-ratio: %w", err)
		}
	}
	if p.config.Plugins.Autoprefixer {
		fmt.Println("Installing autoprefixer...")
		cmd = exec.Command("bun", "add", "-d", "autoprefixer")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install autoprefixer: %w", err)
		}
	}
	if p.config.Plugins.PostCSSNested {
		fmt.Println("Installing postcss-nested...")
		cmd = exec.Command("bun", "add", "-d", "postcss-nested")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install postcss-nested: %w", err)
		}
	}

	// Generate config
	if err := p.generatePostCSSConfig("."); err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}

	// Create input CSS if needed
	cssPath := p.config.Input
	if _, err := os.Stat(cssPath); os.IsNotExist(err) {
		if err := p.generateMainCSS(cssPath); err != nil {
			return fmt.Errorf("failed to generate main CSS: %w", err)
		}
	}

	// Create output directory
	outputDir := filepath.Dir(p.config.Output)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	fmt.Println("\n✓ PostCSS installed!")
	fmt.Printf("  Input:  %s\n", p.config.Input)
	fmt.Printf("  Output: %s\n", p.config.Output)
	fmt.Println("\nUsage:")
	fmt.Println("  gospa dev          # Starts PostCSS watcher in dev mode")
	fmt.Println("  gospa build        # Builds CSS for production")
	fmt.Println("  gospa pc:watch     # Manual watch mode")
	fmt.Println("  gospa pc:build     # Manual build")
	return nil
}

// buildCommand is the CLI command for building.
func (p *PostCSSPlugin) buildCommand(args []string) error {
	projectDir := "."
	if len(args) > 0 {
		projectDir = args[0]
	}
	return p.compile(projectDir)
}

// watchCommand is the CLI command for watching.
func (p *PostCSSPlugin) watchCommand(args []string) error {
	projectDir := "."
	if len(args) > 0 {
		projectDir = args[0]
	}
	p.watchWithContext(projectDir)
	select {}
}

// configCommand generates the PostCSS config.
func (p *PostCSSPlugin) configCommand(args []string) error {
	projectDir := "."
	if len(args) > 0 {
		projectDir = args[0]
	}
	return p.generatePostCSSConfig(projectDir)
}

// watchWithContext starts the PostCSS watcher.
func (p *PostCSSPlugin) watchWithContext(projectDir string) {
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.mu.Unlock()

	fmt.Println("PostCSS: starting watcher...")
	fmt.Printf("  Input:  %s\n", p.config.Input)
	fmt.Printf("  Output: %s\n", p.config.Output)

	args := []string{
		"postcss",
		p.config.Input,
		"--output", p.config.Output,
		"--config", projectDir,
		"--watch",
	}

	if p.config.SourceMap {
		args = append(args, "--map")
	}

	cmd := exec.CommandContext(ctx, "bunx", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	p.mu.Lock()
	p.cmd = cmd
	p.mu.Unlock()

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.Canceled {
			fmt.Println("PostCSS: watcher stopped gracefully")
		} else {
			fmt.Fprintf(os.Stderr, "PostCSS watcher failed: %v\n", err)
		}
	}
}

// compile runs a single PostCSS build.
func (p *PostCSSPlugin) compile(projectDir string) error {
	fmt.Println("PostCSS: compiling...")
	fmt.Printf("  Input:  %s\n", p.config.Input)
	fmt.Printf("  Output: %s\n", p.config.Output)

	args := []string{
		"postcss",
		p.config.Input,
		"--output", p.config.Output,
		"--config", projectDir,
	}

	if p.config.SourceMap {
		args = append(args, "--map")
	}

	cmd := exec.Command("bunx", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("postcss build failed: %w", err)
	}

	fmt.Println("PostCSS: build complete!")
	return nil
}

// generatePostCSSConfig generates a postcss.config.js file.
func (p *PostCSSPlugin) generatePostCSSConfig(projectDir string) error {
	configPath := filepath.Join(projectDir, "postcss.config.js")

	content := `// PostCSS configuration for GoSPA
// Generated by postcss plugin

export default {
  plugins: {
`

	// Tailwind CSS v4 PostCSS plugin (required first)
	content += "    '@tailwindcss/postcss': {},\n"

	// PostCSS Nested
	if p.config.Plugins.PostCSSNested {
		content += "    'postcss-nested': {},\n"
	}

	// Tailwind CSS extensions (no longer PostCSS plugins in v4)
	// Typography, Forms, AspectRatio are now loaded via @plugin in CSS

	// Autoprefixer
	if p.config.Plugins.Autoprefixer {
		content += "    'autoprefixer': {},\n"
	}

	// CSSNano for minification
	if p.config.Plugins.CSSNano {
		content += `    'cssnano': {
      preset: ['default', { discardComments: { removeAll: true } }]
    },
`
	}

	content += `  }
};
`

	return os.WriteFile(configPath, []byte(content), 0644)
}

// generateMainCSS generates a main CSS file with Tailwind imports.
func (p *PostCSSPlugin) generateMainCSS(cssPath string) error {
	content := `/* Main CSS file for GoSPA */
/* Processed by PostCSS with Tailwind CSS v4 */

@import 'tailwindcss';

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

	// Add plugin-specific comments and @plugin imports
	if p.config.Plugins.Typography {
		content += `

/* Typography plugin enabled - use prose classes:
   prose, prose-sm, prose-lg, prose-xl, prose-2xl
   prose-headings, prose-lead, prose-img, etc.
*/
@plugin "@tailwindcss/typography";
`
	}

	if p.config.Plugins.Forms {
		content += `

/* Forms plugin enabled - form elements styled automatically
   Use form-input, form-textarea, form-select, form-checkbox, form-radio
*/
@plugin "@tailwindcss/forms";
`
	}

	if p.config.Plugins.AspectRatio {
		content += `

/* Aspect Ratio plugin enabled - use aspect-{ratio} classes
   aspect-video, aspect-square, aspect-[4/3], etc.
*/
@plugin "@tailwindcss/aspect-ratio";
`
	}

	// Ensure directory exists
	dir := filepath.Dir(cssPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(cssPath, []byte(content), 0644)
}

// Ensure PostCSSPlugin implements CLIPlugin interface.
var _ plugin.CLIPlugin = (*PostCSSPlugin)(nil)
