// Package postcss provides a PostCSS plugin for GoSPA with Tailwind CSS extensions.
// It supports typography, forms, aspect-ratio, and other popular PostCSS plugins.
package postcss

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aydenstechdungeon/gospa/plugin"
)

// PostCSSPlugin provides PostCSS processing with Tailwind CSS extensions.
type PostCSSPlugin struct {
	config         *Config
	enabledPlugins map[string]bool
}

// Config holds PostCSS plugin configuration.
type Config struct {
	// OutputDir is where generated CSS files are written.
	OutputDir string `yaml:"output_dir" json:"outputDir"`

	// SourceDir is where source CSS files are located.
	SourceDir string `yaml:"source_dir" json:"sourceDir"`

	// InputFile is the main CSS input file (default: styles.css).
	InputFile string `yaml:"input_file" json:"inputFile"`

	// OutputFile is the generated CSS output file (default: styles.output.css).
	OutputFile string `yaml:"output_file" json:"outputFile"`

	// Minify enables CSS minification in production builds.
	Minify bool `yaml:"minify" json:"minify"`

	// SourceMap enables source map generation.
	SourceMap bool `yaml:"source_map" json:"sourceMap"`

	// Plugins configures which PostCSS plugins to enable.
	Plugins PluginConfig `yaml:"plugins" json:"plugins"`
}

// PluginConfig configures individual PostCSS plugins.
type PluginConfig struct {
	// Typography enables @tailwindcss/typography (prose classes).
	Typography bool `yaml:"typography" json:"typography"`

	// Forms enables @tailwindcss/forms (form styling).
	Forms bool `yaml:"forms" json:"forms"`

	// AspectRatio enables @tailwindcss/aspect-ratio.
	AspectRatio bool `yaml:"aspect_ratio" json:"aspectRatio"`

	// LineClamp enables @tailwindcss/line-clamp.
	LineClamp bool `yaml:"line_clamp" json:"lineClamp"`

	// ContainerQueries enables container queries (built into Tailwind v4).
	ContainerQueries bool `yaml:"container_queries" json:"containerQueries"`

	// Autoprefixer enables autoprefixer for vendor prefixes.
	Autoprefixer bool `yaml:"autoprefixer" json:"autoprefixer"`

	// CSSNano enables cssnano for minification.
	CSSNano bool `yaml:"cssnano" json:"cssnano"`

	// PostCSSImport enables postcss-import for @import handling.
	PostCSSImport bool `yaml:"postcss_import" json:"postcssImport"`

	// PostCSSNested enables postcss-nested for nested CSS.
	PostCSSNested bool `yaml:"postcss_nested" json:"postcssNested"`

	// PostCSSCustomProperties enables postcss-custom-properties.
	PostCSSCustomProperties bool `yaml:"postcss_custom_properties" json:"postcssCustomProperties"`
}

// DefaultConfig returns the default PostCSS plugin configuration.
func DefaultConfig() *Config {
	return &Config{
		OutputDir:  "static/css",
		SourceDir:  "styles",
		InputFile:  "main.css",
		OutputFile: "main.output.css",
		Minify:     true,
		SourceMap:  true,
		Plugins: PluginConfig{
			Typography:              true,
			Forms:                   true,
			AspectRatio:             true,
			LineClamp:               true,
			ContainerQueries:        true,
			Autoprefixer:            true,
			CSSNano:                 false, // Handled by Tailwind v4
			PostCSSImport:           true,
			PostCSSNested:           true,
			PostCSSCustomProperties: false, // Native in modern browsers
		},
	}
}

// New creates a new PostCSS plugin with the given configuration.
func New(cfg *Config) *PostCSSPlugin {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	p := &PostCSSPlugin{
		config:         cfg,
		enabledPlugins: make(map[string]bool),
	}

	// Track enabled plugins
	p.enabledPlugins["typography"] = cfg.Plugins.Typography
	p.enabledPlugins["forms"] = cfg.Plugins.Forms
	p.enabledPlugins["aspect-ratio"] = cfg.Plugins.AspectRatio
	p.enabledPlugins["line-clamp"] = cfg.Plugins.LineClamp
	p.enabledPlugins["container-queries"] = cfg.Plugins.ContainerQueries
	p.enabledPlugins["autoprefixer"] = cfg.Plugins.Autoprefixer
	p.enabledPlugins["cssnano"] = cfg.Plugins.CSSNano
	p.enabledPlugins["postcss-import"] = cfg.Plugins.PostCSSImport
	p.enabledPlugins["postcss-nested"] = cfg.Plugins.PostCSSNested
	p.enabledPlugins["postcss-custom-properties"] = cfg.Plugins.PostCSSCustomProperties

	return p
}

// Name returns the plugin name.
func (p *PostCSSPlugin) Name() string {
	return "postcss"
}

// Init initializes the PostCSS plugin.
func (p *PostCSSPlugin) Init() error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(p.config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create source directory if it doesn't exist
	if err := os.MkdirAll(p.config.SourceDir, 0755); err != nil {
		return fmt.Errorf("failed to create source directory: %w", err)
	}

	return nil
}

// Dependencies returns the required Bun packages for PostCSS.
func (p *PostCSSPlugin) Dependencies() []plugin.Dependency {
	deps := []plugin.Dependency{
		{Type: plugin.DepBun, Name: "postcss", Version: "latest"},
		{Type: plugin.DepBun, Name: "@tailwindcss/postcss", Version: "latest"},
	}

	// Add optional plugins based on configuration
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

	if p.config.Plugins.LineClamp {
		deps = append(deps, plugin.Dependency{
			Type: plugin.DepBun, Name: "@tailwindcss/line-clamp", Version: "latest",
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

	if p.config.Plugins.PostCSSImport {
		deps = append(deps, plugin.Dependency{
			Type: plugin.DepBun, Name: "postcss-import", Version: "latest",
		})
	}

	if p.config.Plugins.PostCSSNested {
		deps = append(deps, plugin.Dependency{
			Type: plugin.DepBun, Name: "postcss-nested", Version: "latest",
		})
	}

	if p.config.Plugins.PostCSSCustomProperties {
		deps = append(deps, plugin.Dependency{
			Type: plugin.DepBun, Name: "postcss-custom-properties", Version: "latest",
		})
	}

	return deps
}

// OnHook handles lifecycle hooks.
func (p *PostCSSPlugin) OnHook(hook plugin.Hook, ctx map[string]interface{}) error {
	switch hook {
	case plugin.BeforeBuild, plugin.BeforeDev:
		projectDir, _ := ctx["project_dir"].(string)
		if projectDir == "" {
			projectDir = "."
		}

		// Generate postcss.config.js
		if err := p.generatePostCSSConfig(projectDir); err != nil {
			return fmt.Errorf("failed to generate PostCSS config: %w", err)
		}

		// Generate main CSS file if it doesn't exist
		cssPath := filepath.Join(projectDir, p.config.SourceDir, p.config.InputFile)
		if _, err := os.Stat(cssPath); os.IsNotExist(err) {
			if err := p.generateMainCSS(cssPath); err != nil {
				return fmt.Errorf("failed to generate main CSS: %w", err)
			}
		}
	}
	return nil
}

// Commands returns custom CLI commands for the PostCSS plugin.
func (p *PostCSSPlugin) Commands() []plugin.Command {
	return []plugin.Command{
		{
			Name:        "postcss:config",
			Alias:       "pc",
			Description: "Generate PostCSS configuration file",
			Action: func(args []string) error {
				projectDir := "."
				if len(args) > 0 {
					projectDir = args[0]
				}
				return p.generatePostCSSConfig(projectDir)
			},
		},
		{
			Name:        "postcss:init",
			Alias:       "pi",
			Description: "Initialize PostCSS with default CSS file",
			Action: func(args []string) error {
				projectDir := "."
				if len(args) > 0 {
					projectDir = args[0]
				}
				cssPath := filepath.Join(projectDir, p.config.SourceDir, p.config.InputFile)
				return p.generateMainCSS(cssPath)
			},
		},
	}
}

// generatePostCSSConfig generates a postcss.config.js file.
func (p *PostCSSPlugin) generatePostCSSConfig(projectDir string) error {
	configPath := filepath.Join(projectDir, "postcss.config.js")

	// Build plugins list
	content := `// PostCSS configuration for GoSPA
// Generated by postcss plugin

export default {
  plugins: {
`

	// Tailwind CSS v4 PostCSS plugin (required first)
	content += "    '@tailwindcss/postcss': {},\n"

	// PostCSS Import (for @import handling)
	if p.config.Plugins.PostCSSImport {
		content += "    'postcss-import': {},\n"
	}

	// PostCSS Nested (for nested CSS syntax)
	if p.config.Plugins.PostCSSNested {
		content += "    'postcss-nested': {},\n"
	}

	// Tailwind CSS extensions
	if p.config.Plugins.Typography {
		content += "    '@tailwindcss/typography': {},\n"
	}

	if p.config.Plugins.Forms {
		content += "    '@tailwindcss/forms': {},\n"
	}

	if p.config.Plugins.AspectRatio {
		content += "    '@tailwindcss/aspect-ratio': {},\n"
	}

	if p.config.Plugins.LineClamp {
		content += "    '@tailwindcss/line-clamp': {},\n"
	}

	// Autoprefixer for vendor prefixes
	if p.config.Plugins.Autoprefixer {
		content += "    'autoprefixer': {},\n"
	}

	// CSSNano for minification (production only)
	if p.config.Plugins.CSSNano {
		content += `    'cssnano': {
      preset: ['default', { discardComments: { removeAll: true } }]
    },
`
	}

	// PostCSS Custom Properties
	if p.config.Plugins.PostCSSCustomProperties {
		content += "    'postcss-custom-properties': {},\n"
	}

	content += `  }
};
`

	return os.WriteFile(configPath, []byte(content), 0644)
}

// generateMainCSS generates a main CSS file with Tailwind imports.
func (p *PostCSSPlugin) generateMainCSS(cssPath string) error {
	content := `/* Main CSS file for GoSPA */
/* This file is processed by PostCSS with Tailwind CSS v4 */

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

	// Add plugin-specific CSS comments
	if p.config.Plugins.Typography {
		content += `

/* Typography plugin enabled - use prose classes:
   prose, prose-sm, prose-lg, prose-xl, prose-2xl
   prose-headings, prose-lead, prose-img, etc.
*/
`
	}

	if p.config.Plugins.Forms {
		content += `

/* Forms plugin enabled - form elements styled automatically
   Use form-input, form-textarea, form-select, form-checkbox, form-radio
*/
`
	}

	if p.config.Plugins.AspectRatio {
		content += `

/* Aspect Ratio plugin enabled - use aspect-{ratio} classes
   aspect-video, aspect-square, aspect-[4/3], etc.
*/
`
	}

	if p.config.Plugins.ContainerQueries {
		content += `

/* Container Queries enabled (built into Tailwind v4)
   Use @container, @sm:, @md:, @lg:, etc.
*/
`
	}

	// Ensure directory exists
	dir := filepath.Dir(cssPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(cssPath, []byte(content), 0644)
}

// GetConfig returns the current configuration.
func (p *PostCSSPlugin) GetConfig() *Config {
	return p.config
}

// IsPluginEnabled checks if a specific PostCSS plugin is enabled.
func (p *PostCSSPlugin) IsPluginEnabled(name string) bool {
	enabled, ok := p.enabledPlugins[name]
	return ok && enabled
}

// EnablePlugin enables a specific PostCSS plugin.
func (p *PostCSSPlugin) EnablePlugin(name string) {
	p.enabledPlugins[name] = true
}

// DisablePlugin disables a specific PostCSS plugin.
func (p *PostCSSPlugin) DisablePlugin(name string) {
	p.enabledPlugins[name] = false
}

// Ensure PostCSSPlugin implements CLIPlugin interface.
var _ plugin.CLIPlugin = (*PostCSSPlugin)(nil)
