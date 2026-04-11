// Package plugin provides the plugin system for GoSPA.
package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// getPluginCacheDir returns the directory where external plugins are cached.
func getPluginCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".gospa", "plugins")
}

// PluginCacheDir is the directory where external plugins are cached.
var PluginCacheDir = getPluginCacheDir()

// Hook represents a lifecycle event in GoSPA.
type Hook string

const (
	// BeforeGenerate is triggered before code generation starts.
	BeforeGenerate Hook = "before:generate"
	// AfterGenerate is triggered after code generation completes.
	AfterGenerate Hook = "after:generate"
	// BeforeDev is triggered before the development server starts.
	BeforeDev Hook = "before:dev"
	// AfterDev is triggered after the development server stops.
	AfterDev Hook = "after:dev"
	// BeforeBuild is triggered before the production build starts.
	BeforeBuild Hook = "before:build"
	// AfterBuild is triggered after the production build completes.
	AfterBuild Hook = "after:build"
	// BeforeServe is triggered before the HTTP server starts.
	BeforeServe Hook = "before:serve"
	// AfterServe is triggered after the HTTP server starts.
	AfterServe Hook = "after:serve"
	// BeforePrune is triggered before state pruning.
	BeforePrune Hook = "before:prune"
	// AfterPrune is triggered after state pruning.
	AfterPrune Hook = "after:prune"
	// OnError is triggered when an error occurs.
	OnError Hook = "on:error"
)

// DependencyType represents the type of dependency (Go or Bun/JS).
type DependencyType string

const (
	// DepGo is a Go module dependency.
	DepGo DependencyType = "go"
	// DepBun is a Bun/JavaScript package dependency.
	DepBun DependencyType = "bun"
)

// Dependency represents a plugin dependency.
type Dependency struct {
	// Type is the dependency type (go or bun).
	Type DependencyType
	// Name is the package name (e.g., "golang.org/x/oauth2" or "valibot").
	Name string
	// Version is the version constraint (e.g., "latest", "v1.2.3").
	Version string
}

// State represents the current state of a plugin.
type State int

const (
	// StateEnabled means the plugin is active.
	StateEnabled State = iota
	// StateDisabled means the plugin is loaded but inactive.
	StateDisabled
	// StateError means the plugin failed to load.
	StateError
)

// Info provides metadata about a plugin.
type Info struct {
	Name        string // Unique identifier
	Version     string // Semantic version
	Description string // Human-readable description
	Author      string // Plugin author
	State       State  // Current state
}

// Config defines plugin configuration structure.
type Config struct {
	// Schema describes the configuration fields.
	Schema map[string]FieldSchema `json:"schema" yaml:"schema"`
	// Defaults provides default values.
	Defaults map[string]interface{} `json:"defaults" yaml:"defaults"`
}

// FieldSchema describes a single configuration field.
type FieldSchema struct {
	Type        string      `json:"type" yaml:"type"`               // "string", "bool", "number", "array"
	Description string      `json:"description" yaml:"description"` // Human-readable description
	Required    bool        `json:"required" yaml:"required"`       // Is this field required?
	Default     interface{} `json:"default" yaml:"default"`         // Default value
}

// Plugin is the base interface for all GoSPA extensions.
type Plugin interface {
	Name() string
	Init() error
	// Dependencies returns the list of dependencies required by this plugin.
	Dependencies() []Dependency
}

// RuntimePlugin extends Plugin with runtime integration capabilities.
type RuntimePlugin interface {
	Plugin
	// Config returns the plugin configuration schema.
	Config() Config
	// Middlewares returns Fiber handlers to inject into the app.
	Middlewares() []interface{} // []fiber.Handler
	// TemplateFuncs returns template functions to expose to templ components.
	TemplateFuncs() map[string]interface{}
}

// CLIPlugin extends Plugin with CLI-specific functionality.
type CLIPlugin interface {
	Plugin
	// OnHook is called when a lifecycle hook is triggered.
	OnHook(hook Hook, ctx map[string]interface{}) error
	// Commands returns custom CLI commands provided by the plugin.
	Commands() []Command
}

// Command represents a custom CLI command.
type Command struct {
	Name        string
	Alias       string
	Description string
	Action      func(args []string) error
	// Flags defines command-line flags for this command.
	Flags []Flag
}

// Flag represents a command-line flag.
type Flag struct {
	Name        string
	Shorthand   string
	Description string
	Default     interface{}
}

// registry is the global plugin registry protected by a mutex for thread-safety.
var (
	registry   map[string]*registryEntry // name -> entry
	registryMu sync.RWMutex
)

// registryEntry holds a plugin and its metadata.
type registryEntry struct {
	plugin Info
	impl   Plugin
}

// init initializes the registry.
func init() {
	registry = make(map[string]*registryEntry)
}

// Register registers a plugin with GoSPA.
// This function is thread-safe and can be called from multiple goroutines.
// Returns an error if a plugin with the same name is already registered.
func Register(p Plugin) error {
	registryMu.Lock()
	defer registryMu.Unlock()

	name := p.Name()
	if _, exists := registry[name]; exists {
		return fmt.Errorf("plugin %q is already registered", name)
	}

	registry[name] = &registryEntry{
		plugin: Info{
			Name:  name,
			State: StateEnabled,
		},
		impl: p,
	}
	return nil
}

// Unregister removes a plugin from the registry.
func Unregister(name string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	delete(registry, name)
}

// GetPlugin returns a registered plugin by name.
// Returns nil if the plugin is not found.
func GetPlugin(name string) Plugin {
	registryMu.RLock()
	defer registryMu.RUnlock()
	if entry, ok := registry[name]; ok {
		return entry.impl
	}
	return nil
}

// GetPlugins returns all registered plugins.
// This function is thread-safe and returns a copy of the registry.
func GetPlugins() []Plugin {
	registryMu.RLock()
	defer registryMu.RUnlock()

	result := make([]Plugin, 0, len(registry))
	for _, entry := range registry {
		result = append(result, entry.impl)
	}
	return result
}

// GetPluginInfo returns metadata for a specific plugin.
func GetPluginInfo(name string) (Info, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	if entry, ok := registry[name]; ok {
		return entry.plugin, true
	}
	return Info{}, false
}

// GetAllPluginInfo returns metadata for all registered plugins.
func GetAllPluginInfo() []Info {
	registryMu.RLock()
	defer registryMu.RUnlock()

	result := make([]Info, 0, len(registry))
	for _, entry := range registry {
		result = append(result, entry.plugin)
	}
	return result
}

// GetCLIPlugins returns all enabled CLI plugins.
// This function is thread-safe and only returns plugins that are in StateEnabled.
func GetCLIPlugins() []CLIPlugin {
	registryMu.RLock()
	defer registryMu.RUnlock()

	var cliPlugins []CLIPlugin
	for _, entry := range registry {
		if entry.plugin.State == StateEnabled {
			if cp, ok := entry.impl.(CLIPlugin); ok {
				cliPlugins = append(cliPlugins, cp)
			}
		}
	}
	return cliPlugins
}

// GetRuntimePlugins returns all enabled runtime plugins.
// This function is thread-safe and only returns plugins that are in StateEnabled.
func GetRuntimePlugins() []RuntimePlugin {
	registryMu.RLock()
	defer registryMu.RUnlock()

	var runtimePlugins []RuntimePlugin
	for _, entry := range registry {
		if entry.plugin.State == StateEnabled {
			if rp, ok := entry.impl.(RuntimePlugin); ok {
				runtimePlugins = append(runtimePlugins, rp)
			}
		}
	}
	return runtimePlugins
}

// Enable enables a plugin by name.
func Enable(name string) error {
	registryMu.Lock()
	defer registryMu.Unlock()

	if entry, ok := registry[name]; ok {
		entry.plugin.State = StateEnabled
		return nil
	}
	return fmt.Errorf("plugin %q not found", name)
}

// Disable disables a plugin by name.
func Disable(name string) error {
	registryMu.Lock()
	defer registryMu.Unlock()

	if entry, ok := registry[name]; ok {
		entry.plugin.State = StateDisabled
		return nil
	}
	return fmt.Errorf("plugin %q not found", name)
}

// TriggerHook triggers a lifecycle hook for all registered CLI plugins concurrently.
// Only enabled plugins have their hooks triggered.
func TriggerHook(hook Hook, ctx map[string]interface{}) error {
	registryMu.RLock()
	var enabledPlugins []CLIPlugin
	for _, entry := range registry {
		if entry.plugin.State == StateEnabled {
			if cp, ok := entry.impl.(CLIPlugin); ok {
				enabledPlugins = append(enabledPlugins, cp)
			}
		}
	}
	registryMu.RUnlock()

	var wg sync.WaitGroup
	errCh := make(chan error, len(enabledPlugins))

	for _, p := range enabledPlugins {
		wg.Add(1)
		go func(plugin CLIPlugin) {
			defer wg.Done()
			if err := plugin.OnHook(hook, ctx); err != nil {
				errCh <- fmt.Errorf("plugin %s failed on hook %s: %w", plugin.Name(), hook, err)
			}
		}(p)
	}

	wg.Wait()
	close(errCh)

	// Collect all errors
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		// Return aggregated error message
		errMsg := fmt.Sprintf("%d plugin(s) failed on hook %s:", len(errs), hook)
		for _, e := range errs {
			errMsg += fmt.Sprintf("\n  - %v", e)
		}
		return fmt.Errorf("%s", errMsg)
	}
	return nil
}

// TriggerHookForPlugin triggers a lifecycle hook for a specific plugin by name.
func TriggerHookForPlugin(name string, hook Hook, ctx map[string]interface{}) error {
	p := GetPlugin(name)
	if p == nil {
		return fmt.Errorf("plugin %q not found", name)
	}

	cliPlugin, ok := p.(CLIPlugin)
	if !ok {
		return fmt.Errorf("plugin %q does not implement CLIPlugin", name)
	}

	return cliPlugin.OnHook(hook, ctx)
}

// RunCommand executes a custom command from a plugin.
func RunCommand(name string, args []string) (bool, error) {
	for _, p := range GetCLIPlugins() {
		for _, cmd := range p.Commands() {
			if cmd.Name == name || (cmd.Alias != "" && cmd.Alias == name) {
				return true, cmd.Action(args)
			}
		}
	}
	return false, nil
}

// GetAllDependencies returns all dependencies from enabled plugins.
func GetAllDependencies() []Dependency {
	registryMu.RLock()
	defer registryMu.RUnlock()

	var deps []Dependency
	for _, entry := range registry {
		if entry.plugin.State != StateEnabled {
			continue
		}
		// Plugin interface requires Dependencies() method, so the type assertion
		// is always successful - but we check anyway for safety
		if p, ok := entry.impl.(interface{ Dependencies() []Dependency }); ok {
			deps = append(deps, p.Dependencies()...)
		}
	}
	return deps
}

// ResolveDependencies installs all plugin dependencies.
func ResolveDependencies() error {
	deps := GetAllDependencies()

	// Group by type
	var goDeps []Dependency
	var bunDeps []Dependency

	for _, dep := range deps {
		switch dep.Type {
		case DepGo:
			goDeps = append(goDeps, dep)
		case DepBun:
			bunDeps = append(bunDeps, dep)
		}
	}

	// Install Go dependencies
	if len(goDeps) > 0 {
		fmt.Println("Installing Go dependencies for plugins...")
		for _, dep := range goDeps {
			version := dep.Version
			if version == "latest" {
				version = "@latest"
			} else {
				version = "@" + version
			}
			fmt.Printf("  go get %s%s\n", dep.Name, version)
			cmd := exec.Command("go", "get", dep.Name+version) //nolint:gosec // G204: dep.Name and version are from trusted plugin manifest
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to install Go dependency %s: %w", dep.Name, err)
			}
		}
	}

	// Install Bun dependencies
	if len(bunDeps) > 0 {
		fmt.Println("Installing Bun dependencies for plugins...")
		for _, dep := range bunDeps {
			version := dep.Version
			if version == "latest" {
				version = "latest"
			}
			fmt.Printf("  bun add %s@%s\n", dep.Name, version)
			cmd := exec.Command("bun", "add", dep.Name+"@"+version) //nolint:gosec // G204: dep.Name and version are from trusted plugin manifest
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to install Bun dependency %s: %w", dep.Name, err)
			}
		}
	}

	return nil
}
