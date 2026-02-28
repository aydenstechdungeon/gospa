package state

import (
	"sync"
	"testing"
	"time"
)

func TestBatch(t *testing.T) {
	count := NewRune(0)
	var callCount int
	var mu sync.Mutex

	unsub := count.Subscribe(func(v int) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})
	defer unsub()

	// Batch multiple updates
	Batch(func() {
		count.Set(1)
		count.Set(2)
		count.Set(3)
	})

	// Wait for batch to complete
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	// In server-side batching, notifications may be synchronous
	// The exact count depends on implementation
	if callCount == 0 {
		t.Error("Expected at least one notification after batch")
	}
	mu.Unlock()

	// Final value should be 3
	if count.Get() != 3 {
		t.Errorf("Expected final value 3, got %d", count.Get())
	}
}

func TestBatchResult(t *testing.T) {
	count := NewRune(5)

	result := BatchResult(func() int {
		count.Set(10)
		return count.Get() * 2
	})

	if result != 20 {
		t.Errorf("Expected result 20, got %d", result)
	}

	if count.Get() != 10 {
		t.Errorf("Expected count 10, got %d", count.Get())
	}
}

func TestBatchError(t *testing.T) {
	var called bool

	err := BatchError(func() error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !called {
		t.Error("Expected function to be called")
	}
}

func TestNestedBatch(t *testing.T) {
	count := NewRune(0)
	var callCount int
	var mu sync.Mutex

	unsub := count.Subscribe(func(v int) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})
	defer unsub()

	// Nested batches
	Batch(func() {
		count.Set(1)
		Batch(func() {
			count.Set(2)
			Batch(func() {
				count.Set(3)
			})
		})
	})

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if callCount == 0 {
		t.Error("Expected at least one notification after nested batch")
	}
	mu.Unlock()

	if count.Get() != 3 {
		t.Errorf("Expected final value 3, got %d", count.Get())
	}
}

func TestConcurrentBatch(t *testing.T) {
	count := NewRune(0)
	var wg sync.WaitGroup

	// Multiple concurrent batches
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			Batch(func() {
				count.Set(val)
			})
		}(i)
	}

	wg.Wait()

	// Final value should be one of the set values
	finalValue := count.Get()
	if finalValue < 0 || finalValue >= 10 {
		t.Errorf("Expected value in range [0, 10), got %d", finalValue)
	}
}
