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

- **`$state(value)`**: Defines a reactive piece of state.
- **`$derived(expression)`**: Defines a piece of state derived from other reactive values.
- **`$effect(func() { ... })`**: Defines a side-effect that runs whenever its dependencies change (client-only).

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

## Performance Benefits

- **Minimal Hydration**: Only interactive islands ship JavaScript to the client.
- **Tree-shakeable**: The `@gospa/runtime` only includes the reactive primitives actually used in your components.
- **Fast SSR**: Templates are compiled directly to Go code (via Templ), ensuring low latency and high concurrency.
