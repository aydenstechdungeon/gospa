package gospa

import (
	"context"
	"time"

	"github.com/aydenstechdungeon/gospa/routing"
)

// initSemaphore initializes the ISR semaphore if not already done.
func (a *App) initSemaphore() {
	a.isrSemOnce.Do(func() {
		limit := a.Config.ISRSemaphoreLimit
		if limit <= 0 {
			limit = 10
		}
		a.isrSemaphore = make(chan struct{}, limit)
	})
}

func (a *App) backgroundRevalidate(cacheKey string, routeSnap interface{}) {
	route, ok := routeSnap.(*routing.Route)
	if !ok {
		a.Logger().Error("ISR: invalid route snapshot type", "path", cacheKey)
		return
	}
	defer a.isrRevalidating.Delete(cacheKey)
	select {
	case a.isrSemaphore <- struct{}{}:
		defer func() { <-a.isrSemaphore }()
	default:
		return
	}
	timeout := a.Config.ISRTimeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	bgCtx, cancel := context.WithTimeout(a.Context(), timeout)
	defer cancel()
	freshHTML, err := a.buildPageHTML(bgCtx, route, nil)
	if err != nil {
		a.Logger().Error("ISR background render error", "path", cacheKey, "err", err)
		return
	}
	a.storeSsgEntry(cacheKey, freshHTML)
}
