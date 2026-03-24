# .gospa Single File Components (SFC)

The `.gospa` file format is a modern, single-file component system for the GoSPA framework. It allows you to define server-rendered HTML (Go/Templ), client-side reactivity (TypeScript), and component styles in a single file, following a syntax familiar to Svelte or React developer.

---

## Structure

A `.gospa` file is divided into three main sections: `<script>`, `<template>`, and `<style>`.

```svelte
<script lang="go">
  // Go logic for server-side state and hydration
</script>

<template>
  <!-- HTML template with reactive bindings and control flow -->
</template>

<style>
  /* Scoped component styles */
</style>
```

### `<script>` Block

The script block primarily contains Go code that defines the component's state and server-side behavior. It uses reactive primitives that the compiler translates to TypeScript for the client side.

- **`$state(value)`**: Defines reactive state (local or shared).
- **`$derived(expression)`**: Defines state derived from other reactive values.
- **`$effect(func() { ... })`**: Defines a side-effect that runs whenever its dependencies change (client-only).

These primitives work seamlessly across multiple islands without requiring WebSocket connections for local UI state.

### Script Language Architecture (Go + TS/JS)

GoSPA supports two script roles in one `.gospa` file:

1. **`<script lang="go">` (optional, max 1)**  
   - Used for SSR/Templ-side logic.
   - Also used as the source for automatic TS generation **when no explicit TS/JS script is provided**.

2. **`<script lang="ts">` / `<script lang="js">` (optional, max 1 total)**  
   - Used directly as the client hydration script.
   - When this block exists, the compiler does **not** run Go-to-TS DSL rewriting on that block.

So yes—you can have both a Go script and a TS/JS script in the same component.  
What is disallowed is **duplicates of the same role** (e.g., two Go scripts), to keep parsing deterministic and avoid shadowed logic.

### `<template>` Block

The template block uses a syntax similar to Svelte and Templ.

- **Interpolation**: `{ expression }`
- **Control Flow**: 
    - `{#if condition} ... {/if}`
    - `{#each items as item} ... {/each}`
- **Events**: `on:event={ handler }` (e.g., `on:click={ handleClick }`)

### `<style>` Block

Styles defined in the `<style>` block are **scoped** to the component by default. The compiler generates a unique CSS class for each component, ensuring that styles do not leak to other parts of the application.

---

## Example: Counter

```svelte
<script lang="go">
  var count = $state(0)
  var doubled = $derived(count * 2)
  
  $effect(func() {
    fmt.Printf("Count changed to: %d\n", count)
  })

  func increment() {
    count++
  }
</script>

<template>
  <div class="counter-card">
    <h2>Counter</h2>
    <p>Count: {count}</p>
    <p>Doubled: {doubled}</p>
    <button on:click={increment} class="btn-primary">Increment</button>
  </div>
</template>

<style>
  .counter-card {
    padding: 1rem;
    border: 1px solid var(--color-gray-200);
    border-radius: 0.5rem;
  }
  .btn-primary {
    background: var(--color-blue-500);
    color: white;
    padding: 0.5rem 1rem;
    border-radius: 0.25rem;
  }
</style>
```

---

## How it Works

The `.gospa` compiler performs several transformations:

1.  **Server-side (Go/Templ)**:
    - Generates a `.templ` file.
    - `$state(val)` is replaced with `val` for initial rendering.
    - Scoped CSS classes are added to the HTML elements.
2.  **Client-side (TypeScript)**:
    - Generates a TypeScript island module.
    - `$state`, `$derived`, and `$effect` are translated to the reactive system in `@gospa/runtime`.
    - Event handlers automatically attach to the server-rendered elements.
3.  **Scoped CSS**:
    - The CSS is extracted and scoped using a unique component hash.

---

## File-Based Routing

`.gospa` files are fully supported by GoSPA's file-based routing system. You can place them inside the `routes/` directory using the same conventions as `.templ` files (e.g., `page.gospa`, `layout.gospa`, `[id]/page.gospa`).

---

## Cross-Island State

Islands in GoSPA can share state without server-side roundtrips by using global stores. This is ideal for UI-only state like global themes, sidebar visibility, or cart totals.

### Example: Shared Theme Store

```go
// islands/ThemeToggle.gospa
<script lang="go">
  var theme = createStore("theme", "light")
  
  func toggle() {
    if theme.Get() == "light" {
      theme.Set("dark")
    } else {
      theme.Set("light")
    }
  }
</script>
<template>
  <button on:click={toggle}>Current: {theme}</button>
</template>
```

```go
// islands/Header.gospa
<script lang="go">
  var theme = getStore("theme")
  
  $effect(func() {
     fmt.Printf("Header reacting to theme: %s\n", theme)
  })
</script>
<template>
  <header class={theme}>GoSPA Header</header>
</template>
```

---

---

## Security & Best Practices

GoSPA is designed with security in mind, but developers should follow these practices when building SFCs:

1.  **Input Sanitization**: Always sanitize user-provided data before displaying it in templates. GoSPA's interpolation `{ expression }` automatically escapes HTML entities, but be cautious when using `@templ.Raw()`.
2.  **Style Safety**: The SFC compiler automatically scopes and injects styles. While it escapes typical breakouts, avoid using complex dynamic string interpolation directly inside `<style>` blocks if the input comes from an untrusted source.
3.  **Local State**: Use `$state` for UI-only state. For sensitive data, prefer server-side state managed via WebSocket or Remote Actions to keep the logic and data off the client where possible.
4.  **CSP**: We recommend a strong Content Security Policy (CSP). GoSPA works best with policies that allow `style-src 'self' 'unsafe-inline'` for scoped CSS injection, but you can further tighten this by using nonces if using the SSR-only mode.

- **Minimal Hydration**: Only interactive islands ship JavaScript to the client.
- **Tree-shakeable**: The `@gospa/runtime` only includes the reactive primitives actually used in your components.
- **Fast SSR**: Templates are compiled directly to Go code (via Templ), ensuring low latency and high concurrency.
