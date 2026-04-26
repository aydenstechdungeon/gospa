# SFC Templates

The `<template>` block is parsed into a GoSPA template AST and compiled into Templ output.

## Control Flow

### If Blocks

```svelte
{#if isAdmin}
  <button>Delete User</button>
{:else if isModerator}
  <button>Hide Post</button>
{:else}
  <span>No Actions</span>
{/if}
```

### Each Blocks

```svelte
<ul>
  {#each items as item}
    <li>{item.Name}</li>
  {/each}
</ul>
```

### Snippet Blocks

```svelte
{#snippet Badge(label)}
  <span class="badge">{label}</span>
{/snippet}
```

### Await Blocks

```svelte
{#await profilePromise}
  <p>Loading...</p>
{:then profile}
  <p>{profile.Name}</p>
{:catch err}
  <p>{err.Error()}</p>
{/await}
```

## Event Handlers

Bind user interactions using `on:<event>` and a named handler function:

```svelte
<script lang="go">
  func handleClick() {}
</script>

<button on:click={handleClick}>Click Me</button>
```

`on:<event>` is lowered to `data-gospa-on="event:handler"` in generated output.
Use function identifiers as handlers. Inline function literals are not part of the current delegated handler contract.

## Expressions

Use standard curly braces for expressions: `{ variable }`. Go functions and variables from the instance script are accessible.
