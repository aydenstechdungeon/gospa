package gospa

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"

	"encoding/json"
	"github.com/a-h/templ"
	"github.com/aydenstechdungeon/gospa/routing"
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
	params := make(map[string]interface{})

	layouts := a.Router.ResolveLayoutChain(errRoute)
	content = a.wrapWithLayouts(content, layouts, params, path)

	rootLayoutFunc := routing.GetRootLayout()
	var wrappedContent templ.Component
	if rootLayoutFunc != nil {
		tier := a.resolveTier(routing.RouteOptions{}, layouts)
		rootProps := a.buildRootLayoutProps(c, params, tier)
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

func (a *App) buildPageContent(route *routing.Route, params map[string]interface{}, path string) templ.Component {
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

func (a *App) wrapWithLayouts(content templ.Component, layouts []*routing.Route, params map[string]interface{}, path string) templ.Component {
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

func (a *App) buildRootLayoutProps(c gofiber.Ctx, params map[string]interface{}, tier string) map[string]interface{} {
	wsRD, wsMR, wsHB := a.normalizeWSConfig()
	props := map[string]interface{}{
		"appName":             a.Config.AppName,
		"runtimePath":         a.getRuntimePathForTier(tier),
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

func (a *App) buildPageHTML(ctx context.Context, route *routing.Route, params map[string]interface{}) ([]byte, error) {
	layouts := a.Router.ResolveLayoutChain(route)
	if params == nil {
		params = map[string]interface{}{}
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

// getRuntimePathForTier returns the path to the client runtime script for the specified tier.
func (a *App) getRuntimePathForTier(tier string) string {
	if a.Config.RuntimeScript != "" && tier == "" {
		return a.Config.RuntimeScript
	}

	name := "runtime"
	switch strings.ToLower(tier) {
	case string(RuntimeTierMicro):
		name = "runtime-micro"
	case string(RuntimeTierCore):
		name = "runtime-core"
	}

	if a.Config.DevMode {
		return "/_gospa/" + name + ".js"
	}

	if a.Config.BuildManifest != nil {
		if path, ok := a.Config.BuildManifest["static/js/"+name+".js"]; ok {
			return "/" + path
		}
	}

	return "/_gospa/" + name + ".js"
}

// getRuntimePath returns the path to the client runtime script from global config.
func (a *App) getRuntimePath() string {
	return a.getRuntimePathForTier(string(a.Config.RuntimeTier))
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

	host := strings.TrimSpace(string(c.Request().Host()))
	_, portStr, _ := net.SplitHostPort(host)
	port, _ := strconv.Atoi(portStr)

	isPortAllowedInsecure := false
	if port > 0 {
		for _, p := range a.Config.AllowPortsWithInsecureWS {
			if p == port {
				isPortAllowedInsecure = true
				break
			}
		}
	}

	protocol := "ws://"
	shouldUseWSS := (c.Protocol() == "https" || strings.ToLower(c.Get("X-Forwarded-Proto")) == "https")
	if shouldUseWSS && !a.Config.AllowInsecureWS && !isPortAllowedInsecure {
		protocol = "wss://"
	}

	if a.Config.DevMode || a.Config.AllowInsecureWS || isPortAllowedInsecure {
		if validatedHost, ok := a.validatePublicHost(host); ok {
			return protocol + validatedHost + a.Config.WebSocketPath
		}
		if portStr != "" {
			return protocol + "127.0.0.1:" + portStr + a.Config.WebSocketPath
		}
		return protocol + "127.0.0.1" + a.Config.WebSocketPath
	}

	// SECURITY FIX: In production we must NEVER reflect the Host header into the
	// WebSocket URL. An attacker controlling the Host header could perform SSRF.
	// If PublicOrigin is set, it was handled above and we won't reach this point.
	// If PublicOrigin is missing in production, fail safe to loopback and log a
	// hard-error so operators see it immediately.
	a.Logger().Error("CRITICAL: PublicOrigin is not set in production. WebSocket connections will fail. Set PublicOrigin (e.g. https://yourapp.com) to fix this.")

	// Try to include port in the fallback if we're on a non-standard port
	if portStr != "" {
		return protocol + "127.0.0.1:" + portStr + a.Config.WebSocketPath
	}
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
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(v); err != nil {
		return "{}"
	}
	return strings.TrimSuffix(buf.String(), "\n")
}

const noncePlaceholder = "__GOSPA_NONCE_PLACEHOLDER__"

// replaceNonces replaces the nonce placeholder in the HTML with the actual nonce.
func (a *App) replaceNonces(html []byte, nonce string) []byte {
	return bytes.ReplaceAll(html, []byte(noncePlaceholder), []byte(nonce))
}
