# Routing API Reference

Low-level API for parameter extraction, validation, and URL construction.

## Params

Type-safe parameter map with conversion methods.

```go
id := params.GetInt("id")
name := params.GetString("name")
isAdmin := params.GetBool("admin")
```

## QueryParams

Handle URL query string parameters.

```go
qp := routing.NewQueryParams(url.Query())
page := qp.GetInt("page")
search := qp.Get("q")
```

## PathBuilder

Construct safe URLs from patterns and parameters.

```go
builder := routing.NewPathBuilder("/users/:id/posts/:postId")
builder.Param("id", "1")
builder.Param("postId", "42")
path := builder.Build() // "/users/1/posts/42"
```

## Route Matching

### ParamExtractor

```go
extractor := routing.NewParamExtractor("/users/:id")
params, ok := extractor.Extract("/users/123")
```

## Generated Artifacts

### Go Route Registry (`routes/generated_routes.go`)

Automatically registers all routes from your file system.

### TS Routes (`generated/routes.ts`)

Provides `buildPath`, `matchRoute`, and `findRoute` on the client.
