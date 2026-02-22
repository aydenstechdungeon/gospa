// Package state provides batch update support for reactive primitives.
// Note: In a concurrent Go web server, global batching (as in Svelte) shares state
// across requests. Batching is a pass-through in this implementation to ensure safety.
package state

func inBatch() bool {
	return false
}

type notifier interface {
	notifySubscribers()
	ID() string
}

func addToBatch(n notifier) {
}

// Batch executes the given function.
// For server safety, it no longer defers notifications globally.
func Batch(fn func()) {
	fn()
}

// BatchResult allows batching with a return value
func BatchResult[T any](fn func() T) T {
	return fn()
}

// BatchError allows batching with error return
func BatchError(fn func() error) error {
	return fn()
}
