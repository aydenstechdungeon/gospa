# Reactive Primitives (Go)

Server-side reactive primitives for GoSPA, mirroring Svelte's rune system.

## Core Primitives

### Rune[T]

The base reactive primitive, similar to Svelte's `$state` rune.

```go
import "github.com/aydenstechdungeon/gospa/state"

count := state.NewRune(0)
value := count.Get()
count.Set(42)
count.Update(func(v int) int { return v + 1 })
```

### Derived[T]

Computed value that automatically updates when dependencies change.

```go
doubled := state.NewDerived(func() int {
    return count.Get() * 2
})
```

### Effect

Side effect that runs when dependencies change.

```go
effect := state.NewEffect(func() state.CleanupFunc {
    fmt.Println("Count:", count.Get())
    return func() { /* cleanup */ }
})
```

## Batch Updates

Execute multiple updates within a single notification cycle.

```go
state.Batch(func() {
    count.Set(10)
    name.Set("Updated")
})
```

## StateMap

A collection of named observables for component state management.

```go
sm := state.NewStateMap()
sm.Add("count", count)
sm.AddAny("name", "GoSPA")
```
