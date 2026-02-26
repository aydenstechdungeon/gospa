package store

import (
	"sync"
)

// PubSub represents an external publish-subscribe mechanism for multi-process broadcasting.
type PubSub interface {
	Publish(channel string, message []byte) error
	Subscribe(channel string, handler func(message []byte)) error
}

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

	if !ok {
		return nil // No subscribers to this channel
	}

	// Make a copy of the handlers array to avoid holding the lock during execution
	handlersCopy := make([]func(message []byte), len(handlers))
	copy(handlersCopy, handlers)

	// Send to handlers asynchronously to not block the publisher
	for _, handler := range handlersCopy {
		go handler(message)
	}

	return nil
}

// Subscribe registers a handler function for messages on a channel.
func (p *MemoryPubSub) Subscribe(channel string, handler func(message []byte)) error {
	p.mu.Lock()
	p.subscribers[channel] = append(p.subscribers[channel], handler)
	p.mu.Unlock()
	return nil
}
