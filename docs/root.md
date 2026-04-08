# Framework Root & Layouts

GoSPA's core architecture begins with a root layout component that wraps every page and is the entry point for both server-side rendering and client-side hydration.

## Root Layout

The root layout is typically defined in `routes/layout.templ`. This is where you declare your HTML document structure, including the `<html>`, `<head>`, and `<body>` tags.

```templ
package routes

import "github.com/aydenstechdungeon/gospa/fiber"

templ RootLayout(children templ.Component, props map[string]interface{}) {
    <!DOCTYPE html>
    <html lang="en" data-gospa-auto>
        <head>
            <meta charset="utf-8" />
            <title>My GoSPA App</title>
            @fiber.RuntimeScript()
        </head>
        <body>
            <div id="root">
                @children
            </div>
        </body>
    </html>
}
```

## Nested Layouts

Directories under `routes/` can also contain their own `layout.templ` files. These layouts are automatically nested, allowing you to create complex UI hierarchies with ease.

### Layout Nesting example:
1.  **`routes/layout.templ`**: The top-level layout.
2.  **`routes/admin/layout.templ`**: A layout for all admin-specific pages.
3.  **`routes/admin/settings/page.templ`**: The settings page.

The `settings` page will be rendered inside the `admin` layout, which will then be rendered inside the `root` layout.

## Data Loading for Layouts

Like pages, layouts can have their own server-side `Load` function to fetch data that is common to all their children (e.g., user session or site-wide configuration).

```go
func Load(c routing.LoadContext) (map[string]interface{}, error) {
    user := getCurrentUser()
    return map[string]interface{}{"user": user}, nil
}
```

## Special Components

The root layout is the ideal place to include global components that are used on every page:
- **`@fiber.RuntimeScript()`**: Injects the GoSPA client-side runtime.
- **`@fiber.HMRScript()`**: Injects the HMR connection in development mode.
- **Global Error Handlers**: Define how fatal rendering crashes are displayed at the top level.
