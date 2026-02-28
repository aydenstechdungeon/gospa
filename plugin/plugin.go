package plugin

import (
	"fmt"
	"os"
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
	// AfterDev is triggered after the development server starts.
	AfterDev Hook = "after:dev"
	// BeforeBuild is triggered before the production build starts.
	BeforeBuild Hook = "before:build"
	// AfterBuild is triggered after the production build completes.
	AfterBuild Hook = "after:build"
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

// Plugin is the base interface for all GoSPA extensions.
type Plugin interface {
	Name() string
	Init() error
	// Dependencies returns the list of dependencies required by this plugin.
	// This includes both Go modules and Bun packages.
	Dependencies() []Dependency
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
}

// registry is the global plugin registry protected by a mutex for thread-safety.
var (
	registry   []Plugin
	registryMu sync.RWMutex
)

// Register registers a plugin with GoSPA.
// This function is thread-safe and can be called from multiple goroutines.
func Register(p Plugin) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = append(registry, p)
}

// GetPlugins returns all registered plugins.
// This function is thread-safe and returns a copy of the registry.
func GetPlugins() []Plugin {
	registryMu.RLock()
	defer registryMu.RUnlock()
	// Return a copy to prevent external modification
	result := make([]Plugin, len(registry))
	copy(result, registry)
	return result
}

// GetCLIPlugins returns all registered CLI plugins.
// This function is thread-safe.
func GetCLIPlugins() []CLIPlugin {
	registryMu.RLock()
	defer registryMu.RUnlock()
	var cliPlugins []CLIPlugin
	for _, p := range registry {
		if cp, ok := p.(CLIPlugin); ok {
			cliPlugins = append(cliPlugins, cp)
		}
	}
	return cliPlugins
}

// TriggerHook triggers a lifecycle hook for all registered CLI plugins.
func TriggerHook(hook Hook, ctx map[string]interface{}) error {
	for _, p := range GetCLIPlugins() {
		if err := p.OnHook(hook, ctx); err != nil {
			return fmt.Errorf("plugin %s failed on hook %s: %w", p.Name(), hook, err)
		}
	}
	return nil
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
