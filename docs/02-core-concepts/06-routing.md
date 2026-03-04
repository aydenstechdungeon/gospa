# GoSPA Route Parameters

GoSPA provides a comprehensive route parameter handling system for extracting, validating, and building URLs with path and query parameters.

## Overview

The parameter system in `routing/params.go` provides:

- **Params**: Type-safe parameter map with conversion methods
- **QueryParams**: Query string parameter handling
- **ParamExtractor**: Route matching and parameter extraction
- **PathBuilder**: URL path construction with parameters

---

## Params

The `Params` type is a map for storing route parameters with type-safe accessors.

### Basic Usage

```go
import "github.com/aydenstechdungeon/gospa/routing"

// Create params
params := routing.Params{
    "id":    "123",
    "name":  "john",
    "admin": "true",
}

// Get value
id := params.Get("id")  // "123"

// Get with default
name := params.GetDefault("nickname", "anonymous")  // "anonymous" if not found

// Check existence
if params.Has("id") {
    // Parameter exists
}

// Set value
params.Set("email", "john@example.com")

// Delete value
params.Delete("temp")

// Clone
cloned := params.Clone()

// Merge
otherParams := routing.Params{"role": "admin"}
params.Merge(otherParams)
```

### Type Conversion Methods

```go
// String
name := params.GetString("name")  // string

// Integer
id := params.GetInt("id")         // int
id64 := params.GetInt64("id")     // int64

// Float
price := params.GetFloat64("price")  // float64

// Boolean
admin := params.GetBool("admin")  // bool

// Slice (comma-separated)
tags := params.GetSlice("tags")   // []any

// JSON
json, err := params.ToJSON()      // []byte
```

### All Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `Get()` | `Get(key string) any` | Get parameter value |
| `GetDefault()` | `GetDefault(key string, def any) any` | Get with default |
| `GetString()` | `GetString(key string) string` | Get as string |
| `GetInt()` | `GetInt(key string) int` | Get as int |
| `GetInt64()` | `GetInt64(key string) int64` | Get as int64 |
| `GetFloat64()` | `GetFloat64(key string) float64` | Get as float64 |
| `GetBool()` | `GetBool(key string) bool` | Get as bool |
| `GetSlice()` | `GetSlice(key string) []any` | Get as slice |
| `Has()` | `Has(key string) bool` | Check if exists |
| `Set()` | `Set(key string, value any)` | Set parameter |
| `Delete()` | `Delete(key string)` | Delete parameter |
| `Clone()` | `Clone() Params` | Clone params |
| `Merge()` | `Merge(other Params)` | Merge params |
| `ToJSON()` | `ToJSON() ([]byte, error)` | Serialize to JSON |
| `Keys()` | `Keys() []string` | Get all keys |
| `Values()` | `Values() []any` | Get all values |

---

## QueryParams

Handle URL query string parameters.

### Creating QueryParams

```go
// From URL
url, _ := url.Parse("https://example.com/search?q=gospa&page=2")
queryParams := routing.NewQueryParams(url.Query())

// From map
queryParams := routing.NewQueryParamsFromMap(map[string][]string{
    "q":    {"gospa"},
    "page": {"2"},
})

// Empty
queryParams := routing.NewQueryParamsEmpty()
```

### Methods

```go
// Get first value
q := queryParams.Get("q")  // "gospa"

// Get all values (for multi-value params)
tags := queryParams.GetAll("tags")  // []string

// Get with default
page := queryParams.GetDefault("page", "1")  // "2" or "1" if not found

// Type conversions
page := queryParams.GetInt("page")        // int
price := queryParams.GetFloat64("price")  // float64
active := queryParams.GetBool("active")   // bool

// Check existence
if queryParams.Has("q") {
    // Query param exists
}

// Set value
queryParams.Set("sort", "desc")

// Add value (for multi-value params)
queryParams.Add("filter", "active")
queryParams.Add("filter", "pending")  // filter=active&filter=pending

// Delete
queryParams.Del("temp")

// Encode to string
encoded := queryParams.Encode()  // "q=gospa&page=2"

// Clone
cloned := queryParams.Clone()

// To URL values
values := queryParams.Values()  // url.Values
```

### QueryParams Methods Table

| Method | Signature | Description |
|--------|-----------|-------------|
| `Get()` | `Get(key string) string` | Get first value |
| `GetAll()` | `GetAll(key string) []string` | Get all values |
| `GetDefault()` | `GetDefault(key, def string) string` | Get with default |
| `GetInt()` | `GetInt(key string) int` | Get as int |
| `GetInt64()` | `GetInt64(key string) int64` | Get as int64 |
| `GetFloat64()` | `GetFloat64(key string) float64` | Get as float64 |
| `GetBool()` | `GetBool(key string) bool` | Get as bool |
| `Has()` | `Has(key string) bool` | Check if exists |
| `Set()` | `Set(key, value string)` | Set value |
| `Add()` | `Add(key, value string)` | Add value |
| `Del()` | `Del(key string)` | Delete key |
| `Encode()` | `Encode() string` | Encode to string |
| `Clone()` | `Clone() QueryParams` | Clone params |
| `Values()` | `Values() url.Values` | Get as url.Values |

---

## ParamExtractor

Extract parameters from routes during matching.

### Creating an Extractor

```go
// Create an extractor for a specific pattern
extractor := routing.NewParamExtractor("/users/:id")

// Match and extract
params, ok := extractor.Extract("/users/123")
if ok {
    id := params.Get("id")  // "123"
}
```

### Extracting Parameters

```go
// Match with wildcard
extractor = routing.NewParamExtractor("/files/*filepath")
params, ok = extractor.Extract("/files/docs/readme.txt")
if ok {
    filepath := params.Get("filepath")  // "docs/readme.txt"
}
```

### RouteMatch

```go
type RouteMatch struct {
    Pattern string  // Matched pattern
    Params  Params  // Extracted parameters
    Handler any     // Associated handler
}
```

### Pattern Syntax

| Pattern | Description | Example |
|---------|-------------|---------|
| `:name` | Named parameter | `/users/:id` matches `/users/123` |
| `*name` | Wildcard (greedy) | `/files/*path` matches `/files/a/b/c` |
| `{name:regex}` | Regex constraint | `/users/{id:\\d+}` matches digits only |

---

## Route Groups

Route groups allow you to organize routes into logical groups without affecting the URL path. This is useful for:

- Grouping related routes together
- Applying layouts to specific sections
- Keeping feature areas separated

### Syntax

Route groups are created by wrapping a folder name in parentheses: `(name)`

```
routes/
├── (marketing)/
│   ├── about/
│   │   └── page.templ      → /about
│   └── contact/
│       └── page.templ      → /contact
├── (shop)/
│   ├── products/
│   │   └── page.templ      → /products
│   └── cart/
│       └── page.templ      → /cart
└── page.templ              → /
```

### How It Works

When a folder is named with parentheses:

1. The folder name is **not** included in the URL path
2. Routes inside the group behave as if they were at the parent level
3. Multiple groups can exist at the same level

### Example Structure

```
routes/
├── (docs)/
│   ├── layout.templ        → Layout for /docs/* routes
│   ├── getting-started/
│   │   └── page.templ      → /getting-started
│   └── api/
│       └── page.templ      → /api
├── (auth)/
│   ├── login/
│   │   └── page.templ      → /login
│   └── register/
│       └── page.templ      → /register
└── page.templ              → /
```

### Benefits

1. **Organization**: Keep related routes together without URL bloat
2. **Layouts**: Apply layouts to groups of routes
3. **Clean URLs**: Maintain clean URL structures while organizing code

### Comparison with Other Patterns

| Pattern | URL Impact | Use Case |
|---------|------------|----------|
| `(name)` | Not included in URL | Organizational grouping |
| `_name` | Becomes `:name` | Dynamic parameter |
| `[name]` | Becomes `:name` | Dynamic parameter (bracket syntax) |
| `name` | Included as `/name` | Static path segment |

### Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `AddPattern()` | `AddPattern(pattern string)` | Add route pattern |
| `Match()` | `Match(path string) *RouteMatch` | Match path and extract |
| `MatchPattern()` | `MatchPattern(pattern, path string) *RouteMatch` | Match specific pattern |
| `Extract()` | `Extract(pattern, path string) Params` | Extract params only |
| `Patterns()` | `Patterns() []string` | Get all patterns |

---

## PathBuilder

Build URL paths with parameters.

### Creating a PathBuilder

```go
builder := routing.NewPathBuilder("/users/:userId/posts/:postId")
builder.Param("userId", "1")
builder.Param("postId", "42")
path := builder.Build()
// Result: "/users/1/posts/42"
```

### With Query Parameters

```go
builder = routing.NewPathBuilder("/search")
builder.Query("q", "gospa")
builder.Query("page", "1")
path = builder.Build()
// Result: "/search?q=gospa&page=1"
```

### URL Encoding

```go
// Automatic URL encoding
path := builder.Build("/search/:q", routing.Params{
    "q": "hello world",
})
// Result: "/search/hello%20world"
```

### Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `Build()` | `Build(pattern string, params Params) string` | Build path |
| `BuildWithQuery()` | `BuildWithQuery(pattern string, params Params, query QueryParams) string` | Build with query |
| `SetStrictSlash()` | `SetStrictSlash(strict bool) *PathBuilder` | Enable/disable trailing slash |
| `SetURLEncoding()` | `SetURLEncoding(enabled bool) *PathBuilder` | Enable/disable URL encoding |

---

## Complete Example

```go
package main

import (
    "fmt"
    "net/url"
    
    "github.com/aydenstechdungeon/gospa/routing"
)

func main() {
    // Create param extractor
    extractor := routing.NewParamExtractor("/users/:id")
    
    // Match routes
    params, ok := extractor.Extract("/users/123")
    if ok {
        fmt.Printf("User ID: %s\n", params.Get("id"))
    }
    
    // Match with wildcard
    extractor = routing.NewParamExtractor("/api/:version/*path")
    params, ok = extractor.Extract("/api/v1/users/123/profile")
    if ok {
        fmt.Printf("Version: %s\n", params.Get("version"))
        fmt.Printf("Path: %s\n", params.Get("path"))
    }
    
    // Build paths
    builder := routing.NewPathBuilder("/users/:id/posts/:postId")
    builder.Param("id", "1")
    builder.Param("postId", "42")
    path := builder.Build()
    fmt.Println("Built path:", path)
    
    // With query params
    builder = routing.NewPathBuilder("/search")
    builder.Query("q", "gospa")
    builder.Query("page", "1")
    fullPath := builder.Build()
    fmt.Println("Full path:", fullPath)
    
    // Parse and use query params
    u, _ := url.Parse("https://example.com/search?q=gospa&page=2&filter=active&filter=pending")
    qp := routing.NewQueryParams(u.Query())
    
    fmt.Println("Query:", qp.Get("q"))
    fmt.Println("Page:", qp.GetInt("page"))
    fmt.Println("Filters:", qp.GetAll("filter"))
}
```

---

## Integration with Fiber

```go
func GetUser(c *fiber.Ctx) error {
    // Get path params
    id := c.Params("id")
    
    // Get query params
    qp := routing.NewQueryParams(c.Request().URI().QueryArgs())
    page := qp.GetDefault("page", "1")
    
    // Type-safe access
    pageNum := qp.GetInt("page")
    active := qp.GetBool("active")
    
    // ...
}

func Search(c *fiber.Ctx) error {
    // Build URL for redirect
    builder := routing.NewPathBuilder()
    redirectURL := builder.BuildWithQuery("/results", nil, routing.QueryParams{
        "q":    []string{c.Query("q")},
        "page": []string{"1"},
    })
    
    return c.Redirect(redirectURL)
}
```

---

## Best Practices

1. **Use type-safe accessors**: Always use `GetInt`, `GetBool`, etc. for type conversion
2. **Provide defaults**: Use `GetDefault` for optional parameters
3. **Validate parameters**: Check existence with `Has` before processing
4. **URL encode**: Let PathBuilder handle URL encoding
5. **Clone when modifying**: Clone params before modification if original needed
6. **Use wildcards sparingly**: Wildcards are greedy, use specific patterns when possible
