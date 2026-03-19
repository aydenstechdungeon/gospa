# GoSPA Plugin System

GoSPA features a powerful plugin system that allows you to extend and customize your development workflow. Plugins can hook into the build process, add CLI commands, and integrate with the runtime.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Plugin Types](#plugin-types)
- [Built-in Plugins](#built-in-plugins)
- [Plugin Configuration](#plugin-configuration)
- [Creating Custom Plugins](#creating-custom-plugins)
- [External Plugin Loading](#external-plugin-loading)
- [Plugin API Reference](#plugin-api-reference)

## Architecture Overview

The GoSPA plugin system is built around interfaces that define how plugins interact with the CLI, build process, and runtime.

### Core Interfaces

```go
// Plugin is the base interface all plugins must implement
type Plugin interface {
    Name() string        // Returns the plugin name
    Init() error         // Called when plugin is loaded
    Dependencies() []Dependency  // Returns required dependencies
}

// CLIPlugin extends Plugin with hook and command support
type CLIPlugin interface {
    Plugin
    OnHook(hook Hook, ctx map[string]interface{}) error  // Handle lifecycle hooks
    Commands() []Command                                  // Provide CLI commands
}

// RuntimePlugin extends Plugin with runtime integration capabilities
type RuntimePlugin interface {
    Plugin
    Config() PluginConfig                      // Plugin configuration schema
    Middlewares() []interface{}                // Fiber handlers to inject
    TemplateFuncs() map[string]interface{}      // Template functions
}
```

### Lifecycle Hooks

Plugins can respond to lifecycle events:

| Hook | When | Context |
|------|------|---------|
| `BeforeGenerate` | Before code generation | `nil` |
| `AfterGenerate` | After code generation | `nil` |
| `BeforeDev` | Before dev server starts | `nil` |
| `AfterDev` | After dev server stops | `nil` |
| `BeforeBuild` | Before production build | `{"config": *BuildConfig}` |
| `AfterBuild` | After production build | `{"config": *BuildConfig}` |
| `BeforeServe` | Before HTTP server starts | `{"fiber": *fiber.App}` |
| `AfterServe` | After HTTP server starts | `{"fiber": *fiber.App}` |
| `BeforePrune` | Before state pruning/cleanup | `nil` |
| `AfterPrune` | After state pruning/cleanup | `nil` |
| `OnError` | When an error occurs | `{"error": string}` |

### Dependency Types

Plugins declare their dependencies with type information:

```go
type DependencyType string

const (
    DepGo  DependencyType = "go"  // Go module dependency
    DepBun DependencyType = "bun" // Bun/JavaScript dependency
)

type Dependency struct {
    Type    DependencyType
    Name    string    // Package name (e.g., "github.com/example/pkg")
    Version string    // Version constraint (e.g., "v1.2.3", "^4.0.0")
}
```

## Plugin Types

GoSPA supports three types of plugins:

### 1. Base Plugin
The minimal plugin type that provides name, initialization, and dependencies.

```go
type MyPlugin struct{}

func (p *MyPlugin) Name() string        { return "myplugin" }
func (p *MyPlugin) Init() error         { return nil }
func (p *MyPlugin) Dependencies() []plugin.Dependency {
    return []plugin.Dependency{}
}
```

### 2. CLI Plugin
Extends base plugin with CLI hooks and commands.

```go
type MyCLIPlugin struct{}

func (p *MyCLIPlugin) OnHook(hook plugin.Hook, ctx map[string]interface{}) error {
    switch hook {
    case plugin.BeforeBuild:
        // Run before production build
    }
    return nil
}

func (p *MyCLIPlugin) Commands() []plugin.Command {
    return []plugin.Command{
        {Name: "my:command", Alias: "mc", Description: "My command"},
    }
}
```

### 3. Runtime Plugin
Extends base plugin with runtime integration capabilities (middleware, template functions).

```go
type MyRuntimePlugin struct{}

func (p *MyRuntimePlugin) Config() plugin.PluginConfig {
    return plugin.PluginConfig{
        Schema: map[string]plugin.FieldSchema{
            "option1": {Type: "string", Description: "An option"},
        },
        Defaults: map[string]interface{}{"option1": "default"},
    }
}

func (p *MyRuntimePlugin) Middlewares() []interface{} {
    return []interface{}{
        func(c *fiber.Ctx) error { return c.Next() },
    }
}

func (p *MyRuntimePlugin) TemplateFuncs() map[string]interface{} {
    return map[string]interface{}{
        "myHelper": func() string { return "helper output" },
    }
}
```

## Built-in Plugins

### Tailwind CSS

Adds Tailwind CSS v4 support with CSS-first configuration, content scanning, and watch mode.

**Installation:**
```bash
gospa add tailwind
# or
gospa add:tailwind
```

**Configuration (`gospa.yaml`):**
```yaml
plugins:
  tailwind:
    input: ./styles/main.css      # Input CSS file (default: ./styles/main.css)
    output: ./static/css/main.css # Output CSS file (default: ./static/css/main.css)
    content:                      # Content paths for class scanning
      - ./routes/**/*.templ
      - ./components/**/*.templ
      - ./views/**/*.go
    minify: true                  # Minify in production (default: true)
```

**CLI Commands:**
| Command | Alias | Description |
|---------|-------|-------------|
| `add:tailwind` | `at` | Install Tailwind deps and create starter files |
| `tailwind:build` | `tb` | Build CSS for production (with minification) |
| `tailwind:watch` | `tw` | Watch and rebuild CSS on changes |

**Usage:**
1. Run `gospa add:tailwind` to install dependencies and create starter files
2. Create `styles/main.css`:
```css
@import 'tailwindcss';

@theme {
    --font-display: 'Satoshi', sans-serif;
    --color-primary: oklch(0.6 0.2 250);
}
```
3. The plugin automatically runs during `gospa dev` (watch mode) and `gospa build` (production)

**Lifecycle Hooks:**
- `BeforeDev`: Starts Tailwind CLI in watch mode
- `BeforeBuild`: Builds minified CSS for production
- `AfterDev`: Stops the watch process gracefully

**Dependencies:**
- `@tailwindcss/cli` (bun) - Tailwind CSS v4 CLI

---

### PostCSS

PostCSS processing with Tailwind CSS v4 integration and additional plugins.

**Installation:**
```bash
gospa add postcss
# or
gospa add:postcss
```

**Configuration:**
```yaml
plugins:
  postcss:
    input: ./styles/main.css      # Input CSS file (default: ./styles/main.css)
    output: ./static/css/main.css # Output CSS file (default: ./static/css/main.css)
    watch: true                   # Watch mode in dev (default: true)
    minify: true                  # Minify in production (default: true)
    sourceMap: false              # Generate source maps (default: false)
    plugins:                      # PostCSS plugins to enable
      typography: true            # @tailwindcss/typography
      forms: true                 # @tailwindcss/forms
      aspectRatio: true           # @tailwindcss/aspect-ratio
      autoprefixer: true          # Add vendor prefixes
      cssnano: false              # Advanced minification
      postcssNested: true         # Nested CSS support
```

### Critical CSS and Async Loading

The PostCSS plugin supports critical CSS extraction to improve page load performance:

1. **Critical CSS** - Above-the-fold styles that are inlined directly in the HTML `<head>`
2. **Non-critical CSS** - Below-the-fold styles that are loaded asynchronously

**Setup:**

1. Enable critical CSS in your `gospa.yaml`:
```yaml
plugins:
  postcss:
    criticalCSS:
      enabled: true
      criticalOutput: ./static/css/critical.css      # Inlined in HTML head
      nonCriticalOutput: ./static/css/non-critical.css  # Loaded asynchronously
      inlineMaxSize: 14336        # Max bytes for inline CSS (14KB = single round-trip)
```

2. Extract critical CSS:
```bash
gospa postcss:critical
```

3. Import the postcss package in your layout:
```go
import (
    "github.com/aydenstechdungeon/gospa/plugin/postcss"
    // ... other imports
)
```

4. Use the helper functions in your templ files:
```templ
<!-- Inlined Critical CSS (render-blocking, single round-trip) -->
@templ.Raw("<style>" + postcss.CriticalCSS("./static/css/critical.css") + "</style>")

<!-- Async load non-critical CSS (non-blocking) -->
@templ.Raw(postcss.AsyncCSS("/static/css/non-critical.css"))
```

**How it works:**
The `CriticalCSS` helper reads the critical CSS file and returns its content as a string for inlining. The `AsyncCSS` helper generates a preload link that loads the non-critical CSS asynchronously:

```html
<link rel="preload" href="/static/css/non-critical.css" as="style" onload="this.onload=null;this.rel='stylesheet'">
<noscript><link rel="stylesheet" href="/static/css/non-critical.css"></noscript>
```

**Performance:** The 14KB default `inlineMaxSize` ensures critical CSS fits within a single TCP round-trip (HTTP/2 initial window), minimising render-blocking time.
**CSS-safe extraction:** The splitter always cuts at a complete CSS rule boundary (`}`), never mid-declaration.

**Helper Function Reference:**

| Function | Signature | Description |
|----------|-----------|-------------|
| `CriticalCSS` | `func CriticalCSS(path string) string` | Reads critical CSS file; returns empty string if file not found |
| `CriticalCSSWithFallback` | `func CriticalCSSWithFallback(path, fallback string) string` | Like `CriticalCSS` but returns `fallback` if file not found (useful in dev) |
| `AsyncCSS` | `func AsyncCSS(path string) string` | Returns HTML string for async CSS loading with preload + noscript fallback |
| `GenerateCriticalCSSHelper` | `func GenerateCriticalCSSHelper(projectDir, criticalCSSPath string) (string, error)` | Reads critical CSS relative to `projectDir`; returns error on failure |
| `GenerateAsyncCSSScript` | `func GenerateAsyncCSSScript(cssPath string) string` | Low-level helper that generates the preload/noscript HTML string |

### CSS Bundle Splitting

For multi-page applications, you can split CSS into separate bundles to reduce per-page payload:

```yaml
plugins:
  postcss:
    bundles:
      - name: marketing
        input: ./styles/marketing.css
        output: ./static/css/marketing.css
        content:
          - ./routes/marketing/**/*.templ
      - name: app
        input: ./styles/app.css
        output: ./static/css/app.css
        content:
          - ./routes/app/**/*.templ
```

Each bundle:
- Scans only its specified content paths for Tailwind class detection
- Can have its own critical CSS extraction
- Is built independently with `gospa postcss:bundles`

**Usage:**
1. Run `gospa add:postcss` to install dependencies and create `postcss.config.js`
2. The plugin automatically processes CSS during `gospa dev` and `gospa build`

**Generated `postcss.config.js`:**
```javascript
export default {
  plugins: {
    '@tailwindcss/postcss': {},
  },
};
```

**Lifecycle Hooks:**
- `BeforeDev`: Starts PostCSS in watch mode
- `BeforeBuild`: Processes CSS for production
- `AfterDev`: Stops the watch process gracefully

**Dependencies:**
- `postcss` (bun)
- `@tailwindcss/postcss` (bun)
- `@tailwindcss/typography` (bun, optional)
- `@tailwindcss/forms` (bun, optional)

---

### Image Optimization

Optimize images for production with responsive sizes.

**Installation:**
```bash
gospa add image
```

**Configuration:**
```yaml
plugins:
  image:
    input: ./static/images
    output: ./static/images/optimized
    formats:
      - webp
      - avif
      - jpeg
      - png
    widths: [320, 640, 1280, 1920]
    quality: 85
    on_the_fly: false  # Enable runtime optimization
```

**Requirements:** `cgo` enabled with system libraries `libwebp` and `libheif` installed.

**CLI Commands:**
| Command | Alias | Description |
|---------|-------|-------------|
| `image:optimize` | `io` | Optimize all images |
| `image:clean` | `ic` | Clean optimized images |
| `image:sizes` | `is` | List image sizes |

---

### Form Validation

Client and server-side form validation with Valibot and Go validator.

**Installation:**
```bash
gospa add validation
```

**Configuration:**
```yaml
plugins:
  validation:
    schemas_dir: ./schemas
    output_dir: ./generated/validation
```

**CLI Commands:**
| Command | Alias | Description |
|---------|-------|-------------|
| `validation:generate` | `vg` | Generate validation code |
| `validation:create` | `vc` | Create schema file |
| `validation:list` | `vl` | List all schemas |

---

### SEO Optimization

Generate SEO assets including sitemap, meta tags, and structured data.

**Installation:**
```bash
gospa add seo
```

**Configuration:**
```yaml
plugins:
  seo:
    site_url: https://example.com
    site_name: My GoSPA Site
    site_description: A modern web application
    generate_sitemap: true
    generate_robots: true
    default_image: /images/og-default.png
```

**CLI Commands:**
| Command | Alias | Description |
|---------|-------|-------------|
| `seo:generate` | `sg` | Generate sitemap and robots.txt |
| `seo:meta` | `sm` | Generate meta tags |
| `seo:structured` | `ss` | Generate JSON-LD |

---

### Authentication

Complete authentication solution with OAuth2, JWT, and OTP support.

**Installation:**
```bash
gospa add auth
```

**Configuration:**
```yaml
plugins:
  auth:
    jwt_secret: ${JWT_SECRET}  # Use environment variable
    jwt_expiry: 24             # Hours
    oauth_providers:
      - google
      - github
    otp_enabled: true
    otp_issuer: MyGoSPAApp
```

**Usage:**
```go
import "github.com/aydenstechdungeon/gospa/plugin/auth"

// Initialize auth
authPlugin := auth.New(&auth.Config{
    JWTSecret:  "your-secret",
    JWTExpiry:  24,
    OTPEnabled: true,
})

// Create JWT token
token, err := authPlugin.CreateToken(userID, userEmail, role)

// Validate token (includes issuer validation)
claims, err := authPlugin.ValidateToken(token)

// Generate OTP for 2FA
otpSecret, qrURL, err := authPlugin.GenerateOTP(userEmail)

// Verify OTP code
valid := authPlugin.VerifyOTP(secret, code)
```

---

### QR Code

Pure Go QR code generation plugin for URLs, OTP/TOTP setup, and general use.

**Installation:**
```bash
gospa add qrcode
```

**Usage:**

```go
import "github.com/aydenstechdungeon/gospa/plugin/qrcode"

// Generate a QR code as data URL (for HTML img src)
dataURL, err := qrcode.GenerateDataURL("https://example.com")

// Generate for OTP/TOTP setup
otpURL := "otpauth://totp/MyApp:user@example.com?secret=JBSWY3DPEHPK3PXP&issuer=MyApp"
qrDataURL, err := qrcode.ForOTP(otpURL)
```

## Plugin Configuration

Plugins are configured via `gospa.yaml` in your project root:

```yaml
# gospa.yaml

# Global plugin settings
plugins_dir: ./plugins

# Plugin configurations
plugins:
  tailwind:
    input: ./styles/main.css
    output: ./static/css/main.css
  
  image:
    input: ./static/images
    output: ./static/images/optimized
    formats: [webp, avif, jpeg, png]
    widths: [320, 640, 1280]
  
  seo:
    site_url: https://example.com
    site_name: My Site
    generate_sitemap: true
  
  auth:
    jwt_secret: ${JWT_SECRET}
    jwt_expiry: 24
    oauth_providers: [google, github]
    otp_enabled: true
```

### Environment Variables

Use `${VAR_NAME}` syntax to reference environment variables:

```yaml
plugins:
  auth:
    jwt_secret: ${JWT_SECRET}
    github_client_id: ${GITHUB_CLIENT_ID}
```

## Creating Custom Plugins

### Basic Plugin Structure

```go
package myplugin

import (
    "github.com/aydenstechdungeon/gospa/plugin"
)

type MyPlugin struct {
    config Config
}

type Config struct {
    Option1 string `yaml:"option1"`
    Option2 bool   `yaml:"option2"`
}

func New(config Config) *MyPlugin {
    return &MyPlugin{config: config}
}

func (p *MyPlugin) Name() string {
    return "myplugin"
}

func (p *MyPlugin) Init() error {
    // Initialize plugin
    return nil
}

func (p *MyPlugin) Dependencies() []plugin.Dependency {
    return []plugin.Dependency{
        {Type: plugin.DepGo, Name: "github.com/example/pkg", Version: "v1.0.0"},
    }
}
```

### CLI Plugin with Hooks

```go
package myplugin

import (
    "github.com/aydenstechdungeon/gospa/plugin"
)

type MyCLIPlugin struct {
    MyPlugin
}

func (p *MyCLIPlugin) OnHook(hook plugin.Hook, ctx map[string]interface{}) error {
    switch hook {
    case plugin.BeforeBuild:
        // Run before production build
        return p.beforeBuild(ctx)
    case plugin.AfterBuild:
        // Run after production build
        return p.afterBuild(ctx)
    case plugin.BeforeServe:
        // Run before HTTP server starts
        return p.beforeServe(ctx)
    case plugin.OnError:
        // Handle errors
        if err, ok := ctx["error"].(string); ok {
            return p.handleError(err)
        }
    }
    return nil
}

func (p *MyCLIPlugin) Commands() []plugin.Command {
    return []plugin.Command{
        {
            Name:        "myplugin:run",
            Alias:       "mr",
            Description: "Run my plugin",
            Action:      p.runCommand,
        },
    }
}
```

### Runtime Plugin with Middlewares

```go
package myplugin

import (
    "github.com/aydenstechdungeon/gospa/plugin"
    "github.com/gofiber/fiber/v3"
)

type MyRuntimePlugin struct{}

func (p *MyRuntimePlugin) Config() plugin.PluginConfig {
    return plugin.PluginConfig{
        Schema: map[string]plugin.FieldSchema{
            "apiKey": {
                Type:        "string",
                Description: "API key for the service",
                Required:    true,
            },
        },
        Defaults: map[string]interface{}{
            "apiKey": "",
        },
    }
}

func (p *MyRuntimePlugin) Middlewares() []interface{} {
    return []interface{}{
        func(c fiber.Ctx) error {
            // Custom middleware logic
            return c.Next()
        },
    }
}

func (p *MyRuntimePlugin) TemplateFuncs() map[string]interface{} {
    return map[string]interface{}{
        "formatDate": func(date interface{}) string {
            // Custom date formatting
            return "formatted"
        },
    }
}
```

### Registering Plugins

You can register plugins in two ways:

**1. Using the global registry (for CLI plugins):**

```go
import (
    "github.com/aydenstechdungeon/gospa/plugin"
    "your-project/plugins/myplugin"
)

func init() {
    plugin.Register(myplugin.New(myplugin.Config{
        Option1: "value",
        Option2: true,
    }))
}
```

**2. Using App-level registration (recommended for runtime plugins):**

```go
import (
    "github.com/aydenstechdungeon/gospa"
    "your-project/plugins/myplugin"
)

func main() {
    app := gospa.New(config)

    // Register a single plugin
    if err := app.UsePlugin(myplugin.New(myplugin.Config{
        Option1: "value",
    })); err != nil {
        log.Fatal(err)
    }

    // Or register multiple plugins at once
    app.UsePlugins(plugin1, plugin2, plugin3)

    // Run the application
    app.Run(":3000")
}
```

### Plugin State Management

Plugins can be enabled/disabled at runtime:

```go
// Enable a plugin
plugin.Enable("myplugin")

// Disable a plugin
plugin.Disable("myplugin")

// Or via App methods
app.GetPlugin("myplugin")  // Get plugin instance
app.ListPlugins()          // List all registered plugins
```

## External Plugin Loading

GoSPA supports loading plugins from external GitHub repositories:

```go
import "github.com/aydenstechdungeon/gospa/plugin"

// Create a loader
loader := plugin.NewExternalPluginLoader()

// Load a plugin from GitHub
// Supported formats:
// - github.com/owner/repo
// - github.com/owner/repo@version
// - owner/repo
// - owner/repo@version
p, err := loader.LoadFromGitHub("github.com/username/gospa-plugin-example")

// Or use the convenience functions
err := plugin.InstallPlugin("username/gospa-plugin-example")
err := plugin.UninstallPlugin("username/gospa-plugin-example")

// List installed plugins
entries, err := plugin.ListInstalledPlugins()

// Discover available plugins
entries, err := plugin.DiscoverPlugins()

// Search plugins
results, err := plugin.SearchPlugins("tailwind")
```

External plugins are cached in `~/.gospa/plugins/` by default.

## Plugin API Reference

### Plugin Interface

```go
type Plugin interface {
    Name() string
    Init() error
    Dependencies() []Dependency
}
```

### RuntimePlugin Interface

```go
type RuntimePlugin interface {
    Plugin
    Config() PluginConfig
    Middlewares() []interface{}
    TemplateFuncs() map[string]interface{}
}
```

### CLIPlugin Interface

```go
type CLIPlugin interface {
    Plugin
    OnHook(hook Hook, ctx map[string]interface{}) error
    Commands() []Command
}
```

### Command Structure

```go
type Command struct {
    Name        string
    Alias       string
    Description string
    Action      func(args []string) error
    Flags       []Flag
}

type Flag struct {
    Name        string
    Shorthand   string
    Description string
    Default     interface{}
}
```

### Hook Types

```go
type Hook string

const (
    BeforeGenerate Hook = "before:generate"
    AfterGenerate  Hook = "after:generate"
    BeforeDev      Hook = "before:dev"
    AfterDev       Hook = "after:dev"
    BeforeBuild    Hook = "before:build"
    AfterBuild     Hook = "after:build"
    BeforeServe    Hook = "before:serve"
    AfterServe     Hook = "after:serve"
    BeforePrune    Hook = "before:prune"
    AfterPrune     Hook = "after:prune"
    OnError        Hook = "on:error"
)
```

### Plugin State

```go
type PluginState int

const (
    StateEnabled  PluginState = iota  // Plugin is active
    StateDisabled                     // Plugin is loaded but inactive
    StateError                        // Plugin failed to load
)

type PluginInfo struct {
    Name        string
    Version     string
    Description string
    Author      string
    State       PluginState
}
```

### Plugin Configuration Schema

```go
type PluginConfig struct {
    Schema   map[string]FieldSchema
    Defaults map[string]interface{}
}

type FieldSchema struct {
    Type        string
    Description string
    Required    bool
    Default     interface{}
}
```

### Registry Functions

```go
// Register/unregister plugins
func Register(p Plugin) error
func Unregister(name string)

// Get plugins
func GetPlugin(name string) Plugin
func GetPlugins() []Plugin
func GetPluginInfo(name string) (PluginInfo, bool)
func GetAllPluginInfo() []PluginInfo

// Get plugins by type
func GetCLIPlugins() []CLIPlugin
func GetRuntimePlugins() []RuntimePlugin

// Plugin state
func Enable(name string) error
func Disable(name string) error

// Trigger hooks
func TriggerHook(hook Hook, ctx map[string]interface{}) error
func TriggerHookForPlugin(name string, hook Hook, ctx map[string]interface{}) error

// Run commands
func RunCommand(name string, args []string) (bool, error)

// Dependencies
func GetAllDependencies() []Dependency
func ResolveDependencies() error
```

### App Plugin Methods

```go
func (a *App) UsePlugin(p plugin.Plugin) error
func (a *App) UsePlugins(plugins ...plugin.Plugin) error
func (a *App) GetPlugin(name string) (plugin.Plugin, bool)
func (a *App) ListPlugins() []plugin.PluginInfo
```

## Best Practices

1. **Keep plugins focused**: Each plugin should do one thing well
2. **Document configuration**: Provide clear YAML configuration examples
3. **Handle errors gracefully**: Return meaningful error messages
4. **Use semantic versioning**: Follow semver for plugin versions
5. **Test with multiple GoSPA versions**: Ensure compatibility
6. **Minimize dependencies**: Only include necessary dependencies
7. **Provide CLI commands**: Make common tasks accessible via CLI
8. **Support environment variables**: Allow sensitive values via env vars
9. **Implement RuntimePlugin**: For runtime integration, implement Middlewares() and TemplateFuncs()
10. **Use App registration**: Prefer `app.UsePlugin()` over global `plugin.Register()` for runtime plugins

## Troubleshooting

### Plugin Not Loading

1. Check plugin is registered in `init()` function or via `app.UsePlugin()`
2. Verify dependencies are installed
3. Check `gospa.yaml` configuration
4. Review the CLI output for plugin initialization errors

### Dependency Issues

```bash
# Install Go dependencies
go get github.com/example/pkg@v1.0.0

# Install Bun dependencies
bun add package-name
```

### Hook Not Firing

1. Ensure plugin implements `CLIPlugin` interface
2. Check hook is registered correctly
3. Verify plugin is loaded before hook fires

### Configuration Not Applied

1. Check YAML syntax is correct
2. Verify environment variables are set
3. Re-run the command and inspect any configuration parsing errors
