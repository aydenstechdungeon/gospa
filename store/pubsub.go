// Package store provides pubsub state backends for GoSPA.
package store

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// PubSub represents an external publish-subscribe mechanism for multi-process broadcasting.
type PubSub interface {
	Publish(ctx context.Context, channel string, message []byte) error
	Subscribe(ctx context.Context, channel string, handler func(message []byte)) (Unsubscribe, error)
}

// Unsubscribe is a function to cancel a subscription.
type Unsubscribe func()

// subscriber holds a handler and a unique ID for identification.
type subscriber struct {
	id      uint64
	handler func(message []byte)
}

// MemoryPubSub provides an in-memory implementation of the PubSub interface.
// It is intended for single-process environments where external infrastructure is not needed.
type MemoryPubSub struct {
	subscribers map[string][]subscriber
	mu          sync.RWMutex
	nextID      uint64
}

// NewMemoryPubSub creates a new in-memory PubSub system.
func NewMemoryPubSub() *MemoryPubSub {
	return &MemoryPubSub{
		subscribers: make(map[string][]subscriber),
	}
}

// Publish sends a message to all subscribers of a channel.
func (p *MemoryPubSub) Publish(_ context.Context, channel string, message []byte) error {
	p.mu.RLock()
	handlers, ok := p.subscribers[channel]
	p.mu.RUnlock()

	if !ok || len(handlers) == 0 {
		return nil
	}

	// Message copy to prevent race conditions or mutation by handlers.
	msgCopy := make([]byte, len(message))
	copy(msgCopy, message)

	// Dispatch to handlers asynchronously to avoid blocking the publisher.
	for _, sub := range handlers {
		go func(h func(message []byte)) {
			defer func() {
				// Guard against consumer panics in the memory backend
				if r := recover(); r != nil {
					// Use a logger if available, otherwise just drop the panic
					fmt.Printf("MemoryPubSub: consumer panicked: %v\n", r)
				}
			}()
			h(msgCopy)
		}(sub.handler)
	}

	return nil
}

// Subscribe registers a handler and returns an Unsubscribe function.
func (p *MemoryPubSub) Subscribe(_ context.Context, channel string, handler func(message []byte)) (Unsubscribe, error) {
	id := atomic.AddUint64(&p.nextID, 1)
	sub := subscriber{id: id, handler: handler}

	p.mu.Lock()
	p.subscribers[channel] = append(p.subscribers[channel], sub)
	p.mu.Unlock()

	return func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		handlers := p.subscribers[channel]
		for i, s := range handlers {
			if s.id == id {
				p.subscribers[channel] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
	}, nil
}
