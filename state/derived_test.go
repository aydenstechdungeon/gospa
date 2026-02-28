package state

import (
	"testing"
	"time"
)

func TestNewDerived(t *testing.T) {
	count := NewRune(5)
	doubled := NewDerived(func() int {
		return count.Get() * 2
	})

	// Initial computation
	if doubled.Get() != 10 {
		t.Errorf("Expected initial derived value 10, got %d", doubled.Get())
	}
}

func TestDerivedRecomputation(t *testing.T) {
	count := NewRune(5)
	doubled := NewDerived(func() int {
		return count.Get() * 2
	})

	// Set up dependency tracking
	doubled.DependOn(count)

	if doubled.Get() != 10 {
		t.Errorf("Expected initial value 10, got %d", doubled.Get())
	}

	count.Set(10)
	time.Sleep(100 * time.Millisecond) // Wait for async notification

	if doubled.Get() != 20 {
		t.Errorf("Expected recomputed value 20, got %d", doubled.Get())
	}
}

func TestDerivedSubscribe(t *testing.T) {
	count := NewRune(5)
	doubled := NewDerived(func() int {
		return count.Get() * 2
	})
	doubled.DependOn(count)

	var received []int
	unsub := doubled.Subscribe(func(v int) {
		received = append(received, v)
	})
	defer unsub()

	count.Set(10)
	time.Sleep(100 * time.Millisecond)

	if len(received) != 1 || received[0] != 20 {
		t.Errorf("Expected [20], got %v", received)
	}
}

func TestDerivedNoRecomputeOnSameValue(t *testing.T) {
	count := NewRune(5)
	callCount := 0

	doubled := NewDerived(func() int {
		callCount++
		return count.Get() * 2
	})
	doubled.DependOn(count)

	// Initial computation
	_ = doubled.Get()
	if callCount != 1 {
		t.Errorf("Expected 1 initial computation, got %d", callCount)
	}

	// Get again without change
	_ = doubled.Get()
	if callCount != 1 {
		t.Errorf("Expected no recomputation, got %d calls", callCount)
	}

	// Change and get
	count.Set(10)
	time.Sleep(100 * time.Millisecond)
	_ = doubled.Get()
	if callCount != 2 {
		t.Errorf("Expected 2 computations after change, got %d", callCount)
	}
}

func TestDerivedFrom(t *testing.T) {
	count := NewRune(5)
	doubled := DerivedFrom(func() int {
		return count.Get() * 2
	}, count)

	if doubled.Get() != 10 {
		t.Errorf("Expected value 10, got %d", doubled.Get())
	}

	count.Set(15)
	time.Sleep(100 * time.Millisecond)

	if doubled.Get() != 30 {
		t.Errorf("Expected recomputed value 30, got %d", doubled.Get())
	}
}

func TestDerived2(t *testing.T) {
	a := NewRune(3)
	b := NewRune(4)
	sum := Derived2(a, b, func(x, y int) int {
		return x + y
	})

	if sum.Get() != 7 {
		t.Errorf("Expected sum 7, got %d", sum.Get())
	}

	a.Set(10)
	time.Sleep(100 * time.Millisecond)

	if sum.Get() != 14 {
		t.Errorf("Expected sum 14, got %d", sum.Get())
	}
}

func TestDerived3(t *testing.T) {
	a := NewRune(1)
	b := NewRune(2)
	c := NewRune(3)

	sum := Derived3(a, b, c, func(x, y, z int) int {
		return x + y + z
	})

	if sum.Get() != 6 {
		t.Errorf("Expected sum 6, got %d", sum.Get())
	}
}

func TestDerivedID(t *testing.T) {
	d1 := NewDerived(func() int { return 1 })
	d2 := NewDerived(func() int { return 2 })

	if d1.ID() == d2.ID() {
		t.Error("Expected different IDs for different derived values")
	}

	if d1.ID() == "" {
		t.Error("Expected non-empty ID")
	}
}

func TestDerivedMarshalJSON(t *testing.T) {
	d := NewDerived(func() int { return 42 })
	data, err := d.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	expected := `{"id":"` + d.ID() + `","value":42}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

func TestDerivedGetAny(t *testing.T) {
	d := NewDerived(func() int { return 42 })
	v := d.GetAny()
	if v != 42 {
		t.Errorf("Expected 42 from GetAny, got %v", v)
	}
}

func TestDerivedDispose(t *testing.T) {
	count := NewRune(5)
	doubled := NewDerived(func() int {
		return count.Get() * 2
	})
	doubled.DependOn(count)

	doubled.Dispose()

	// After dispose, should still return last computed value
	if doubled.Get() != 10 {
		t.Errorf("Expected value 10 after dispose, got %d", doubled.Get())
	}
}

func TestDerivedSubscribeAny(t *testing.T) {
	count := NewRune(5)
	doubled := NewDerived(func() int {
		return count.Get() * 2
	})
	doubled.DependOn(count)

	var received []any
	unsub := doubled.SubscribeAny(func(v any) {
		received = append(received, v)
	})
	defer unsub()

	count.Set(10)
	time.Sleep(100 * time.Millisecond)

	if len(received) != 1 || received[0] != 20 {
		t.Errorf("Expected [20], got %v", received)
	}
}
