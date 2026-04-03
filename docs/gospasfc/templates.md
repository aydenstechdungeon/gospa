# SFC Templates

The `<template>` block uses Go's logic via **Templ** integration, with Svelte-aligned control flow syntax.

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
  {#each items as item, index}
    <li>{index + 1}: {item.Name}</li>
  {/each}
</ul>
```

### Await Blocks

```svelte
{#await promise}
  <p>Loading...</p>
{:then result}
  <p>Result: {result}</p>
{:catch error}
  <p>Error: {error.message}</p>
{/await}
```

## Event Handlers

Bind user interactions using `on:<event>`:

```svelte
<button on:click={handleClick}>Click Me</button>
<input on:input={func(e *gospa.Event) { value = e.Value }} />
```

## Expressions

Use standard curly braces for expressions: `{ variable }`. Go functions and variables are directly accessible.
