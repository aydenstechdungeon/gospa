# .gospa Single File Components (SFC - ALPHA)

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
Directly use Svelte-aligned logic blocks:

```svelte
<template>
  {#if isAdmin}
    <button>Delete User</button>
  {:else if isModerator}
    <button>Hide Post</button>
  {:else}
    <span>No Actions</span>
  {/if}

  <ul>
    {#each items as item, index}
      <li>{index + 1}: {item.Name}</li>
    {/each}
  </ul>

  {#await promise}
    <p>Loading...</p>
  {:then result}
    <p>Result: {result}</p>
  {:catch error}
    <p>Error: {error.message}</p>
  {/await}
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
## Technical Deep Dive: Go -> TS Transpilation

GoSPA components use a custom compiler to transform Go logic into efficient TypeScript for client-side hydration. Here are the primary rules:

### 1. Prop Destructuring
Inside `<script lang="go">`, use the `$props()` rune to access component properties. 

```go
// Source (.gospa)
var { title, count } = $props()
```

The compiler transforms this into TypeScript destructuring while maintaining type safety:

```typescript
// Generated (.ts)
const { title, count } = $props();
```

### 2. Type Stripping
Go-specific types and syntax that don't exist in TypeScript are stripped or mapped:
- `int`, `float64`, `int32` → `number`
- `string` → `string`
- `bool` → `boolean`
- `map[string]any` → `Record<string, any>`

### 3. Expression Translation
The compiler performs basic regex-based translation for common Go patterns to their JavaScript equivalents:
- `fmt.Printf(...)` → `console.log(...)`
- `len(arr)` → `arr.length`
- `append(arr, item)` → `[...arr, item]`

### 4. Rune Synchronization
Any variable marked with `$state()` is automatically synchronized with the server-side state map. The compiler generates the necessary WebSocket bridge code to ensure that updates on the server are reflected in the client's reactive runes.

---

## Security & Trust Boundary

> **`.gospa` files are source code, not user content.**

The `.gospa` compiler executes the contents of `<script lang="go">` and `<script lang="ts">` blocks directly into generated Go/Templ output. This means:

- **Trusted source only.** Compile `.gospa` files only from sources you control. Never compile tenant-supplied or user-provided SFC content in a shared build pipeline, CI, or runtime context.
- **Build pipeline isolation.** If you must process untrusted templates, run the compiler in an isolated sandboxed worker (e.g., a container with no network/filesystem access).
- **SafeMode compiler option.** For semi-trusted sources (e.g., CMS-generated SFCs), enable `SafeMode` on `CompileOptions`:

```go
compiler := compiler.NewCompiler()
templOut, tsOut, err := compiler.Compile(compiler.CompileOptions{
    Name:     "UserWidget",
    SafeMode: true,   // enables AST validation and dangerous-pattern detection
}, userSFCInput)
```

In `SafeMode`, the compiler:
1. Parses the Go script content with `go/parser` to ensure syntactic validity.
2. Rejects scripts containing imports of disallowed packages (`os/exec`, `unsafe`, `syscall`, `os`, `plugin`, `reflect`).
3. Rejects scripts matching known dangerous patterns (`exec.Command`, `os.WriteFile`, `syscall.Exec`, etc.).

### SFC Parser Constraints

The parser enforces these structural rules:

| Constraint | Behavior |
|---|---|
| Exactly one `<template>` block | Rejected with error if duplicate |
| At most one `<script lang="go">` | Rejected with error if duplicate |
| At most one `<script lang="ts\|js">` | Rejected with error if duplicate |
| At most one `<style>` block | Rejected with error if duplicate |
| Maximum input size | 2 MB; rejected if exceeded |
| Backtick strings with HTML tags inside | Skipped during parsing to avoid false-positive block extraction |
