// Package state provides batch update support for reactive primitives.
// Batching is request-scoped: notifications are deferred until the batch completes,
// then all subscribers are notified at once. This prevents intermediate state updates
// from triggering multiple re-renders or WebSocket messages.
package state

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// batchContextKey is used to store batch state in context
type batchContextKey struct{}

// batchState tracks runes that have pending notifications within a batch
type batchState struct {
	mu     sync.Mutex
	dirty  map[string]notifier // map of ID -> notifier
	active bool
}

var batchStatePool = sync.Pool{
	New: func() any {
		return &batchState{
			dirty: make(map[string]notifier),
		}
	},
}

func getBatchStateFromPool() *batchState {
	bs := batchStatePool.Get().(*batchState)
	bs.active = true
	if bs.dirty == nil {
		bs.dirty = make(map[string]notifier)
	}
	return bs
}

func putBatchStateToPool(bs *batchState) {
	bs.active = false
	for k := range bs.dirty {
		delete(bs.dirty, k)
	}
	batchStatePool.Put(bs)
}

// notifier interface for objects that can be batched
type notifier interface {
	notifySubscribers()
	ID() string
}

// activeBatches maps goroutine ID to *batchState
var activeBatches sync.Map

// activeSyncBatchCount tracks the number of currently active synchronous Batch()
// calls across all goroutines. This allows getBatchState to skip the expensive
// runtime.Stack / getGID() call when no synchronous batch is active.
var activeSyncBatchCount atomic.Int64

// activeContextBatches maps context identities to active batch states.
// This allows BatchWithContext to work with the caller-provided ctx even when
// fn does not receive the enriched context directly.
var activeContextBatches sync.Map

// getGID returns the current goroutine ID by parsing a runtime stack trace.
// WARNING: This is an intentional — but limited — use of goroutine-local state.
// It is ONLY safe for synchronous Batch() calls where the entire batch lifecycle
// (start → mutations → flush) runs in the same goroutine without yielding to the
// scheduler. Never call Batch() and then hand work off to spawned goroutines and
// expect them to inherit the batch state — they will not. Use BatchWithContext
// (passing the enriched ctx to sub-functions) for that pattern instead.
// GID parsing is expensive (runtime.Stack). Optimizing the string parsing.
func getGID() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	if n < 10 {
		return 0
	}
	// skip "goroutine "
	s := string(buf[10:n])
	spaceIdx := strings.IndexByte(s, ' ')
	if spaceIdx == -1 {
		return 0
	}
	id, _ := strconv.ParseInt(s[:spaceIdx], 10, 64)
	return id
}

// getBatchState retrieves the batch state from context or goroutine-local storage.
func getBatchState(ctx context.Context) *batchState {
	// 1. Check context first (explicitly scoped) — this is always cheap.
	if ctx != nil {
		if bs, ok := ctx.Value(batchContextKey{}).(*batchState); ok {
			return bs
		}
		// Fallback to checking the context pointer in the active map
		if bs, ok := activeContextBatches.Load(contextKeyOnly(ctx)); ok {
			return bs.(*batchState)
		}
	}

	// PERF FIX: Skip the expensive runtime.Stack call (getGID) entirely when
	// no synchronous Batch() is active.
	if activeSyncBatchCount.Load() == 0 {
		return nil
	}

	// 2. Fallback to goroutine-local storage — only reached inside a Batch()
	// or when mutations happen inside BatchWithContext without passing ctx.
	gid := getGID()
	if bs, ok := activeBatches.Load(gid); ok {
		return bs.(*batchState)
	}
	return nil
}

func contextKeyOnly(ctx context.Context) string {
	return fmt.Sprintf("%p", ctx)
}

// inBatch returns true if the current goroutine is within a batch operation.
func inBatch() bool {
	return getBatchState(context.Background()) != nil
}

// addToBatch adds a notifier to the current batch for deferred notification.
func addToBatch(n notifier) {
	bs := getBatchState(context.Background())
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
// All state mutations inside fn() are deferred and flushed atomically when fn returns.
// This is safe when fn() runs synchronously in the calling goroutine. If fn() spawns
// new goroutines that also mutate state, use BatchWithContext and pass the ctx down.
func Batch(fn func()) {
	bs := getBatchStateFromPool()
	defer putBatchStateToPool(bs)

	gid := getGID()
	activeSyncBatchCount.Add(1)
	activeBatches.Store(gid, bs)
	defer func() {
		activeBatches.Delete(gid)
		activeSyncBatchCount.Add(-1)
	}()

	fn()

	flushBatch(bs)
}

// BatchWithContext executes the given function with notification batching using context.
// The enriched context is stored so that any code receiving it can call
// getBatchState(ctx) and find the active batch even from a different goroutine.
// For sub-goroutines to participate in the same batch, pass batchCtx down to them.
func BatchWithContext(ctx context.Context, fn func() error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	bs := getBatchStateFromPool()
	defer putBatchStateToPool(bs)

	activeSyncBatchCount.Add(1)
	ctxKey := contextKeyOnly(ctx)
	activeContextBatches.Store(ctxKey, bs)

	gid := getGID()
	activeBatches.Store(gid, bs)

	defer func() {
		activeContextBatches.Delete(ctxKey)
		activeBatches.Delete(gid)
		activeSyncBatchCount.Add(-1)
	}()

	if err := fn(); err != nil {
		return err
	}

	flushBatch(bs)
	return nil
}

// BatchWithContextFn is an alternative to BatchWithContext where fn receives the
// enriched context directly, enabling sub-functions and spawned goroutines to
// participate in the same batch by passing batchCtx to their getBatchState calls.
func BatchWithContextFn(ctx context.Context, fn func(batchCtx context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	bs := getBatchStateFromPool()
	defer putBatchStateToPool(bs)

	batchCtx := context.WithValue(ctx, batchContextKey{}, bs)

	activeSyncBatchCount.Add(1)
	ctxKey := contextKeyOnly(ctx)
	activeContextBatches.Store(ctxKey, bs)

	defer func() {
		activeContextBatches.Delete(ctxKey)
		activeSyncBatchCount.Add(-1)
	}()

	if err := fn(batchCtx); err != nil {
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
