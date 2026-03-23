package gospa

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/aydenstechdungeon/gospa/routing"
	templpkg "github.com/aydenstechdungeon/gospa/templ"
)

func (a *App) storePprShell(key string, shell []byte) {
	if a.Config.Storage != nil && !a.Config.Prefork {
		_ = a.Config.Storage.Set("gospa:ppr:"+key, shell, 0)
		return
	}

	a.pprShellMu.Lock()
	defer a.pprShellMu.Unlock()

	maxEntries := a.Config.SSGCacheMaxEntries
	if maxEntries <= 0 {
		maxEntries = 500
	}

	if len(a.pprShellCache) >= maxEntries && len(a.pprShellKeys) > 0 {
		evictCount := maxEntries / 10
		if evictCount < 1 {
			evictCount = 1
		}
		for i := 0; i < evictCount && i < len(a.pprShellKeys); i++ {
			delete(a.pprShellCache, a.pprShellKeys[i])
		}
		a.pprShellKeys = append([]string(nil), a.pprShellKeys[evictCount:]...)
	}

	for i, k := range a.pprShellKeys {
		if k == key {
			a.pprShellKeys = append(a.pprShellKeys[:i], a.pprShellKeys[i+1:]...)
			break
		}
	}
	a.pprShellKeys = append(a.pprShellKeys, key)
	a.pprShellCache[key] = pprEntry{html: shell, createdAt: time.Now()}
}

func (a *App) applyPPRSlots(route *routing.Route, shell []byte, path string, opts routing.RouteOptions) ([]byte, error) {
	_, params := a.Router.Match(path)
	if params == nil {
		params = map[string]string{}
	}

	result := shell
	for _, slotName := range opts.DynamicSlots {
		slotFn := routing.GetSlot(route.Path, slotName)
		if slotFn == nil {
			continue
		}
		slotProps := map[string]interface{}{"path": path}
		for k, v := range params {
			slotProps[k] = v
		}
		var slotBuf bytes.Buffer
		if err := slotFn(slotProps).Render(context.Background(), &slotBuf); err != nil {
			a.Logger().Error("PPR slot render error", "slot", slotName, "err", err)
			continue
		}
		placeholder := []byte(templpkg.SlotPlaceholder(slotName))
		open := []byte(fmt.Sprintf(`<div data-gospa-slot="%s">`, slotName))
		closeTag := []byte(`</div>`)
		replacement := make([]byte, 0, len(open)+slotBuf.Len()+len(closeTag))
		replacement = append(replacement, open...)
		replacement = append(replacement, slotBuf.Bytes()...)
		replacement = append(replacement, closeTag...)
		result = bytes.ReplaceAll(result, placeholder, replacement)
	}
	return result, nil
}
