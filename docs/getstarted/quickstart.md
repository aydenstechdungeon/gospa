# Quick Start

Get your first reactive GoSPA application running in under five minutes.

## Prerequisites

- **Go 1.26.0+**
- **Bun** (for client-side builds)
- **Templ** CLI (`go install github.com/a-h/templ/cmd/templ@latest`)

## 1. Install GoSPA CLI

The CLI is the recommended way to manage GoSPA projects.

```bash
go install github.com/aydenstechdungeon/gospa/cmd/gospa@latest
```
or
```bash
go run github.com/aydenstechdungeon/gospa/cmd/gospa@latest
```

## 2. Scaffold a Project

Create a new project using the `create` command. This sets up the directory structure and recommended configuration.

```bash
gospa create my-app
cd my-app
go mod tidy
```
or
```bash
go run github.com/aydenstechdungeon/gospa/cmd/gospa@latest create my-app
cd my-app
go mod tidy
```

## 3. Launch Development Server

GoSPA's dev server handles hot reloading, route generation, and runtime builds automatically.

```bash
gospa dev
```

Your app is now running at `http://localhost:3000`.

## 4. Create your first SFC

Single File Components (`.gospa`) co-locate your logic, template, and styles. Create `islands/Counter.gospa`:

```svelte
<script lang="go">
    var count = $state(0)
    func increment() { count++ }
</script>

<template>
    <div class="p-8 border rounded-2xl glass">
        <h2 class="text-2xl font-bold">Counter: {count}</h2>
        <button on:click={increment} class="mt-4 px-6 py-2 bg-[var(--accent-primary)] text-white rounded-full font-bold transition-all hover:scale-105">
            Increment
        </button>
    </div>
</template>

<style>
    div { transition: all 0.3s ease; }
    button { box-shadow: 0 4px 12px var(--accent-primary-alpha); }
</style>
```

### What's happening here?
1. **`<script lang="go">`**: Defines component logic and reactive state using the `$state` rune.
2. **`$state(0)`**: Creates a reactive variable that synchronized between server and client.
3. **`on:click={increment}`**: Binds the click event to your Go function.
4. **Scoping**: Styles in the `<style>` block are automatically scoped to this component.

## 5. Use the Component

Open `routes/page.templ` and import/use your new island:

```go
package routes

import "myapp/generated/islands"

templ Page() {
    <div class="p-12">
        <h1 class="text-4xl font-extrabold mb-8">Welcome to GoSPA</h1>
        @islands.Counter()
    </div>
}
```

## Next Steps

- **[Core Concepts: State](../state-management/server.md)** — Deep dive into Runes, Derived values, and Effects.
- **[Routing](../routing.md)** — Dynamic parameters and layout nesting.
- **[CLI Reference](../cli.md)** — Master the `gospa` command.
