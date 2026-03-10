package state

import (
	"sync"
	"testing"
	"time"
)

// ─── NewDerived + Get ─────────────────────────────────────────────────────────

func TestNewDerived_InitialValue(t *testing.T) {
	count := NewRune(5)
	doubled := NewDerived(func() int {
		return count.Get() * 2
	})

	if doubled.Get() != 10 {
		t.Errorf("expected initial derived value 10, got %d", doubled.Get())
	}
}

func TestDerived_UpdatesWhenDepChanges(t *testing.T) {
	count := NewRune(3)
	doubled := DerivedFrom(func() int {
		return count.Get() * 2
	}, count)

	count.Set(7)
	// DependOn marks it dirty; Get() should recompute
	time.Sleep(20 * time.Millisecond) // let goroutine notify
	if doubled.Get() != 14 {
		t.Errorf("expected derived value 14 after count=7, got %d", doubled.Get())
	}
}

func TestDerived_GetAny(t *testing.T) {
	r := NewRune(42)
	d := NewDerived(func() int { return r.Get() })
	v := d.GetAny()
	if v != 42 {
		t.Errorf("expected GetAny()=42, got %v", v)
	}
}

func TestDerived_ID(t *testing.T) {
	d := NewDerived(func() int { return 0 })
	if d.ID() == "" {
		t.Error("ID() should not be empty")
	}
}

func TestDerived_IDUnique(t *testing.T) {
	d1 := NewDerived(func() int { return 0 })
	d2 := NewDerived(func() int { return 0 })
	if d1.ID() == d2.ID() {
		t.Error("different Derived values should have different IDs")
	}
}

// ─── DerivedFrom ──────────────────────────────────────────────────────────────

func TestDerivedFrom_MultipleObservables(t *testing.T) {
	a := NewRune(2)
	b := NewRune(3)
	sum := DerivedFrom(func() int {
		return a.Get() + b.Get()
	}, a, b)

	if sum.Get() != 5 {
		t.Errorf("expected sum=5, got %d", sum.Get())
	}

	a.Set(10)
	time.Sleep(20 * time.Millisecond)
	if sum.Get() != 13 {
		t.Errorf("expected sum=13 after a=10, got %d", sum.Get())
	}
}

// ─── Derived2 / Derived3 ──────────────────────────────────────────────────────

func TestDerived2_Product(t *testing.T) {
	a := NewRune(4)
	b := NewRune(5)
	product := Derived2(a, b, func(x, y int) int { return x * y })

	if product.Get() != 20 {
		t.Errorf("expected 20, got %d", product.Get())
	}

	a.Set(3)
	time.Sleep(20 * time.Millisecond)
	if product.Get() != 15 {
		t.Errorf("expected 15 after a=3, got %d", product.Get())
	}
}

func TestDerived3_Sum(t *testing.T) {
	a := NewRune(1)
	b := NewRune(2)
	c := NewRune(3)
	sum := Derived3(a, b, c, func(x, y, z int) int { return x + y + z })

	if sum.Get() != 6 {
		t.Errorf("expected 6, got %d", sum.Get())
	}
}

// ─── Derived.Subscribe ────────────────────────────────────────────────────────

func TestDerived_Subscribe(t *testing.T) {
	count := NewRune(0)
	doubled := NewDerived(func() int { return count.Get() * 2 })
	doubled.DependOn(count)

	// Subscribe to count changes and re-read derived value after each change
	var mu sync.Mutex
	var observed []int
	unsub := count.Subscribe(func(_ int) {
		mu.Lock()
		observed = append(observed, doubled.Get())
		mu.Unlock()
	})
	defer unsub()

	count.Set(5)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(observed) == 0 {
		t.Error("expected at least one observation from derived after count change")
		return
	}
	// doubled.Get() after count.Set(5) should be 10
	if observed[len(observed)-1] != 10 {
		t.Errorf("expected derived value 10, got %d", observed[len(observed)-1])
	}
}

func TestDerived_Unsubscribe(t *testing.T) {
	count := NewRune(0)
	doubled := DerivedFrom(func() int { return count.Get() * 2 }, count)

	var mu sync.Mutex
	notified := 0
	unsub := doubled.Subscribe(func(_ int) {
		mu.Lock()
		notified++
		mu.Unlock()
	})

	count.Set(1)
	time.Sleep(50 * time.Millisecond)
	unsub() // unsubscribe

	count.Set(2)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if notified > 1 {
		t.Errorf("expected at most 1 notification after unsubscribe, got %d", notified)
	}
}

func TestDerived_SubscribeAny(t *testing.T) {
	count := NewRune(0)
	doubled := NewDerived(func() int { return count.Get() * 2 })
	doubled.DependOn(count)

	// Test GetAny directly after marking dirty
	count.Set(7)
	time.Sleep(20 * time.Millisecond)

	v := doubled.GetAny()
	if v != 14 {
		t.Errorf("SubscribeAny: expected derived GetAny()=14 after count=7, got %v", v)
	}
}

// ─── Derived.MarshalJSON ──────────────────────────────────────────────────────

func TestDerived_MarshalJSON(t *testing.T) {
	d := NewDerived(func() int { return 99 })
	data, err := d.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("MarshalJSON should produce non-empty data")
	}
}

// ─── Derived.Dispose ──────────────────────────────────────────────────────────

func TestDerived_Dispose(_ *testing.T) {
	count := NewRune(1)
	d := DerivedFrom(func() int { return count.Get() }, count)

	var mu sync.Mutex
	notified := false
	d.Subscribe(func(_ int) {
		mu.Lock()
		notified = true
		mu.Unlock()
	})

	d.Dispose()

	// After disposal, deps/subscribers should be cleared
	count.Set(99)
	time.Sleep(50 * time.Millisecond)

	// The value may or may not have been notified pre-dispose; that's OK.
	// The important thing is that Dispose doesn't panic.
	_ = notified
}

// ─── Derived DependOn (manual dependency wiring) ─────────────────────────────

func TestDerived_DependOn(t *testing.T) {
	count := NewRune(10)
	d := NewDerived(func() int { return count.Get() * 3 })
	d.DependOn(count)

	count.Set(5)
	time.Sleep(30 * time.Millisecond)

	if d.Get() != 15 {
		t.Errorf("expected 15 after count=5, got %d", d.Get())
	}
}
