package state

import (
	"sync"
	"testing"
	"time"
)

func TestNewRune(t *testing.T) {
	r := NewRune(42)
	if r.Get() != 42 {
		t.Errorf("Expected initial value 42, got %d", r.Get())
	}

	s := NewRune("hello")
	if s.Get() != "hello" {
		t.Errorf("Expected initial value 'hello', got %s", s.Get())
	}
}

func TestRuneSet(t *testing.T) {
	r := NewRune(0)
	r.Set(10)
	if r.Get() != 10 {
		t.Errorf("Expected value 10, got %d", r.Get())
	}

	// Setting same value should not cause issues
	r.Set(10)
	if r.Get() != 10 {
		t.Errorf("Expected value 10 after duplicate set, got %d", r.Get())
	}
}

func TestRuneSubscribe(t *testing.T) {
	r := NewRune(0)
	var received []int
	var mu sync.Mutex

	unsub := r.Subscribe(func(v int) {
		mu.Lock()
		received = append(received, v)
		mu.Unlock()
	})
	defer unsub()

	r.Set(1)
	r.Set(2)
	r.Set(3)

	// Wait for notifications
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if len(received) != 3 {
		t.Errorf("Expected 3 notifications, got %d", len(received))
	}
	if received[0] != 1 || received[1] != 2 || received[2] != 3 {
		t.Errorf("Expected [1, 2, 3], got %v", received)
	}
	mu.Unlock()
}

func TestRuneUnsubscribe(t *testing.T) {
	r := NewRune(0)
	var count int
	var mu sync.Mutex

	unsub := r.Subscribe(func(v int) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	r.Set(1)
	unsub()
	r.Set(2)
	r.Set(3)

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if count != 1 {
		t.Errorf("Expected 1 notification after unsubscribe, got %d", count)
	}
	mu.Unlock()
}

func TestRuneUpdate(t *testing.T) {
	r := NewRune(5)
	r.Update(func(v int) int {
		return v * 2
	})
	if r.Get() != 10 {
		t.Errorf("Expected value 10 after update, got %d", r.Get())
	}
}

func TestRuneConcurrentAccess(t *testing.T) {
	r := NewRune(0)
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			r.Set(val)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.Get()
		}()
	}

	wg.Wait()
}

func TestRuneID(t *testing.T) {
	r1 := NewRune(1)
	r2 := NewRune(2)

	if r1.ID() == r2.ID() {
		t.Error("Expected different IDs for different runes")
	}

	if r1.ID() == "" {
		t.Error("Expected non-empty ID")
	}
}

func TestRuneMarshalJSON(t *testing.T) {
	r := NewRune(42)
	data, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	expected := `{"id":"` + r.ID() + `","value":42}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

func TestRuneSetAny(t *testing.T) {
	r := NewRune(0)
	err := r.SetAny(42)
	if err != nil {
		t.Fatalf("SetAny failed: %v", err)
	}
	if r.Get() != 42 {
		t.Errorf("Expected value 42, got %d", r.Get())
	}

	// Test type conversion via JSON
	r2 := NewRune(0)
	err = r2.SetAny(float64(100))
	if err != nil {
		t.Fatalf("SetAny with float64 failed: %v", err)
	}
	if r2.Get() != 100 {
		t.Errorf("Expected value 100, got %d", r2.Get())
	}
}

func TestRuneGetAny(t *testing.T) {
	r := NewRune(42)
	v := r.GetAny()
	if v != 42 {
		t.Errorf("Expected 42 from GetAny, got %v", v)
	}
}

func TestRuneSubscribeAny(t *testing.T) {
	r := NewRune(0)
	var received []any
	var mu sync.Mutex

	unsub := r.SubscribeAny(func(v any) {
		mu.Lock()
		received = append(received, v)
		mu.Unlock()
	})
	defer unsub()

	r.Set(1)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if len(received) != 1 || received[0] != 1 {
		t.Errorf("Expected [1], got %v", received)
	}
	mu.Unlock()
}

func TestRuneNoNotificationOnEqualValue(t *testing.T) {
	r := NewRune(5)
	var count int
	var mu sync.Mutex

	unsub := r.Subscribe(func(v int) {
		mu.Lock()
		count++
		mu.Unlock()
	})
	defer unsub()

	r.Set(5) // Same value
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if count != 0 {
		t.Errorf("Expected no notification for equal value, got %d", count)
	}
	mu.Unlock()
}

func TestRuneStringEquality(t *testing.T) {
	r := NewRune("hello")
	var count int
	var mu sync.Mutex

	unsub := r.Subscribe(func(v string) {
		mu.Lock()
		count++
		mu.Unlock()
	})
	defer unsub()

	r.Set("hello") // Same value
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if count != 0 {
		t.Errorf("Expected no notification for equal string, got %d", count)
	}
	mu.Unlock()

	r.Set("world")
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if count != 1 {
		t.Errorf("Expected 1 notification for different string, got %d", count)
	}
	mu.Unlock()
}

func TestRuneSliceEquality(t *testing.T) {
	r := NewRune([]int{1, 2, 3})
	var count int
	var mu sync.Mutex

	unsub := r.Subscribe(func(v []int) {
		mu.Lock()
		count++
		mu.Unlock()
	})
	defer unsub()

	r.Set([]int{1, 2, 3}) // Same value
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if count != 0 {
		t.Errorf("Expected no notification for equal slice, got %d", count)
	}
	mu.Unlock()

	r.Set([]int{1, 2, 4})
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if count != 1 {
		t.Errorf("Expected 1 notification for different slice, got %d", count)
	}
	mu.Unlock()
}
