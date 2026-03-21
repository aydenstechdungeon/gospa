// Package plugin provides external plugin loading capabilities for GoSPA.
// This enables loading plugins from external sources like GitHub repositories.
package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ExternalPluginLoader handles loading plugins from external sources.
type ExternalPluginLoader struct {
	cacheDir string
}

// NewExternalPluginLoader creates a new loader with the default cache directory.
func NewExternalPluginLoader() *ExternalPluginLoader {
	return &ExternalPluginLoader{
		cacheDir: PluginCacheDir,
	}
}

// NewExternalPluginLoaderWithCache creates a new loader with a custom cache directory.
func NewExternalPluginLoaderWithCache(cacheDir string) *ExternalPluginLoader {
	return &ExternalPluginLoader{
		cacheDir: cacheDir,
	}
}

// ParsePluginRef parses a plugin reference string into owner, repo, and version.
// Supports formats:
//   - github.com/owner/repo
//   - github.com/owner/repo@version
//   - owner/repo (shorthand)
//   - owner/repo@version (shorthand)
func ParsePluginRef(ref string) (owner, repo, version string, err error) {
	// Remove .git suffix if present
	ref = strings.TrimSuffix(ref, ".git")

	// Check for version suffix
	version = "latest"
	if strings.Contains(ref, "@") {
		parts := strings.Split(ref, "@")
		ref = parts[0]
		version = parts[1]
	}

	// Parse the reference
	switch {
	case strings.HasPrefix(ref, "github.com/"):
		// Full GitHub URL format: github.com/owner/repo
		parts := strings.TrimPrefix(ref, "github.com/")
		owner, repo, found := strings.Cut(parts, "/")
		if !found || owner == "" || repo == "" {
			return "", "", "", fmt.Errorf("invalid GitHub URL: %s", ref)
		}
		return owner, repo, version, nil
	case strings.Contains(ref, "/"):
		// Shorthand format: owner/repo
		owner, repo, found := strings.Cut(ref, "/")
		if !found || owner == "" || repo == "" {
			return "", "", "", fmt.Errorf("invalid plugin reference: %s", ref)
		}
		return owner, repo, version, nil
	default:
		return "", "", "", fmt.Errorf("invalid plugin reference: %s (use owner/repo or github.com/owner/repo)", ref)
	}
}

// LoadFromGitHub downloads and loads a plugin from a GitHub repository.
// The ref can be in the following formats:
//   - github.com/owner/repo
//   - github.com/owner/repo@version
//   - owner/repo
//   - owner/repo@version
func (l *ExternalPluginLoader) LoadFromGitHub(ref string) (Plugin, error) {
	owner, repo, version, err := ParsePluginRef(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plugin reference: %w", err)
	}

	// Validate to prevent command injection
	if err := validatePluginName(owner); err != nil {
		return nil, fmt.Errorf("failed to validate owner: %w", err)
	}
	if err := validatePluginName(repo); err != nil {
		return nil, fmt.Errorf("failed to validate repo: %w", err)
	}

	// Ensure cache directory exists
	if err := validatePluginVersion(version); err != nil {
		return nil, fmt.Errorf("failed to validate version: %w", err)
	}
	if err := os.MkdirAll(l.cacheDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create plugin cache directory: %w", err)
	}

	// Check if already cached
	pluginPath := filepath.Join(l.cacheDir, owner, repo, version)
	pluginDataPath := filepath.Join(pluginPath, "plugin.json")

	if _, err := os.Stat(pluginDataPath); err == nil {
		// Load from cache
		return l.loadFromPath(pluginPath)
	}

	// Download the plugin
	if err := l.download(owner, repo, version); err != nil {
		return nil, fmt.Errorf("failed to download plugin: %w", err)
	}

	return l.loadFromPath(pluginPath)
}

// validatePluginName validates that a plugin owner/repo name is safe.
func validatePluginName(name string) error {
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid name %q: must contain only alphanumeric characters, hyphens, and underscores", name)
	}
	return nil
}

func validatePluginVersion(version string) error {
	validVersion := regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)
	if version == "" || !validVersion.MatchString(version) || strings.Contains(version, "..") {
		return fmt.Errorf("invalid version %q", version)
	}
	return nil
}

// download clones or downloads a plugin from GitHub.
func (l *ExternalPluginLoader) download(owner, repo, version string) error {
	pluginPath := filepath.Join(l.cacheDir, owner, repo, version)

	// Create plugin directory
	if err := os.MkdirAll(pluginPath, 0750); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// Check if git is available
	if _, err := exec.LookPath("git"); err == nil {
		gitURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
		cloneArgs := []string{"clone", "--depth", "1"}
		if version != "latest" {
			cloneArgs = append(cloneArgs, "--branch", version)
		}
		cloneArgs = append(cloneArgs, gitURL, pluginPath)
		cmd := exec.Command("git", cloneArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
	} else {
		// Fallback: require git - archive extraction not implemented
		return fmt.Errorf("git is not installed and archive extraction is not supported. Please install git to download plugins")
	}

	//nolint:gosec // pluginPath is validated by validatePluginName and validatePluginVersion
	resolvedRefCmd := exec.Command("git", "-C", pluginPath, "rev-parse", "HEAD")
	resolvedRefOut, err := resolvedRefCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to resolve plugin commit: %w", err)
	}

	metadata := Metadata{
		Name:        repo,
		Version:     version,
		Source:      fmt.Sprintf("github.com/%s/%s", owner, repo),
		ResolvedRef: strings.TrimSpace(string(resolvedRefOut)),
	}

	metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to create plugin metadata: %w", err)
	}

	if err := os.WriteFile(filepath.Join(pluginPath, "plugin.json"), metadataBytes, 0600); err != nil {
		return fmt.Errorf("failed to write plugin metadata: %w", err)
	}

	return nil
}

// loadFromPath loads a plugin from a local path.
func (l *ExternalPluginLoader) loadFromPath(pluginPath string) (Plugin, error) {
	// Read plugin metadata - use filepath.Clean to prevent path traversal
	metadataPath := filepath.Clean(filepath.Join(pluginPath, "plugin.json"))
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin metadata: %w", err)
	}

	var metadata Metadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse plugin metadata: %w", err)
	}

	// Look for a compiled plugin binary (.so file)
	pluginFile := filepath.Join(pluginPath, "plugin.so")
	if _, err := os.Stat(pluginFile); err == nil {
		return l.loadCompiledPlugin(pluginFile, metadata.Name)
	}

	// Look for a Go module
	goModPath := filepath.Join(pluginPath, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		// This is a Go-based plugin - it needs to be built first
		return nil, fmt.Errorf("plugin %s is a Go module and must be built before loading", metadata.Name)
	}

	return nil, fmt.Errorf("no loadable plugin found in %s", pluginPath)
}

// loadCompiledPlugin loads a compiled Go plugin from a .so file.
func (l *ExternalPluginLoader) loadCompiledPlugin(_ string, _ string) (Plugin, error) {
	// Note: Go's plugin package only works on Linux/macOS and requires the same Go version
	// This is a simplified implementation - production code would need more error handling
	/*
		plugin, err := plugin.Open(pluginFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open plugin: %w", err)
		}

		symPlugin, err := plugin.Lookup("Plugin")
		if err != nil {
			return nil, fmt.Errorf("failed to find Plugin symbol: %w", err)
		}

		p, ok := symPlugin.(Plugin)
		if !ok {
			return nil, fmt.Errorf("plugin does not implement Plugin interface")
		}

		return p, nil
	*/

	// Placeholder - actual implementation requires plugin package
	return nil, fmt.Errorf("compiled plugin loading not yet implemented (requires Go plugin package)")
}

// Metadata holds metadata about an external plugin.
type Metadata struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	Author      string `json:"author,omitempty"`
	Source      string `json:"source"`
	ResolvedRef string `json:"resolvedRef,omitempty"`
}

// RegistryEntry represents a plugin in the registry.
type RegistryEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Version     string `json:"version"`
	Installed   bool   `json:"installed"`
}

// DiscoverPlugins queries a plugin registry for available plugins.
// This is a placeholder that would connect to an actual registry service.
func DiscoverPlugins() ([]RegistryEntry, error) {
	// In production, this would connect to a plugin registry service
	// For now, return built-in plugins
	return []RegistryEntry{
		{Name: "tailwind", Description: "Tailwind CSS v4 integration", URL: "github.com/aydenstechdungeon/gospa-plugin-tailwind", Installed: true},
		{Name: "postcss", Description: "PostCSS processing with Tailwind v4", URL: "github.com/aydenstechdungeon/gospa-plugin-postcss", Installed: true},
		{Name: "image", Description: "Image optimization", URL: "github.com/aydenstechdungeon/gospa-plugin-image", Installed: true},
		{Name: "seo", Description: "SEO helpers and sitemap generation", URL: "github.com/aydenstechdungeon/gospa-plugin-seo", Installed: true},
		{Name: "validation", Description: "Form validation with Valibot", URL: "github.com/aydenstechdungeon/gospa-plugin-validation", Installed: true},
		{Name: "auth", Description: "Authentication with OAuth2/JWT/OTP", URL: "github.com/aydenstechdungeon/gospa-plugin-auth", Installed: true},
		{Name: "qrcode", Description: "QR code generation", URL: "github.com/aydenstechdungeon/gospa-plugin-qrcode", Installed: true},
	}, nil
}

// SearchPlugins searches the registry for plugins matching a query.
func SearchPlugins(query string) ([]RegistryEntry, error) {
	entries, err := DiscoverPlugins()
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	var results []RegistryEntry

	for _, entry := range entries {
		if strings.Contains(strings.ToLower(entry.Name), query) ||
			strings.Contains(strings.ToLower(entry.Description), query) {
			results = append(results, entry)
		}
	}

	return results, nil
}

// InstallPlugin installs a plugin from a remote source.
func InstallPlugin(ref string) error {
	loader := NewExternalPluginLoader()
	_, err := loader.LoadFromGitHub(ref)
	return err
}

// UninstallPlugin removes a cached plugin.
func UninstallPlugin(name string) error {
	// Parse the name to get owner/repo
	owner, repo, _, err := ParsePluginRef(name)
	if err != nil {
		return err
	}

	pluginPath := filepath.Join(PluginCacheDir, owner, repo)
	return os.RemoveAll(pluginPath)
}

// ListInstalledPlugins returns all installed external plugins.
func ListInstalledPlugins() ([]RegistryEntry, error) {
	var entries []RegistryEntry

	// Check if cache directory exists
	if _, err := os.Stat(PluginCacheDir); os.IsNotExist(err) {
		return entries, nil
	}

	// Read all subdirectories
	entriesFs, err := os.ReadDir(PluginCacheDir)
	if err != nil {
		return nil, err
	}

	for _, ownerEntry := range entriesFs {
		if !ownerEntry.IsDir() {
			continue
		}

		ownerPath := filepath.Join(PluginCacheDir, ownerEntry.Name())
		repoEntries, err := os.ReadDir(ownerPath)
		if err != nil {
			continue
		}

		for _, repoEntry := range repoEntries {
			if !repoEntry.IsDir() {
				continue
			}

			repoPath := filepath.Join(ownerPath, repoEntry.Name())
			// Clean the path to prevent path traversal attacks
			metadataPath := filepath.Clean(filepath.Join(repoPath, "plugin.json"))

			var metadata Metadata
			if data, err := os.ReadFile(metadataPath); err == nil {
				if err := json.Unmarshal(data, &metadata); err != nil {
					// Log but continue - don't fail the whole operation
					fmt.Printf("Warning: failed to parse plugin metadata: %v\n", err)
				}
			}

			entries = append(entries, RegistryEntry{
				Name:        metadata.Name,
				Description: metadata.Description,
				URL:         metadata.Source,
				Version:     metadata.Version,
				Installed:   true,
			})
		}
	}

	return entries, nil
}

// ValidatePluginRef validates a plugin reference string.
func ValidatePluginRef(ref string) error {
	// Check for valid characters
	validRef := regexp.MustCompile(`^[a-zA-Z0-9_./@-]+$`)
	if !validRef.MatchString(ref) {
		return fmt.Errorf("invalid characters in plugin reference")
	}

	// Try to parse
	_, _, _, err := ParsePluginRef(ref)
	return err
}
