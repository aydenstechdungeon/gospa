# .gospa SFC Getting Started

The `.gospa` file format is a modern, single-file component system for the GoSPA framework. It allows you to define server-rendered HTML (Go/Templ), optional client-side reactivity (TypeScript), and component styles in a single file, following a syntax familiar to Svelte developers.

## Structure

A `.gospa` file is divided into four main sections: **Frontmatter** (optional), `<script>`, `<template>`, and `<style>`.

```svelte
---
type: island
hydrate: true
---

<script lang="go">
  // Go logic for server-side state and hydration
  var count = $state(0)
  func increment() { count++ }
</script>

<template>
  <button on:click={increment}>
    Count is {count}
  </button>
</template>

<style>
  button { padding: 1rem; border-radius: 8px; }
</style>
```

## Frontmatter Reference

Configure the compiler by adding a YAML block at the very top of your file.

| Parameter | Options | Default | Description |
|-----------|---------|---------|-------------|
| `type` | `island`, `page`, `layout`, `static`, `server` | `island` | Determines the hydration role and wrapper. |
| `hydrate` | `true`, `false`, `immediate`, `visible`, `idle`, `interaction` | `true` (island), implied by type | Enables/disables hydration, or sets hydration mode directly. |
| `hydrate_mode` | `immediate`, `visible`, `idle`, `interaction` | `immediate` for `island`/`page` | Explicit hydration mode. |
| `server_only`| `true`, `false` | `false` | If true, skips TS generation even for islands. |
| `package` | string | (folder name) | Custom Go package name for the generated `.templ` file. |

Compiler behavior:

- If `hydrate` is neither `true` nor `false`, it is treated as a hydration mode.
- `hydrate_mode` wins if both keys are present.
- `js` script blocks are normalized to `ts`.

## Component Types

- **`island`**: Interactive, hydrated UI components.
- **`page`**: Individual route pages.
- **`layout`**: Shared structure that wraps children.
- **`static`**: Pure server-rendered output with no JS.
- **`server`**: Logic-only components or fragments.

## Route Server Module Syntax

Route SFC files can export server behavior with a module script:

```svelte
<script context="module" lang="go">
  func Load(c routing.LoadContext) (map[string]interface{}, error) {
    return map[string]interface{}{
      "userID": c.Param("id"),
    }, nil
  }

  func ActionDefault(c routing.LoadContext) (interface{}, error) {
    return map[string]interface{}{"ok": true}, nil
  }

  func ActionDelete(c routing.LoadContext) (interface{}, error) {
    return nil, kit.Redirect(303, "/users")
  }
</script>
```

Rules:

- Only one `<script context="module" lang="go">` block is allowed.
- Module script must use `lang="go"`.
- Export signatures must match:
  - `Load(c routing.LoadContext) (map[string]interface{}, error)` (or `map[string]any`)
  - `ActionDefault(c routing.LoadContext) (interface{}, error)` (or `any`)
  - `Action<Name>(c routing.LoadContext) (interface{}, error)` (or `any`)
  - action suffix must start with uppercase (`ActionSave`, not `Actionsave`)

## End-to-End `+page.gospa` Example

```svelte
---
type: page
package: users
---

<script context="module" lang="go">
import (
  "github.com/aydenstechdungeon/gospa/routing"
  "github.com/aydenstechdungeon/gospa/routing/kit"
)

func Load(c routing.LoadContext) (map[string]interface{}, error) {
  id := c.Param("id")
  if id == "" {
    return nil, kit.Fail(400, map[string]interface{}{"message": "missing id"})
  }
  return map[string]interface{}{"id": id}, nil
}

func ActionDefault(c routing.LoadContext) (interface{}, error) {
  return map[string]interface{}{"saved": true}, nil
}

func ActionDelete(c routing.LoadContext) (interface{}, error) {
  return nil, kit.Redirect(303, "/users")
}
</script>

<script lang="go">
  var notice = $state("Ready")
</script>

<template>
  <h1>User {id}</h1>
  <p>{notice}</p>
  <form method="post">
    <button type="submit">Save</button>
  </form>
  <form method="post" action="?_action=delete">
    <button type="submit">Delete</button>
  </form>
</template>
```

## Conflict Behavior with `+*.server.go`

If `+page.gospa` (or `+layout.gospa`) exports module server functions and the same route also has `+page.server.go` (or `+layout.server.go`), route generation fails. Remove one side to resolve.
