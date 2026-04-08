# Dynamic Routing & Groups

GoSPA features a flexible routing system that supports dynamic segments, wildcards, and regex constraints.

## Dynamic Segments

Use underscores or brackets in filenames to create dynamic route parameters.

| Pattern | Filename | URL Example |
|---------|----------|-------------|
| `:id` | `_id/page.templ` | `/users/123` |
| `*path` | `*path/page.templ` | `/files/a/b/c` |

## Route Groups

Route groups allow you to organize routes into logical groups without affecting the URL path. Create groups by wrapping a folder name in parentheses: `(name)`.

```
routes/
├── (marketing)/
│   ├── about/page.templ      → /about
│   └── contact/page.templ    → /contact
├── (shop)/
│   ├── products/page.templ   → /products
│   └── cart/page.templ       → /cart
```

## Regex Constraints

You can define custom regex for parameters in your components or via manual registration:

```go
// Matches digits only
extractor := routing.NewParamExtractor("/users/{id:\\d+}")
```

## Performance

GoSPA uses an optimized `Router` with static path indexing. Exact path lookups are $O(1)$ and dynamic routes are matched with optimized regex patterns.
