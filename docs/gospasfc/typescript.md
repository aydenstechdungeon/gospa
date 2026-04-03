# SFC TypeScript and JavaScript

Add custom client-side logic using `<script lang="ts">` or `<script lang="js">`.

## Mixed Scripts

You can have both a Go script (for SSR/Runes) and a TS script for manual DOM logic.

```svelte
<script lang="go">
  var initialVersion = "1.0.0"
</script>

<script lang="ts">
  import { onMount } from '@gospa/client';
  
  onMount(() => {
    console.log("Component hydrated!");
  });
</script>
```

## Hydration Hooks

The generated TypeScript code includes lifecycle hooks for hydration.

- **`onMount`**: Runs after the component is hydrated in the DOM.
- **`onDestroy`**: Runs before the component is removed.

## Type Generation

The SFC compiler generates TypeScript interfaces corresponding to your Go state and props.
