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
	mu         sync.RWMutex
	store      map[string]memoryEntry
	stop       chan struct{}
	once       sync.Once
	maxEntries int // Max entries for zero-TTL keys to prevent unbounded growth
}

// memoryEntry stores a value and its expiration time.
type memoryEntry struct {
	val []byte
	exp time.Time
}

// NewMemoryStorage creates a new in-memory storage.
func NewMemoryStorage() *MemoryStorage {
	s := &MemoryStorage{
		store:      make(map[string]memoryEntry),
		stop:       make(chan struct{}),
		maxEntries: 10000, // Safety limit for zero-TTL entries
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
		// RACE FIX: Perform check and delete atomically under write lock
		s.mu.Lock()
		entry, exists = s.store[key] // Re-check
		if exists && !entry.exp.IsZero() && time.Now().After(entry.exp) {
			delete(s.store, key)
		}
		s.mu.Unlock()
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
	} else if s.maxEntries > 0 {
		// Zero-TTL entries never expire; evict oldest zero-exp entries if at capacity
		zeroTTLCount := 0
		for _, entry := range s.store {
			if entry.exp.IsZero() {
				zeroTTLCount++
			}
		}
		if zeroTTLCount >= s.maxEntries {
			// Evict the first zero-exp entry found (approximate LRU)
			for k, entry := range s.store {
				if entry.exp.IsZero() {
					delete(s.store, k)
					break
				}
			}
		}
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
	ticker := time.NewTicker(2 * time.Minute) // Increased frequency, decreased work per tick
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.prune()
		case <-s.stop:
			return // Stop processing when closed
		}
	}
}

// prune handles the actual deletion.
func (s *MemoryStorage) prune() {
	s.mu.RLock()
	now := time.Now()
	var expired []string
	i := 0
	const maxScan = 1000 // Only scan up to 1000 entries per tick to prevent blocking
	for key, entry := range s.store {
		if !entry.exp.IsZero() && now.After(entry.exp) {
			expired = append(expired, key)
		}
		i++
		if i >= maxScan {
			break
		}
	}
	s.mu.RUnlock()

	if len(expired) > 0 {
		s.mu.Lock()
		for _, key := range expired {
			delete(s.store, key)
		}
		s.mu.Unlock()
	}
}

// Close explicitly stops the background pruning loop to prevent goroutine leaks.
func (s *MemoryStorage) Close() error {
	s.once.Do(func() {
		close(s.stop)
	})
	return nil
}
