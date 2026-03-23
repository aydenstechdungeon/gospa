package gospa

import (
	"time"
)

func (a *App) storeSsgEntry(key string, html []byte) {
	if a.Config.Storage != nil && !a.Config.Prefork {
		entry := ssgEntry{html: html, createdAt: time.Now()}
		_ = a.Config.Storage.Set("gospa:ssg:"+key, encodeSsgEntry(entry), 0)
		return
	}

	a.ssgCacheMu.Lock()
	defer a.ssgCacheMu.Unlock()

	maxEntries := a.Config.SSGCacheMaxEntries
	if maxEntries <= 0 {
		maxEntries = 500
	}

	if len(a.ssgCache) >= maxEntries && len(a.ssgCacheKeys) > 0 {
		// Convert map for generic helper or just inline here to avoid interface{} complexity
		evictCount := maxEntries / 10
		if evictCount < 1 {
			evictCount = 1
		}
		for i := 0; i < evictCount && i < len(a.ssgCacheKeys); i++ {
			delete(a.ssgCache, a.ssgCacheKeys[i])
		}
		a.ssgCacheKeys = append([]string(nil), a.ssgCacheKeys[evictCount:]...)
	}

	for i, k := range a.ssgCacheKeys {
		if k == key {
			a.ssgCacheKeys = append(a.ssgCacheKeys[:i], a.ssgCacheKeys[i+1:]...)
			break
		}
	}
	a.ssgCacheKeys = append(a.ssgCacheKeys, key)
	a.ssgCache[key] = ssgEntry{html: html, createdAt: time.Now()}
}
