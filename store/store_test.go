package store

import (
	"bytes"
	"sync"
	"testing"
	"time"
)

// ─── MemoryStorage ────────────────────────────────────────────────────────────

func TestMemoryStorage_SetAndGet(t *testing.T) {
	s := NewMemoryStorage()
	data := []byte("hello")
	if err := s.Set("key1", data, 0); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	got, err := s.Get("key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("expected %q, got %q", data, got)
	}
}

func TestMemoryStorage_GetMissing(t *testing.T) {
	s := NewMemoryStorage()
	_, err := s.Get("missing")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound for missing key, got %v", err)
	}
}

func TestMemoryStorage_Delete(t *testing.T) {
	s := NewMemoryStorage()
	_ = s.Set("key", []byte("val"), 0)
	if err := s.Delete("key"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err := s.Get("key")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMemoryStorage_DeleteNonExistent(t *testing.T) {
	s := NewMemoryStorage()
	// Should not error
	if err := s.Delete("nonexistent"); err != nil {
		t.Errorf("Delete of non-existent key should not error, got %v", err)
	}
}

func TestMemoryStorage_ReturnsCopyOnGet(t *testing.T) {
	s := NewMemoryStorage()
	original := []byte("data")
	_ = s.Set("key", original, 0)

	got, _ := s.Get("key")
	// Mutate the returned slice
	got[0] = 'X'

	// Re-read and verify the stored value is unchanged
	got2, _ := s.Get("key")
	if got2[0] == 'X' {
		t.Error("Get should return a copy, not a reference to internal storage")
	}
}

func TestMemoryStorage_Expiry(t *testing.T) {
	s := NewMemoryStorage()
	_ = s.Set("expiring", []byte("value"), 50*time.Millisecond)

	// Should exist immediately
	got, err := s.Get("expiring")
	if err != nil || !bytes.Equal(got, []byte("value")) {
		t.Error("key should exist before expiry")
	}

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)

	_, err = s.Get("expiring")
	if err != ErrNotFound {
		t.Error("key should be expired and return ErrNotFound")
	}
}

func TestMemoryStorage_NoExpiry(t *testing.T) {
	s := NewMemoryStorage()
	_ = s.Set("persistent", []byte("value"), 0)

	time.Sleep(50 * time.Millisecond)

	_, err := s.Get("persistent")
	if err != nil {
		t.Errorf("key with no expiry should persist, got: %v", err)
	}
}

func TestMemoryStorage_Overwrite(t *testing.T) {
	s := NewMemoryStorage()
	_ = s.Set("key", []byte("first"), 0)
	_ = s.Set("key", []byte("second"), 0)

	got, err := s.Get("key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !bytes.Equal(got, []byte("second")) {
		t.Errorf("expected 'second', got %q", got)
	}
}

func TestMemoryStorage_ConcurrentAccess(t *testing.T) {
	s := NewMemoryStorage()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "concurrent"
			_ = s.Set(key, []byte("value"), 0)
			_, _ = s.Get(key)
			_ = s.Delete(key)
		}(i)
	}
	wg.Wait()
}

func TestMemoryStorage_EmptyValue(t *testing.T) {
	s := NewMemoryStorage()
	_ = s.Set("empty", []byte{}, 0)
	got, err := s.Get("empty")
	if err != nil {
		t.Fatalf("Get for empty value failed: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

func TestMemoryStorage_LargeValue(t *testing.T) {
	s := NewMemoryStorage()
	large := make([]byte, 1024*1024) // 1MB
	for i := range large {
		large[i] = byte(i % 256)
	}
	_ = s.Set("large", large, 0)
	got, err := s.Get("large")
	if err != nil {
		t.Fatalf("Get for large value failed: %v", err)
	}
	if !bytes.Equal(got, large) {
		t.Error("large value round-trip failed")
	}
}

// ─── MemoryPubSub ─────────────────────────────────────────────────────────────

func TestMemoryPubSub_PublishSubscribe(t *testing.T) {
	ps := NewMemoryPubSub()
	received := make(chan []byte, 1)

	if err := ps.Subscribe("channel1", func(msg []byte) {
		received <- msg
	}); err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if err := ps.Publish("channel1", []byte("hello")); err != nil {
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

func TestMemoryPubSub_PublishToUnsubscribedChannel(t *testing.T) {
	ps := NewMemoryPubSub()
	// Should not error on publish to channel with no subscribers
	if err := ps.Publish("empty", []byte("msg")); err != nil {
		t.Errorf("Publish to empty channel should not error, got %v", err)
	}
}

func TestMemoryPubSub_MultipleSubscribers(t *testing.T) {
	ps := NewMemoryPubSub()
	received := make(chan int, 10)

	for i := 0; i < 3; i++ {
		idx := i
		_ = ps.Subscribe("multi", func(msg []byte) {
			received <- idx
		})
	}

	_ = ps.Publish("multi", []byte("broadcast"))

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
	if len(got) != 3 {
		t.Errorf("expected 3 subscribers to receive, got %d", len(got))
	}
}

func TestMemoryPubSub_MultipleChannels(t *testing.T) {
	ps := NewMemoryPubSub()
	ch1 := make(chan []byte, 1)
	ch2 := make(chan []byte, 1)

	_ = ps.Subscribe("ch1", func(msg []byte) { ch1 <- msg })
	_ = ps.Subscribe("ch2", func(msg []byte) { ch2 <- msg })

	_ = ps.Publish("ch1", []byte("msg1"))
	_ = ps.Publish("ch2", []byte("msg2"))

	select {
	case msg := <-ch1:
		if string(msg) != "msg1" {
			t.Errorf("ch1 expected 'msg1', got %q", msg)
		}
	case <-time.After(300 * time.Millisecond):
		t.Error("ch1 timed out")
	}

	select {
	case msg := <-ch2:
		if string(msg) != "msg2" {
			t.Errorf("ch2 expected 'msg2', got %q", msg)
		}
	case <-time.After(300 * time.Millisecond):
		t.Error("ch2 timed out")
	}
}

func TestMemoryPubSub_ConcurrentPublish(t *testing.T) {
	ps := NewMemoryPubSub()
	received := make(chan struct{}, 100)

	_ = ps.Subscribe("concurrent", func(msg []byte) {
		received <- struct{}{}
	})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = ps.Publish("concurrent", []byte("msg"))
		}()
	}
	wg.Wait()

	// Collect up to 20 messages
	count := 0
	timeout := time.After(500 * time.Millisecond)
	for count < 20 {
		select {
		case <-received:
			count++
		case <-timeout:
			goto done
		}
	}
done:
	if count < 20 {
		t.Errorf("expected 20 messages received, got %d", count)
	}
}

func TestMemoryPubSub_PublishDoesNotBlockSubscriber(t *testing.T) {
	ps := NewMemoryPubSub()
	done := make(chan struct{})

	// Subscriber that blocks for a while
	_ = ps.Subscribe("blocking", func(msg []byte) {
		time.Sleep(200 * time.Millisecond)
		close(done)
	})

	// Publish should return quickly (async delivery)
	start := time.Now()
	_ = ps.Publish("blocking", []byte("go"))
	elapsed := time.Since(start)

	if elapsed > 50*time.Millisecond {
		t.Errorf("Publish should not block on subscriber; took %v", elapsed)
	}

	// Wait for subscriber to finish
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Error("subscriber never finished")
	}
}
