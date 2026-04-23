# GoSPA SFC (Single File Components)

GoSPA supports `.gospa` Single File Components, but this system is currently **alpha**.

For production apps today, prefer `.templ` route/components and use `.gospa` only when you specifically want SFC ergonomics and are comfortable with alpha behavior.

## Current recommendation

- Stable/default path: `.templ` + Go + islands/runtime features.
- Experimental path: `.gospa` SFCs compiled into Templ and hydration code.

## `.gospa` format (alpha)

```svelte
<script lang="go">
  var count = $state(0)
  func increment() { count++ }
</script>

<template>
  <button on:click={increment}>Count is {count}</button>
</template>

<style>
  button { padding: 1rem; border-radius: 8px; }
</style>
```

The compiler turns this into generated templ output and client hydration code.

## Stable templ-based equivalent

If you want mature behavior today, use templ components and GoSPA runtime attributes directly:

```templ
package components

templ Counter(initial int) {
  <div data-gospa-component="Counter" data-gospa-state={ templ.JSONString(map[string]interface{}{"count": initial}) }>
    <button data-on:click="count++">Increment</button>
    <span data-bind="text:count">{ fmt.Sprint(initial) }</span>
  </div>
}
```

## Runtime bindings and events

GoSPA runtime attributes supported in templ include:

- `data-bind="text:key"`
- `data-bind="html:key"` (sanitized)
- `data-bind="class:name:key"`
- `data-model="key"`
- `data-on:<event>` handlers with modifiers like `.prevent`, `.stop`, `.debounce.*`, `.throttle.*`
