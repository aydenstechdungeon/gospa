package state

import (
	"testing"
	"time"
)

func TestStateMapOnChangeDispatchesThroughBoundedQueue(t *testing.T) {
	sm := NewStateMap()
	r := NewRune(0)
	done := make(chan struct{}, 1)

	sm.OnChange = func(key string, value any) {
		if key != "count" {
			t.Fatalf("unexpected key %q", key)
		}
		if value != 1 {
			t.Fatalf("unexpected value %v", value)
		}
		done <- struct{}{}
	}

	sm.Add("count", r)
	r.Set(1)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for OnChange notification")
	}
}
