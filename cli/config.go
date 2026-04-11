package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	configFileName    = "gospa.config"
	configFileVersion = "1"
)

// GoSPAConfig represents the full gospa configuration file format.
type GoSPAConfig struct {
	Version  string          `yaml:"version"`
	Project  ProjectSection  `yaml:"project"`
	Dev      DevSection      `yaml:"dev"`
	Build    BuildSection    `yaml:"build"`
	Generate GenerateSection `yaml:"generate"`
	Serve    ServeSection    `yaml:"serve"`
	Plugins  []PluginConfig  `yaml:"plugins"`
	BuildAll BuildAllSection `yaml:"build-all"`
}

// ProjectSection holds project-level configuration.
type ProjectSection struct {
	Name   string `yaml:"name"`
	Module string `yaml:"module"`
}

// DevSection holds development server configuration.
type DevSection struct {
	Port       int           `yaml:"port"`
	Host       string        `yaml:"host"`
	Open       bool          `yaml:"open"`
	RoutesDir  string        `yaml:"routes_dir"`
	WatchPaths []string      `yaml:"watch_paths"`
	Proxy      string        `yaml:"proxy"`
	HMRPort    int           `yaml:"hmr_port"`
	Debounce   time.Duration `yaml:"debounce"`
	Timeout    time.Duration `yaml:"timeout"`
}

// BuildSection holds build configuration.
type BuildSection struct {
	Output    string   `yaml:"output"`
	Minify    bool     `yaml:"minify"`
	Compress  bool     `yaml:"compress"`
	SourceMap bool     `yaml:"sourcemap"`
	CGO       bool     `yaml:"cgo"`
	Env       string   `yaml:"env"`
	AssetsDir string   `yaml:"assets_dir"`
	LDFlags   string   `yaml:"ldflags"`
	Tags      string   `yaml:"tags"`
	Targets   []Target `yaml:"targets"`
}

// Target represents a build target platform/arch combination.
type Target struct {
	Platform string `yaml:"platform"`
	Arch     string `yaml:"arch"`
}

// GenerateSection holds code generation configuration.
type GenerateSection struct {
	Output string `yaml:"output"`
	Type   string `yaml:"type"`
	Strict bool   `yaml:"strict"`
}

// ServeSection holds production server configuration.
type ServeSection struct {
	Port    int               `yaml:"port"`
	Gzip    bool              `yaml:"gzip"`
	Brotli  bool              `yaml:"brotli"`
	Cache   bool              `yaml:"cache"`
	Headers map[string]string `yaml:"headers"`
}

// BuildAllSection holds multi-platform build configuration.
type BuildAllSection struct {
	Targets   []string `yaml:"targets"`
	OutputDir string   `yaml:"output_dir"`
	Compress  bool     `yaml:"compress"`
	Manifest  bool     `yaml:"manifest"`
	Parallel  int      `yaml:"parallel"`
}

// PluginConfig holds plugin configuration.
type PluginConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

// ConfigFileSpec defines where to look for config files and their priority.
var configFileSpecs = []struct {
	name    string
	formats []string
}{
	{"gospa.config", []string{"yaml", "yml", "json", "toml"}},
	{".gospa", []string{"yaml", "yml", "json", "toml"}},
}

// FindConfigFile searches for config files in the current directory.
// Returns the path to the first found config file, or empty string if none found.
func FindConfigFile() string {
	for _, spec := range configFileSpecs {
		for _, format := range spec.formats {
			path := filepath.Join(".", spec.name+"."+format)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}
	return ""
}

// LoadConfig loads the gospa configuration from file.
// If no config file is found, returns default config.
func LoadConfig(path string) (*GoSPAConfig, error) {
	if path == "" {
		path = FindConfigFile()
	}
	if path == "" {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path) //nolint:gosec // G304: path is validated config file path
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Determine format from extension
	ext := strings.ToLower(filepath.Ext(path))
	var config GoSPAConfig

	switch ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &config)
	case ".json":
		err = yaml.Unmarshal(data, &config) // YAML parser also handles JSON
	case ".toml":
		err = fmt.Errorf("TOML format not yet supported")
	default:
		// Try YAML as default
		err = yaml.Unmarshal(data, &config)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set version if not set
	if config.Version == "" {
		config.Version = configFileVersion
	}

	return &config, nil
}

// SaveConfig saves the gospa configuration to a file.
func SaveConfig(config *GoSPAConfig, path string) error {
	if config.Version == "" {
		config.Version = configFileVersion
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *GoSPAConfig {
	return &GoSPAConfig{
		Version: configFileVersion,
		Project: ProjectSection{
			Name: filepath.Base(getCurrentDir()),
		},
		Dev: DevSection{
			Port:      3000,
			Host:      "localhost",
			Open:      false,
			RoutesDir: "./routes",
			Debounce:  100 * time.Millisecond,
			Timeout:   30 * time.Second,
		},
		Build: BuildSection{
			Output:    "dist",
			Minify:    true,
			Compress:  true,
			SourceMap: false,
			CGO:       false,
			Env:       "production",
			AssetsDir: "static",
			LDFlags:   "-s -w",
		},
		Generate: GenerateSection{
			Output: "./generated",
			Type:   "island",
			Strict: false,
		},
		Serve: ServeSection{
			Port:   8080,
			Gzip:   true,
			Brotli: true,
			Cache:  true,
		},
		BuildAll: BuildAllSection{
			OutputDir: "./releases",
			Compress:  true,
			Manifest:  true,
			Parallel:  4,
		},
	}
}

// MergeWithEnv merges config values with environment variables.
// Environment variables take precedence over config file values.
func (c *GoSPAConfig) MergeWithEnv() {
	// Dev config env overrides
	if v := os.Getenv("GOSPA_DEV_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &c.Dev.Port) //nolint:errcheck,gosec
	}
	if v := os.Getenv("GOSPA_DEV_HOST"); v != "" {
		c.Dev.Host = v
	}
	if v := os.Getenv("GOSPA_DEV_ROUTES_DIR"); v != "" {
		c.Dev.RoutesDir = v
	}
	if v := os.Getenv("GOSPA_DEV_PROXY"); v != "" {
		c.Dev.Proxy = v
	}
	if v := os.Getenv("GOSPA_DEV_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Dev.Timeout = d
		}
	}

	// Build config env overrides
	if v := os.Getenv("GOSPA_BUILD_OUTPUT"); v != "" {
		c.Build.Output = v
	}
	if v := os.Getenv("GOSPA_BUILD_MINIFY"); v != "" {
		c.Build.Minify = v == "true" || v == "1"
	}
	if v := os.Getenv("GOSPA_BUILD_ENV"); v != "" {
		c.Build.Env = v
	}

	// Generate config env overrides
	if v := os.Getenv("GOSPA_GENERATE_OUTPUT"); v != "" {
		c.Generate.Output = v
	}

	// Serve config env overrides
	if v := os.Getenv("GOSPA_SERVE_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &c.Serve.Port) //nolint:errcheck,gosec
	}
}

func getCurrentDir() string {
	dir, _ := os.Getwd()
	return dir
}

// ToDevConfig converts the config to DevConfig format.
func (c *GoSPAConfig) ToDevConfig() *DevConfig {
	return &DevConfig{
		Port:       c.Dev.Port,
		Host:       c.Dev.Host,
		Open:       c.Dev.Open,
		RoutesDir:  c.Dev.RoutesDir,
		WatchPaths: c.Dev.WatchPaths,
		Proxy:      c.Dev.Proxy,
		HMRPort:    c.Dev.HMRPort,
		Debounce:   c.Dev.Debounce,
		Timeout:    c.Dev.Timeout,
	}
}

// ToBuildConfig converts the config to BuildConfig format.
func (c *GoSPAConfig) ToBuildConfig() *BuildConfig {
	return &BuildConfig{
		OutputDir:  c.Build.Output,
		Minify:     c.Build.Minify,
		Compress:   c.Build.Compress,
		SourceMap:  c.Build.SourceMap,
		CGO:        c.Build.CGO,
		Env:        c.Build.Env,
		AssetsDir:  c.Build.AssetsDir,
		LDFlags:    c.Build.LDFlags,
		Tags:       c.Build.Tags,
		NoManifest: false,
		NoStatic:   false,
		NoCompress: false,
	}
}

// ToGenerateConfig converts the config to GenerateConfig format.
func (c *GoSPAConfig) ToGenerateConfig() *GenerateConfig {
	return &GenerateConfig{
		OutputDir:     c.Generate.Output,
		ComponentType: c.Generate.Type,
		Strict:        c.Generate.Strict,
	}
}

// ToServeConfig converts the config to ServeConfig format.
func (c *GoSPAConfig) ToServeConfig() *ServeConfig {
	return &ServeConfig{
		Port:    c.Serve.Port,
		Gzip:    c.Serve.Gzip,
		Brotli:  c.Serve.Brotli,
		Cache:   c.Serve.Cache,
		Headers: c.Serve.Headers,
	}
}

// ToBuildAllConfig converts the config to BuildAllConfig format.
func (c *GoSPAConfig) ToBuildAllConfig() *BuildAllConfig {
	targets := c.BuildAll.Targets
	if len(targets) == 0 {
		targets = []string{
			"linux/amd64",
			"linux/arm64",
			"darwin/amd64",
			"darwin/arm64",
			"windows/amd64",
			"windows/arm64",
		}
	}
	return &BuildAllConfig{
		Targets:   targets,
		OutputDir: c.BuildAll.OutputDir,
		Compress:  c.BuildAll.Compress,
		Manifest:  c.BuildAll.Manifest,
		Parallel:  c.BuildAll.Parallel,
	}
}
