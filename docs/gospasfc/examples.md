# SFC Examples

## Counter Island

```svelte
<script lang="go">
  var count = $state(0)
</script>

<template>
  <div class="counter">
    <p>Count: {count}</p>
    <button on:click={func() { count++ }}>+</button>
    <button on:click={func() { count-- }}>-</button>
  </div>
</template>

<style>
  .counter { border: 1px solid #ccc; padding: 1rem; }
</style>
```

## Form with Validation

```svelte
<script lang="go">
  var email = $state("")
  var error = $derived(len(email) < 5 && len(email) > 0)
</script>

<template>
  <form>
    <input type="email" data-model="email" />
    {#if error}
      <span class="error">Email too short</span>
    {/if}
  </form>
</template>
```

## Route `+page.gospa` with `Load` + Named Actions

```svelte
<script context="module" lang="go">
import (
  "github.com/aydenstechdungeon/gospa/routing"
  "github.com/aydenstechdungeon/gospa/routing/kit"
)

func Load(c routing.LoadContext) (map[string]interface{}, error) {
  return map[string]interface{}{"slug": c.Param("slug")}, nil
}

func ActionDefault(c routing.LoadContext) (interface{}, error) {
  return map[string]interface{}{"saved": true}, nil
}

func ActionArchive(c routing.LoadContext) (interface{}, error) {
  return nil, kit.Redirect(303, "/posts")
}
</script>

<template>
  <h1>Post {slug}</h1>
  <form method="post">
    <button type="submit">Save</button>
  </form>
  <form method="post" action="?_action=archive">
    <button type="submit">Archive</button>
  </form>
</template>
```
