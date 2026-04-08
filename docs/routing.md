# File-Based Routing

GoSPA uses an opinionated file-based routing system inspired by SvelteKit and Next.js. This system simplifies the creation of pages, layouts, and API routes.

## Pages and Layouts

Every directory under your `routes/` folder represents a route. The actual content is defined in `page.templ` and `layout.templ` files.

### Route structure example:
```text
routes/
├── layout.templ      (Root layout for all pages)
├── page.templ        (Home page at /)
├── about/
│   └── page.templ    (About page at /about)
└── dashboard/
    ├── layout.templ  (Nested dashboard layout)
    └── settings/
        └── page.templ (Settings page at /dashboard/settings)
```

## Special Files

- **`page.templ`**: Defines the main component for a route.
- **`layout.templ`**: Defines a wrapper for all child pages in that directory.
- **`loading.templ`**: Defines a component to show while the page's `Load` function is running.
- **`error.templ`**: Defines a component to show if a route or layout crashes.
- **`server.go`**: Contains server-side Go logic like `Load` and `Action` functions for the route.

## Path Parameters

Dynamic routes use square brackets to indicate parameters.

### Example:
- `routes/blog/[slug]/page.templ` matches `/blog/my-first-post` where `slug` is "my-first-post".

You can access parameters in your `Load` function:
```go
func Load(c routing.LoadContext) (map[string]interface{}, error) {
    slug := c.Param("slug")
    return map[string]interface{}{"post": getPost(slug)}, nil
}
```

## Rendering Strategies

GoSPA supports multiple rendering strategies per route:

1.  **SSR (Server Side Rendering)**: Renders fresh on every request (default).
2.  **SSG (Static Site Generation)**: Renders once and caches the result.
3.  **ISR (Incremental Static Regeneration)**: Stale-while-revalidate strategy for periodic updates.
4.  **PPR (Partial Prerendering)**: Renders a static shell and streams dynamic "slots" per-request.

### Configuring Strategies
Strategies are configured in your route's `server.go`:
```go
routing.RegisterPageWithOptions("/blog", MyBlogPage, routing.RouteOptions{
    Strategy: routing.StrategyISR,
    RevalidateAfter: 1 * time.Hour,
})
```

## Middleware

Route middleware allows you to intercept requests before they reach your components. Standard Fiber handlers can be registered as middleware.

```go
routing.RegisterMiddleware("/admin", auth.AuthMiddleware)
```
GoSPA ensures middleare for `/admin` correctly chains into `/admin/settings`.
