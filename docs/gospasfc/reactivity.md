# SFC Reactivity: Runes

GoSPA SFC uses **Runes** to define reactive logic. These are available in the Go script block and translated to TypeScript.

## $state()

Declares a reactive state variable.

```go
var count = $state(0)
```

## $derived()

Computes a value from other states. It automatically updates when dependencies change.

```go
var first = $state("John")
var last = $state("Doe")
var full = $derived(first + " " + last)
```

## $effect()

Runs a side effect on the client whenever its dependencies change.

```go
$effect(func() {
    fmt.Printf("Count is now %d\n", count)
})
```

Current Go-script form is strict: `$effect(func() { ... })`.

## $props()

Access component properties passed from the parent.

```go
var { title, initialCount } = $props()
```

## Notes

- `$state`, `$derived`, and `$effect` are transformed during compilation for server output and hydration code generation.
- `$props()` supports destructured prop declarations from component inputs.
