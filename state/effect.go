// Package state provides Effect for reactive side effects.
// Effects run when their dependencies change, similar to Svelte's $effect rune.
package state

import (
	"sync"
)

// CleanupFunc is returned by effects to clean up resources
type CleanupFunc func()

// EffectFn is the function passed to an effect
type EffectFn func() CleanupFunc

// Effect represents a reactive side effect that runs when dependencies change.
// Similar to Svelte's $effect rune, it automatically tracks dependencies and
// re-runs when they change.
type Effect struct {
	mu       sync.RWMutex
	fn       EffectFn
	cleanup  CleanupFunc
	deps     []*Rune[any]
	unsubs   []Unsubscribe
	active   bool
	disposed bool
}

// NewEffect creates a new effect that runs the given function.
// The function can return a cleanup function that runs before the next execution
// or when the effect is disposed.
//
// Example:
//
//	count := state.NewRune(0)
//	effect := state.NewEffect(func() state.CleanupFunc {
//	    fmt.Println("Count is:", count.Get())
//	    return func() {
//	        fmt.Println("Cleaning up")
//	    }
//	})
//	defer effect.Dispose()
func NewEffect(fn EffectFn) *Effect {
	e := &Effect{
		fn:     fn,
		deps:   make([]*Rune[any], 0),
		unsubs: make([]Unsubscribe, 0),
		active: true,
	}
	// Run immediately
	e.run()
	return e
}

// run executes the effect function with cleanup
func (e *Effect) run() {
	e.mu.Lock()
	// Run cleanup from previous execution
	if e.cleanup != nil {
		e.cleanup()
		e.cleanup = nil
	}
	e.mu.Unlock()

	// Run the effect and capture cleanup
	cleanup := e.fn()

	e.mu.Lock()
	e.cleanup = cleanup
	e.mu.Unlock()
}

// DependOn adds a rune as a dependency of this effect.
// When the rune changes, the effect will re-run.
func (e *Effect) DependOn(r *Rune[any]) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.disposed || !e.active {
		return
	}

	// Subscribe to the rune
	unsub := r.Subscribe(func(_ any) {
		if e.IsActive() {
			e.run()
		}
	})

	e.deps = append(e.deps, r)
	e.unsubs = append(e.unsubs, unsub)
}

// IsActive returns whether the effect is currently active
func (e *Effect) IsActive() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.active && !e.disposed
}

// Pause temporarily stops the effect from running
func (e *Effect) Pause() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.active = false
}

// Resume reactivates a paused effect
func (e *Effect) Resume() {
	e.mu.Lock()
	wasInactive := !e.active && !e.disposed
	e.active = true
	e.mu.Unlock()

	// Re-run if we were inactive
	if wasInactive {
		e.run()
	}
}

// Dispose permanently stops the effect and cleans up resources
func (e *Effect) Dispose() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.disposed {
		return
	}

	e.disposed = true
	e.active = false

	// Run final cleanup
	if e.cleanup != nil {
		e.cleanup()
		e.cleanup = nil
	}

	// Unsubscribe from all dependencies
	for _, unsub := range e.unsubs {
		unsub()
	}
	e.unsubs = nil
	e.deps = nil
}

// EffectOn creates an effect that depends on one or more runes.
// This is a convenience function that automatically sets up dependencies.
//
// Example:
//
//	count := state.NewRune(0)
//	effect := state.EffectOn(func() state.CleanupFunc {
//	    fmt.Println("Count is:", count.Get())
//	    return nil
//	}, count)
//	defer effect.Dispose()
func EffectOn(fn EffectFn, runes ...*Rune[any]) *Effect {
	e := NewEffect(fn)
	for _, r := range runes {
		e.DependOn(r)
	}
	return e
}

// Watch creates an effect that watches a single rune and calls a callback with its value.
//
// Example:
//
//	count := state.NewRune(0)
//	unsub := state.Watch(count, func(v int) {
//	    fmt.Println("Count changed to:", v)
//	})
//	defer unsub()
func Watch[T any](r *Rune[T], callback func(T)) Unsubscribe {
	effect := EffectOn(func() CleanupFunc {
		callback(r.Get())
		return nil
	}, anyRune(r))

	return func() {
		effect.Dispose()
	}
}

// Watch2 creates an effect that watches two runes.
func Watch2[A, B any](a *Rune[A], b *Rune[B], callback func(A, B)) Unsubscribe {
	effect := EffectOn(func() CleanupFunc {
		callback(a.Get(), b.Get())
		return nil
	}, anyRune(a), anyRune(b))

	return func() {
		effect.Dispose()
	}
}

// Watch3 creates an effect that watches three runes.
func Watch3[A, B, C any](a *Rune[A], b *Rune[B], c *Rune[C], callback func(A, B, C)) Unsubscribe {
	effect := EffectOn(func() CleanupFunc {
		callback(a.Get(), b.Get(), c.Get())
		return nil
	}, anyRune(a), anyRune(b), anyRune(c))

	return func() {
		effect.Dispose()
	}
}
