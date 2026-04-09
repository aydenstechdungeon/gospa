package store

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"
)

// ─── MemoryStorage ────────────────────────────────────────────────────────────

func TestMemoryStorage_SetAndGet(t *testing.T) {
	s := NewMemoryStorage()
	data := []byte("hello")
	ctx := context.Background()
	if err := s.Set(ctx, "key1", data, 0); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	got, err := s.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("expected %q, got %q", data, got)
	}
}

func TestMemoryStorage_GetMissing(t *testing.T) {
	s := NewMemoryStorage()
	_, err := s.Get(context.Background(), "missing")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound for missing key, got %v", err)
	}
}

func TestMemoryStorage_Delete(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()
	_ = s.Set(ctx, "key", []byte("val"), 0)
	if err := s.Delete(ctx, "key"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err := s.Get(ctx, "key")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMemoryStorage_LRUEviction(t *testing.T) {
	s := NewMemoryStorage()
	s.maxEntries = 2 // Small limit for testing
	ctx := context.Background()

	_ = s.Set(ctx, "k1", []byte("1"), 0)
	_ = s.Set(ctx, "k2", []byte("2"), 0)

	// Access k1 to make it MRU
	_, _ = s.Get(ctx, "k1")

	// Set k3, should evict k2 (oldest LRU)
	_ = s.Set(ctx, "k3", []byte("3"), 0)

	if _, err := s.Get(ctx, "k2"); err != ErrNotFound {
		t.Error("k2 should have been evicted by k3")
	}
	if _, err := s.Get(ctx, "k1"); err != nil {
		t.Error("k1 should still exist (MRU)")
	}
}

func TestMemoryStorage_ReturnsCopyOnGet(t *testing.T) {
	s := NewMemoryStorage()
	original := []byte("data")
	ctx := context.Background()
	_ = s.Set(ctx, "key", original, 0)

	got, _ := s.Get(ctx, "key")
	// Mutate the returned slice
	got[0] = 'X'

	// Re-read and verify the stored value is unchanged
	got2, _ := s.Get(ctx, "key")
	if got2[0] == 'X' {
		t.Error("Get should return a copy, not a reference to internal storage")
	}
}

func TestMemoryStorage_Expiry(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()
	_ = s.Set(ctx, "expiring", []byte("value"), 50*time.Millisecond)

	// Should exist immediately
	got, err := s.Get(ctx, "expiring")
	if err != nil || !bytes.Equal(got, []byte("value")) {
		t.Error("key should exist before expiry")
	}

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)

	_, err = s.Get(ctx, "expiring")
	if err != ErrNotFound {
		t.Error("key should be expired and return ErrNotFound")
	}
}

func TestMemoryStorage_Overwrite(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()
	_ = s.Set(ctx, "key", []byte("first"), 0)
	_ = s.Set(ctx, "key", []byte("second"), 0)

	got, err := s.Get(ctx, "key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !bytes.Equal(got, []byte("second")) {
		t.Errorf("expected 'second', got %q", got)
	}
}

func TestMemoryStorage_ConcurrentAccess(_ *testing.T) {
	s := NewMemoryStorage()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			ctx := context.Background()
			key := "concurrent"
			_ = s.Set(ctx, key, []byte("value"), 0)
			_, _ = s.Get(ctx, key)
			_ = s.Delete(ctx, key)
		}(i)
	}
	wg.Wait()
}

// ─── MemoryPubSub ─────────────────────────────────────────────────────────────

func TestMemoryPubSub_PublishSubscribe(t *testing.T) {
	ps := NewMemoryPubSub()
	received := make(chan []byte, 1)
	ctx := context.Background()

	if _, err := ps.Subscribe(ctx, "channel1", func(msg []byte) {
		received <- msg
	}); err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if err := ps.Publish(ctx, "channel1", []byte("hello")); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	select {
	case msg := <-received:
		if !bytes.Equal(msg, []byte("hello")) {
			t.Errorf("expected 'hello', got %q", msg)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("timed out waiting for published message")
	}
}

func TestMemoryPubSub_Unsubscribe(t *testing.T) {
	ps := NewMemoryPubSub()
	received := make(chan []byte, 10)
	ctx := context.Background()

	unsub, err := ps.Subscribe(ctx, "unsub", func(msg []byte) {
		received <- msg
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	_ = ps.Publish(ctx, "unsub", []byte("msg1"))
	unsub()
	_ = ps.Publish(ctx, "unsub", []byte("msg2"))

	time.Sleep(50 * time.Millisecond)
	if len(received) != 1 {
		t.Errorf("expected 1 message received after unsubscription, got %d", len(received))
	}
}

func TestMemoryPubSub_MultipleSubscribers(t *testing.T) {
	ps := NewMemoryPubSub()
	received := make(chan int, 10)

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		idx := i
		_, _ = ps.Subscribe(ctx, "multi", func(_ []byte) {
			received <- idx
		})
	}

	_ = ps.Publish(ctx, "multi", []byte("broadcast"))

	got := make(map[int]bool)
	timeout := time.After(500 * time.Millisecond)
	for len(got) < 3 {
		select {
		case idx := <-received:
			got[idx] = true
		case <-timeout:
			t.Errorf("timed out: only received from %d subscribers", len(got))
			return
		}
	}
}
