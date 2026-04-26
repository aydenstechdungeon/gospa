package gospa

import "strings"

func (a *App) defaultCacheTags(routePath, strategy string) []string {
	normalized := strings.TrimSpace(routePath)
	if normalized == "" {
		normalized = "/"
	}
	strategy = strings.ToLower(strings.TrimSpace(strategy))
	if strategy == "" {
		strategy = "ssr"
	}
	return []string{
		"route:" + normalized,
		"strategy:" + strategy,
	}
}

func (a *App) defaultCacheKeys(routePath string) []string {
	normalized := strings.TrimSpace(routePath)
	if normalized == "" {
		normalized = "/"
	}
	return []string{
		"path:" + normalized,
		normalized,
	}
}

func dependencyTags(depends []string) []string {
	if len(depends) == 0 {
		return nil
	}
	out := make([]string, 0, len(depends))
	for _, dep := range depends {
		trimmed := strings.TrimSpace(dep)
		if trimmed == "" {
			continue
		}
		out = append(out, "dep:"+trimmed)
	}
	return out
}

func dependencyKeys(depends []string) []string {
	if len(depends) == 0 {
		return nil
	}
	out := make([]string, 0, len(depends))
	for _, dep := range depends {
		trimmed := strings.TrimSpace(dep)
		if trimmed == "" {
			continue
		}
		out = append(out, "dep:"+trimmed)
	}
	return out
}

func (a *App) indexCacheEntry(cacheKey string, tags, keys []string) {
	a.cacheIndexMu.Lock()
	defer a.cacheIndexMu.Unlock()

	for _, tag := range tags {
		if tag == "" {
			continue
		}
		if a.cacheTagIndex[tag] == nil {
			a.cacheTagIndex[tag] = make(map[string]struct{})
		}
		a.cacheTagIndex[tag][cacheKey] = struct{}{}
	}
	for _, key := range keys {
		if key == "" {
			continue
		}
		if a.cacheKeyIndex[key] == nil {
			a.cacheKeyIndex[key] = make(map[string]struct{})
		}
		a.cacheKeyIndex[key][cacheKey] = struct{}{}
	}
}

func (a *App) dropCacheIndex(cacheKey string) {
	a.cacheIndexMu.Lock()
	defer a.cacheIndexMu.Unlock()

	for tag, keys := range a.cacheTagIndex {
		delete(keys, cacheKey)
		if len(keys) == 0 {
			delete(a.cacheTagIndex, tag)
		}
	}
	for indexKey, keys := range a.cacheKeyIndex {
		delete(keys, cacheKey)
		if len(keys) == 0 {
			delete(a.cacheKeyIndex, indexKey)
		}
	}
}

func (a *App) collectCacheKeysByTag(tag string) []string {
	a.cacheIndexMu.RLock()
	defer a.cacheIndexMu.RUnlock()

	keys := a.cacheTagIndex[tag]
	if len(keys) == 0 {
		return nil
	}
	out := make([]string, 0, len(keys))
	for key := range keys {
		out = append(out, key)
	}
	return out
}

func (a *App) collectCacheKeysByKey(indexKey string) []string {
	a.cacheIndexMu.RLock()
	defer a.cacheIndexMu.RUnlock()

	keys := a.cacheKeyIndex[indexKey]
	if len(keys) == 0 {
		return nil
	}
	out := make([]string, 0, len(keys))
	for key := range keys {
		out = append(out, key)
	}
	return out
}

// Invalidate removes cache entries associated with the provided route path.
func (a *App) Invalidate(path string) int {
	if path == "" {
		return 0
	}
	return a.invalidateCacheKey(path)
}

// InvalidateTag removes all cache entries indexed under the provided tag.
func (a *App) InvalidateTag(tag string) int {
	if tag == "" {
		return 0
	}
	keys := a.collectCacheKeysByTag(tag)
	count := 0
	for _, key := range keys {
		if a.invalidateCacheKey(key) > 0 {
			count++
		}
	}
	return count
}

// InvalidateKey removes all cache entries indexed under the provided key.
func (a *App) InvalidateKey(key string) int {
	if key == "" {
		return 0
	}
	keys := a.collectCacheKeysByKey(key)
	count := 0
	for _, cacheKey := range keys {
		if a.invalidateCacheKey(cacheKey) > 0 {
			count++
		}
	}
	return count
}

// InvalidateAll removes all in-memory route caches and clears index mappings.
// Returns number of entries removed across SSG and PPR caches.
func (a *App) InvalidateAll() int {
	invalidated := 0

	a.ssgCacheMu.Lock()
	invalidated += len(a.ssgCache)
	a.ssgCache = make(map[string]ssgEntry)
	a.ssgCacheKeys = a.ssgCacheKeys[:0]
	a.ssgCacheIndex = make(map[string]struct{})
	a.ssgCacheMu.Unlock()

	a.pprShellMu.Lock()
	invalidated += len(a.pprShellCache)
	a.pprShellCache = make(map[string]pprEntry)
	a.pprShellKeys = a.pprShellKeys[:0]
	a.pprShellIndex = make(map[string]struct{})
	a.pprShellMu.Unlock()

	a.cacheIndexMu.Lock()
	a.cacheTagIndex = make(map[string]map[string]struct{})
	a.cacheKeyIndex = make(map[string]map[string]struct{})
	a.cacheIndexMu.Unlock()

	return invalidated
}

func (a *App) invalidateCacheKey(cacheKey string) int {
	invalidated := 0

	a.ssgCacheMu.Lock()
	if _, ok := a.ssgCache[cacheKey]; ok {
		delete(a.ssgCache, cacheKey)
		delete(a.ssgCacheIndex, cacheKey)
		for i, k := range a.ssgCacheKeys {
			if k == cacheKey {
				a.ssgCacheKeys = append(a.ssgCacheKeys[:i], a.ssgCacheKeys[i+1:]...)
				break
			}
		}
		invalidated++
	}
	a.ssgCacheMu.Unlock()

	a.pprShellMu.Lock()
	if _, ok := a.pprShellCache[cacheKey]; ok {
		delete(a.pprShellCache, cacheKey)
		delete(a.pprShellIndex, cacheKey)
		for i, k := range a.pprShellKeys {
			if k == cacheKey {
				a.pprShellKeys = append(a.pprShellKeys[:i], a.pprShellKeys[i+1:]...)
				break
			}
		}
		invalidated++
	}
	a.pprShellMu.Unlock()

	if a.Config.Storage != nil {
		_ = a.Config.Storage.Delete(a.Context(), "gospa:ssg:"+cacheKey)
		_ = a.Config.Storage.Delete(a.Context(), "gospa:ppr:"+cacheKey)
	}

	if invalidated > 0 {
		a.recordCacheInvalidation(cacheKey)
		a.dropCacheIndex(cacheKey)
	}
	return invalidated
}
