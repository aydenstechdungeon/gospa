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

Put global rules in app-level/global stylesheets (`styles/main.css`, route-level shared CSS, or Tailwind layers). SFC `<style>` blocks are component-scoped by default.

```css
/* styles/main.css */
body { background: #000; }
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
