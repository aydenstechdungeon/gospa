# SFC Advanced Usage

## Go -> TS Transpilation

The compiler transforms Go logic into efficient TypeScript.

### Type Mapping

- `int`, `float64` → `number`
- `string` → `string`
- `bool` → `boolean`
- `map[string]any` → `Record<string, any>`

### Expression Translation

- `fmt.Printf(...)` → `console.log(...)`
- `len(arr)` → `arr.length`
- `append(arr, item)` → `[...arr, item]`

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

## Parser Constraints

- Exactly one `<template>`
- At most one Go script
- At most one TS/JS script
- At most one `<style>`
- Max size: 2 MB
