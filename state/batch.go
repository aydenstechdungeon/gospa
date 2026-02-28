// Package state provides batch update support for reactive primitives.
// Batching is request-scoped: notifications are deferred until the batch completes,
// then all subscribers are notified at once. This prevents intermediate state updates
// from triggering multiple re-renders or WebSocket messages.
package state

import (
	"context"
	"sync"
)

// batchContextKey is used to store batch state in context
type batchContextKey struct{}

// batchState tracks runes that have pending notifications within a batch
type batchState struct {
	mu     sync.Mutex
	dirty  map[string]notifier // map of ID -> notifier
	active bool
}

// notifier interface for objects that can be batched
type notifier interface {
	notifySubscribers()
	ID() string
}

// batchManager manages the current batch state using context
var batchManager = &batchManagerInstance{}

type batchManagerInstance struct {
	mu sync.RWMutex
	// ctx tracks which goroutine/context has an active batch
	ctx context.Context
}

// setBatchContext sets the active batch context
func (bm *batchManagerInstance) setBatchContext(ctx context.Context) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.ctx = ctx
}

// clearBatchContext clears the active batch context
func (bm *batchManagerInstance) clearBatchContext() {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.ctx = nil
}

// isInBatch checks if we're currently in a batch for the given context
func (bm *batchManagerInstance) isInBatch(ctx context.Context) bool {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	if bm.ctx == nil {
		return false
	}
	return bm.ctx == ctx
}

// getBatchState retrieves the batch state from context if it exists
func getBatchState(ctx context.Context) *batchState {
	if ctx == nil {
		return nil
	}
	if bs, ok := ctx.Value(batchContextKey{}).(*batchState); ok {
		return bs
	}
	return nil
}

// inBatch returns true if the current goroutine is within a batch operation
// This checks the background context by default; use BatchWithContext for proper scoping
func inBatch() bool {
	batchManager.mu.RLock()
	defer batchManager.mu.RUnlock()
	return batchManager.ctx != nil
}

// addToBatch adds a notifier to the current batch for deferred notification
func addToBatch(n notifier) {
	batchManager.mu.RLock()
	ctx := batchManager.ctx
	batchManager.mu.RUnlock()

	if ctx == nil {
		return
	}

	bs := getBatchState(ctx)
	if bs == nil {
		return
	}

	bs.mu.Lock()
	defer bs.mu.Unlock()

	if bs.dirty == nil {
		bs.dirty = make(map[string]notifier)
	}
	bs.dirty[n.ID()] = n
}

// Batch executes the given function with notification batching enabled.
// All state changes within the function will have their subscriber notifications
// deferred until the batch completes, at which point all subscribers are notified
// at once. This prevents cascading updates and reduces re-renders.
//
// Example:
//
//	Batch(func() {
//	    count.Set(10)
//	    name.Set("Alice")  // Subscribers notified after both updates complete
//	})
func Batch(fn func()) {
	// Create a context for this batch
	ctx := context.Background()
	bs := &batchState{
		dirty:  make(map[string]notifier),
		active: true,
	}
	ctx = context.WithValue(ctx, batchContextKey{}, bs)

	// Set as active batch
	batchManager.setBatchContext(ctx)
	defer batchManager.clearBatchContext()

	// Execute the batch function
	fn()

	// Flush all pending notifications
	bs.mu.Lock()
	dirtyList := make([]notifier, 0, len(bs.dirty))
	for _, n := range bs.dirty {
		dirtyList = append(dirtyList, n)
	}
	bs.dirty = nil
	bs.active = false
	bs.mu.Unlock()

	// Notify all subscribers outside the lock
	for _, n := range dirtyList {
		n.notifySubscribers()
	}
}

// BatchWithContext executes the given function with notification batching
// using the provided context. This allows for request-scoped batching in
// HTTP handlers. The context must support value storage.
//
// Example in a Fiber handler:
//
//	app.Get("/api/update", func(c *fiber.Ctx) error {
//	    return state.BatchWithContext(c.Context(), func() error {
//	        counter.Set(counter.Get() + 1)
//	        lastUpdated.Set(time.Now())
//	        return nil
//	    })
//	})
func BatchWithContext(ctx context.Context, fn func() error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	bs := &batchState{
		dirty:  make(map[string]notifier),
		active: true,
	}
	ctx = context.WithValue(ctx, batchContextKey{}, bs)

	// Set as active batch
	batchManager.setBatchContext(ctx)
	defer batchManager.clearBatchContext()

	// Execute the batch function
	if err := fn(); err != nil {
		return err
	}

	// Flush all pending notifications
	bs.mu.Lock()
	dirtyList := make([]notifier, 0, len(bs.dirty))
	for _, n := range bs.dirty {
		dirtyList = append(dirtyList, n)
	}
	bs.dirty = nil
	bs.active = false
	bs.mu.Unlock()

	// Notify all subscribers outside the lock
	for _, n := range dirtyList {
		n.notifySubscribers()
	}

	return nil
}

// BatchResult executes the given function with batching and returns its result.
// All notifications are deferred until the function completes.
func BatchResult[T any](fn func() T) T {
	var result T
	Batch(func() {
		result = fn()
	})
	return result
}

// BatchError executes the given function with batching and returns any error.
// All state changes within the function will be batched and notifications
// deferred until the batch completes, regardless of error status.
func BatchError(fn func() error) error {
	var flushErr error

	Batch(func() {
		flushErr = fn()
	})

	return flushErr
}

// FlushPendingNotifications immediately sends all pending notifications
// for the given context. This is useful when you need to ensure updates
// are sent before a long-running operation or before returning from a handler.
func FlushPendingNotifications(ctx context.Context) {
	bs := getBatchState(ctx)
	if bs == nil {
		return
	}

	bs.mu.Lock()
	dirtyList := make([]notifier, 0, len(bs.dirty))
	for _, n := range bs.dirty {
		dirtyList = append(dirtyList, n)
	}
	bs.dirty = make(map[string]notifier) // Reset but keep batch active
	bs.mu.Unlock()

	for _, n := range dirtyList {
		n.notifySubscribers()
	}
}

// IsInBatch returns true if there is an active batch operation
func IsInBatch() bool {
	return inBatch()
}
