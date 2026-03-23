package gospa

import (
	"context"
	"time"

	"github.com/aydenstechdungeon/gospa/routing"
)

// isrSemaphore limits concurrent ISR background revalidations.
var isrSemaphore chan struct{}

// initSemaphore initializes the ISR semaphore if not already done.
func (a *App) initSemaphore() {
	if isrSemaphore == nil {
		limit := a.Config.ISRSemaphoreLimit
		if limit <= 0 {
			limit = 10
		}
		isrSemaphore = make(chan struct{}, limit)
	}
}

func (a *App) backgroundRevalidate(cacheKey string, routeSnap interface{}) {
	route := routeSnap.(*routing.Route)
	defer a.isrRevalidating.Delete(cacheKey)
	select {
	case isrSemaphore <- struct{}{}:
		defer func() { <-isrSemaphore }()
	default:
		return
	}
	timeout := a.Config.ISRTimeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	bgCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	freshHTML, err := a.buildPageHTML(bgCtx, route, nil)
	if err != nil {
		a.Logger().Error("ISR background render error", "path", cacheKey, "err", err)
		return
	}
	a.storeSsgEntry(cacheKey, freshHTML)
}
