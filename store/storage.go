package store

import (
	"container/list"
	"context"
	"errors"
	"sync"
	"time"
)

// ErrNotFound is returned when a key is not found in the storage.
var ErrNotFound = errors.New("key not found")

// Storage represents an external key-value store for session and state data.
type Storage interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, val []byte, exp time.Duration) error
	Delete(ctx context.Context, key string) error
}

// MemoryStorage provides an in-memory implementation of the Storage interface.
type MemoryStorage struct {
	mu         sync.RWMutex
	store      map[string]memoryEntry
	stop       chan struct{}
	once       sync.Once
	maxEntries int // Max entries for zero-TTL keys to prevent unbounded growth

	// Optimization for zero-TTL entries (LRU-ish eviction)
	zeroTTLCount int
	lru          *list.List               // Tracks zero-TTL key order for eviction
	lruElements  map[string]*list.Element // key -> list element
}

// memoryEntry stores a value and its expiration time.
type memoryEntry struct {
	val []byte
	exp time.Time
}

// NewMemoryStorage creates a new in-memory storage.
func NewMemoryStorage() *MemoryStorage {
	s := &MemoryStorage{
		store:       make(map[string]memoryEntry),
		stop:        make(chan struct{}),
		maxEntries:  10000,
		lru:         list.New(),
		lruElements: make(map[string]*list.Element),
	}
	// Start cleanup goroutine
	go s.pruneLoop()
	return s
}

// Get retrieves a value from the in-memory store.
func (s *MemoryStorage) Get(_ context.Context, key string) ([]byte, error) {
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

	// If it's a zero-TTL key, move it to the front of LRU (most recently used)
	if entry.exp.IsZero() {
		s.mu.Lock()
		if el, ok := s.lruElements[key]; ok {
			s.lru.MoveToFront(el)
		}
		s.mu.Unlock()
	}

	// Return a defensive copy to prevent callers from mutating internal storage.
	buf := make([]byte, len(entry.val))
	copy(buf, entry.val)
	return buf, nil
}

// Set stores a value in the in-memory store.
func (s *MemoryStorage) Set(_ context.Context, key string, val []byte, exp time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var expiration time.Time
	if exp > 0 {
		expiration = time.Now().Add(exp)
		// If it was previously a zero-TTL key, remove from LRU
		if old, exists := s.store[key]; exists && old.exp.IsZero() {
			s.removeFromLRU(key)
		}
	} else {
		// Zero-TTL entry
		if old, exists := s.store[key]; exists && old.exp.IsZero() {
			// Update existing zero-TTL
			s.lru.MoveToFront(s.lruElements[key])
		} else {
			// New or converting to zero-TTL
			if (!exists || !old.exp.IsZero()) && s.maxEntries > 0 && s.zeroTTLCount >= s.maxEntries {
				// Evict oldest zero-exp entry (O(1))
				s.evictOldestZeroTTL()
			}
			s.zeroTTLCount++
			el := s.lru.PushFront(key)
			s.lruElements[key] = el
		}
	}

	s.store[key] = memoryEntry{val: val, exp: expiration}
	return nil
}

// Delete removes a value from the in-memory store.
func (s *MemoryStorage) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, exists := s.store[key]; exists {
		if entry.exp.IsZero() {
			s.removeFromLRU(key)
		}
		delete(s.store, key)
	}
	return nil
}

func (s *MemoryStorage) removeFromLRU(key string) {
	if el, ok := s.lruElements[key]; ok {
		s.lru.Remove(el)
		delete(s.lruElements, key)
		s.zeroTTLCount--
	}
}

func (s *MemoryStorage) evictOldestZeroTTL() {
	el := s.lru.Back()
	if el != nil {
		key := el.Value.(string)
		s.lru.Remove(el)
		delete(s.lruElements, key)
		delete(s.store, key)
		s.zeroTTLCount--
	}
}

// pruneLoop removes expired entries periodically.
func (s *MemoryStorage) pruneLoop() {
	ticker := time.NewTicker(2 * time.Minute)
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
