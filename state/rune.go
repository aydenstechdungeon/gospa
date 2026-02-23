// Package state provides Svelte rune-like reactive primitives for Go.
// Rune[T] is the base reactive primitive that holds a value and notifies subscribers on changes.
package state

import (
	"encoding/json"
	"reflect"
	"strconv"
	"sync"
)

// Unsubscribe is a function returned by Subscribe to remove the subscription.
type Unsubscribe func()

// Subscriber is a callback function that receives value updates.
type Subscriber[T any] func(T)

// Observable provides a type-erased interface for state primitives.
// This allows storing mixed-type Runes in a single collection.
type Observable interface {
	SubscribeAny(func(any)) Unsubscribe
	GetAny() any
}

// Settable extends Observable for types that can be updated.
type Settable interface {
	Observable
	SetAny(any) error
}

type subEntry[T any] struct {
	id uint64
	fn Subscriber[T]
}

// Rune is the base reactive primitive, similar to Svelte's $state rune.
// It holds a value of type T and notifies all subscribers when the value changes.
type Rune[T any] struct {
	mu          sync.RWMutex
	value       T
	subscribers []subEntry[T]
	// ID uniquely identifies this rune for client-side synchronization
	id string
	// dirty marks if the rune has uncommitted changes in batch mode
	dirty     bool
	nextSubID uint64
}

// runeIDCounter is used to generate unique IDs for runes
var runeIDCounter uint64
var runeIDMu sync.Mutex

// generateRuneID creates a unique identifier for a rune
func generateRuneID() string {
	runeIDMu.Lock()
	defer runeIDMu.Unlock()
	runeIDCounter++
	return strconv.FormatUint(runeIDCounter, 10)
}

// NewRune creates a new Rune with the given initial value.
// This is equivalent to Svelte's $state rune.
//
// Example:
//
//	count := state.NewRune(0)
//	name := state.NewRune("hello")
func NewRune[T any](initial T) *Rune[T] {
	return &Rune[T]{
		value:       initial,
		subscribers: make([]subEntry[T], 0),
		id:          generateRuneID(),
		nextSubID:   1,
	}
}

// Get returns the current value of the rune.
// This is thread-safe and can be called concurrently.
func (r *Rune[T]) Get() T {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.value
}

// GetAny returns the current value of the rune as an interface{}.
// This implements the Observable interface.
func (r *Rune[T]) GetAny() any {
	return r.Get()
}

// Set updates the rune's value and notifies all subscribers.
// If the value is being batched, notification is deferred until the batch completes.
// This is thread-safe and can be called concurrently.
func (r *Rune[T]) Set(value T) {
	r.mu.Lock()
	// Check if value actually changed
	if equal(r.value, value) {
		r.mu.Unlock()
		return
	}
	r.value = value
	r.dirty = true

	// If we're in a batch, don't notify yet
	if inBatch() {
		addToBatch(r)
		r.mu.Unlock()
		return
	}

	// Copy subscribers to avoid holding lock during callbacks
	subs := make([]subEntry[T], len(r.subscribers))
	copy(subs, r.subscribers)
	r.dirty = false
	r.mu.Unlock()

	// Notify subscribers outside the lock
	r.notify(subs, value)
}

// SetAny updates the rune's value from an interface{}.
// This implements the Settable interface.
func (r *Rune[T]) SetAny(value any) error {
	var newValue T
	if v, ok := value.(T); ok {
		newValue = v
	} else {
		// Try JSON fallback for converting types (e.g., float64 to int)
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &newValue); err != nil {
			return err
		}
	}
	r.Set(newValue)
	return nil
}

// Subscribe registers a callback that will be called whenever the value changes.
// Returns an Unsubscribe function to remove the subscription.
//
// Example:
//
//	unsub := count.Subscribe(func(v int) {
//	    fmt.Println("Count changed to:", v)
//	})
//	defer unsub()
func (r *Rune[T]) Subscribe(fn Subscriber[T]) Unsubscribe {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.nextSubID
	r.nextSubID++

	r.subscribers = append(r.subscribers, subEntry[T]{id: id, fn: fn})

	// Return unsubscribe function
	return func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		for i, sub := range r.subscribers {
			if sub.id == id {
				r.subscribers = append(r.subscribers[:i], r.subscribers[i+1:]...)
				break
			}
		}
	}
}

// SubscribeAny registers a type-erased callback.
// This implements the Observable interface.
func (r *Rune[T]) SubscribeAny(fn func(any)) Unsubscribe {
	return r.Subscribe(func(v T) {
		fn(v)
	})
}

// notify calls all subscribers with the new value.
// This is called outside the lock to prevent deadlocks.
func (r *Rune[T]) notify(subs []subEntry[T], value T) {
	for _, sub := range subs {
		sub.fn(value)
	}
}

// notifySubscribers is called during batch flush to notify all subscribers.
func (r *Rune[T]) notifySubscribers() {
	r.mu.Lock()
	if !r.dirty {
		r.mu.Unlock()
		return
	}
	value := r.value
	subs := make([]subEntry[T], len(r.subscribers))
	copy(subs, r.subscribers)
	r.dirty = false
	r.mu.Unlock()

	r.notify(subs, value)
}

// ID returns the unique identifier for this rune.
func (r *Rune[T]) ID() string {
	return r.id
}

// MarshalJSON implements json.Marshaler for serialization to client.
func (r *Rune[T]) MarshalJSON() ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return json.Marshal(map[string]interface{}{
		"id":    r.id,
		"value": r.value,
	})
}

// Update applies a function to the current value and sets the result.
// This is useful for updates that depend on the current value.
//
// Example:
//
//	count.Update(func(v int) int {
//	    return v + 1
//	})
func (r *Rune[T]) Update(fn func(T) T) {
	r.mu.Lock()
	newValue := fn(r.value)
	if equal(r.value, newValue) {
		r.mu.Unlock()
		return
	}
	r.value = newValue
	r.dirty = true

	if inBatch() {
		addToBatch(r)
		r.mu.Unlock()
		return
	}

	subs := make([]subEntry[T], len(r.subscribers))
	copy(subs, r.subscribers)
	r.dirty = false
	r.mu.Unlock()

	r.notify(subs, newValue)
}

// equal compares two values of any type for equality using reflection.
// This handles both comparable and non-comparable types.
func equal[T any](a, b T) bool {
	// Try type assertion for common types to avoid reflection overhead
	var aAny interface{} = a
	var bAny interface{} = b

	switch aVal := aAny.(type) {
	case string:
		if bVal, ok := bAny.(string); ok {
			return aVal == bVal
		}
	case int:
		if bVal, ok := bAny.(int); ok {
			return aVal == bVal
		}
	case int64:
		if bVal, ok := bAny.(int64); ok {
			return aVal == bVal
		}
	case int32:
		if bVal, ok := bAny.(int32); ok {
			return aVal == bVal
		}
	case uint:
		if bVal, ok := bAny.(uint); ok {
			return aVal == bVal
		}
	case float64:
		if bVal, ok := bAny.(float64); ok {
			return aVal == bVal
		}
	case float32:
		if bVal, ok := bAny.(float32); ok {
			return aVal == bVal
		}
	case bool:
		if bVal, ok := bAny.(bool); ok {
			return aVal == bVal
		}
	}

	// Fallback to deeply comparing everything else
	return reflect.DeepEqual(a, b)
}
