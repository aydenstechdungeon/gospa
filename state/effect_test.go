package state

import (
	"sync"
	"testing"
	"time"
)

func TestNewEffect(t *testing.T) {
	count := NewRune(0)
	var callCount int
	var mu sync.Mutex

	effect := NewEffect(func() CleanupFunc {
		mu.Lock()
		callCount++
		mu.Unlock()
		return nil
	})
	effect.DependOn(count)
	defer effect.Dispose()

	// Effect runs immediately
	mu.Lock()
	if callCount != 1 {
		t.Errorf("Expected 1 initial run, got %d", callCount)
	}
	mu.Unlock()

	// Change dependency
	count.Set(1)
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if callCount != 2 {
		t.Errorf("Expected 2 runs after change, got %d", callCount)
	}
	mu.Unlock()
}

func TestEffectCleanup(t *testing.T) {
	count := NewRune(0)
	var cleanupCount int
	var mu sync.Mutex

	effect := NewEffect(func() CleanupFunc {
		return func() {
			mu.Lock()
			cleanupCount++
			mu.Unlock()
		}
	})
	effect.DependOn(count)
	defer effect.Dispose()

	// No cleanup on first run
	mu.Lock()
	if cleanupCount != 0 {
		t.Errorf("Expected 0 cleanups initially, got %d", cleanupCount)
	}
	mu.Unlock()

	// Change triggers cleanup then new run
	count.Set(1)
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if cleanupCount != 1 {
		t.Errorf("Expected 1 cleanup after change, got %d", cleanupCount)
	}
	mu.Unlock()
}

func TestEffectPauseResume(t *testing.T) {
	count := NewRune(0)
	var callCount int
	var mu sync.Mutex

	effect := NewEffect(func() CleanupFunc {
		mu.Lock()
		callCount++
		mu.Unlock()
		return nil
	})
	effect.DependOn(count)
	defer effect.Dispose()

	// Initial run
	time.Sleep(50 * time.Millisecond)

	// Pause
	effect.Pause()

	// Change while paused
	count.Set(1)
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	pausedCount := callCount
	mu.Unlock()

	if pausedCount != 1 {
		t.Errorf("Expected 1 call while paused, got %d", pausedCount)
	}

	// Resume - should run immediately
	effect.Resume()
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if callCount != 2 {
		t.Errorf("Expected 2 calls after resume, got %d", callCount)
	}
	mu.Unlock()
}

func TestEffectIsActive(t *testing.T) {
	effect := NewEffect(func() CleanupFunc {
		return nil
	})

	if !effect.IsActive() {
		t.Error("Expected effect to be active initially")
	}

	effect.Pause()
	if effect.IsActive() {
		t.Error("Expected effect to be inactive after pause")
	}

	effect.Resume()
	if !effect.IsActive() {
		t.Error("Expected effect to be active after resume")
	}

	effect.Dispose()
	if effect.IsActive() {
		t.Error("Expected effect to be inactive after dispose")
	}
}

func TestEffectDispose(t *testing.T) {
	var cleanupCalled bool

	effect := NewEffect(func() CleanupFunc {
		return func() {
			cleanupCalled = true
		}
	})

	effect.Dispose()

	if !cleanupCalled {
		t.Error("Expected cleanup to be called on dispose")
	}
}

func TestEffectOn(t *testing.T) {
	count := NewRune(0)
	var callCount int
	var mu sync.Mutex

	effect := EffectOn(func() CleanupFunc {
		mu.Lock()
		callCount++
		mu.Unlock()
		return nil
	}, count)
	defer effect.Dispose()

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if callCount != 1 {
		t.Errorf("Expected 1 run, got %d", callCount)
	}
	mu.Unlock()

	count.Set(1)
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if callCount != 2 {
		t.Errorf("Expected 2 runs after change, got %d", callCount)
	}
	mu.Unlock()
}

func TestWatch(t *testing.T) {
	count := NewRune(0)
	var received []int
	var mu sync.Mutex

	unsub := Watch(count, func(v int) {
		mu.Lock()
		received = append(received, v)
		mu.Unlock()
	})
	defer unsub()

	// Watch calls immediately with current value
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if len(received) != 1 || received[0] != 0 {
		t.Errorf("Expected initial [0], got %v", received)
	}
	mu.Unlock()

	count.Set(1)
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if len(received) != 2 || received[1] != 1 {
		t.Errorf("Expected [0, 1], got %v", received)
	}
	mu.Unlock()
}

func TestWatch2(t *testing.T) {
	a := NewRune(1)
	b := NewRune(2)
	var received [][2]int
	var mu sync.Mutex

	unsub := Watch2(a, b, func(x, y int) {
		mu.Lock()
		received = append(received, [2]int{x, y})
		mu.Unlock()
	})
	defer unsub()

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if len(received) != 1 || received[0] != [2]int{1, 2} {
		t.Errorf("Expected initial [[1 2]], got %v", received)
	}
	mu.Unlock()

	a.Set(10)
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if len(received) != 2 || received[1] != [2]int{10, 2} {
		t.Errorf("Expected [[1 2] [10 2]], got %v", received)
	}
	mu.Unlock()
}

func TestWatch3(t *testing.T) {
	a := NewRune(1)
	b := NewRune(2)
	c := NewRune(3)
	var callCount int
	var mu sync.Mutex

	unsub := Watch3(a, b, c, func(x, y, z int) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})
	defer unsub()

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if callCount != 1 {
		t.Errorf("Expected 1 initial call, got %d", callCount)
	}
	mu.Unlock()
}
