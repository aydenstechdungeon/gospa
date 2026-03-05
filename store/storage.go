package store

import (
	"errors"
	"sync"
	"time"
)

// ErrNotFound is returned when a key is not found in the storage.
var ErrNotFound = errors.New("key not found")

// Storage represents an external key-value store for session and state data.
type Storage interface {
	Get(key string) ([]byte, error)
	Set(key string, val []byte, exp time.Duration) error
	Delete(key string) error
}

// MemoryStorage provides an in-memory implementation of the Storage interface.
type MemoryStorage struct {
	mu    sync.RWMutex
	store map[string]memoryEntry
	stop  chan struct{}
}

// memoryEntry stores a value and its expiration time.
type memoryEntry struct {
	val []byte
	exp time.Time
}

// NewMemoryStorage creates a new in-memory storage.
func NewMemoryStorage() *MemoryStorage {
	s := &MemoryStorage{
		store: make(map[string]memoryEntry),
		stop:  make(chan struct{}),
	}
	// Start cleanup goroutine
	go s.pruneLoop()
	return s
}

// Get retrieves a value from the in-memory store.
func (s *MemoryStorage) Get(key string) ([]byte, error) {
	s.mu.RLock()
	entry, exists := s.store[key]
	s.mu.RUnlock()

	if !exists {
		return nil, ErrNotFound
	}
	if !entry.exp.IsZero() && time.Now().After(entry.exp) {
		_ = s.Delete(key)
		return nil, ErrNotFound
	}
	// Return a defensive copy to prevent callers from mutating internal storage.
	buf := make([]byte, len(entry.val))
	copy(buf, entry.val)
	return buf, nil
}

// Set stores a value in the in-memory store.
func (s *MemoryStorage) Set(key string, val []byte, exp time.Duration) error {
	s.mu.Lock()
	var expiration time.Time
	if exp > 0 {
		expiration = time.Now().Add(exp)
	}
	s.store[key] = memoryEntry{val: val, exp: expiration}
	s.mu.Unlock()
	return nil
}

// Delete removes a value from the in-memory store.
func (s *MemoryStorage) Delete(key string) error {
	s.mu.Lock()
	delete(s.store, key)
	s.mu.Unlock()
	return nil
}

// pruneLoop removes expired entries periodically.
func (s *MemoryStorage) pruneLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for key, entry := range s.store {
				if !entry.exp.IsZero() && now.After(entry.exp) {
					delete(s.store, key)
				}
			}
			s.mu.Unlock()
		case <-s.stop:
			return // Stop processing when closed
		}
	}
}

// Close explicitly stops the background pruning loop to prevent goroutine leaks.
func (s *MemoryStorage) Close() error {
	// Send close signal
	close(s.stop)
	return nil
}
