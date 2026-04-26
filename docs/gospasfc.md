# GoSPA SFC (Single File Components)

GoSPA supports `.gospa` Single File Components as a first-class route/component authoring format.

## SFC references

- [SFC API Reference](gospasfc/api-reference.md)
- [SFC System Reference](gospasfc/system-reference.md)
- [SvelteKit Migration Guide](migration/sveltekit-to-gospa.md)

## Current recommendation

- Default path: `.gospa` SFCs compiled into Templ and hydration code.
- Compatibility path: `.templ` + Go + islands/runtime features for teams migrating incrementally.

## `.gospa` format

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

## What the parser/compiler supports today

- Frontmatter (optional, YAML-like `key: value` pairs)
- One `<template>` block
- Zero or one `<script lang="go">` block
- Zero or one `<script lang="ts">` or `<script lang="js">` block (`js` normalizes to `ts`)
- Zero or one `<script context="module" lang="go">` block (route exports)
- Zero or one `<style>` block
- Maximum SFC input size: 2 MB

If no explicit `<template>` exists, top-level markup is treated as implicit template content.

## Frontmatter keys used by compiler

```yaml
type: island | page | layout | static | server
hydrate: true | false | immediate | visible | idle | interaction
hydrate_mode: immediate | visible | idle | interaction
server_only: true | false
package: <go package name>
```

Notes:

- `hydrate: <mode>` enables hydration and sets mode in one key.
- `hydrate_mode` overrides mode if both are present.
- For `island`, hydration defaults to enabled unless `server_only: true` or `hydrate: false`.
- Default hydrate mode is `immediate` for `island` and `page`.

## Route Server Module (SvelteKit-style)

For route SFC files (`+page.gospa`, `+layout.gospa`), GoSPA supports a module script:

```svelte
<script context="module" lang="go">
  func Load(c routing.LoadContext) (map[string]interface{}, error) {
    return map[string]interface{}{"title": "Dashboard"}, nil
  }

  func ActionDefault(c routing.LoadContext) (interface{}, error) {
    return nil, nil
  }

  func ActionSave(c routing.LoadContext) (interface{}, error) {
    return map[string]interface{}{"saved": true}, nil
  }
</script>
```

Supported exports:

- `Load(c routing.LoadContext) (map[string]interface{}, error)` (also accepts `map[string]any`)
- `ActionDefault(c routing.LoadContext) (interface{}, error)` (also accepts `any`)
- `Action<Name>(c routing.LoadContext) (interface{}, error)` (also accepts `any`)

### Conflict Rule

If a route defines module exports in `+page.gospa` or `+layout.gospa` **and** also has `+page.server.go` / `+layout.server.go`, generation fails with a conflict error. Keep one source of truth per route.

### Redirect/Fail Helpers

Use helper control-flow errors in load/actions:

```go
import "github.com/aydenstechdungeon/gospa/routing/kit"

return nil, kit.Redirect(303, "/login")
return nil, kit.Fail(422, map[string]interface{}{"fieldErrors": map[string]string{"email": "invalid"}})
```

`routing.ActionResponse` still works for compatibility during migration.

## Stable templ-based equivalent

If you want mature behavior today, use templ components and GoSPA runtime attributes directly:

```templ
package components

templ Counter(initial int) {
  <div data-gospa-component="Counter" data-gospa-state={ templ.JSONString(map[string]interface{}{"count": initial}) }>
    <button data-on="click:increment">Increment</button>
    <span data-bind="text:count">{ fmt.Sprint(initial) }</span>
  </div>
}
```

## Runtime bindings and events

GoSPA runtime attributes supported in templ include:

- `data-bind="text:key"`
- `data-bind="html:key"` (sanitized)
- `data-model="key"`
- `data-on="event:action"` for navigation/runtime action dispatch

In SFC templates, `on:<event>` is lowered to runtime delegation attributes during compile:

- `on:click={increment}` -> `data-gospa-on="click:increment"`
