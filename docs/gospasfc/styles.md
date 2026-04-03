# SFC Styles

Styles in the `<style>` block are automatically scoped to the component.

## Scoped CSS

Selectors only affect elements within the component template.

```css
<style>
  h1 { color: red; } /* Only affects h1 in this component */
</style>
```

## Global Styles

Use `:global()` to define styles that affect the whole page:

```css
<style>
  :global(body) { background: #000; }
</style>
```

## Integration with Tailwind

You can use Tailwind classes directly in your templates. The compiler scans `.gospa` files for Tailwind classes.

```svelte
<template>
  <div class="p-4 bg-blue-500 text-white">
    Tailwind integrated!
  </div>
</template>
```
