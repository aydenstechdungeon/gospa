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
</script>

<template>
  <button on:click={func() { count++ }}>
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
| `hydrate` | `true`, `false` | `true` | Enables/disables TypeScript generation for `island`. |
| `server_only`| `true`, `false` | `false` | If true, skips TS generation even for islands. |
| `package` | string | (folder name) | Custom Go package name for the generated `.templ` file. |

## Component Types

- **`island`**: Interactive, hydrated UI components.
- **`page`**: Individual route pages.
- **`layout`**: Shared structure that wraps children.
- **`static`**: Pure server-rendered output with no JS.
- **`server`**: Logic-only components or fragments.
