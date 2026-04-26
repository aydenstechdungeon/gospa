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
  {#each items as item, index}
    <li>{index + 1}: {item.Name}</li>
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

Bind user interactions using `on:<event>`:

```svelte
<button on:click={handleClick}>Click Me</button>
<input on:input={func(e *gospa.Event) { value = e.Value }} />
```

`on:<event>` is lowered to `data-gospa-on="event:handler"` in generated output.

## Expressions

Use standard curly braces for expressions: `{ variable }`. Go functions and variables from the instance script are accessible.
