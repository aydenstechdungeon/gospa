# Go State Primitives Reference

Server-side reactive primitives for GoSPA, mirroring Svelte's rune system for server-side state management.

## Overview

The `state` package provides Svelte rune-like reactive primitives for Go. These primitives enable reactive state management on the server side, with automatic change notification and dependency tracking.

```go
import "github.com/aydenstechdungeon/gospa/state"
```

## Core Types

### Rune[T]

The base reactive primitive, similar to Svelte's `$state` rune. Holds a value and notifies subscribers on changes.

#### Constructor

```go
func NewRune[T any](initial T) *Rune[T]
```

Creates a new Rune with the given initial value.

**Example:**
```go
count := state.NewRune(0)
name := state.NewRune("hello")
items := state.NewRune([]string{})
```

#### Methods

##### Get

```go
func (r *Rune[T]) Get() T
```

Returns the current value. Thread-safe for concurrent access.

**Example:**
```go
value := count.Get()
fmt.Println("Current count:", value)
```

##### Set

```go
func (r *Rune[T]) Set(value T)
```

Updates the value and notifies all subscribers. Skips notification if value unchanged. Defers notification in batch mode.

**Example:**
```go
count.Set(42)
name.Set("updated")
```

##### Update

```go
func (r *Rune[T]) Update(fn func(T) T)
```

Applies a function to the current value and sets the result. Useful for updates that depend on current value.

**Example:**
```go
count.Update(func(v int) int {
    return v + 1
})

items.Update(func(v []string) []string {
    return append(v, "new item")
})
```

##### Subscribe

```go
func (r *Rune[T]) Subscribe(fn Subscriber[T]) Unsubscribe
```

Registers a callback invoked on value changes. Returns unsubscribe function.

**Example:**
```go
unsub := count.Subscribe(func(v int) {
    fmt.Println("Count changed to:", v)
})
defer unsub()
```

##### ID

```go
func (r *Rune[T]) ID() string
```

Returns unique identifier for client-side synchronization.

##### MarshalJSON

```go
func (r *Rune[T]) MarshalJSON() ([]byte, error)
```

Implements `json.Marshaler` for serialization to client.

---

### Derived[T]

Computed value that automatically updates when dependencies change. Similar to Svelte's `$derived` rune.

#### Constructor

```go
func NewDerived[T any](compute func() T) *Derived[T]
```

Creates a derived value from a compute function. Called immediately for initial value.

**Example:**
```go
count := state.NewRune(5)
doubled := state.NewDerived(func() int {
    return count.Get() * 2
})
```

#### Methods

##### Get

```go
func (d *Derived[T]) Get() T
```

Returns current computed value. Recomputes if dependencies changed.

##### Subscribe

```go
func (d *Derived[T]) Subscribe(fn Subscriber[T]) Unsubscribe
```

Registers callback for derived value changes.

##### DependOn

```go
func (d *Derived[T]) DependOn(o Observable)
```

Adds an observable as a dependency. When it changes, derived value marked dirty.

**Example:**
```go
count := state.NewRune(5)
doubled := state.NewDerived(func() int {
    return count.Get() * 2
})
doubled.DependOn(count) // Auto-recompute when count changes
```

##### Dispose

```go
func (d *Derived[T]) Dispose()
```

Cleans up all subscriptions to dependencies. Call when no longer needed.

##### ID

```go
func (d *Derived[T]) ID() string
```

Returns unique identifier.

---

### Effect

Reactive side effect that runs when dependencies change. Similar to Svelte's `$effect` rune.

#### Constructor

```go
func NewEffect(fn EffectFn) *Effect
```

Where `EffectFn` is:
```go
type EffectFn func() CleanupFunc
```

Creates effect that runs immediately. Return cleanup function for resource cleanup.

**Example:**
```go
count := state.NewRune(0)
effect := state.NewEffect(func() state.CleanupFunc {
    fmt.Println("Count is:", count.Get())
    return func() {
        fmt.Println("Cleaning up")
    }
})
defer effect.Dispose()
```

#### Methods

##### DependOn

```go
func (e *Effect) DependOn(o Observable)
```

Adds observable as dependency. Effect re-runs when it changes.

**Example:**
```go
effect := state.NewEffect(func() state.CleanupFunc {
    fmt.Println("Count:", count.Get())
    return nil
})
effect.DependOn(count)
```

##### IsActive

```go
func (e *Effect) IsActive() bool
```

Returns whether effect is currently active.

##### Pause

```go
func (e *Effect) Pause()
```

Temporarily stops effect from running.

##### Resume

```go
func (e *Effect) Resume()
```

Reactivates a paused effect. Re-runs if was inactive.

##### Dispose

```go
func (e *Effect) Dispose()
```

Permanently stops effect and cleans up resources.

---

## Convenience Functions

### DerivedFrom

```go
func DerivedFrom[T any](compute func() T, observables ...Observable) *Derived[T]
```

Creates derived value with automatic dependency setup.

**Example:**
```go
count := state.NewRune(5)
doubled := state.DerivedFrom(func() int {
    return count.Get() * 2
}, count)
```

### Derived2

```go
func Derived2[A, B, T any](a *Rune[A], b *Rune[B], combine func(A, B) T) *Derived[T]
```

Creates derived from two runes.

**Example:**
```go
firstName := state.NewRune("John")
lastName := state.NewRune("Doe")
fullName := state.Derived2(firstName, lastName, func(a, b string) string {
    return a + " " + b
})
```

### Derived3

```go
func Derived3[A, B, C, T any](a *Rune[A], b *Rune[B], c *Rune[C], combine func(A, B, C) T) *Derived[T]
```

Creates derived from three runes.

### EffectOn

```go
func EffectOn(fn EffectFn, observables ...Observable) *Effect
```

Creates effect with automatic dependency setup.

**Example:**
```go
effect := state.EffectOn(func() state.CleanupFunc {
    fmt.Println("Count:", count.Get())
    return nil
}, count)
defer effect.Dispose()
```

### Watch

```go
func Watch[T any](r *Rune[T], callback func(T)) Unsubscribe
```

Watches a single rune with callback.

**Example:**
```go
unsub := state.Watch(count, func(v int) {
    fmt.Println("Count changed to:", v)
})
defer unsub()
```

### Watch2

```go
func Watch2[A, B any](a *Rune[A], b *Rune[B], callback func(A, B)) Unsubscribe
```

Watches two runes.

### Watch3

```go
func Watch3[A, B, C any](a *Rune[A], b *Rune[B], c *Rune[C], callback func(A, B, C)) Unsubscribe
```

Watches three runes.

---

## Interfaces

### Observable

Type-erased interface for state primitives. Allows storing mixed-type Runes in single collection.

```go
type Observable interface {
    SubscribeAny(func(any)) Unsubscribe
    GetAny() any
}
```

### Settable

Extends Observable for types that can be updated.

```go
type Settable interface {
    Observable
    SetAny(any) error
}
```

### Serializable

Values that can be serialized to JSON.

```go
type Serializable interface {
    Serialize() ([]byte, error)
}
```

---

## StateMap

Collection of observables for component state management.

### Constructor

```go
func NewStateMap() *StateMap
```

### Methods

#### Add

```go
func (sm *StateMap) Add(name string, obs Observable) *StateMap
```

Adds observable to collection. Returns self for chaining.

**Example:**
```go
stateMap := state.NewStateMap()
stateMap.Add("count", count).Add("name", name)
```

#### AddAny

```go
func (sm *StateMap) AddAny(name string, value interface{}) *StateMap
```

Adds primitive value as rune.

**Example:**
```go
stateMap.AddAny("initialized", true)
stateMap.AddAny("items", []string{"a", "b"})
```

#### Get

```go
func (sm *StateMap) Get(name string) (Observable, bool)
```

Retrieves observable by name.

#### ForEach

```go
func (sm *StateMap) ForEach(fn func(key string, value any))
```

Iterates over all observables.

#### ToMap

```go
func (sm *StateMap) ToMap() map[string]any
```

Returns all state values as plain map.

#### MarshalJSON

```go
func (sm *StateMap) MarshalJSON() ([]byte, error)
```

Serializes state map to JSON.

#### ToJSON

```go
func (sm *StateMap) ToJSON() (string, error)
```

Returns state as JSON string.

### OnChange Callback

```go
stateMap.OnChange = func(key string, value any) {
    fmt.Printf("State changed: %s = %v\n", key, value)
}
```

---

## Batch Updates

### Batch

```go
func Batch(fn func())
```

Executes function. For server safety, notifications are pass-through (not deferred globally).

**Example:**
```go
state.Batch(func() {
    count.Set(1)
    name.Set("updated")
    // Notifications happen immediately for server safety
})
```

### BatchResult

```go
func BatchResult[T any](fn func() T) T
```

Batch with return value.

### BatchError

```go
func BatchError(fn func() error) error
```

Batch with error return.

---

## Serialization

### SerializeState

```go
func SerializeState(runes map[string]interface{}) ([]byte, error)
```

Serializes multiple runes into JSON object.

**Example:**
```go
data, err := state.SerializeState(map[string]interface{}{
    "count": count,
    "name":  name,
})
```

---

## State Messages

### StateSnapshot

Snapshot of component state at a point in time.

```go
type StateSnapshot struct {
    ComponentID string                 `json:"componentId"`
    State       map[string]interface{} `json:"state"`
    Timestamp   int64                  `json:"timestamp"`
}
```

#### Constructor

```go
func NewSnapshot(componentID string, state map[string]interface{}) *StateSnapshot
```

### StateDiff

Represents a change in state.

```go
type StateDiff struct {
    ComponentID string      `json:"componentId"`
    Key         string      `json:"key"`
    OldValue    interface{} `json:"oldValue,omitempty"`
    NewValue    interface{} `json:"newValue"`
    Timestamp   int64       `json:"timestamp"`
}
```

#### Constructor

```go
func NewStateDiff(componentID, key string, oldValue, newValue interface{}) *StateDiff
```

### StateMessage

Message sent between server and client.

```go
type StateMessage struct {
    Type        string      `json:"type"` // "init", "update", "sync", "error"
    ComponentID string      `json:"componentId,omitempty"`
    Key         string      `json:"key,omitempty"`
    Value       interface{} `json:"value,omitempty"`
    State       interface{} `json:"state,omitempty"`
    Error       string      `json:"error,omitempty"`
    Timestamp   int64       `json:"timestamp"`
}
```

#### Message Constructors

```go
func NewInitMessage(componentID string, state interface{}) *StateMessage
func NewUpdateMessage(componentID, key string, value interface{}) *StateMessage
func NewSyncMessage(componentID string, state interface{}) *StateMessage
func NewErrorMessage(componentID, errMsg string) *StateMessage
```

#### ParseMessage

```go
func ParseMessage(data []byte) (*StateMessage, error)
```

Parses JSON message.

---

## Validation

### StateValidator

Validates state values.

```go
validator := state.NewStateValidator()
validator.AddValidator("age", func(v interface{}) error {
    if age, ok := v.(int); !ok || age < 0 {
        return fmt.Errorf("invalid age")
    }
    return nil
})

err := validator.Validate("age", 25)
err := validator.ValidateAll(map[string]interface{}{
    "age": 25,
    "name": "John",
})
```

---

## Type Definitions

### Unsubscribe

```go
type Unsubscribe func()
```

Function returned by Subscribe to remove subscription.

### Subscriber

```go
type Subscriber[T any] func(T)
```

Callback function that receives value updates.

### CleanupFunc

```go
type CleanupFunc func()
```

Returned by effects for cleanup.

### Validator

```go
type Validator func(interface{}) error
```

Validates a state value.

---

## Thread Safety

All primitives are thread-safe:
- `Rune[T]` uses `sync.RWMutex` for concurrent access
- `Derived[T]` uses `sync.RWMutex` for concurrent access
- `Effect` uses `sync.RWMutex` and `sync.Mutex` for safe execution
- `StateMap` uses `sync.RWMutex` for concurrent access

---

## Complete Example

```go
package main

import (
    "fmt"
    "github.com/aydenstechdungeon/gospa/state"
)

func main() {
    // Create reactive state
    count := state.NewRune(0)
    name := state.NewRune("World")
    
    // Create derived value
    greeting := state.DerivedFrom(func() string {
        return fmt.Sprintf("Hello, %s! Count: %d", name.Get(), count.Get())
    }, count, name)
    
    // Watch for changes
    unsubGreeting := greeting.Subscribe(func(v string) {
        fmt.Println("Greeting:", v)
    })
    defer unsubGreeting()
    
    // Effect with cleanup
    effect := state.EffectOn(func() state.CleanupFunc {
        fmt.Printf("Effect: count=%d, name=%s\n", count.Get(), name.Get())
        return func() {
            fmt.Println("Effect cleanup")
        }
    }, count, name)
    defer effect.Dispose()
    
    // State map for component
    stateMap := state.NewStateMap()
    stateMap.Add("count", count)
    stateMap.Add("name", name)
    stateMap.OnChange = func(key string, value any) {
        fmt.Printf("State changed: %s = %v\n", key, value)
    }
    
    // Update state
    count.Set(1)
    name.Set("GoSPA")
    count.Update(func(v int) int { return v + 1 })
    
    // Serialize state
    json, _ := stateMap.ToJSON()
    fmt.Println("State JSON:", json)
}
```

---

## Comparison with Client-Side Primitives

| Feature | Go Server | TypeScript Client |
|---------|-----------|-------------------|
| Basic State | `Rune[T]` | `Rune<T>` |
| Computed | `Derived[T]` | `Derived<T>` |
| Side Effects | `Effect` | `Effect` |
| Batch Updates | `Batch()` | `batch()` |
| State Collection | `StateMap` | `StateMap` |
| Async Resources | Manual | `Resource<T>` |
| Raw State | N/A | `RuneRaw<T>` |
| Pre-Effects | N/A | `PreEffect` |

The Go implementation focuses on server-side concerns:
- Thread safety for concurrent requests
- JSON serialization for client sync
- No async primitives (use goroutines directly)
- No DOM-related features
