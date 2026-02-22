package plugin

import (
	"fmt"
)

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

// Plugin is the base interface for all GoSPA extensions.
type Plugin interface {
	Name() string
	Init() error
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

var registry []Plugin

// Register registers a plugin with GoSPA.
func Register(p Plugin) {
	registry = append(registry, p)
}

// GetPlugins returns all registered plugins.
func GetPlugins() []Plugin {
	return registry
}

// GetCLIPlugins returns all registered CLI plugins.
func GetCLIPlugins() []CLIPlugin {
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
