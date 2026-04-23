package gospa

import (
	"time"
)

func (a *App) storeSsgEntry(key string, html []byte, tags, keys []string) {
	if a.Config.Storage != nil && !a.Config.Prefork {
		entry := ssgEntry{html: html, createdAt: time.Now()}
		_ = a.Config.Storage.Set(a.Context(), "gospa:ssg:"+key, encodeSsgEntry(entry), 0)
		a.indexCacheEntry(key, tags, keys)
		return
	}

	a.ssgCacheMu.Lock()
	defer a.ssgCacheMu.Unlock()

	maxEntries := a.Config.SSGCacheMaxEntries
	if maxEntries == 0 {
		maxEntries = 500
	}

	if maxEntries > 0 && len(a.ssgCache) >= maxEntries && len(a.ssgCacheKeys) > 0 {
		evictCount := maxEntries / 10
		if evictCount < 1 {
			evictCount = 1
		}
		// Evict oldest keys using FIFO order.
		for i := 0; i < evictCount && i < len(a.ssgCacheKeys); i++ {
			evictedKey := a.ssgCacheKeys[i]
			delete(a.ssgCache, evictedKey)
			// PERF FIX: O(1) removal from the index map instead of O(n) scan.
			delete(a.ssgCacheIndex, evictedKey)
			a.dropCacheIndex(evictedKey)
		}
		a.ssgCacheKeys = append([]string(nil), a.ssgCacheKeys[evictCount:]...)
	}

	// PERF FIX: Use the O(1) index map to check for existing key instead of
	// iterating the entire ssgCacheKeys slice (previously O(n) on every write).
	if _, alreadyTracked := a.ssgCacheIndex[key]; alreadyTracked {
		// Key exists in index — remove from slice to re-append at tail (LRU update).
		for i, k := range a.ssgCacheKeys {
			if k == key {
				a.ssgCacheKeys = append(a.ssgCacheKeys[:i], a.ssgCacheKeys[i+1:]...)
				break
			}
		}
	}

	a.ssgCacheKeys = append(a.ssgCacheKeys, key)
	a.ssgCacheIndex[key] = struct{}{}
	a.ssgCache[key] = ssgEntry{html: html, createdAt: time.Now()}
	a.indexCacheEntry(key, tags, keys)
}
