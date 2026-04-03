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
    <input type="email" bind:value={email} />
    {#if error}
      <span class="error">Email too short</span>
    {/if}
  </form>
</template>
```
