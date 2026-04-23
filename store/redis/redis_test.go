package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/aydenstechdungeon/gospa/store"
	goredis "github.com/redis/go-redis/v9"
)

func newTestRedis(t *testing.T) (*miniredis.Miniredis, *goredis.Client) {
	t.Helper()
	srv, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	client := goredis.NewClient(&goredis.Options{Addr: srv.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		srv.Close()
	})
	return srv, client
}

func TestStoreGetSetDelete(t *testing.T) {
	_, client := newTestRedis(t)
	s := NewStore(client)
	ctx := context.Background()

	if err := s.Set(ctx, "k1", []byte("v1"), 0); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, err := s.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(got) != "v1" {
		t.Fatalf("unexpected value: %q", got)
	}

	if err := s.Delete(ctx, "k1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = s.Get(ctx, "k1")
	if err != store.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got: %v", err)
	}
}

func TestPubSubPublishSubscribe(t *testing.T) {
	_, client := newTestRedis(t)
	p := NewPubSub(client)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	received := make(chan string, 1)
	unsub, err := p.Subscribe(ctx, "topic", func(message []byte) {
		received <- string(message)
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer unsub()

	if err := p.Publish(ctx, "topic", []byte("hello")); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	select {
	case msg := <-received:
		if msg != "hello" {
			t.Fatalf("unexpected message: %q", msg)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for pubsub message")
	}
}

func TestConsumeRateLimitToken(t *testing.T) {
	_, client := newTestRedis(t)
	s := NewStore(client)
	ctx := context.Background()

	key := "ratelimit:user-1"
	maxTokens := 1.0
	refillRate := 1.0 // 1 token / second
	ttl := 10 * time.Second

	now := time.Now()
	allowed, err := s.ConsumeRateLimitToken(ctx, key, now, maxTokens, refillRate, ttl)
	if err != nil {
		t.Fatalf("first ConsumeRateLimitToken failed: %v", err)
	}
	if !allowed {
		t.Fatal("expected first request to be allowed")
	}

	allowed, err = s.ConsumeRateLimitToken(ctx, key, now, maxTokens, refillRate, ttl)
	if err != nil {
		t.Fatalf("second ConsumeRateLimitToken failed: %v", err)
	}
	if allowed {
		t.Fatal("expected second immediate request to be denied")
	}

	allowed, err = s.ConsumeRateLimitToken(ctx, key, now.Add(2*time.Second), maxTokens, refillRate, ttl)
	if err != nil {
		t.Fatalf("third ConsumeRateLimitToken failed: %v", err)
	}
	if !allowed {
		t.Fatal("expected token to refill after enough elapsed time")
	}
}
