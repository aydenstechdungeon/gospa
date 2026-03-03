// Package state provides batch update support for reactive primitives.
// Batching is request-scoped: notifications are deferred until the batch completes,
// then all subscribers are notified at once. This prevents intermediate state updates
// from triggering multiple re-renders or WebSocket messages.
package state

import (
	"context"
	"runtime"
	"strconv"
	"strings"
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

// activeBatches maps goroutine ID to *batchState
var activeBatches sync.Map

// getGID returns the current goroutine ID.
func getGID() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, _ := strconv.ParseInt(idField, 10, 64)
	return id
}

// getBatchState retrieves the batch state from context or goroutine-local storage.
func getBatchState(ctx context.Context) *batchState {
	// 1. Check context first (explicitly scoped)
	if ctx != nil {
		if bs, ok := ctx.Value(batchContextKey{}).(*batchState); ok {
			return bs
		}
	}

	// 2. Fallback to goroutine-local storage
	gid := getGID()
	if bs, ok := activeBatches.Load(gid); ok {
		return bs.(*batchState)
	}
	return nil
}

// inBatch returns true if the current goroutine is within a batch operation.
func inBatch() bool {
	return getBatchState(context.TODO()) != nil
}

// addToBatch adds a notifier to the current batch for deferred notification.
func addToBatch(n notifier) {
	bs := getBatchState(context.TODO())
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
func Batch(fn func()) {
	bs := &batchState{
		dirty:  make(map[string]notifier),
		active: true,
	}
	gid := getGID()
	activeBatches.Store(gid, bs)
	defer activeBatches.Delete(gid)

	fn()

	flushBatch(bs)
}

// BatchWithContext executes the given function with notification batching using context.
func BatchWithContext(ctx context.Context, fn func() error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	bs := &batchState{
		dirty:  make(map[string]notifier),
		active: true,
	}
	_ = context.WithValue(ctx, batchContextKey{}, bs)

	gid := getGID()
	activeBatches.Store(gid, bs)
	defer activeBatches.Delete(gid)

	if err := fn(); err != nil {
		return err
	}

	flushBatch(bs)
	return nil
}

func flushBatch(bs *batchState) {
	bs.mu.Lock()
	dirtyList := make([]notifier, 0, len(bs.dirty))
	for _, n := range bs.dirty {
		dirtyList = append(dirtyList, n)
	}
	bs.dirty = nil
	bs.active = false
	bs.mu.Unlock()

	for _, n := range dirtyList {
		n.notifySubscribers()
	}
}

// BatchResult executes the given function with batching and returns its result.
func BatchResult[T any](fn func() T) T {
	var result T
	Batch(func() {
		result = fn()
	})
	return result
}

// BatchError executes the given function with batching and returns any error.
func BatchError(fn func() error) error {
	var flushErr error
	Batch(func() {
		flushErr = fn()
	})
	return flushErr
}

// FlushPendingNotifications immediately sends all pending notifications for the given context.
func FlushPendingNotifications(ctx context.Context) {
	bs := getBatchState(ctx)
	if bs == nil {
		return
	}
	flushBatch(bs)
}

// IsInBatch returns true if there is an active batch operation.
func IsInBatch() bool {
	return inBatch()
}
