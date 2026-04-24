package gospa

import (
	"strings"
	"time"

	gofiber "github.com/gofiber/fiber/v3"
)

type routeCacheStats struct {
	Hits          int `json:"hits"`
	Misses        int `json:"misses"`
	StaleServed   int `json:"staleServed"`
	Revalidations int `json:"revalidations"`
	Invalidations int `json:"invalidations"`
}

type slotCacheStat struct {
	Renders int `json:"renders"`
	Errors  int `json:"errors"`
}

type cacheStatsSnapshot struct {
	GeneratedAt string                     `json:"generatedAt"`
	Routes      map[string]routeCacheStats `json:"routes"`
	Slots       map[string]slotCacheStat   `json:"slots"`
}

func normalizeCacheStatsPath(path string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		return "/"
	}
	return p
}

func (a *App) ensureRouteCacheStats(path string) *routeCacheStats {
	normalized := normalizeCacheStatsPath(path)
	stats := a.routeCacheStats[normalized]
	if stats == nil {
		stats = &routeCacheStats{}
		a.routeCacheStats[normalized] = stats
	}
	return stats
}

func (a *App) ensureSlotCacheStats(path, slot string) *slotCacheStat {
	key := normalizeCacheStatsPath(path) + "#" + slot
	stats := a.slotCacheStats[key]
	if stats == nil {
		stats = &slotCacheStat{}
		a.slotCacheStats[key] = stats
	}
	return stats
}

func (a *App) recordCacheHit(path string) {
	a.cacheStatsMu.Lock()
	defer a.cacheStatsMu.Unlock()
	a.ensureRouteCacheStats(path).Hits++
}

func (a *App) recordCacheMiss(path string) {
	a.cacheStatsMu.Lock()
	defer a.cacheStatsMu.Unlock()
	a.ensureRouteCacheStats(path).Misses++
}

func (a *App) recordCacheStaleServed(path string) {
	a.cacheStatsMu.Lock()
	defer a.cacheStatsMu.Unlock()
	a.ensureRouteCacheStats(path).StaleServed++
}

func (a *App) recordCacheRevalidation(path string) {
	a.cacheStatsMu.Lock()
	defer a.cacheStatsMu.Unlock()
	a.ensureRouteCacheStats(path).Revalidations++
}

func (a *App) recordCacheInvalidation(path string) {
	a.cacheStatsMu.Lock()
	defer a.cacheStatsMu.Unlock()
	a.ensureRouteCacheStats(path).Invalidations++
}

func (a *App) recordSlotRender(path, slot string, hadError bool) {
	a.cacheStatsMu.Lock()
	defer a.cacheStatsMu.Unlock()
	stats := a.ensureSlotCacheStats(path, slot)
	stats.Renders++
	if hadError {
		stats.Errors++
	}
}

func (a *App) cacheStatsSnapshot() cacheStatsSnapshot {
	a.cacheStatsMu.RLock()
	defer a.cacheStatsMu.RUnlock()
	out := cacheStatsSnapshot{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Routes:      make(map[string]routeCacheStats, len(a.routeCacheStats)),
		Slots:       make(map[string]slotCacheStat, len(a.slotCacheStats)),
	}
	for k, v := range a.routeCacheStats {
		out.Routes[k] = *v
	}
	for k, v := range a.slotCacheStats {
		out.Slots[k] = *v
	}
	return out
}

func (a *App) handleCacheStats(c gofiber.Ctx) error {
	if !a.Config.DevMode {
		return c.SendStatus(gofiber.StatusNotFound)
	}
	return c.JSON(a.cacheStatsSnapshot())
}

func (a *App) handleTransportPoll(c gofiber.Ctx) error {
	return c.JSON(gofiber.Map{
		"messages":  []any{},
		"transport": "polling",
		"ts":        time.Now().UnixMilli(),
	})
}
