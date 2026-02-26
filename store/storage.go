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

type memoryEntry struct {
	val []byte
	exp time.Time
}

// MemoryStorage provides an in-memory implementation of the Storage interface.
type MemoryStorage struct {
	store map[string]memoryEntry
	mu    sync.RWMutex
}

// NewMemoryStorage creates a new in-memory storage.
func NewMemoryStorage() *MemoryStorage {
	s := &MemoryStorage{
		store: make(map[string]memoryEntry),
	}
	go s.pruneLoop()
	return s
}

// Get retrieves a value from the in-memory store.
func (s *MemoryStorage) Get(key string) ([]byte, error) {
	s.mu.RLock()
	entry, ok := s.store[key]
	s.mu.RUnlock()

	if !ok {
		return nil, ErrNotFound
	}

	if !entry.exp.IsZero() && time.Now().After(entry.exp) {
		_ = s.Delete(key)
		return nil, ErrNotFound
	}

	// Return a copy to prevent accidental mutation of stored data
	valCopy := make([]byte, len(entry.val))
	copy(valCopy, entry.val)
	return valCopy, nil
}

// Set stores a value in the in-memory store.
// If exp is > 0, the entry will be removed after the given duration.
func (s *MemoryStorage) Set(key string, val []byte, exp time.Duration) error {
	var expiresAt time.Time
	if exp > 0 {
		expiresAt = time.Now().Add(exp)
	}

	// Store a copy to prevent accidental mutation by caller
	valCopy := make([]byte, len(val))
	copy(valCopy, val)

	s.mu.Lock()
	s.store[key] = memoryEntry{
		val: valCopy,
		exp: expiresAt,
	}
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
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, entry := range s.store {
			if !entry.exp.IsZero() && now.After(entry.exp) {
				delete(s.store, key)
			}
		}
		s.mu.Unlock()
	}
}
