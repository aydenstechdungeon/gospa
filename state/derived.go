// Package state provides Derived[T] for computed/derived reactive state.
// Derived values automatically recalculate when their dependencies change.
package state

import (
	"encoding/json"
	"log"
	"reflect"
	"runtime/debug"
	"sync"
)

// Derived is a computed value that automatically updates when dependencies change.
// Similar to Svelte's $derived rune, it recalculates its value based on a compute function
// and notifies subscribers when the computed value changes.
type Derived[T any] struct {
	mu          sync.RWMutex
	value       T
	compute     func() T
	subscribers map[uint64]subEntry[T]
	// deps tracks the observables this derived value depends on
	deps []dependency
	// dirty marks if dependencies have changed and value needs recomputation
	dirty bool
	// id uniquely identifies this derived value
	id        string
	nextSubID uint64
	version   uint64
	// lastDepVersions stores the version of each dependency when last computed
	lastDepVersions map[string]uint64
	// computing tracks if we are currently recomputing to detect circularity
	computing bool
}

// dependency represents a dependency on an observable
type dependency struct {
	observable  Observable
	unsubscribe Unsubscribe
}

// NewDerived creates a new derived value from a compute function.
// The compute function is called immediately to get the initial value.
//
// Example:
//
//	count := state.NewRune(5)
//	doubled := state.NewDerived(func() int {
//	    return count.Get() * 2
//	})
func NewDerived[T any](compute func() T) *Derived[T] {
	d := &Derived[T]{
		compute:         compute,
		subscribers:     make(map[uint64]subEntry[T]),
		deps:            make([]dependency, 0),
		id:              generateRuneID(),
		dirty:           true,
		nextSubID:       1,
		lastDepVersions: make(map[string]uint64),
	}
	// Compute initial value
	d.recompute()
	return d
}

// recompute recalculates the derived value.
// CRITICAL: New value is calculated entirely outside the lock.
func (d *Derived[T]) recompute() {
	d.mu.Lock()
	if d.computing {
		d.mu.Unlock()
		log.Printf("gospa: circular dependency detected in derived rune %s\n%s", d.id, debug.Stack())
		return
	}
	d.computing = true
	d.mu.Unlock()

	newValue := d.compute()

	var subs []subEntry[T]
	var changed bool

	d.mu.Lock()
	d.computing = false
	if !equal(d.value, newValue) {
		d.value = newValue
		changed = true
		d.version++ // Increment version on change
		if len(d.subscribers) > 0 {
			subs = make([]subEntry[T], 0, len(d.subscribers))
			for _, sub := range d.subscribers {
				subs = append(subs, sub)
			}
		}
	}

	// Update snapshot of dependency versions
	for _, dep := range d.deps {
		d.lastDepVersions[dep.observable.ID()] = dep.observable.Version()
	}

	d.dirty = false
	d.mu.Unlock()

	if changed {
		for _, sub := range subs {
			func(fn Subscriber[T]) {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("gospa: recovered panic in derived subscriber: %v\n%s", r, debug.Stack())
					}
				}()
				fn(newValue)
			}(sub.fn)
		}
	}
}

// Get returns the current computed value.
// If dependencies have changed, it recomputes first.
func (d *Derived[T]) Get() T {
	d.mu.RLock()
	stale := d.dirty
	if !stale {
		for _, dep := range d.deps {
			if dep.observable.Version() > d.lastDepVersions[dep.observable.ID()] {
				stale = true
				break
			}
		}
	}
	d.mu.RUnlock()

	if stale {
		d.recompute()
	}

	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.value
}

// Version returns the current version of the derived value.
func (d *Derived[T]) Version() uint64 {
	d.mu.RLock()
	// During a Version call, we might want to check if we are stale too?
	// But usually Version is used for dependency checking.
	// We can trust the last version if we are not being recomputed.
	defer d.mu.RUnlock()
	return d.version
}

// GetAny returns the current value of the derivative as an interface{}.
// This implements the Observable interface.
func (d *Derived[T]) GetAny() any {
	return d.Get()
}

// Subscribe registers a callback for when the derived value changes.
// Returns an unsubscribe function.
func (d *Derived[T]) Subscribe(fn Subscriber[T]) Unsubscribe {
	d.mu.Lock()
	defer d.mu.Unlock()

	id := d.nextSubID
	d.nextSubID++

	d.subscribers[id] = subEntry[T]{id: id, fn: fn}

	return func() {
		d.mu.Lock()
		defer d.mu.Unlock()
		delete(d.subscribers, id)
	}
}

// SubscribeAny registers a type-erased callback.
// This implements the Observable interface.
func (d *Derived[T]) SubscribeAny(fn func(any)) Unsubscribe {
	return d.Subscribe(func(v T) {
		fn(v)
	})
}

// DependOn adds an observable as a dependency of this derived value.
// When the observable changes, this derived value will be marked dirty.
//
// Example:
//
//	count := state.NewRune(5)
//	doubled := state.NewDerived(func() int {
//	    return count.Get() * 2
//	})
//	doubled.DependOn(count)
func (d *Derived[T]) DependOn(o Observable) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, dep := range d.deps {
		if sameObservable(dep.observable, o) {
			return
		}
	}

	// Subscribe to the observable's changes
	unsub := o.SubscribeAny(func(_ any) {
		d.markDirty()
	})

	d.deps = append(d.deps, dependency{
		observable:  o,
		unsubscribe: unsub,
	})
}

func sameObservable(a, b Observable) bool {
	if a == nil || b == nil {
		return a == b
	}
	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)
	if va.IsValid() && vb.IsValid() && va.Type() == vb.Type() && va.Type().Comparable() {
		return va.Interface() == vb.Interface()
	}
	return false
}

// markDirty marks this derived value as needing recomputation and notifies
// any subscribers of the new computed value. It is called when a dependency
// changes. recompute() is invoked without holding d.mu (it acquires it
// internally) to avoid a deadlock with the caller's subscription goroutine.
func (d *Derived[T]) markDirty() {
	d.mu.Lock()
	if d.dirty {
		d.mu.Unlock()
		return
	}
	d.dirty = true
	d.mu.Unlock()

	// Recompute immediately — this sets dirty=false, computes the new value,
	// takes a snapshot of current subscribers, and fires notify() in a goroutine.
	d.recompute()
}

// ID returns the unique identifier for this derived value
func (d *Derived[T]) ID() string {
	return d.id
}

// MarshalJSON implements json.Marshaler for serialization
func (d *Derived[T]) MarshalJSON() ([]byte, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return json.Marshal(map[string]interface{}{
		"id":    d.id,
		"value": d.value,
	})
}

// Dispose cleans up all subscriptions to dependencies
func (d *Derived[T]) Dispose() {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, dep := range d.deps {
		dep.unsubscribe()
	}
	d.deps = nil
	d.subscribers = nil
}

// DerivedFrom creates a derived value that depends on one or more observables.
// This is a convenience function that automatically sets up dependencies.
//
// Example:
//
//	count := state.NewRune(5)
//	doubled := state.DerivedFrom(func() int {
//	    return count.Get() * 2
//	}, count)
func DerivedFrom[T any](compute func() T, observables ...Observable) *Derived[T] {
	d := NewDerived(compute)
	for _, o := range observables {
		d.DependOn(o)
	}
	return d
}

// Derived2 creates a derived value from two runes with a combine function.
func Derived2[A, B, T any](a *Rune[A], b *Rune[B], combine func(A, B) T) *Derived[T] {
	return DerivedFrom(func() T {
		return combine(a.Get(), b.Get())
	}, a, b)
}

// Derived3 creates a derived value from three runes with a combine function.
func Derived3[A, B, C, T any](a *Rune[A], b *Rune[B], c *Rune[C], combine func(A, B, C) T) *Derived[T] {
	return DerivedFrom(func() T {
		return combine(a.Get(), b.Get(), c.Get())
	}, a, b, c)
}
