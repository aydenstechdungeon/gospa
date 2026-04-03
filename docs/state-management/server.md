# Server-side State Management

GoSPA provides a robust server-side state management system using reactive primitives and the `StateMap` container.

## Lifecycle of Server State

1.  **Request Initialization**: A new `StateMap` is typically created at the start of a request or within a component.
2.  **Initial Values**: Values are added using `Add` or `AddAny`.
3.  **Reactivity**: `Derived` and `Effect` primitives are used for server-side logic and validation.
4.  **Hydration**: The `StateMap` is serialized to JSON and sent to the client as part of the initial HTML.

## StateMap

The `StateMap` is the central container for component state. It manages a collection of `Observable` primitives.

```go
stateMap := state.NewStateMap()
stateMap.Add("count", countRune)
stateMap.AddAny("user", currentUser)
```

### Computed State

Add computed variables that depend on other keys in the map:

```go
stateMap.AddComputed("fullName", []string{"first", "last"}, func(vals map[string]any) any {
    return vals["first"].(string) + " " + vals["last"].(string)
})
```

## Batching

Server-side batching defers all state change notifications until the batch block completes, ensuring consistency and reducing redundant work.

```go
state.Batch(func() {
    runeA.Set(true)
    runeB.Set(false)
})
```
