# GoSPA Plugin System

GoSPA features a powerful plugin system that allows you to extend and customize your development workflow. Plugins can hook into the build process, add CLI commands, and integrate with external tools.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Built-in Plugins](#built-in-plugins)
- [Plugin Configuration](#plugin-configuration)
- [Creating Custom Plugins](#creating-custom-plugins)
- [Plugin API Reference](#plugin-api-reference)

## Architecture Overview

The GoSPA plugin system is built around interfaces that define how plugins interact with the CLI and build process.

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
```

### Lifecycle Hooks

Plugins can respond to lifecycle events:

| Hook | When | Context |
|------|------|---------|
| `BeforeDev` | Before dev server starts | `nil` |
| `AfterDev` | After dev server stops | `nil` |
| `BeforeBuild` | Before production build | `{"config": *BuildConfig}` |
| `AfterBuild` | After production build | `{"config": *BuildConfig}` |
| `BeforeGenerate` | Before code generation | `nil` |
| `AfterGenerate` | After code generation | `nil` |

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

    # Critical CSS extraction for performance
    criticalCSS:
      enabled: false              # Enable critical CSS extraction
      criticalOutput: ./static/css/critical.css      # Inlined in HTML
      nonCriticalOutput: ./static/css/non-critical.css  # Async loaded
      inlineMaxSize: 14336        # Max bytes for inline CSS (14KB default)
      dimensions:                 # Viewport sizes for critical CSS
        - width: 1300
          height: 900
          name: desktop
        - width: 500
          height: 900
          name: mobile

    # CSS bundle splitting for multi-page apps
    bundles:
      - name: marketing
        input: ./styles/marketing.css
        output: ./static/css/marketing.css
        content:                    # Content paths for this bundle
          - ./routes/marketing/**/*.templ
        criticalCSS:
          enabled: true
          criticalOutput: ./static/css/marketing.critical.css
          nonCriticalOutput: ./static/css/marketing.non-critical.css
      - name: dashboard
        input: ./styles/dashboard.css
        output: ./static/css/dashboard.css
        content:
          - ./routes/dashboard/**/*.templ
```

**CLI Commands:**
| Command | Alias | Description |
|---------|-------|-------------|
| `add:postcss` | `ap` | Install PostCSS deps and create config |
| `postcss:build` | `pb` | Build CSS for production |
| `postcss:watch` | `pw` | Watch and rebuild CSS on changes |
| `postcss:config` | `pc` | Generate PostCSS configuration file |
| `postcss:critical` | `pcr` | Extract critical CSS for above-the-fold content |
| `postcss:bundles` | `pbd` | Build all CSS bundles |

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
      dimensions:                 # Viewport sizes for critical CSS detection
        - width: 1300
          height: 900
          name: desktop
        - width: 500
          height: 900
          name: mobile
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

<!-- Note: Path is relative to your app's working directory -->
<!-- For apps running from root: "./website/static/css/critical.css" -->
<!-- For apps running from website dir: "./static/css/critical.css" -->

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

**Note on Tailwind v4 Plugins:**
Typography, forms, and aspect-ratio plugins are no longer added via `postcss.config.js`. Instead, they are automatically added to your `styles/main.css` via the `@plugin` directive when enabled in the GoSPA PostCSS configuration.

**Lifecycle Hooks:**
- `BeforeDev`: Starts PostCSS in watch mode
- `BeforeBuild`: Processes CSS for production
- `AfterDev`: Stops the watch process gracefully

**Dependencies:**
- `postcss` (bun)
- `postcss-cli` (bun)
- `@tailwindcss/postcss` (bun)
- `@tailwindcss/typography` (bun, optional)
- `@tailwindcss/forms` (bun, optional)
- `@tailwindcss/aspect-ratio` (bun, optional)

**Note:** Container queries and line-clamp are built into Tailwind CSS v4, so separate plugins are no longer needed for those features.

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
      - jpeg
    widths: [320, 640, 1280, 1920]
    quality: 85
    on_the_fly: false  # Enable runtime optimization
```

**CLI Commands:**
| Command | Alias | Description |
|---------|-------|-------------|
| `image:optimize` | `io` | Optimize all images |
| `image:clean` | `ic` | Clean optimized images |
| `image:sizes` | `is` | List image sizes |

**Features:**
- Build-time optimization (default)
- Optional on-the-fly processing
- WebP, JPEG, PNG support
- Responsive srcset generation
- No external dependencies (stdlib only)

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

**Usage:**

1. Define schema (`schemas/user.json`):
```json
{
  "name": "UserSchema",
  "fields": {
    "email": {"type": "string", "format": "email", "required": true},
    "password": {"type": "string", "minLength": 8, "required": true},
    "age": {"type": "integer", "min": 0, "max": 150}
  }
}
```

2. Generate validation:
```bash
gospa validation:generate
```

3. Use in Go:
```go
import "your-project/generated/validation"

user, err := validation.ValidateUser(data)
```

4. Use in TypeScript:
```typescript
import { UserSchema } from './generated/validation';
import * as v from 'valibot';

const result = v.safeParse(UserSchema, data);
```

**Dependencies:**
- `github.com/go-playground/validator/v10` (go)
- `valibot` (bun)

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

**Features:**
- Automatic sitemap.xml generation
- robots.txt with configurable rules
- Meta tags (title, description, keywords)
- Open Graph tags for social sharing
- Twitter Cards
- JSON-LD structured data (Organization, WebSite, Article, etc.)

**Usage in Templates:**
```go
import "github.com/aydenstechdungeon/gospa/plugin/seo"

// Generate meta tags
meta := seo.MetaTags(seo.MetaConfig{
    Title:       "Page Title",
    Description: "Page description",
    Image:       "/images/page.png",
    URL:         "https://example.com/page",
})

// Generate structured data
jsonLD := seo.StructuredData("Article", seo.ArticleData{
    Headline:   "Article Title",
    Author:     "John Doe",
    DatePublished: "2024-01-15",
})
```

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
      - facebook
      - microsoft
      - discord
      - telegram
      - twitter
    otp_enabled: true
    otp_issuer: MyGoSPAApp
    backup_code_count: 10
```

**CLI Commands:**
| Command | Alias | Description |
|---------|-------|-------------|
| `auth:generate` | `ag` | Generate auth code |
| `auth:secret` | `as` | Generate JWT secret |
| `auth:otp` | `ao` | Generate OTP secret + QR URL |
| `auth:backup` | `ab` | Generate backup codes |
| `auth:verify` | `av` | Verify OTP code |

**OAuth2 Provider Setup:**

1. **Google:**
   ```env
   GOOGLE_CLIENT_ID=your-client-id
   GOOGLE_CLIENT_SECRET=your-client-secret
   ```

2. **GitHub:**
   ```env
   GITHUB_CLIENT_ID=your-client-id
   GITHUB_CLIENT_SECRET=your-client-secret
   ```

3. **Facebook:**
   ```env
   FACEBOOK_CLIENT_ID=your-client-id
   FACEBOOK_CLIENT_SECRET=your-client-secret
   ```

4. **Microsoft:**
   ```env
   MICROSOFT_CLIENT_ID=your-client-id
   MICROSOFT_CLIENT_SECRET=your-client-secret
   ```

5. **Discord:**
   ```env
   DISCORD_CLIENT_ID=your-client-id
   DISCORD_CLIENT_SECRET=your-client-secret
   ```

6. **Telegram:**
   ```env
   TELEGRAM_BOT_TOKEN=your-bot-token
   ```
   Note: Telegram uses Login Widget flow (non-standard OAuth2). Create a bot via [@BotFather](https://t.me/botfather) and set your domain as the login domain.

7. **Twitter/X:**
   ```env
   TWITTER_CLIENT_ID=your-client-id
   TWITTER_CLIENT_SECRET=your-client-secret
   ```
   Note: Twitter uses OAuth 2.0 with PKCE flow.

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

// Validate token
claims, err := authPlugin.ValidateToken(token)

// Generate OTP for 2FA
otpSecret, qrURL, err := authPlugin.GenerateOTP(userEmail)

// Verify OTP code
valid := authPlugin.VerifyOTP(secret, code)

// Generate backup codes
backupCodes, err := auth.GenerateBackupCodes(10)
```

**Dependencies:**
- `github.com/golang-jwt/jwt/v5` (go)
- `golang.org/x/oauth2` (go)
- `github.com/pquerna/otp` (go)

---

### QR Code

Pure Go QR code generation plugin for URLs, OTP/TOTP setup, and general use.

**Installation:**
```bash
gospa add qrcode
```

**Configuration:**
```yaml
plugins:
  qrcode:
    default_size: 256        # Default QR code size in pixels
    default_level: medium    # Error correction: low, medium, quartile, high
```

**Usage:**

```go
import "github.com/aydenstechdungeon/gospa/plugin/qrcode"

// Generate a QR code as data URL (for HTML img src)
dataURL, err := qrcode.GenerateDataURL("https://example.com")
if err != nil {
    log.Fatal(err)
}
// Use in HTML: <img src="{{ .DataURL }}" />

// Generate with custom options
dataURL, err := qrcode.GenerateDataURL("https://example.com",
    qrcode.WithSize(512),
    qrcode.WithLevel(qrcode.LevelHigh),
)

// Generate PNG bytes
pngBytes, err := qrcode.GeneratePNG("https://example.com")

// Generate for OTP/TOTP setup
otpURL := "otpauth://totp/MyApp:user@example.com?secret=JBSWY3DPEHPK3PXP&issuer=MyApp"
qrDataURL, err := qrcode.ForOTP(otpURL)

// Create plugin instance with custom defaults
plugin := qrcode.NewWithConfig(qrcode.Config{
    DefaultSize:  400,
    DefaultLevel: "high",
})

// Use plugin instance
dataURL, err := plugin.GenerateDataURL("https://example.com")
```

**Package Functions:**

| Function | Description |
|----------|-------------|
| `Generate(content, ...Option)` | Generate QR as image.Image |
| `GeneratePNG(content, ...Option)` | Generate QR as PNG bytes |
| `GenerateBase64(content, ...Option)` | Generate QR as base64 string |
| `GenerateDataURL(content, ...Option)` | Generate QR as data URL |
| `ForOTP(otpURL, ...Option)` | Generate QR for OTP setup (300px default) |

**Options:**

| Option | Description | Default |
|--------|-------------|---------|
| `WithSize(int)` | Image size in pixels | 256 |
| `WithLevel(Level)` | Error correction level | LevelMedium |
| `WithColors(fg, bg)` | Foreground/background colors | Black/White |

**Error Correction Levels:**

| Level | Recovery | Use Case |
|-------|----------|----------|
| `LevelLow` | 7% | Clean environments |
| `LevelMedium` | 15% | General use (default) |
| `LevelQuartile` | 25% | Moderate damage risk |
| `LevelHigh` | 30% | High damage risk, logos/overlays |

**Features:**
- Multiple output formats: Image, PNG bytes, Base64, Data URL
- Configurable error correction levels
- Customizable size and colors
- Built-in OTP/TOTP QR code generation
- Functional options pattern for flexible configuration
- Integrates with Auth plugin for 2FA flows

**Dependencies:**
- `github.com/skip2/go-qrcode` (go)

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
    formats: [webp, jpeg]
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

func (p *MyCLIPlugin) runCommand(args []string) error {
    // Command implementation
    return nil
}
```

### Registering Plugins

Register your plugin in your application's main package:

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

## Plugin API Reference

### Plugin Interface

```go
type Plugin interface {
    Name() string
    Init() error
    Dependencies() []Dependency
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
    Name        string                           // Full command name
    Alias       string                           // Short alias
    Description string                           // Help text
    Action      func(args []string) error        // Command handler
    Flags       []Flag                           // Command flags
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
    BeforeDev      Hook = "before:dev"
    AfterDev       Hook = "after:dev"
    BeforeBuild    Hook = "before:build"
    AfterBuild     Hook = "after:build"
    BeforeGenerate Hook = "before:generate"
    AfterGenerate  Hook = "after:generate"
)
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

## Scope Note

This document covers the built-in plugin model that exists in the current codebase. Dynamic external plugin installation and shared-library plugin loading are not documented here because they are not part of the currently implemented CLI surface.

## Troubleshooting

### Plugin Not Loading

1. Check plugin is registered in `init()` function
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
