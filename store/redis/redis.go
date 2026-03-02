package redis

import (
	"context"
	"time"

	"github.com/aydenstechdungeon/gospa/store"
	goredis "github.com/redis/go-redis/v9"
)

// Store provides a Redis-backed implementation of the store.Storage interface.
type Store struct {
	client *goredis.Client
	ctx    context.Context
}

// NewStore creates a new Redis storage.
func NewStore(client *goredis.Client) *Store {
	return &Store{
		client: client,
		ctx:    context.Background(), // Can be injected externally or derived
	}
}

// Get retrieves a key from Redis.
func (s *Store) Get(key string) ([]byte, error) {
	val, err := s.client.Get(s.ctx, key).Bytes()
	if err == goredis.Nil {
		return nil, store.ErrNotFound
	}
	return val, err
}

// Set stores a key in Redis with an optional expiration time.
func (s *Store) Set(key string, val []byte, exp time.Duration) error {
	return s.client.Set(s.ctx, key, val, exp).Err()
}

// Delete removes a key from Redis.
func (s *Store) Delete(key string) error {
	return s.client.Del(s.ctx, key).Err()
}

// PubSub provides a Redis-backed implementation of the store.PubSub interface.
type PubSub struct {
	client *goredis.Client
	ctx    context.Context
}

// NewPubSub creates a new Redis PubSub.
func NewPubSub(client *goredis.Client) *PubSub {
	return &PubSub{
		client: client,
		ctx:    context.Background(),
	}
}

// Publish publishes a message to a Redis channel.
func (p *PubSub) Publish(channel string, message []byte) error {
	return p.client.Publish(p.ctx, channel, message).Err()
}

// Subscribe subscribes to a Redis channel and invokes the handler for each message.
func (p *PubSub) Subscribe(channel string, handler func(message []byte)) error {
	pubsub := p.client.Subscribe(p.ctx, channel)

	// Wait for confirmation that subscription is created
	_, err := pubsub.Receive(p.ctx)
	if err != nil {
		return err
	}

	go func() {
		defer func() { _ = pubsub.Close() }()
		ch := pubsub.Channel()
		for msg := range ch {
			handler([]byte(msg.Payload))
		}
	}()

	return nil
}
