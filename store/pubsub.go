// Package store provides pubsub state backends for GoSPA.
package store

import (
	"reflect"
	"sync"
)

// PubSub represents an external publish-subscribe mechanism for multi-process broadcasting.
type PubSub interface {
	Publish(channel string, message []byte) error
	Subscribe(channel string, handler func(message []byte)) (Unsubscribe, error)
}

// Unsubscribe is a function to cancel a subscription.
type Unsubscribe func()

// MemoryPubSub provides an in-memory implementation of the PubSub interface.
// It is intended for single-process environments where external infrastructure is not needed.
type MemoryPubSub struct {
	subscribers map[string][]func(message []byte)
	mu          sync.RWMutex
}

// NewMemoryPubSub creates a new in-memory PubSub system.
func NewMemoryPubSub() *MemoryPubSub {
	return &MemoryPubSub{
		subscribers: make(map[string][]func(message []byte)),
	}
}

// Publish sends a message to all subscribers of a channel.
func (p *MemoryPubSub) Publish(channel string, message []byte) error {
	p.mu.RLock()
	handlers, ok := p.subscribers[channel]
	p.mu.RUnlock()

	if !ok || len(handlers) == 0 {
		return nil
	}

	// SECURITY FIX: Copy the message to prevent race conditions or mutation by handlers.
	msgCopy := make([]byte, len(message))
	copy(msgCopy, message)

	// Make a copy of the handlers array to avoid holding the lock during execution
	handlersCopy := make([]func(message []byte), len(handlers))
	copy(handlersCopy, handlers)

	// Send to handlers asynchronously
	for _, handler := range handlersCopy {
		go handler(msgCopy)
	}

	return nil
}

// Subscribe registers a handler and returns an Unsubscribe function.
func (p *MemoryPubSub) Subscribe(channel string, handler func(message []byte)) (Unsubscribe, error) {
	p.mu.Lock()
	p.subscribers[channel] = append(p.subscribers[channel], handler)
	p.mu.Unlock()

	return func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		handlers := p.subscribers[channel]
		for i, h := range handlers {
			// Comparing function pointers is hacky in Go but works for simple registration.
			// A better way would be using a unique ID for each sub.
			if reflect.ValueOf(h).Pointer() == reflect.ValueOf(handler).Pointer() {
				p.subscribers[channel] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
	}, nil
}
