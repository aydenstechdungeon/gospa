// Package redis provides a Redis-backed implementation of the store.Storage interface.
package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aydenstechdungeon/gospa/store"
	goredis "github.com/redis/go-redis/v9"
)

// Store provides a Redis-backed implementation of the store.Storage interface.
type Store struct {
	client *goredis.Client
}

// NewStore creates a new Redis storage.
func NewStore(client *goredis.Client) *Store {
	return &Store{
		client: client,
	}
}

// Get retrieves a key from Redis.
func (s *Store) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := s.client.Get(ctx, key).Bytes()
	if err == goredis.Nil {
		return nil, store.ErrNotFound
	}
	return val, err
}

// Set stores a key in Redis with an optional expiration time.
func (s *Store) Set(ctx context.Context, key string, val []byte, exp time.Duration) error {
	return s.client.Set(ctx, key, val, exp).Err()
}

// Delete removes a key from Redis.
func (s *Store) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}

// PubSub provides a Redis-backed implementation of the store.PubSub interface.
type PubSub struct {
	client *goredis.Client
}

// NewPubSub creates a new Redis PubSub.
func NewPubSub(client *goredis.Client) *PubSub {
	return &PubSub{
		client: client,
	}
}

// Publish publishes a message to a Redis channel.
func (p *PubSub) Publish(ctx context.Context, channel string, message []byte) error {
	return p.client.Publish(ctx, channel, message).Err()
}

// Subscribe subscribes to a Redis channel and invokes the handler for each message.
// Returns an unsubscribe function to stop the subscription.
func (p *PubSub) Subscribe(ctx context.Context, channel string, handler func(message []byte)) (store.Unsubscribe, error) {
	// If the parent context is canceled, the subscription will naturally end
	subCtx, cancel := context.WithCancel(ctx)
	
	err := p.SubscribeWithContext(subCtx, channel, handler)
	if err != nil {
		cancel()
		return nil, err
	}
	
	return store.Unsubscribe(cancel), nil
}

// SubscribeWithContext subscribes to a Redis channel and automatically unsubscribes
// when the provided context is canceled.
func (p *PubSub) SubscribeWithContext(ctx context.Context, channel string, handler func(message []byte)) error {
	pubsub := p.client.Subscribe(ctx, channel)

	// Wait for confirmation that subscription is created
	_, err := pubsub.Receive(ctx)
	if err != nil {
		_ = pubsub.Close()
		return err
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Guard against consumer panics from crashing the whole process
				fmt.Printf("Redis PubSub: consumer panicked: %v\n", r)
			}
			_ = pubsub.Close()
		}()
		
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				handler([]byte(msg.Payload))
			}
		}
	}()

	return nil
}

var consumeRateLimitTokenScript = goredis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local max_tokens = tonumber(ARGV[2])
local refill_rate = tonumber(ARGV[3])
local ttl_ms = tonumber(ARGV[4])

local data = redis.call("HMGET", key, "tokens", "last_refill")
local tokens = tonumber(data[1])
local last_refill = tonumber(data[2])

if not tokens or not last_refill then
  tokens = max_tokens
  last_refill = now
end

local elapsed = math.max(0, (now - last_refill) / 1000.0)
tokens = math.min(max_tokens, tokens + (elapsed * refill_rate))
local allowed = 0
if tokens >= 1.0 then
  tokens = tokens - 1.0
  allowed = 1
end

redis.call("HSET", key, "tokens", tokens, "last_refill", now)
redis.call("PEXPIRE", key, ttl_ms)
return allowed
`)

// ConsumeRateLimitToken atomically consumes a token for distributed rate limiting.
func (s *Store) ConsumeRateLimitToken(ctx context.Context, key string, now time.Time, maxTokens float64, refillRate float64, ttl time.Duration) (bool, error) {
	result, err := consumeRateLimitTokenScript.Run(
		ctx,
		s.client,
		[]string{key},
		strconv.FormatInt(now.UnixMilli(), 10),
		strconv.FormatFloat(maxTokens, 'f', -1, 64),
		strconv.FormatFloat(refillRate, 'f', -1, 64),
		strconv.FormatInt(ttl.Milliseconds(), 10),
	).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}
