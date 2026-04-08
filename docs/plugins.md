# Plugin Architecture

GoSPA's plugin system allows developers to extend the framework's core capabilities. It provides a modular way to add new features, middleware, and CLI commands.

## Plugin Interface

All plugins must implement the basic `Plugin` interface:

```go
type Plugin interface {
    Name() string
    Init() error
    Dependencies() []Dependency
}
```

## Runtime Plugins

Runtime plugins can inject middleware and template functions into the application.

```go
type RuntimePlugin interface {
    Plugin
    Config() Config
    Middlewares() []interface{} // []fiber.Handler
    TemplateFuncs() map[string]interface{}
}
```

### Dependency Management
If your plugin depends on external Go modules or Bun packages, describe them in your `Dependencies` function:

```go
func (p *MyPlugin) Dependencies() []plugin.Dependency {
    return []plugin.Dependency{
        { Type: plugin.DepGo, Name: "golang.org/x/oauth2", Version: "latest" },
        { Type: plugin.DepBun, Name: "valibot", Version: "latest" },
    }
}
```

## CLI Plugins

CLI plugins allow you to register new lifecycle hooks or custom commands.

```go
type CLIPlugin interface {
    Plugin
    OnHook(hook Hook, ctx map[string]interface{}) error
    Commands() []Command
}
```

### Lifecycle Hooks
A CLI plugin can interact with several lifecycle hooks:
- `BeforeGenerate`: Run before code generation.
- `AfterGenerate`: Run after code generation.
- `BeforeDev`: Run before the dev server starts.
- `BeforeBuild`: Run before production build.
- `OnError`: Run when a fatal error occurs.

## Registration

Plugins are registered in your `gospa.App` during initialization:

```go
app := gospa.New(config)
app.UsePlugin(new(auth.AuthPlugin))
app.UsePlugins(new(image.ImagePlugin), new(seo.SEOPlugin))
```

GoSPA automatically handles the initialization and dependency resolution for all registered plugins.
