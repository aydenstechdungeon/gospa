// Package postcss provides a PostCSS plugin for GoSPA with Tailwind CSS v4 support.
// It processes CSS through PostCSS with configurable plugins.
package postcss

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/aydenstechdungeon/gospa/plugin"
	"gopkg.in/yaml.v3"
)

// PostCSSPlugin provides PostCSS processing with Tailwind CSS v4 support.
//
//nolint:revive // changing name would break API
type PostCSSPlugin struct {
	mu      sync.Mutex
	cmds    []*exec.Cmd
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
	// CriticalCSS enables critical CSS extraction for above-the-fold content.
	CriticalCSS CriticalCSSConfig `yaml:"criticalCSS" json:"criticalCSS"`
	// Bundles defines multiple CSS entry points for code splitting.
	Bundles []BundleEntry `yaml:"bundles" json:"bundles"`
}

// CriticalCSSConfig configures critical CSS extraction.
type CriticalCSSConfig struct {
	// Enabled enables critical CSS extraction.
	Enabled bool `yaml:"enabled" json:"enabled"`
	// CriticalOutput is the path for critical CSS (inlined in HTML).
	CriticalOutput string `yaml:"criticalOutput" json:"criticalOutput"`
	// NonCriticalOutput is the path for non-critical CSS (async loaded).
	NonCriticalOutput string `yaml:"nonCriticalOutput" json:"nonCriticalOutput"`
	// Dimensions defines viewport sizes for critical CSS detection.
	// Default: 1300x900 (desktop), 500x900 (mobile).
	Dimensions []Dimension `yaml:"dimensions" json:"dimensions"`
	// InlineMaxSize is the max size (in bytes) for inlining critical CSS.
	// Default: 14KB (gzip) for single round-trip.
	InlineMaxSize int `yaml:"inlineMaxSize" json:"inlineMaxSize"`
	// CriticalContent lists template file paths/globs (e.g., "../routes/*.templ")
	// whose classes are above-the-fold and should be prioritized in critical CSS.
	// When empty, falls back to byte-size-only splitting.
	CriticalContent []string `yaml:"criticalContent" json:"criticalContent"`
}

// Dimension defines a viewport size for critical CSS extraction.
type Dimension struct {
	Width  int    `yaml:"width" json:"width"`
	Height int    `yaml:"height" json:"height"`
	Name   string `yaml:"name" json:"name"`
}

// BundleEntry defines a CSS bundle for code splitting.
type BundleEntry struct {
	// Name is the bundle identifier (e.g., "marketing", "dashboard").
	Name string `yaml:"name" json:"name"`
	// Input is the source CSS file.
	Input string `yaml:"input" json:"input"`
	// Output is the processed CSS file.
	Output string `yaml:"output" json:"output"`
	// Content paths for Tailwind to scan (globs).
	Content []string `yaml:"content" json:"content"`
	// CriticalCSS enables critical extraction for this bundle.
	CriticalCSS *CriticalCSSConfig `yaml:"criticalCSS" json:"criticalCSS"`
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
		CriticalCSS: CriticalCSSConfig{
			Enabled:           false,
			CriticalOutput:    "static/css/critical.css",
			NonCriticalOutput: "static/css/non-critical.css",
			Dimensions: []Dimension{
				{Width: 1300, Height: 900, Name: "desktop"},
				{Width: 500, Height: 900, Name: "mobile"},
			},
			InlineMaxSize: 14336, // 14KB for single round-trip
		},
		Bundles: nil,
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
		loadConfigFromYaml(cfg)
	}
	return &PostCSSPlugin{config: cfg}
}

// loadConfigFromYaml reads standard configurations if gospa.yaml is present.
func loadConfigFromYaml(cfg *Config) {
	data, err := os.ReadFile("gospa.yaml")
	if err != nil {
		// Try root directory if not found in current (helpful for some build setups)
		data, err = os.ReadFile("../../gospa.yaml")
		if err != nil {
			return
		}
	}

	// Wrapper to match the gospa.yaml structure
	var wrapper struct {
		Plugins struct {
			PostCSS Config `yaml:"postcss"`
		} `yaml:"plugins"`
	}

	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		return
	}

	pcfg := wrapper.Plugins.PostCSS

	// Merge values if they are set in YAML
	if pcfg.Input != "" {
		cfg.Input = pcfg.Input
	}
	if pcfg.Output != "" {
		cfg.Output = pcfg.Output
	}
	cfg.Watch = pcfg.Watch
	cfg.Minify = pcfg.Minify
	cfg.SourceMap = pcfg.SourceMap
	cfg.Plugins = pcfg.Plugins

	// Merge critical CSS config
	if pcfg.CriticalCSS.Enabled {
		cfg.CriticalCSS = pcfg.CriticalCSS
		if cfg.CriticalCSS.InlineMaxSize == 0 {
			cfg.CriticalCSS.InlineMaxSize = 32768 // Increase default to 32KB
		}
	}

	// Merge bundles
	if len(pcfg.Bundles) > 0 {
		cfg.Bundles = pcfg.Bundles
		for i := range cfg.Bundles {
			if cfg.Bundles[i].CriticalCSS != nil && cfg.Bundles[i].CriticalCSS.Enabled {
				if cfg.Bundles[i].CriticalCSS.InlineMaxSize == 0 {
					cfg.Bundles[i].CriticalCSS.InlineMaxSize = 32768
				}
			}
		}
	}
}

// Name returns the plugin name.
func (p *PostCSSPlugin) Name() string {
	return "postcss"
}

// Init initializes the PostCSS plugin.
func (p *PostCSSPlugin) Init() error {
	// Create output directory
	outputDir := filepath.Dir(p.config.Output)
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	// Create input directory
	inputDir := filepath.Dir(p.config.Input)
	if err := os.MkdirAll(inputDir, 0750); err != nil {
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
	if p.config.Plugins.CSSNano || p.config.Minify {
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
		// Build
		if len(p.config.Bundles) > 0 {
			return p.bundlesCommand([]string{projectDir})
		}
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
	for _, cmd := range p.cmds {
		if cmd != nil && cmd.Process != nil {
			// Try graceful shutdown first with SIGINT, then force kill
			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				_ = cmd.Process.Kill()
			}
		}
	}
	fmt.Println("PostCSS: watcher(s) stopped")
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
			Alias:       "pb",
			Description: "Build CSS with PostCSS",
			Action:      p.buildCommand,
		},
		{
			Name:        "postcss:watch",
			Alias:       "pw",
			Description: "Watch and rebuild CSS on changes",
			Action:      p.watchCommand,
		},
		{
			Name:        "postcss:config",
			Alias:       "pc",
			Description: "Generate PostCSS configuration file",
			Action:      p.configCommand,
		},
		{
			Name:        "postcss:critical",
			Alias:       "pcr",
			Description: "Extract critical CSS for above-the-fold content",
			Action:      p.criticalCommand,
		},
		{
			Name:        "postcss:bundles",
			Alias:       "pbd",
			Description: "Build all CSS bundles",
			Action:      p.bundlesCommand,
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
func (p *PostCSSPlugin) install(_ []string) error {
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
	if err := os.MkdirAll(outputDir, 0750); err != nil {
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

	// Stop existing watchers if running to prevent process and context leaks
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	for _, cmd := range p.cmds {
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Signal(os.Interrupt)
		}
	}
	p.cmds = nil

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.mu.Unlock()

	fmt.Println("PostCSS: starting watcher(s)...")

	startWatcher := func(input, output string) {
		fmt.Printf("  Watching: %s -> %s\n", input, output)

		args := []string{
			"postcss",
			input,
			"--output", output,
			"--config", projectDir,
			"--watch",
		}

		if p.config.SourceMap {
			args = append(args, "--map")
		}

		cmd := exec.CommandContext(ctx, "bun", append([]string{"x"}, args...)...) // #nosec //nolint:gosec
		cmd.Dir = projectDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		p.mu.Lock()
		p.cmds = append(p.cmds, cmd)
		p.mu.Unlock()

		if err := cmd.Run(); err != nil {
			if ctx.Err() == context.Canceled {
				fmt.Printf("PostCSS: watcher for %s stopped gracefully\n", input)
			} else {
				fmt.Fprintf(os.Stderr, "PostCSS watcher for %s failed: %v\n", input, err)
			}
		}
	}

	if len(p.config.Bundles) > 0 {
		for _, bundle := range p.config.Bundles {
			go startWatcher(bundle.Input, bundle.Output)
		}
	} else {
		go startWatcher(p.config.Input, p.config.Output)
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

	cmd := exec.Command("bun", append([]string{"x"}, args...)...) // #nosec //nolint:gosec
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
	if p.config.Plugins.CSSNano || p.config.Minify {
		content += `    'cssnano': {
      preset: ['default', { discardComments: { removeAll: true } }]
    },
`
	}

	content += `  }
};
`

	return os.WriteFile(configPath, []byte(content), 0600)
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
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	return os.WriteFile(cssPath, []byte(content), 0600)
}

// criticalCommand extracts critical CSS for above-the-fold content.
func (p *PostCSSPlugin) criticalCommand(args []string) error {
	projectDir := "."
	if len(args) > 0 {
		projectDir = args[0]
	}

	// Enable critical CSS extraction
	p.config.CriticalCSS.Enabled = true

	fmt.Println("PostCSS: extracting critical CSS...")
	fmt.Printf("  Input:  %s\n", p.config.Input)
	fmt.Printf("  Critical: %s\n", p.config.CriticalCSS.CriticalOutput)
	fmt.Printf("  Non-critical: %s\n", p.config.CriticalCSS.NonCriticalOutput)

	// First, build the full CSS
	if err := p.compile(projectDir); err != nil {
		return fmt.Errorf("failed to compile CSS: %w", err)
	}

	// Read the compiled CSS
	fullCSS, err := os.ReadFile(p.config.Output)
	if err != nil {
		return fmt.Errorf("failed to read compiled CSS: %w", err)
	}

	// Create output directories
	criticalDir := filepath.Dir(p.config.CriticalCSS.CriticalOutput)
	if err := os.MkdirAll(criticalDir, 0750); err != nil {
		return fmt.Errorf("failed to create critical CSS directory: %w", err)
	}

	nonCriticalDir := filepath.Dir(p.config.CriticalCSS.NonCriticalOutput)
	if err := os.MkdirAll(nonCriticalDir, 0750); err != nil {
		return fmt.Errorf("failed to create non-critical CSS directory: %w", err)
	}

	// Use shared splitting logic
	criticalClasses := extractClassesFromTempl(projectDir, p.config.CriticalCSS.CriticalContent)
	criticalCSS, nonCriticalCSS := splitCSS(fullCSS, p.config.CriticalCSS.InlineMaxSize, criticalClasses)

	// Use cleaned path to prevent path traversal
	criticalOutputPath := filepath.Clean(p.config.CriticalCSS.CriticalOutput)
	nonCriticalOutputPath := filepath.Clean(p.config.CriticalCSS.NonCriticalOutput)

	// Ensure paths do not escape the project directory
	if !p.isPathSafe(projectDir, criticalOutputPath) || !p.isPathSafe(projectDir, nonCriticalOutputPath) {
		return fmt.Errorf("safety violation: critical CSS paths must be within project directory")
	}

	if err := os.WriteFile(criticalOutputPath, criticalCSS, 0600); err != nil { // #nosec //nolint:gosec // path from config, validated
		return fmt.Errorf("failed to write critical CSS: %w", err)
	}

	// Write non-critical CSS
	if err := os.WriteFile(nonCriticalOutputPath, nonCriticalCSS, 0600); err != nil { // #nosec //nolint:gosec // path from config, validated
		return fmt.Errorf("failed to write non-critical CSS: %w", err)
	}

	fmt.Println("✓ Critical CSS extracted!")
	fmt.Printf("  Critical:    %s (%d bytes)\n", p.config.CriticalCSS.CriticalOutput, len(criticalCSS))
	fmt.Printf("  Non-critical: %s (%d bytes)\n", p.config.CriticalCSS.NonCriticalOutput, len(nonCriticalCSS))

	// Print usage example
	fmt.Println("\nUsage in templates:")
	fmt.Println("  {{ CriticalCSS . }}  // Inline critical CSS")
	fmt.Println("  {{ AsyncCSS . \"/css/non-critical.css\" }}  // Async load non-critical")

	return nil
}

// bundlesCommand builds all CSS bundles defined in config.
func (p *PostCSSPlugin) bundlesCommand(args []string) error {
	projectDir := "."
	if len(args) > 0 {
		projectDir = args[0]
	}

	if len(p.config.Bundles) == 0 {
		fmt.Println("No bundles defined in config. Building main bundle only.")
		return p.compile(projectDir)
	}

	fmt.Printf("PostCSS: building %d bundles...\n", len(p.config.Bundles))

	for _, bundle := range p.config.Bundles {
		fmt.Printf("\n  Building: %s\n", bundle.Name)
		fmt.Printf("    Input:  %s\n", bundle.Input)
		fmt.Printf("    Output: %s\n", bundle.Output)

		// Generate bundle-specific CSS file if it doesn't exist
		inputPath := filepath.Join(projectDir, bundle.Input)
		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			if err := p.generateBundleCSS(inputPath, bundle); err != nil {
				return fmt.Errorf("failed to generate bundle CSS for %s: %w", bundle.Name, err)
			}
		}

		// Compile the bundle
		args := []string{
			"postcss",
			bundle.Input,
			"--output", bundle.Output,
			"--config", projectDir,
		}

		if p.config.SourceMap {
			args = append(args, "--map")
		}

		cmd := exec.Command("bun", append([]string{"x"}, args...)...) // #nosec //nolint:gosec
		cmd.Dir = projectDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to build bundle %s: %w", bundle.Name, err)
		}

		// If critical CSS is enabled for this bundle, extract it
		if bundle.CriticalCSS != nil && bundle.CriticalCSS.Enabled {
			if err := p.extractCriticalForBundle(projectDir, bundle); err != nil {
				return fmt.Errorf("failed to extract critical CSS for %s: %w", bundle.Name, err)
			}
		}
	}

	fmt.Println("\n✓ All bundles built successfully!")
	return nil
}

// generateBundleCSS generates a bundle-specific CSS file.
func (p *PostCSSPlugin) generateBundleCSS(cssPath string, bundle BundleEntry) error {
	content := fmt.Sprintf(`/* CSS bundle: %s */
	/* Processed by PostCSS with Tailwind CSS v4 */
	
	@import 'tailwindcss';
	`, bundle.Name)

	// Add @source directives for bundle-specific content
	if len(bundle.Content) > 0 {
		content += "\n/* Content paths for this bundle */\n"
		for _, path := range bundle.Content {
			content += fmt.Sprintf("@source \"%s\";\n", path)
		}
	}

	content += `
	/* Custom theme configuration */
	@theme {
	  /* Add your custom theme values here */
	}
	
	/* Your custom styles below */
	`

	// Ensure directory exists
	dir := filepath.Dir(cssPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	return os.WriteFile(cssPath, []byte(content), 0600)
}

// extractCriticalForBundle extracts critical CSS for a specific bundle.
func (p *PostCSSPlugin) extractCriticalForBundle(projectDir string, bundle BundleEntry) error {
	fullCSSPath := filepath.Join(projectDir, bundle.Output)
	fullCSS, err := os.ReadFile(filepath.Clean(fullCSSPath))
	if err != nil {
		return fmt.Errorf("failed to read bundle CSS: %w", err)
	}

	// Determine critical/non-critical output paths
	criticalOutput := bundle.CriticalCSS.CriticalOutput
	if criticalOutput == "" {
		ext := filepath.Ext(bundle.Output)
		base := bundle.Output[:len(bundle.Output)-len(ext)]
		criticalOutput = base + ".critical" + ext
	}

	nonCriticalOutput := bundle.CriticalCSS.NonCriticalOutput
	if nonCriticalOutput == "" {
		ext := filepath.Ext(bundle.Output)
		base := bundle.Output[:len(bundle.Output)-len(ext)]
		nonCriticalOutput = base + ".non-critical" + ext
	}

	// Validate paths to prevent traversal
	cleanCriticalOutput := filepath.Clean(criticalOutput)
	cleanNonCriticalOutput := filepath.Clean(nonCriticalOutput)
	outputDir := filepath.Dir(bundle.Output)
	if !strings.HasPrefix(cleanCriticalOutput, filepath.Clean(outputDir)) || !strings.HasPrefix(cleanNonCriticalOutput, filepath.Clean(outputDir)) {
		return fmt.Errorf("invalid output path: critical=%s, non-critical=%s", criticalOutput, nonCriticalOutput)
	}

	// Create output directories
	criticalDir := filepath.Dir(cleanCriticalOutput)
	if err := os.MkdirAll(criticalDir, 0750); err != nil {
		return fmt.Errorf("failed to create critical CSS directory: %w", err)
	}

	nonCriticalDir := filepath.Dir(cleanNonCriticalOutput)
	if err := os.MkdirAll(nonCriticalDir, 0750); err != nil {
		return fmt.Errorf("failed to create non-critical CSS directory: %w", err)
	}

	// Use shared safe splitting logic
	criticalSize := bundle.CriticalCSS.InlineMaxSize
	criticalClasses := extractClassesFromTempl(projectDir, bundle.CriticalCSS.CriticalContent)
	criticalCSS, nonCriticalCSS := splitCSS(fullCSS, criticalSize, criticalClasses)

	// Write critical CSS
	if err := os.WriteFile(cleanCriticalOutput, criticalCSS, 0600); err != nil { // #nosec //nolint:gosec // path validated above
		return fmt.Errorf("failed to write critical CSS: %w", err)
	}

	// Write non-critical CSS
	if err := os.WriteFile(cleanNonCriticalOutput, nonCriticalCSS, 0600); err != nil { // #nosec //nolint:gosec // path validated above
		return fmt.Errorf("failed to write non-critical CSS: %w", err)
	}

	fmt.Printf("    ✓ Critical: %s (%d bytes)\n", criticalOutput, len(criticalCSS))
	fmt.Printf("    ✓ Non-critical: %s (%d bytes)\n", nonCriticalOutput, len(nonCriticalCSS))

	return nil
}

// GenerateCriticalCSSHelper generates a Go helper function for templating.
// This can be used in your template files to inline critical CSS.
func GenerateCriticalCSSHelper(projectDir, criticalCSSPath string) (string, error) {
	fullPath := filepath.Join(projectDir, criticalCSSPath)
	// #nosec //nolint:gosec
	css, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read critical CSS: %w", err)
	}
	return string(css), nil
}

// GenerateAsyncCSSScript generates the HTML for async loading non-critical CSS.
func GenerateAsyncCSSScript(cssPath string) string {
	return fmt.Sprintf(`<link rel="preload" href="%s" as="style" onload="this.onload=null;this.rel='stylesheet'" data-gospa-async-css="1">
<noscript><link rel="stylesheet" href="%s"></noscript>`, cssPath, cssPath)
}

// CriticalCSS reads the critical CSS file and returns its content for inlining in templates.
// This is a template helper that can be used directly in templ files.
// Usage in templates: @CriticalCSS("./static/css/critical.css")
func CriticalCSS(path string) string {
	// Try to read from the file system at runtime
	// This allows the critical CSS to be extracted at build time and read at runtime
	// #nosec //nolint:gosec
	css, err := os.ReadFile(path)
	if err != nil {
		// Return empty string if file doesn't exist (will be handled gracefully)
		return ""
	}
	return string(css)
}

// AsyncCSS generates the HTML markup for async loading non-critical CSS.
// This is a template helper that can be used directly in templ files.
// Usage in templates: @templ.Raw(AsyncCSS("/static/css/non-critical.css"))
func AsyncCSS(path string) string {
	return GenerateAsyncCSSScript(path)
}

// CriticalCSSWithFallback returns critical CSS from the given path,
// or returns a fallback message if the file doesn't exist.
// Useful for development where critical CSS might not be extracted yet.
func CriticalCSSWithFallback(path, fallback string) string {
	// #nosec //nolint:gosec
	css, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	return string(css)
}

// classRegex matches class="..." with literal string values in templ files.
var classRegex = regexp.MustCompile(`class="([^"]*)"`)

// extractClassesFromTempl extracts CSS class names from templ template files
// matching the provided glob patterns. It parses class="string" attributes
// and returns a set of individual class names.
func extractClassesFromTempl(projectDir string, globs []string) map[string]bool {
	classes := make(map[string]bool)

	for _, pattern := range globs {
		if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(projectDir, pattern)
		}

		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, path := range matches {
			data, err := os.ReadFile(filepath.Clean(path)) // #nosec //nolint:gosec // glob from config
			if err != nil {
				continue
			}

			for _, match := range classRegex.FindAllSubmatch(data, -1) {
				raw := string(match[1])
				for _, cls := range strings.Fields(raw) {
					if cls != "" {
						classes[cls] = true
					}
				}
			}
		}
	}

	return classes
}

// classToEscapedSelector converts a class name to its Tailwind CSS selector form.
// Tailwind v4 escapes special characters: / → \/, [ → \[, ] → \], : → \:, etc.
func classToEscapedSelector(className string) string {
	var b strings.Builder
	b.WriteByte('.')
	for _, ch := range className {
		switch ch {
		case '/', '[', ']', '(', ')', ':', '.', '#', '%', ',', ';', '=', '!', '*', '~', '$', '^', '{', '}':
			b.WriteByte('\\')
			b.WriteRune(ch)
		default:
			b.WriteRune(ch)
		}
	}
	return b.String()
}

// splitCSS safely splits CSS into critical and non-critical parts at a safe boundary.
// It ensures we don't split in the middle of a rule or media query by tracking brace depth.
// When criticalClasses is non-empty, rules matching those classes are prioritized
// in the critical portion by reordering before non-matching rules.
func splitCSS(fullCSS []byte, maxSize int, criticalClasses map[string]bool) ([]byte, []byte) {
	if maxSize <= 0 {
		maxSize = 14336 // 14KB default
	}

	// If the entire CSS is smaller than the target size, no split needed
	if len(fullCSS) <= maxSize {
		return fullCSS, []byte{}
	}

	// When no critical classes provided, fall back to byte-size-only splitting
	if len(criticalClasses) == 0 {
		return splitCSSBySize(fullCSS, maxSize)
	}

	// Build set of escaped selectors from critical classes
	criticalSelectors := make(map[string]bool, len(criticalClasses))
	for cls := range criticalClasses {
		criticalSelectors[classToEscapedSelector(cls)] = true
	}

	// Find the @layer utilities block boundaries
	utilStart, utilEnd := findUtilitiesLayer(fullCSS)
	if utilStart == -1 {
		// No utilities layer found, fall back to size-only split
		return splitCSSBySize(fullCSS, maxSize)
	}

	// Split CSS into: before, @layer utilities{ wrapper, inner rules, closing }, after
	utilPrefix := []byte("@layer utilities{")
	innerStart := utilStart + len(utilPrefix)
	innerEnd := utilEnd - 1 // position of closing }

	before := fullCSS[:utilStart]
	innerContent := fullCSS[innerStart:innerEnd]
	after := fullCSS[utilEnd:]

	// Parse individual rules within the inner utilities content
	rules := parseRules(innerContent)

	// Partition rules into critical-matching and the rest
	var matched, unmatched [][]byte
	for _, rule := range rules {
		if matchesAnySelector(rule, criticalSelectors) {
			matched = append(matched, rule)
		} else {
			unmatched = append(unmatched, rule)
		}
	}

	// Reorder: matched rules first, then unmatched
	reordered := make([]byte, 0, len(innerContent))
	for _, r := range matched {
		reordered = append(reordered, r...)
	}
	for _, r := range unmatched {
		reordered = append(reordered, r...)
	}

	// Calculate how much space is available for inner utility rules in critical CSS
	// Account for: before content, "@layer utilities{", and closing "}"
	overhead := len(before) + len(utilPrefix) + 1 // +1 for closing }
	innerMaxSize := maxSize - overhead
	if innerMaxSize < 0 {
		innerMaxSize = 0
	}

	// Split the reordered inner rules at the byte boundary
	criticalInner, nonCriticalInner := splitCSSBySize(reordered, innerMaxSize)

	// Build critical CSS: before + @layer utilities{ + criticalInnerRules + }
	critical := make([]byte, 0, len(before)+len(utilPrefix)+len(criticalInner)+1)
	critical = append(critical, before...)
	critical = append(critical, utilPrefix...)
	critical = append(critical, criticalInner...)
	critical = append(critical, '}')

	// Build non-critical CSS: @layer utilities{ + nonCriticalInnerRules + } + after
	var nonCritical []byte
	if len(nonCriticalInner) > 0 {
		nonCritical = make([]byte, 0, len(utilPrefix)+len(nonCriticalInner)+1+len(after))
		nonCritical = append(nonCritical, utilPrefix...)
		nonCritical = append(nonCritical, nonCriticalInner...)
		nonCritical = append(nonCritical, '}')
		nonCritical = append(nonCritical, after...)
	}

	return critical, nonCritical
}

// splitCSSBySize splits CSS at a byte boundary, finding the first safe brace-depth-0
// position after maxSize. This is the original splitting behavior.
func splitCSSBySize(fullCSS []byte, maxSize int) ([]byte, []byte) {
	splitPoint := -1
	braceDepth := 0
	inComment := false

	for i := 0; i < len(fullCSS); i++ {
		if i > 0 && fullCSS[i-1] == '/' && fullCSS[i] == '*' {
			inComment = true
		}
		if i > 0 && fullCSS[i-1] == '*' && fullCSS[i] == '/' {
			inComment = false
		}

		if !inComment {
			if fullCSS[i] == '{' {
				braceDepth++
			} else if fullCSS[i] == '}' {
				braceDepth--
				if braceDepth == 0 && i >= maxSize {
					splitPoint = i + 1
					break
				}
			}
		}
	}

	if splitPoint == -1 || splitPoint >= len(fullCSS) {
		return fullCSS, []byte{}
	}

	return fullCSS[:splitPoint], fullCSS[splitPoint:]
}

// findUtilitiesLayer locates the @layer utilities{...} block in minified CSS.
// Returns the start and end byte offsets of the utilities block content
// (including the @layer prefix).
func findUtilitiesLayer(css []byte) (int, int) {
	prefix := []byte("@layer utilities{")
	start := bytes.Index(css, prefix)
	if start == -1 {
		return -1, -1
	}

	// Find matching closing brace
	braceDepth := 0
	inComment := false
	contentStart := start + len(prefix) - 1 // position of '{'

	for i := contentStart; i < len(css); i++ {
		if i > 0 && css[i-1] == '/' && css[i] == '*' {
			inComment = true
		}
		if i > 0 && css[i-1] == '*' && css[i] == '/' {
			inComment = false
		}

		if !inComment {
			switch css[i] {
			case '{':
				braceDepth++
			case '}':
				braceDepth--
				if braceDepth == 0 {
					return start, i + 1
				}
			}
		}
	}

	return -1, -1
}

// parseRules splits minified CSS rule content into individual rules at braceDepth==0 boundaries.
// The input should be the inner utilities content (without the @layer utilities{ wrapper).
// Each returned element is a complete CSS rule (selector + body), potentially with nested
// @supports or @media blocks.
func parseRules(css []byte) [][]byte {
	var rules [][]byte
	start := -1
	braceDepth := 0
	inComment := false

	for i := 0; i < len(css); i++ {
		if i > 0 && css[i-1] == '/' && css[i] == '*' {
			inComment = true
		}
		if i > 0 && css[i-1] == '*' && css[i] == '/' {
			inComment = false
		}

		if !inComment {
			switch css[i] {
			case '{':
				if braceDepth == 0 && start == -1 {
					start = i
				}
				braceDepth++
			case '}':
				braceDepth--
				if braceDepth == 0 && start != -1 {
					rules = append(rules, css[start:i+1])
					start = -1
				}
			default:
				if braceDepth == 0 && start == -1 {
					start = i
				}
			}
		}
	}

	return rules
}

// matchesAnySelector checks if any critical selector appears in the rule's selector portion.
func matchesAnySelector(rule []byte, selectors map[string]bool) bool {
	// Get just the selector part (before first '{')
	idx := bytes.IndexByte(rule, '{')
	if idx == -1 {
		return false
	}
	selector := string(rule[:idx])
	for sel := range selectors {
		if strings.Contains(selector, sel) {
			return true
		}
	}
	return false
}

// isPathSafe checks if the given path is within the project directory.
func (p *PostCSSPlugin) isPathSafe(projectDir, path string) bool {
	fullPath := path
	if !filepath.IsAbs(path) {
		fullPath = filepath.Join(projectDir, path)
	}
	rel, err := filepath.Rel(projectDir, fullPath)
	return err == nil && !strings.HasPrefix(rel, "..")
}

// Ensure PostCSSPlugin implements CLIPlugin interface.
var _ plugin.CLIPlugin = (*PostCSSPlugin)(nil)

func init() {
	if err := plugin.Register(New()); err != nil {
		panic("failed to register postcss plugin: " + err.Error())
	}
}
