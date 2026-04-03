# Routing Layouts & Special Files

GoSPA uses special filenames within the `routes/` directory to construct the application layout, middleware chain, error boundaries, and loading states automatically.

## Special Routing Files

| Filename | Purpose | Scope |
|----------|---------|-------|
| `page.templ` / `page.gospa` | Renders the primary component for a route directory. | Current route |
| `layout.templ` / `layout.gospa` | Wraps all nested child pages inside a particular directory segment. | Segment and children |
| `root_layout.templ` | The outermost HTML wrapper (`<html>`, `<body>`). Must include the GoSPA scripts. | Global (root only) |
| `_middleware.go` | Segment-scoped middleware intercepting requests before they hit pages. | Segment and children |
| `_error.templ` / `_error.gospa` | Error boundary. If a page panics or returns an error during SSR, it falls back to this. | Segment and children |
| `_loading.templ` / `_loading.gospa` | Automatically compiled as the default static shell during PPR (Partial Page Rendering). | Segment and children |

## Root Layout (`root_layout.templ`)

The root layout is the entry point for your application's HTML. It must include the GoSPA runtime script:

```templ
package routes

import "github.com/aydenstechdungeon/gospa"

templ RootLayout(content templ.Component) {
    <!DOCTYPE html>
    <html lang="en">
        <head>
            <meta charset="UTF-8" />
            <title>My GoSPA App</title>
            @gospa.Scripts()
        </head>
        <body>
            <div data-gospa-root>
                @content
            </div>
        </body>
    </html>
}
```

## Nested Layouts (`layout.templ`)

Layouts wrap all pages in their directory. They receive the child page via the `children` prop.

# Middleware Files (`_middleware.go`)

Middleware files automatically apply their `Handler` to all routes in their directory and subdirectories.

```go
func init() {
    routing.RegisterMiddleware("/admin", func(c fiber.Ctx) error {
        if !isAdmin(c) {
            return c.Redirect().Status(302).To("/login")
        }
        return c.Next()
    })
}
```
