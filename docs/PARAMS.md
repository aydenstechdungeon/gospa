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
import "github.com/gospa/gospa/routing"

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
extractor := routing.NewParamExtractor()

// Add route pattern
extractor.AddPattern("/users/:id")
extractor.AddPattern("/posts/:postId/comments/:commentId")
extractor.AddPattern("/files/*filepath")  // Wildcard
```

### Extracting Parameters

```go
// Match and extract
match := extractor.Match("/users/123")
if match != nil {
    params := match.Params
    id := params.Get("id")  // "123"
}

// Match with wildcard
match := extractor.Match("/files/docs/readme.txt")
if match != nil {
    params := match.Params
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
builder := routing.NewPathBuilder()

// Build from pattern
path := builder.Build("/users/:id", routing.Params{
    "id": "123",
})
// Result: "/users/123"

// Build with multiple params
path := builder.Build("/users/:userId/posts/:postId", routing.Params{
    "userId":  "1",
    "postId":  "42",
})
// Result: "/users/1/posts/42"
```

### With Query Parameters

```go
path := builder.BuildWithQuery("/search", routing.Params{
    "q": "gospa",
}, routing.QueryParams{
    "page":  []string{"1"},
    "sort":  []string{"desc"},
})
// Result: "/search?q=gospa&page=1&sort=desc"
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
    
    "github.com/gospa/gospa/routing"
)

func main() {
    // Create param extractor
    extractor := routing.NewParamExtractor()
    extractor.AddPattern("/users/:id")
    extractor.AddPattern("/posts/:postId/comments/:commentId")
    extractor.AddPattern("/api/:version/*path")
    
    // Match routes
    match := extractor.Match("/users/123")
    if match != nil {
        fmt.Printf("Pattern: %s\n", match.Pattern)
        fmt.Printf("User ID: %s\n", match.Params.Get("id"))
    }
    
    // Match with wildcard
    match = extractor.Match("/api/v1/users/123/profile")
    if match != nil {
        fmt.Printf("Version: %s\n", match.Params.Get("version"))
        fmt.Printf("Path: %s\n", match.Params.Get("path"))
    }
    
    // Build paths
    builder := routing.NewPathBuilder()
    
    path := builder.Build("/users/:id/posts/:postId", routing.Params{
        "id":     "1",
        "postId": "42",
    })
    fmt.Println("Built path:", path)
    
    // With query params
    queryParams := routing.NewQueryParamsFromMap(map[string][]string{
        "page": {"1"},
        "sort": {"desc"},
    })
    
    fullPath := builder.BuildWithQuery("/search", routing.Params{
        "q": "gospa",
    }, queryParams)
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
