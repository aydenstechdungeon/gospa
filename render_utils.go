package gospa

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"

	"github.com/a-h/templ"
	"github.com/aydenstechdungeon/gospa/embed"
	"github.com/aydenstechdungeon/gospa/routing"
	json "github.com/goccy/go-json"
	gofiber "github.com/gofiber/fiber/v3"
)

func (a *App) validatePublicHost(host string) (string, bool) {
	if host == "" || len(host) > 253 || strings.Contains(host, "@") || strings.Contains(host, "://") {
		return "", false
	}

	// Filter out dangerous characters that shouldn't be in a hostname
	for _, r := range host {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '.' && r != '-' && r != ':' {
			return "", false
		}
	}

	candidate := host
	if !strings.Contains(candidate, ":") || strings.Count(candidate, ":") > 1 {
		candidate = net.JoinHostPort(host, "80")
	}

	parsedHost, _, err := net.SplitHostPort(candidate)
	if err != nil {
		return "", false
	}
	if parsedHost == "" {
		return "", false
	}

	if a.Config.PublicOrigin != "" {
		if parsedURL, err := url.Parse(a.Config.PublicOrigin); err == nil && parsedURL.Host != "" {
			expectedHost := parsedURL.Hostname()
			if !strings.EqualFold(parsedHost, expectedHost) {
				return "", false
			}
		}
	} else if !a.Config.DevMode {
		// In production, if PublicOrigin is NOT set, we do NOT trust the Host header
		return "", false
	}

	return host, true
}

func (a *App) renderError(c gofiber.Ctx, statusCode int, errToDisplay error) error {
	path := c.Path()
	message := "Internal Server Error"
	if a.Config.DevMode && errToDisplay != nil {
		message = errToDisplay.Error()
	}
	errRoute := a.Router.GetErrorRoute(path)
	if errRoute == nil {
		return c.Status(statusCode).SendString(message)
	}

	errCompFn := routing.GetError(errRoute.Path)
	if errCompFn == nil {
		return c.Status(statusCode).SendString(message)
	}

	props := map[string]interface{}{
		"error": message,
		"code":  statusCode,
		"path":  path,
	}

	content := errCompFn(props)
	params := make(map[string]string)

	layouts := a.Router.ResolveLayoutChain(errRoute)
	content = a.wrapWithLayouts(content, layouts, params, path)

	rootLayoutFunc := routing.GetRootLayout()
	var wrappedContent templ.Component
	if rootLayoutFunc != nil {
		rootProps := a.buildRootLayoutProps(c, params)
		wrappedContent = rootLayoutFunc(content, rootProps)
	} else {
		wrappedContent = content
	}

	var buf bytes.Buffer
	if rerr := wrappedContent.Render(c.Context(), &buf); rerr != nil {
		a.Logger().Error("Error rendering error boundary", "err", rerr)
		return c.Status(statusCode).SendString("Internal Server Error")
	}

	c.Set("Content-Type", "text/html")
	return c.Status(statusCode).Send(buf.Bytes())
}

func (a *App) buildPageContent(route *routing.Route, params map[string]string, path string) templ.Component {
	pageFunc := routing.GetPage(route.Path)
	if pageFunc != nil {
		props := map[string]interface{}{"path": path}
		for k, v := range params {
			props[k] = v
		}
		return pageFunc(props)
	}
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, _ = fmt.Fprintf(w, `<div data-gospa-page="%s">Page: %s</div>`, route.Path, route.Path)
		return nil
	})
}

func (a *App) wrapWithLayouts(content templ.Component, layouts []*routing.Route, params map[string]string, path string) templ.Component {
	for i := len(layouts) - 1; i >= 0; i-- {
		layout := layouts[i]
		layoutFunc := routing.GetLayout(layout.Path)
		if layoutFunc != nil {
			props := map[string]interface{}{"path": path}
			for k, v := range params {
				props[k] = v
			}
			content = layoutFunc(content, props)
		} else {
			children := content
			lp := layout.Path
			content = templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
				_, _ = fmt.Fprintf(w, `<div data-gospa-layout="%s">`, lp)
				if err := children.Render(ctx, w); err != nil {
					return err
				}
				_, _ = fmt.Fprint(w, `</div>`)
				return nil
			})
		}
	}
	return content
}

func (a *App) buildRootLayoutProps(c gofiber.Ctx, params map[string]string) map[string]interface{} {
	wsRD, wsMR, wsHB := a.normalizeWSConfig()
	props := map[string]interface{}{
		"appName":             a.Config.AppName,
		"runtimePath":         a.getRuntimePath(),
		"path":                c.Path(),
		"debug":               a.Config.DevMode,
		"wsUrl":               a.getWSUrl(c),
		"hydrationMode":       a.Config.HydrationMode,
		"hydrationTimeout":    a.Config.HydrationTimeout,
		"wsReconnectDelay":    wsRD,
		"wsMaxReconnect":      wsMR,
		"wsHeartbeat":         wsHB,
		"serializationFormat": a.Config.SerializationFormat,
		"navigationOptions":   a.Config.NavigationOptions,
		"disableSanitization": a.Config.DisableSanitization,
	}
	for k, v := range params {
		props[k] = v
	}
	return props
}

func (a *App) buildPageHTML(ctx context.Context, route *routing.Route, params map[string]string) ([]byte, error) {
	layouts := a.Router.ResolveLayoutChain(route)
	if params == nil {
		params = map[string]string{}
	}
	path := route.Path
	content := a.buildPageContent(route, params, path)
	content = a.wrapWithLayouts(content, layouts, params, path)

	rootLayoutFunc := routing.GetRootLayout()
	if rootLayoutFunc == nil {
		var buf bytes.Buffer
		if err := content.Render(ctx, &buf); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	wsRD, wsMR, wsHB := a.normalizeWSConfig()
	rootProps := map[string]interface{}{
		"appName":             a.Config.AppName,
		"runtimePath":         a.getRuntimePath(),
		"path":                path,
		"debug":               false,
		"wsUrl":               a.Config.WebSocketPath,
		"hydrationMode":       a.Config.HydrationMode,
		"hydrationTimeout":    a.Config.HydrationTimeout,
		"wsReconnectDelay":    wsRD,
		"wsMaxReconnect":      wsMR,
		"wsHeartbeat":         wsHB,
		"serializationFormat": string(a.Config.SerializationFormat),
	}
	for k, v := range params {
		rootProps[k] = v
	}

	wrapped := rootLayoutFunc(content, rootProps)
	var buf bytes.Buffer
	if err := wrapped.Render(ctx, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (a *App) getRuntimePath() string {
	if a.Config.RuntimeScript != "/_gospa/runtime.js" && a.Config.RuntimeScript != "" {
		return a.Config.RuntimeScript
	}

	name := "runtime"
	if a.Config.SimpleRuntime {
		name = "runtime-simple"
	}

	if h, err := embed.RuntimeHash(a.Config.SimpleRuntime); err == nil {
		return fmt.Sprintf("/_gospa/%s.%s.js", name, h)
	}
	h := fmt.Sprintf("%x", sha256.Sum256([]byte(Version)))
	return fmt.Sprintf("/_gospa/%s.%s.js", name, h[:8])
}

func (a *App) getWSUrl(c gofiber.Ctx) string {
	if publicOrigin := strings.TrimSpace(a.Config.PublicOrigin); publicOrigin != "" {
		if parsed, err := url.Parse(publicOrigin); err == nil && parsed.Host != "" {
			scheme := "ws"
			if strings.EqualFold(parsed.Scheme, "https") {
				scheme = "wss"
			}
			return scheme + "://" + parsed.Host + a.Config.WebSocketPath
		}
	}

	protocol := "ws://"
	if (c.Protocol() == "https" || strings.ToLower(c.Get("X-Forwarded-Proto")) == "https") && !a.Config.AllowInsecureWS {
		protocol = "wss://"
	}

	if a.Config.DevMode {
		host := strings.TrimSpace(string(c.Request().Host()))
		if validatedHost, ok := a.validatePublicHost(host); ok {
			return protocol + validatedHost + a.Config.WebSocketPath
		}
		return protocol + "localhost" + a.Config.WebSocketPath
	}

	// Production fallback — use current host if PublicOrigin is missing
	if host := strings.TrimSpace(string(c.Request().Host())); host != "" {
		if a.Config.AllowInsecureWS {
			return protocol + host + a.Config.WebSocketPath
		}
	}

	a.Logger().Error("CRITICAL: PublicOrigin is not set in production. WebSocket connections will fail or use loopback. Set PublicOrigin for security.")
	return protocol + "127.0.0.1" + a.Config.WebSocketPath
}

func (a *App) normalizeWSConfig() (rd, mr, hb int) {
	rd = int(a.Config.WSReconnectDelay.Milliseconds())
	if rd <= 0 {
		rd = 1000
	}
	mr = a.Config.WSMaxReconnect
	if mr <= 0 {
		mr = 10
	}
	hb = int(a.Config.WSHeartbeat.Milliseconds())
	if hb <= 0 {
		hb = 30000
	}
	return
}

func toJS(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
