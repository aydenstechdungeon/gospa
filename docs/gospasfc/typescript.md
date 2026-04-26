# SFC TypeScript and JavaScript

Add custom client-side logic using `<script lang="ts">` or `<script lang="js">`.

`lang="js"` is accepted and normalized to TypeScript handling internally.

## Mixed Scripts

You can have both a Go script (for SSR/Runes) and a TS script for manual DOM logic.

```svelte
<script lang="go">
  var initialVersion = "1.0.0"
</script>

<script lang="ts">
  const btn = document.querySelector("button");
  btn?.addEventListener("click", () => {
    console.log("clicked");
  });
</script>
```

Note: `on:<event>` template handler names are resolved through the generated island handler registry. Prefer named handlers declared in the Go instance script for delegated template events.

## Generated Runtime Bridge

Generated SFC code injects runtime helpers when available:

- `__gospa_state(...)` -> runtime `$state(...)`
- `__gospa_derived(...)` -> runtime `$derived(...)`
- `__gospa_effect(...)` -> runtime `$effect(...)`

## Type Generation

The SFC compiler generates TypeScript interfaces corresponding to your Go state and props.
