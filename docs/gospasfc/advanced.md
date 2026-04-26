# SFC Advanced Usage

## Go -> TS Transpilation

The compiler transforms Go logic into efficient TypeScript.

### Type Mapping

- `int`, `float64` → `number`
- `string` → `string`
- `bool` → `boolean`
- other Go types currently fall back to `any`

### Expression Translation

Current compiler rewrites include:

- `fmt.Printf(...)` → `console.log(...)`
- `fmt.Sprint(...)` / `fmt.Sprintf(...)` → `String(...)` (best-effort fallback)
- `for _, item := range items` → `for (const item of items)`
- `:=` → `=`

The compiler does not currently implement full Go→TS semantic translation (for example, `len(...)` and `append(...)` are not special-cased).

## Security & Trust Boundary

> [!IMPORTANT]
> **.gospa files are source code, not user content.** Compile files only from trusted sources.

### SafeMode Compiler Option

For semi-trusted sources, enable `SafeMode` to perform AST validation and reject dangerous patterns (e.g., `os/exec`).

```go
compiler.Compile(compiler.CompileOptions{
    SafeMode: true,
}, input)
```

SafeMode validation currently covers:

- Go scripts (`<script lang="go">` and module scripts) via AST + call-pattern checks
- TS/JS scripts (`<script lang="ts">` / `lang="js"`) via syntax and pattern checks
- Template expressions and directive expressions

Examples of blocked imports/patterns include `os/exec`, `unsafe`, `syscall`, `reflect`, `C`, and process/filesystem execution patterns.

## Parser Constraints

- Exactly one `<template>`
- At most one Go script
- At most one module Go script (`<script context="module" lang="go">`)
- At most one TS/JS script
- At most one `<style>`
- Max size: 2 MB

## Redirect/Fail Control Flow

Use `kit.Redirect` and `kit.Fail` from `github.com/aydenstechdungeon/gospa/routing/kit` in `Load` and action exports for explicit status semantics:

- `kit.Redirect(status, "/path")` for controlled redirects.
- `kit.Fail(status, data)` for non-500 failures (validation/business errors).

These helpers work with progressive enhancement and `?__data=1` data responses.

## Migrating from `Actions` Map

Existing `+page.server.go` routes using:

- `var Actions = map[string]routing.ActionFunc{...}`
- `routing.ActionResponse`

continue to work. For SFC module scripts, prefer named exports:

- `ActionDefault`
- `Action<Name>`
