# .gospa Single File Components (SFC)

The `.gospa` file format is a modern, single-file component system for the GoSPA framework. It allows you to define server-rendered HTML (Go/Templ), optional client-side reactivity (TypeScript), and component styles in a single file, following a syntax familiar to Svelte developers.

---

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

### Frontmatter Reference

Configure the compiler by adding a YAML block at the very top of your file.

| Parameter | Options | Default | Description |
|-----------|---------|---------|-------------|
| `type` | `island`, `page`, `layout`, `static`, `server` | `island` | Determines the hydration role and wrapper. |
| `hydrate` | `true`, `false` | `true` | Enables/disables TypeScript generation for `island`. |
| `server_only`| `true`, `false` | `false` | If true, skips TS generation even for islands. |
| `package` | string | (folder name) | Custom Go package name for the generated `.templ` file. |

---

## Reactivity: Runes

GoSPA SFC uses **Runes** to define reactive logic. These are available in the `<script lang="go">` block and are automatically translated to TypeScript for client-side hydration.

- **`$state(value)`**: Declares a reactive state variable.
- **`$derived(expression)`**: Computes a value from other states. It automatically updates when dependencies change.
- **`$effect(func())`**: Runs a side effect on the client whenever its dependencies change.

### Example: Derived State
```go
<script lang="go">
  var first = $state("John")
  var last = $state("Doe")
  var full = $derived(first + " " + last)
</script>

<template>
  <p>Full name: {full}</p>
</template>
```

---

## Template Syntax

The `<template>` block uses Go's logic via **Templ** integration.

### Control Flow
Directly use Go's `if` and `for` blocks:

```svelte
<template>
  {if isAdmin}
    <button>Delete User</button>
  {/if}

  <ul>
    {for _, item := range items}
      <li>{item.Name}</li>
    {/for}
  </ul>
</template>
```

### Event Handlers
Bind user interactions using `on:<event>`:

```svelte
<template>
  <button on:click={handleClick}>Click Me</button>
  <input on:input={func(e *gospa.Event) { value = e.Value }} />
</template>
```

---

## Style Scoping

Styles in the `<style>` block are automatically scoped. This means a selector like `h1` will only affect `<h1>` elements *inside* that specific component.

### Global Escaping
Use `:global()` to define styles that affect the whole page:

```css
<style>
  h1 { color: red; } /* Scoped */
  :global(body) { background: #000; } /* Global */
</style>
```

---

## Advanced Usage & Patterns

### 1. Mixed Scripts (Go + TS)
You can include both a Go script and a TypeScript script in the same file. The Go script handles SSR logic, while the TS script provides the hydration logic.

```svelte
<script lang="go">
  var initialVersion = "1.0.0"
</script>

<script lang="ts">
  console.log("Hydrated on client!");
</script>
```

### 2. File-Based Routing Integration
`.gospa` files are first-class citizens in the router:
- `page.gospa`: Main route entry.
- `layout.gospa`: Nested layout for the directory.
- `error.gospa`: Error boundary for the segment.

### 3. Component Types Deep Dive

- **`island`**: The workhorse of GoSPA. Provides interactive, hydrated UI berries.
- **`page`**: Individual route pages. Wrapped in a scoped div but typically not hydrated (unless marked as such).
- **`layout`**: Shared structure that wraps children using `{ children }`.
- **`static`**: Pure server-rendered output with no wrapper or JS.
- **`server`**: Logic-only components or fragments.

---

## Best Practices

1. **Keep Islands Small**: Use `static` or `page` types for non-interactive content to keep the client-side JS bundle small.
2. **Use Tailwind for Layout**: Combine scoped CSS for component-specific visuals and Tailwind for overall layout and spacing.
3. **Leverage Go Types**: Take advantage of Go's strong typing in your script blocks for robust data handling.
