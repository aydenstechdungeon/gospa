# Route Parameters

GoSPA supports dynamic route parameters that allow you to capture segments of a URL and use them in your components and data loaders.

## Parameter Syntax

### 1. File-System Syntax
Used in your `routes/` directory structure:
- **Named Parameters**: `[id]`. Matches a single segment.
- **Optional Parameters**: `[[id]]`. Matches zero or one segment.
- **Catch-all Parameters**: `[...rest]`. Matches one or more segments.
- **Optional Catch-all**: `[[...rest]]`. Matches zero or more segments.

### 2. Programmatic / Router Syntax
Used when matching routes internally, in `BuildURL`, or when manually registering routes:
- **Named Parameters**: `:id`.
- **Optional Parameters**: `:id?`.
- **Catch-all Parameters**: `*rest`.
- **Catch-all (Tail)**: `*`.

## Accessing Parameters

### In Components

Parameters are passed directly to your component functions as part of the `props` map.

```svelte
// routes/blog/[slug]/page.templ
<script lang="go">
    param slug string
</script>

<template>
    <h1>Post: {slug}</h1>
</template>
```

### In Data Loaders

Use the `LoadContext` to access parameters in your `+page.server.go` or `+layout.server.go` files.

```go
func Load(c routing.LoadContext) (map[string]interface{}, error) {
    slug := c.Param("slug")
    post, err := db.GetPostBySlug(slug)
    return map[string]interface{}{
        "post": post,
    }, err
}
```

## Priority & Matching

GoSPA uses a scoring system to determine which route matches a given URL:

1. **Static segments** have the highest priority.
2. **Required dynamic segments** (`:param`) are next.
3. **Optional dynamic segments** (`:?param`) are lower priority.
4. **Catch-all segments** (`*param`) have the lowest priority.

If multiple routes match, the one with the highest priority (lowest score) is chosen.
