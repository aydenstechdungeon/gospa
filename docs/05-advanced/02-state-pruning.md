# StatePruner — API Reference

`github.com/aydenstechdungeon/gospa/state`

## Overview

`StatePruner` automatically removes stale `Rune` values from a `StateMap` when they haven't changed for a configurable duration. This prevents unbounded state map growth in long-lived applications where keys are added dynamically.

---

## Types

```go
// StatePruner periodically removes stale entries from a StateMap.
type StatePruner struct {
    // ...
}

// PrunerConfig configures the StatePruner.
type PrunerConfig struct {
    // Interval is how often the pruner runs (default: 5m).
    Interval time.Duration
    // MaxAge is how long a value must be unchanged before it is pruned (default: 30m).
    MaxAge time.Duration
    // OnPrune is an optional callback fired for each pruned key.
    OnPrune func(key string)
}
```

---

## Constructor

### `NewStatePruner(sm *StateMap, config PrunerConfig) *StatePruner`

Creates a new `StatePruner` targeting the given `StateMap`. Does not start pruning until `Start()` is called.

**Example:**

```go
sm := state.NewStateMap()

pruner := state.NewStatePruner(sm, state.PrunerConfig{
    Interval: 10 * time.Minute,
    MaxAge:   1 * time.Hour,
    OnPrune: func(key string) {
        log.Printf("Pruned stale key: %s", key)
    },
})
pruner.Start()
defer pruner.Stop()
```

---

## Methods

### `(*StatePruner) Start()`

Begins the background pruning loop. Safe to call multiple times (no-op if already running).

### `(*StatePruner) Stop()`

Stops the background pruning loop and blocks until it exits. Safe to call multiple times.

### `(*StatePruner) Prune() int`

Runs a single pruning pass immediately (outside the normal interval). Returns the number of keys pruned.

---

## Behavior Notes

- The pruner tracks the last modification time of each Rune via a subscription. It does **not** poll Rune values directly.
- Keys added after `Start()` are automatically tracked.
- Pruned keys are deleted from the `StateMap`. The corresponding Rune is also disposed if it implements `Disposable`.
- The `OnPrune` callback is invoked synchronously during the prune pass. Keep it fast; heavy work should be dispatched asynchronously.
- The pruner does **not** protect against re-adding pruned keys — if code adds them back, they will be tracked again.

---

## Interaction with `WebSocket` state sync

If you use a `StateMap` for per-session WebSocket state, attach a pruner to avoid unbounded growth of per-user keys as sessions expire:

```go
// Per-session state example
sessionState := state.NewStateMap()

pruner := state.NewStatePruner(sessionState, state.PrunerConfig{
    Interval: 5 * time.Minute,
    MaxAge:   fiber.SessionTTL, // match session TTL from websocket package
})
pruner.Start()
defer pruner.Stop()
```

---

## See Also

- [`StateMap`](API.md#statemap)
- [`Rune`](API.md#runet)
- [`SessionTTL`](API.md#websocket-hub) in `fiber` package
