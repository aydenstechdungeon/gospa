package gospa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"strings"
	"time"

	gospafiber "github.com/aydenstechdungeon/gospa/fiber"
	"github.com/aydenstechdungeon/gospa/routing"
	"github.com/aydenstechdungeon/gospa/state"
	templpkg "github.com/aydenstechdungeon/gospa/templ"
	gofiber "github.com/gofiber/fiber/v3"
)

// renderRoute renders a route with its layout chain.
func (a *App) renderRoute(c gofiber.Ctx, route *routing.Route, routeParams map[string]interface{}) error {
	cacheKey := c.Path()
	ctx := c.Context()
	opts := routing.GetRouteOptions(route.Path)

	effStrategy := opts.Strategy
	if effStrategy == "" {
		effStrategy = a.Config.DefaultRenderStrategy
	}
	if effStrategy == "" {
		effStrategy = routing.StrategySSR
	}

	// 1. SSG Strategy
	if a.Config.CacheTemplates && effStrategy == routing.StrategySSG {
		var entry ssgEntry
		var hit bool
		if a.Config.Storage != nil && !a.Config.Prefork {
			if data, err := a.Config.Storage.Get(c.Context(), "gospa:ssg:"+cacheKey); err == nil {
				entry, hit = decodeSsgEntry(data)
			}
		} else {
			a.ssgCacheMu.RLock()
			entry, hit = a.ssgCache[cacheKey]
			a.ssgCacheMu.RUnlock()
		}

		if hit && a.Config.SSGCacheTTL > 0 && time.Since(entry.createdAt) >= a.Config.SSGCacheTTL {
			hit = false
		}

		if hit {
			c.Set("Content-Type", "text/html")
			c.Set("Cache-Control", "public, max-age=31536000, immutable")

			// SECURITY: Replace nonces in cached HTML with the current request's nonce
			// to ensure CSP remains valid across different requests and sessions.
			currentNonce, _ := c.Locals("gospa.csp_nonce").(string)
			if currentNonce != "" {
				return c.Send(a.replaceNonces(entry.html, currentNonce))
			}

			return c.Send(entry.html)
		}
	}

	// 2. ISR Strategy
	if a.Config.CacheTemplates && effStrategy == routing.StrategyISR {
		a.initSemaphore()
		ttl := opts.RevalidateAfter
		if ttl == 0 {
			ttl = a.Config.DefaultRevalidateAfter
		}
		ttlSec := int(ttl.Seconds())
		if ttlSec <= 0 {
			ttlSec = 1
		}

		var entry ssgEntry
		var hit bool
		if a.Config.Storage != nil && !a.Config.Prefork {
			if data, err := a.Config.Storage.Get(c.Context(), "gospa:ssg:"+cacheKey); err == nil {
				entry, hit = decodeSsgEntry(data)
			}
		} else {
			a.ssgCacheMu.RLock()
			entry, hit = a.ssgCache[cacheKey]
			a.ssgCacheMu.RUnlock()
		}

		if hit && a.Config.SSGCacheTTL > 0 && time.Since(entry.createdAt) >= a.Config.SSGCacheTTL {
			hit = false
		}

		if hit {
			age := time.Since(entry.createdAt)
			if ttl > 0 && age >= ttl {
				if _, alreadyRunning := a.isrRevalidating.LoadOrStore(cacheKey, true); !alreadyRunning {
					go a.backgroundRevalidate(cacheKey, route) // #nosec //nolint:gosec // intentional: background revalidation uses independent context
				}
			}
			c.Set("Content-Type", "text/html")
			c.Set("Cache-Control", fmt.Sprintf("public, s-maxage=%d, stale-while-revalidate=%d", ttlSec, ttlSec))

			// SECURITY: Replace nonces in cached HTML with the current request's nonce.
			currentNonce, _ := c.Locals("gospa.csp_nonce").(string)
			if currentNonce != "" {
				return c.Send(a.replaceNonces(entry.html, currentNonce))
			}

			return c.Send(entry.html)
		}
	}

	// 3. PPR Strategy
	if a.Config.CacheTemplates && effStrategy == routing.StrategyPPR {
		var shell []byte
		var shellHit bool
		if a.Config.Storage != nil && !a.Config.Prefork {
			if data, err := a.Config.Storage.Get(c.Context(), "gospa:ppr:"+cacheKey); err == nil {
				shell = data
				shellHit = true
			}
		} else {
			a.pprShellMu.RLock()
			p, hit := a.pprShellCache[cacheKey]
			if hit && (a.Config.SSGCacheTTL <= 0 || time.Since(p.createdAt) < a.Config.SSGCacheTTL) {
				shell = p.html
				shellHit = true
			}
			a.pprShellMu.RUnlock()
		}

		if shellHit {
			result, err := a.applyPPRSlots(ctx, route, shell, c.Path(), opts)
			if err != nil {
				a.Logger().Error("PPR slot error", "err", err)
			}
			c.Set("Content-Type", "text/html")
			c.Set("Cache-Control", "no-store")

			// SECURITY: Replace nonces in cached shell before applying slots.
			currentNonce, _ := c.Locals("gospa.csp_nonce").(string)
			if currentNonce != "" {
				result = a.replaceNonces(result, currentNonce)
			}

			return c.Send(result)
		}
	}

	layouts := a.Router.ResolveLayoutChain(route)
	if routeParams == nil {
		routeParams = map[string]interface{}{}
	}

	// Resolve data load chain
	loadedProps, err := a.resolveLoadChain(c, route, layouts)
	if err != nil {
		a.Logger().Error("Load error", "err", err)
		return a.renderError(c, gofiber.StatusInternalServerError, err)
	}

	// Merge with route params (route params take precedence for ID fields etc)
	for k, v := range routeParams {
		loadedProps[k] = v
	}

	// 4. Inject Flash messages into the component state
	for k, v := range gospafiber.GetFlashes(c) {
		loadedProps[k] = v
	}
	if nonce, ok := c.Locals("gospa.csp_nonce").(string); ok && nonce != "" {
		ctx = templpkg.WithNonce(ctx, nonce)
	}
	registry := state.NewRegistry()
	ctx = context.WithValue(ctx, state.RegistryContextKey, registry)

	content := a.buildPageContent(route, loadedProps, c.Path())
	content = a.wrapWithLayouts(content, layouts, loadedProps, c.Path())

	c.Set("Content-Type", "text/html")

	tier := a.resolveTier(opts, layouts)
	rootLayoutFunc := routing.GetRootLayout()
	if rootLayoutFunc != nil {
		rootProps := a.buildRootLayoutProps(c, routeParams, tier)
		// Merge loaded props into root props if they don't conflict
		for k, v := range loadedProps {
			if _, ok := rootProps[k]; !ok {
				rootProps[k] = v
			}
		}
		wrappedContent := rootLayoutFunc(content, rootProps)

		if a.Config.CacheTemplates && effStrategy == routing.StrategySSG {
			var buf bytes.Buffer
			if err := wrappedContent.Render(ctx, &buf); err != nil {
				a.Logger().Error("SSG render error", "err", err)
				return a.renderError(c, gofiber.StatusInternalServerError, err)
			}

			htmlBytes := buf.Bytes()
			// Prepare for caching: replace the current nonce with a placeholder.
			if nonce, ok := c.Locals("gospa.csp_nonce").(string); ok && nonce != "" {
				htmlBytes = bytes.ReplaceAll(htmlBytes, []byte(nonce), []byte("__GOSPA_NONCE_PLACEHOLDER__"))
			}

			a.storeSsgEntry(cacheKey, htmlBytes)
			c.Set("Cache-Control", "public, max-age=31536000, immutable")
			return c.Send(buf.Bytes())
		}

		if a.Config.CacheTemplates && effStrategy == routing.StrategyISR {
			ttl := opts.RevalidateAfter
			if ttl == 0 {
				ttl = a.Config.DefaultRevalidateAfter
			}
			ttlSec := int(ttl.Seconds())
			if ttlSec <= 0 {
				ttlSec = 1
			}
			var buf bytes.Buffer
			if err := wrappedContent.Render(ctx, &buf); err != nil {
				a.Logger().Error("ISR render error", "err", err)
				return a.renderError(c, gofiber.StatusInternalServerError, err)
			}

			htmlBytes := buf.Bytes()
			// Prepare for caching: replace the current nonce with a placeholder.
			if nonce, ok := c.Locals("gospa.csp_nonce").(string); ok && nonce != "" {
				htmlBytes = bytes.ReplaceAll(htmlBytes, []byte(nonce), []byte("__GOSPA_NONCE_PLACEHOLDER__"))
			}

			a.storeSsgEntry(cacheKey, htmlBytes)
			c.Set("Cache-Control", fmt.Sprintf("public, s-maxage=%d, stale-while-revalidate=%d", ttlSec, ttlSec))
			return c.Send(buf.Bytes())
		}

		if a.Config.CacheTemplates && effStrategy == routing.StrategyPPR {
			done := make(chan struct{})
			actual, loaded := a.pprShellBuilding.LoadOrStore(cacheKey, done)
			if !loaded {
				defer func() {
					close(done)
					a.pprShellBuilding.Delete(cacheKey)
				}()
				shellCtx := templpkg.WithPPRShellBuild(ctx)
				shellContent := wrappedContent
				if loadingFn := routing.GetLoading(route.Path); loadingFn != nil {
					ld := loadingFn(map[string]interface{}{})
					ld = a.wrapWithLayouts(ld, layouts, loadedProps, c.Path())
					rootProps := a.buildRootLayoutProps(c, loadedProps, tier)
					// Merge loaded props into root props if they don't conflict
					for k, v := range loadedProps {
						if _, ok := rootProps[k]; !ok {
							rootProps[k] = v
						}
					}
					shellContent = rootLayoutFunc(ld, rootProps)
				}

				var shellBuf bytes.Buffer
				if err := shellContent.Render(shellCtx, &shellBuf); err != nil {
					a.Logger().Error("PPR shell render error", "err", err)
					return a.renderError(c, gofiber.StatusInternalServerError, err)
				}

				shellBytes := shellBuf.Bytes()
				// Prepare for caching: replace current nonce with a placeholder.
				if nonce, ok := c.Locals("gospa.csp_nonce").(string); ok && nonce != "" {
					shellBytes = bytes.ReplaceAll(shellBytes, []byte(nonce), []byte("__GOSPA_NONCE_PLACEHOLDER__"))
				}

				a.storePprShell(cacheKey, shellBytes)
				result, err := a.applyPPRSlots(ctx, route, shellBuf.Bytes(), c.Path(), opts)
				if err != nil {
					a.Logger().Error("PPR slot error", "err", err)
					return a.renderError(c, gofiber.StatusInternalServerError, err)
				}
				c.Set("Cache-Control", "no-store")
				return c.Send(result)
			}
			<-actual.(chan struct{})

			var shellHTML []byte
			var shellOk bool
			if a.Config.Storage != nil && !a.Config.Prefork {
				if data, err := a.Config.Storage.Get(c.Context(), "gospa:ppr:"+cacheKey); err == nil {
					shellHTML, shellOk = data, true
				}
			} else {
				a.pprShellMu.RLock()
				p, hit := a.pprShellCache[cacheKey]
				if hit && (a.Config.SSGCacheTTL <= 0 || time.Since(p.createdAt) < a.Config.SSGCacheTTL) {
					shellHTML, shellOk = p.html, true
				}
				a.pprShellMu.RUnlock()
			}
			if shellOk {
				result, err := a.applyPPRSlots(ctx, route, shellHTML, c.Path(), opts)
				if err != nil {
					a.Logger().Error("PPR slot error", "err", err)
					return a.renderError(c, gofiber.StatusInternalServerError, err)
				}
				c.Set("Cache-Control", "no-store")
				return c.Send(result)
			}

			var fallbackBuf bytes.Buffer
			if err := wrappedContent.Render(ctx, &fallbackBuf); err != nil {
				a.Logger().Error("PPR fallback render error", "err", err)
				return a.renderError(c, gofiber.StatusInternalServerError, err)
			}
			c.Set("Cache-Control", "no-store")
			return c.Send(fallbackBuf.Bytes())
		}

		c.Set("Cache-Control", "no-store")
		var buf bytes.Buffer
		if err := wrappedContent.Render(ctx, &buf); err != nil {
			a.Logger().Error("render error", "err", err)
			return a.renderError(c, gofiber.StatusInternalServerError, err)
		}
		return c.Send(buf.Bytes())
	}

	wsURL := a.getWSUrl(c)
	runtimePath := a.getRuntimePath()
	wsRD, wsMR, wsHB := a.normalizeWSConfig()

	c.Set("Cache-Control", "no-store")
	cspNonce, _ := c.Locals("gospa.csp_nonce").(string)
	nonceFmt := ""
	if cspNonce != "" {
		nonceFmt = ` nonce="` + html.EscapeString(cspNonce) + `"`
	}
	var out bytes.Buffer
	_, _ = fmt.Fprint(&out, `<!DOCTYPE html><html lang="en" data-gospa-auto><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>`)
	// SECURITY: Escape AppName to prevent XSS via title injection.
	_, _ = fmt.Fprint(&out, html.EscapeString(a.Config.AppName))
	_, _ = fmt.Fprint(&out, `</title></head><body><div id="app" data-gospa-root><main>`)
	if err := content.Render(ctx, &out); err != nil {
		a.Logger().Error("render error", "err", err)
		return a.renderError(c, gofiber.StatusInternalServerError, err)
	}
	_, _ = fmt.Fprint(&out, `</main></div>`)

	// Determine the highest required runtime tier for this page and all its layouts
	maxTierLevel := tierToLevel(opts.RuntimeTier)
	for _, l := range layouts {
		if lTier := routing.GetLayoutTier(l.Path); lTier != "" {
			if level := tierToLevel(lTier); level > maxTierLevel {
				maxTierLevel = level
			}
		}
	}
	if rootTier := routing.GetLayoutTier(""); rootTier != "" {
		if level := tierToLevel(rootTier); level > maxTierLevel {
			maxTierLevel = level
		}
	}
	tier = levelToTier(maxTierLevel)
	runtimePathForPage := runtimePath
	if tier != "" && tier != "full" && strings.HasPrefix(runtimePath, "/_gospa/runtime.js") {
		runtimePathForPage = "/_gospa/runtime-" + tier + ".js"
	}

	_, _ = fmt.Fprintf(&out, `<script src="%s" type="module"%s></script>`, runtimePathForPage, nonceFmt)
	_, _ = fmt.Fprintf(&out, `<script type="module"%s>
import * as runtime from %s;
window.__GOSPA_CONFIG__ = {
	navigationOptions: %s,
	csrfToken: %s,
};
runtime.init({
	wsUrl: %s,
	serializationFormat: %s,
	debug: %v,
	simpleRuntimeSVGs: %v,
	disableSanitization: %v,
	wsReconnectDelay: %d,
	wsMaxReconnect: %d,
	wsHeartbeat: %d,
	hydration: {
		mode: %s,
		timeout: %d
	}
});
</script>`, nonceFmt, toJS(runtimePathForPage), toJS(a.Config.NavigationOptions), toJS(c.Locals("gospa.csrf_token")), toJS(wsURL), toJS(string(a.Config.SerializationFormat)), a.Config.DevMode, a.Config.SimpleRuntimeSVGs, a.Config.DisableSanitization, wsRD, wsMR, wsHB, toJS(a.Config.HydrationMode), a.Config.HydrationTimeout)

	// Islands bundle — loads and registers all island setup functions
	// Only include if the file exists (islands are optional)
	islandsPath := a.Config.IslandsBundlePath
	if islandsPath == "" {
		islandsPath = "static/js/islands.js"
	}
	if _, err := os.Stat(islandsPath); err == nil {
		_, _ = fmt.Fprintf(&out, `<script src="/%s" type="module"%s></script>`, html.EscapeString(islandsPath), nonceFmt)
	}

	// Centralized State Registry
	data, _ := json.Marshal(registry.GetData())
	_, _ = fmt.Fprintf(&out, `<script id="__GOSPA_DATA__" type="application/json"%s>%s</script>`, nonceFmt, string(data))

	// Handle Deferred Slots
	for _, slotName := range opts.DeferredSlots {
		_, _ = out.WriteString(a.renderDeferredSlotToBuffer(route, slotName, routeParams, c.Path(), nonceFmt))
	}

	_, _ = fmt.Fprint(&out, `</body></html>`)
	return c.Send(out.Bytes())
}

func extractRouteParams(c gofiber.Ctx, route *routing.Route) map[string]interface{} {
	if len(route.Params) == 0 {
		return map[string]interface{}{}
	}
	params := make(map[string]interface{}, len(route.Params))
	for _, key := range route.Params {
		params[key] = c.Params(key)
	}
	return params
}

// renderDeferredSlotToBuffer renders a deferred slot and returns the HTML/script chunk for injection.
func (a *App) renderDeferredSlotToBuffer(route *routing.Route, slotName string, params map[string]interface{}, path string, nonce string) string {
	slotFn := routing.GetSlot(route.Path, slotName)
	if slotFn == nil {
		return ""
	}
	slotProps := map[string]interface{}{"path": path}
	for k, v := range params {
		slotProps[k] = v
	}

	var buf bytes.Buffer
	if err := slotFn(slotProps).Render(context.Background(), &buf); err != nil {
		a.Logger().Error("Deferred slot render error", "slot", slotName, "err", err)
		return ""
	}

	// SECURITY: Use proper JS escaping for slot names and IDs.
	// buf.String() contains templ-rendered content which is already HTML-safe.
	safeSlotName := html.EscapeString(slotName)
	jsSlotName := toJS(slotName)
	var chunk bytes.Buffer
	_, _ = fmt.Fprintf(&chunk, `<template id="gospa-deferred-content-%s">%s</template>`, safeSlotName, buf.String())
	_, _ = fmt.Fprintf(&chunk, `<script%s>if(window.__GOSPA_STREAM__){__GOSPA_STREAM__({type:'html', id:'gospa-deferred-'+%s, content: document.getElementById('gospa-deferred-content-'+%s).innerHTML})}</script>`, nonce, jsSlotName, jsSlotName)
	return chunk.String()
}

// resolveLoadChain executes the load functions for a route and its layout chain.
func (a *App) resolveLoadChain(c gofiber.Ctx, route *routing.Route, layouts []*routing.Route) (map[string]interface{}, error) {
	props := make(map[string]interface{})
	lc := &fiberLoadContext{c: c}

	// 1. Root Layout Loader
	if loader := routing.GetLayoutLoad(""); loader != nil {
		data, err := loader(lc)
		if err != nil {
			return nil, err
		}
		for k, v := range data {
			props[k] = v
		}
	}

	// 2. Nested Layout Loaders
	for _, layout := range layouts {
		if loader := routing.GetLayoutLoad(layout.Path); loader != nil {
			data, err := loader(lc)
			if err != nil {
				return nil, err
			}
			for k, v := range data {
				props[k] = v
			}
		}
	}

	// 3. Page Loader
	if loader := routing.GetLoad(route.Path); loader != nil {
		data, err := loader(lc)
		if err != nil {
			return nil, err
		}
		for k, v := range data {
			props[k] = v
		}
	}

	return props, nil
}

func (a *App) resolveTier(opts routing.RouteOptions, layouts []*routing.Route) string {
	maxLevel := tierToLevel(string(a.Config.RuntimeTier))
	if pLevel := tierToLevel(opts.RuntimeTier); pLevel > maxLevel {
		maxLevel = pLevel
	}
	for _, l := range layouts {
		if lTier := routing.GetLayoutTier(l.Path); lTier != "" {
			if level := tierToLevel(lTier); level > maxLevel {
				maxLevel = level
			}
		}
	}
	if rootTier := routing.GetLayoutTier(""); rootTier != "" {
		if level := tierToLevel(rootTier); level > maxLevel {
			maxLevel = level
		}
	}
	return levelToTier(maxLevel)
}

// tierToLevel converts a runtime tier string to a numeric level for comparison.
func tierToLevel(tier string) int {
	switch strings.ToLower(tier) {
	case "full":
		return 3
	case "core":
		return 2
	case "micro":
		return 1
	default:
		return 0
	}
}

// levelToTier converts a numeric level back to a runtime tier string.
func levelToTier(level int) string {
	switch level {
	case 3:
		return "full"
	case 2:
		return "core"
	case 1:
		return "micro"
	default:
		return "full"
	}
}
