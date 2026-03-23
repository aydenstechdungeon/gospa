package gospa

import (
	"bufio"
	"bytes"
	"fmt"
	"time"

	"github.com/aydenstechdungeon/gospa/routing"
	templpkg "github.com/aydenstechdungeon/gospa/templ"
	gofiber "github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

// renderRoute renders a route with its layout chain.
func (a *App) renderRoute(c gofiber.Ctx, route *routing.Route) error {
	cacheKey := c.Path()
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
			if data, err := a.Config.Storage.Get("gospa:ssg:" + cacheKey); err == nil {
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
			if data, err := a.Config.Storage.Get("gospa:ssg:" + cacheKey); err == nil {
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
			return c.Send(entry.html)
		}
	}

	// 3. PPR Strategy
	if a.Config.CacheTemplates && effStrategy == routing.StrategyPPR {
		var shell []byte
		var shellHit bool
		if a.Config.Storage != nil && !a.Config.Prefork {
			if data, err := a.Config.Storage.Get("gospa:ppr:" + cacheKey); err == nil {
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
			result, err := a.applyPPRSlots(route, shell, c.Path(), opts)
			if err != nil {
				a.Logger().Error("PPR slot error", "err", err)
			}
			c.Set("Content-Type", "text/html")
			c.Set("Cache-Control", "no-store")
			return c.Send(result)
		}
	}

	layouts := a.Router.ResolveLayoutChain(route)
	_, params := a.Router.Match(c.Path())
	ctx := c.Context()

	content := a.buildPageContent(route, params, c.Path())
	content = a.wrapWithLayouts(content, layouts, params, c.Path())

	c.Set("Content-Type", "text/html")

	rootLayoutFunc := routing.GetRootLayout()
	if rootLayoutFunc != nil {
		rootProps := a.buildRootLayoutProps(c, params)
		wrappedContent := rootLayoutFunc(content, rootProps)

		if a.Config.CacheTemplates && effStrategy == routing.StrategySSG {
			var buf bytes.Buffer
			if err := wrappedContent.Render(ctx, &buf); err != nil {
				a.Logger().Error("SSG render error", "err", err)
				return a.renderError(c, gofiber.StatusInternalServerError, err)
			}
			a.storeSsgEntry(cacheKey, buf.Bytes())
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
			a.storeSsgEntry(cacheKey, buf.Bytes())
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
					ld = a.wrapWithLayouts(ld, layouts, params, c.Path())
					rootProps := a.buildRootLayoutProps(c, params)
					shellContent = rootLayoutFunc(ld, rootProps)
				}

				var shellBuf bytes.Buffer
				if err := shellContent.Render(shellCtx, &shellBuf); err != nil {
					a.Logger().Error("PPR shell render error", "err", err)
					return a.renderError(c, gofiber.StatusInternalServerError, err)
				}
				a.storePprShell(cacheKey, shellBuf.Bytes())
				result, err := a.applyPPRSlots(route, shellBuf.Bytes(), c.Path(), opts)
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
				if data, err := a.Config.Storage.Get("gospa:ppr:" + cacheKey); err == nil {
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
				result, err := a.applyPPRSlots(route, shellHTML, c.Path(), opts)
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
		c.Response().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
			defer func() {
				if r := recover(); r != nil {
					a.Logger().Error("panic during streaming render", "err", r)
				}
			}()
			if err := wrappedContent.Render(ctx, w); err != nil {
				a.Logger().Error("streaming render error", "err", err)
			}
			_ = w.Flush()
		}))
		return nil
	}

	wsURL := a.getWSUrl(c)
	runtimePath := a.getRuntimePath()
	wsRD, wsMR, wsHB := a.normalizeWSConfig()

	c.Set("Cache-Control", "no-store")
	c.Response().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		_, _ = fmt.Fprint(w, `<!DOCTYPE html><html lang="en" data-gospa-auto><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>`)
		_, _ = fmt.Fprint(w, a.Config.AppName)
		_, _ = fmt.Fprint(w, `</title></head><body><div id="app" data-gospa-root><main>`)
		if err := content.Render(ctx, w); err != nil {
			a.Logger().Error("streaming render error", "err", err)
		}
		_, _ = fmt.Fprint(w, `</main></div>`)
		_, _ = fmt.Fprintf(w, `<script src="%s" type="module"></script>`, runtimePath)
		_, _ = fmt.Fprintf(w, `<script type="module">
import * as runtime from %s;
window.__GOSPA_CONFIG__ = {
	navigationOptions: %s,
};
runtime.init({
	wsUrl: %s,
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
</script>`, toJS(runtimePath), toJS(a.Config.NavigationOptions), toJS(wsURL), a.Config.DevMode, a.Config.SimpleRuntimeSVGs, a.Config.DisableSanitization, wsRD, wsMR, wsHB, toJS(a.Config.HydrationMode), a.Config.HydrationTimeout)
		_, _ = fmt.Fprint(w, `</body></html>`)
		_ = w.Flush()
	}))
	return nil
}
