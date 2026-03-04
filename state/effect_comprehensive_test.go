package state

import (
	"sync"
	"testing"
	"time"
)

// ─── NewEffect ────────────────────────────────────────────────────────────────

func TestNewEffect_RunsImmediately(t *testing.T) {
	ran := false
	e := NewEffect(func() CleanupFunc {
		ran = true
		return nil
	})
	defer e.Dispose()

	if !ran {
		t.Error("NewEffect should run the function immediately")
	}
}

func TestNewEffect_RunsOnDependencyChange(t *testing.T) {
	count := NewRune(0)
	var mu sync.Mutex
	runCount := 0

	e := EffectOn(func() CleanupFunc {
		_ = count.Get()
		mu.Lock()
		runCount++
		mu.Unlock()
		return nil
	}, count)
	defer e.Dispose()

	count.Set(1)
	count.Set(2)
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	// Should have run at least 2 times (initial + 1 or 2 changes)
	if runCount < 2 {
		t.Errorf("expected Effect to run at least 2 times, got %d", runCount)
	}
}

// ─── Effect.Pause / Resume ────────────────────────────────────────────────────

func TestEffect_PauseStop(t *testing.T) {
	count := NewRune(0)
	var mu sync.Mutex
	runCount := 0

	e := EffectOn(func() CleanupFunc {
		_ = count.Get()
		mu.Lock()
		runCount++
		mu.Unlock()
		return nil
	}, count)

	// Get initial run count
	time.Sleep(10 * time.Millisecond)
	e.Pause()

	mu.Lock()
	beforePause := runCount
	mu.Unlock()

	count.Set(99)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	afterPause := runCount
	mu.Unlock()

	if afterPause > beforePause {
		t.Errorf("Effect should not run while paused (before=%d, after=%d)", beforePause, afterPause)
	}
}

func TestEffect_Resume(t *testing.T) {
	count := NewRune(0)
	var mu sync.Mutex
	runCount := 0

	e := EffectOn(func() CleanupFunc {
		_ = count.Get()
		mu.Lock()
		runCount++
		mu.Unlock()
		return nil
	}, count)

	e.Pause()
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	before := runCount
	mu.Unlock()

	e.Resume()
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	after := runCount
	mu.Unlock()

	// Resuming should trigger a re-run
	if after <= before {
		t.Errorf("Resume should trigger a re-run (before=%d, after=%d)", before, after)
	}
}

// ─── Effect.Dispose ───────────────────────────────────────────────────────────

func TestEffect_Dispose(t *testing.T) {
	count := NewRune(0)
	var mu sync.Mutex
	runCount := 0

	e := EffectOn(func() CleanupFunc {
		_ = count.Get()
		mu.Lock()
		runCount++
		mu.Unlock()
		return nil
	}, count)

	time.Sleep(10 * time.Millisecond)
	e.Dispose()

	mu.Lock()
	atDispose := runCount
	mu.Unlock()

	count.Set(100)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	atEnd := runCount
	mu.Unlock()

	if atEnd > atDispose {
		t.Errorf("Effect should not run after Dispose (at dispose=%d, at end=%d)", atDispose, atEnd)
	}
}

func TestEffect_DoubleDispose_NoPanic(t *testing.T) {
	e := NewEffect(func() CleanupFunc { return nil })
	e.Dispose()
	e.Dispose() // second dispose should not panic
}

// ─── Effect.IsActive ──────────────────────────────────────────────────────────

func TestEffect_IsActive(t *testing.T) {
	e := NewEffect(func() CleanupFunc { return nil })
	if !e.IsActive() {
		t.Error("new Effect should be active")
	}
	e.Pause()
	// Paused means active=false
	if e.IsActive() {
		t.Error("paused Effect should not be active")
	}
	e.Resume()
	if !e.IsActive() {
		t.Error("resumed Effect should be active")
	}
	e.Dispose()
	if e.IsActive() {
		t.Error("disposed Effect should not be active")
	}
}

// ─── Effect cleanup ───────────────────────────────────────────────────────────

func TestEffect_CleanupRunsOnReexecution(t *testing.T) {
	count := NewRune(0)
	var mu sync.Mutex
	cleanupCount := 0

	e := EffectOn(func() CleanupFunc {
		_ = count.Get()
		return func() {
			mu.Lock()
			cleanupCount++
			mu.Unlock()
		}
	}, count)
	defer e.Dispose()

	count.Set(1)
	time.Sleep(60 * time.Millisecond)

	mu.Lock()
	c := cleanupCount
	mu.Unlock()

	// Cleanup should have been called before re-execution
	if c == 0 {
		t.Error("cleanup function should have been called at least once on re-execution")
	}
}

func TestEffect_CleanupRunsOnDispose(t *testing.T) {
	cleaned := false
	e := NewEffect(func() CleanupFunc {
		return func() {
			cleaned = true
		}
	})
	e.Dispose()

	if !cleaned {
		t.Error("cleanup should run on Dispose")
	}
}

// ─── Watch ───────────────────────────────────────────────────────────────────

func TestWatch_Comprehensive(t *testing.T) {
	count := NewRune(0)
	var mu sync.Mutex
	var received []int

	unsub := Watch(count, func(v int) {
		mu.Lock()
		received = append(received, v)
		mu.Unlock()
	})
	defer unsub()

	count.Set(1)
	count.Set(2)
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	// Should have received values (at least once including initial)
	if len(received) == 0 {
		t.Error("Watch should call callback at least once")
	}
}

func TestWatch_Unsubscribe(t *testing.T) {
	count := NewRune(0)
	var mu sync.Mutex
	callCount := 0

	unsub := Watch(count, func(v int) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	before := callCount
	mu.Unlock()

	unsub() // dispose the watch
	count.Set(99)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	after := callCount
	mu.Unlock()

	if after > before {
		t.Errorf("Watch should not fire after Unsubscribe (before=%d, after=%d)", before, after)
	}
}

// ─── Watch2 ───────────────────────────────────────────────────────────────────

func TestWatch2_Comprehensive(t *testing.T) {
	a := NewRune(1)
	b := NewRune(2)
	var mu sync.Mutex
	lastSum := 0

	unsub := Watch2(a, b, func(x, y int) {
		mu.Lock()
		lastSum = x + y
		mu.Unlock()
	})
	defer unsub()

	a.Set(5)
	time.Sleep(60 * time.Millisecond)

	mu.Lock()
	s := lastSum
	mu.Unlock()

	if s == 0 {
		t.Error("Watch2 should have fired callback")
	}
}

// ─── Watch3 ───────────────────────────────────────────────────────────────────

func TestWatch3_Comprehensive(t *testing.T) {
	a := NewRune(1)
	b := NewRune(2)
	c := NewRune(3)
	var mu sync.Mutex
	called := false

	unsub := Watch3(a, b, c, func(x, y, z int) {
		mu.Lock()
		called = true
		mu.Unlock()
	})
	defer unsub()

	// Initial run
	time.Sleep(30 * time.Millisecond)
	mu.Lock()
	if !called {
		t.Error("Watch3 should call callback on initial run")
	}
	mu.Unlock()
}

// ─── EffectOn with no observables ─────────────────────────────────────────────

func TestEffectOn_NoObservables(t *testing.T) {
	ran := false
	e := EffectOn(func() CleanupFunc {
		ran = true
		return nil
	})
	defer e.Dispose()

	if !ran {
		t.Error("EffectOn with no observables should still run immediately")
	}
}

// ─── Multiple subscribers on same observable ──────────────────────────────────

func TestEffect_MultipleEffectsOnSameRune(t *testing.T) {
	count := NewRune(0)
	var mu sync.Mutex
	runs := [2]int{}

	e1 := EffectOn(func() CleanupFunc {
		_ = count.Get()
		mu.Lock()
		runs[0]++
		mu.Unlock()
		return nil
	}, count)
	e2 := EffectOn(func() CleanupFunc {
		_ = count.Get()
		mu.Lock()
		runs[1]++
		mu.Unlock()
		return nil
	}, count)
	defer e1.Dispose()
	defer e2.Dispose()

	count.Set(42)
	time.Sleep(80 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if runs[0] < 2 || runs[1] < 2 {
		t.Errorf("both effects should have run at least twice; got %v", runs)
	}
}
