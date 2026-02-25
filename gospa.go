// Package gospa provides a modern SPA framework for Go with Fiber and Templ.
// It brings Svelte-like reactivity and state management to Go.
package gospa

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sync"

	"time"

	"github.com/a-h/templ"
	"github.com/aydenstechdungeon/gospa/embed"
	"github.com/aydenstechdungeon/gospa/fiber"
	"github.com/aydenstechdungeon/gospa/routing"
	"github.com/aydenstechdungeon/gospa/state"
	fiberpkg "github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// StateSerializerFunc defines a function for state serialization
type StateSerializerFunc func(interface{}) ([]byte, error)

// StateDeserializerFunc defines a function for state deserialization
type StateDeserializerFunc func([]byte, interface{}) error

// Version is the current version of GoSPA.
const Version = "0.1.0"

// Config holds the application configuration.
type Config struct {
	// RoutesDir is the directory containing route files.
	RoutesDir string
	// RoutesFS is the filesystem containing route files (optional). Takes precedence over RoutesDir if provided.
	RoutesFS fs.FS
	// DevMode enables development features.
	DevMode bool
	// RuntimeScript is the path to the client runtime script.
	RuntimeScript string
	// StaticDir is the directory for static files.
	StaticDir string
	// StaticPrefix is the URL prefix for static files.
	StaticPrefix string
	// AppName is the application name.
	AppName string
	// DefaultState is the initial state for new sessions.
	DefaultState map[string]interface{}
	// EnableWebSocket enables WebSocket support.
	EnableWebSocket bool
	// WebSocketPath is the WebSocket endpoint path.
	WebSocketPath string
	// WebSocketMiddleware allows injecting session/auth middleware before WebSocket upgrade.
	WebSocketMiddleware fiberpkg.Handler

	// Performance Options
	// NOTE: CompressState is planned but not yet implemented — setting this has no effect.
	CompressState bool
	// NOTE: StateDiffing is planned but not yet implemented — setting this has no effect.
	StateDiffing   bool
	CacheTemplates bool // Cache compiled templates (SSG only)
	SimpleRuntime  bool // Use lightweight runtime without DOMPurify (~6KB smaller)
	// SimpleRuntimeSVGs allows SVG elements in the simple runtime sanitizer.
	// WARNING: Only enable if your content is fully trusted and never user-generated.
	SimpleRuntimeSVGs bool

	// WebSocket Options
	// NOTE: WSReconnectDelay, WSMaxReconnect, WSHeartbeat are planned but not yet implemented.
	// The client-side WebSocket reconnect logic uses its own defaults (1s delay, 10 max attempts, 30s heartbeat).
	WSReconnectDelay time.Duration // Initial reconnect delay (planned)
	WSMaxReconnect   int           // Max reconnect attempts (planned)
	WSHeartbeat      time.Duration // Heartbeat interval (planned)

	// Hydration Options
	// HydrationMode controls when components become interactive.
	// Supported values: "immediate" | "lazy" | "visible" | "idle" (default: "immediate").
	HydrationMode    string
	HydrationTimeout int // ms before force hydrate (used with "visible" and "idle" modes)

	// Serialization Options
	// NOTE: StateSerializer and StateDeserializer are planned but not yet implemented.
	StateSerializer   StateSerializerFunc
	StateDeserializer StateDeserializerFunc

	// Routing Options
	DisableSPA bool // Disable SPA navigation completely
	// NOTE: SSR (global SSR mode) is planned but not yet implemented — currently all pages are SSR by default.
	SSR bool

	// Remote Action Options
	MaxRequestBodySize int    // Maximum allowed size for remote action request bodies
	RemotePrefix       string // Prefix for remote action endpoints (default "/_gospa/remote")

	// Security Options
	AllowedOrigins []string // Allowed CORS origins
	EnableCSRF     bool     // Enable automatic CSRF protection (requires CSRFSetTokenMiddleware + CSRFTokenMiddleware)
	// SSGCacheMaxEntries caps the SSG page cache size. Oldest entries are evicted when full.
	// Default: 500. Set to -1 to disable eviction (unbounded, not recommended in production).
	SSGCacheMaxEntries int
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		RoutesDir:          "./routes",
		DevMode:            false,
		RuntimeScript:      "/_gospa/runtime.js",
		StaticDir:          "./static",
		StaticPrefix:       "/static",
		AppName:            "GoSPA App",
		DefaultState:       make(map[string]interface{}),
		EnableWebSocket:    true,
		WebSocketPath:      "/_gospa/ws",
		RemotePrefix:       "/_gospa/remote",
		MaxRequestBodySize: 4 * 1024 * 1024, // Default 4MB
	}
}

// getRuntimePath returns the versioned path for the runtime script.
func (a *App) getRuntimePath() string {
	if a.Config.RuntimeScript != "/_gospa/runtime.js" && a.Config.RuntimeScript != "" {
		return a.Config.RuntimeScript
	}

	name := "runtime"
	if a.Config.SimpleRuntime {
		name = "runtime-simple"
	}

	// Try to get hash from content for better cache busting
	if h, err := embed.RuntimeHash(a.Config.SimpleRuntime); err == nil {
		return fmt.Sprintf("/_gospa/%s.%s.js", name, h)
	}
	// Fallback to version-based hash
	h := fmt.Sprintf("%x", sha256.Sum256([]byte(Version)))
	return fmt.Sprintf("/_gospa/%s.%s.js", name, h[:8])
}

// getWSUrl returns the WebSocket URL for the current request.
func (a *App) getWSUrl(c *fiberpkg.Ctx) string {
	protocol := "ws://"
	if c.Secure() {
		protocol = "wss://"
	}
	return protocol + string(c.Request().Host()) + a.Config.WebSocketPath
}

// App is the main GoSPA application.
type App struct {
	// Config is the application configuration.
	Config Config
	// Router is the file-based router.
	Router *routing.Router
	// Fiber is the underlying Fiber app.
	Fiber *fiberpkg.App
	// Hub is the WebSocket hub for real-time updates.
	Hub *fiber.WSHub
	// StateMap is the global state map.
	StateMap *state.StateMap
	// ssgCache stores pre-rendered SSG pages
	ssgCache map[string][]byte
	// ssgCacheKeys tracks insertion order for FIFO eviction
	ssgCacheKeys []string
	// ssgCacheMu protects ssgCache and ssgCacheKeys
	ssgCacheMu sync.RWMutex
}

// New creates a new GoSPA application.
func New(config Config) *App {
	// Apply defaults
	if config.RoutesDir == "" {
		config.RoutesDir = "./routes"
	}
	if config.RuntimeScript == "" {
		config.RuntimeScript = "/_gospa/runtime.js"
	}
	if config.StaticDir == "" {
		config.StaticDir = "./static"
	}
	if config.StaticPrefix == "" {
		config.StaticPrefix = "/static"
	}
	if config.DefaultState == nil {
		config.DefaultState = make(map[string]interface{})
	}
	if config.WebSocketPath == "" {
		config.WebSocketPath = "/_gospa/ws"
	}

	if config.RemotePrefix == "" {
		config.RemotePrefix = "/_gospa/remote"
	}
	if config.MaxRequestBodySize == 0 {
		config.MaxRequestBodySize = 4 * 1024 * 1024
	}

	// Create router
	var routerSource interface{}
	if config.RoutesFS != nil {
		routerSource = config.RoutesFS
	} else {
		routerSource = config.RoutesDir
	}
	router := routing.NewRouter(routerSource)

	// Create Fiber app
	fiberConfig := fiberpkg.Config{
		AppName: config.AppName,
	}
	if config.DevMode {
		fiberConfig.EnablePrintRoutes = true
	}
	fiberApp := fiberpkg.New(fiberConfig)

	// Create WebSocket hub (always enabled by default - WebSocket is a core feature)
	// Note: Go can't distinguish between "unset" and "explicitly false" for bools,
	// so we always create the hub. Users who don't want WebSocket can simply not use it.
	hub := fiber.NewWSHub()
	go hub.Run()

	// Create state map
	stateMap := state.NewStateMap()
	for k, v := range config.DefaultState {
		r := state.NewRune(v)
		stateMap.Add(k, r)
	}

	app := &App{
		Config:       config,
		Router:       router,
		Fiber:        fiberApp,
		Hub:          hub,
		StateMap:     stateMap,
		ssgCache:     make(map[string][]byte),
		ssgCacheKeys: make([]string, 0),
	}

	// Setup middleware
	app.setupMiddleware()

	// Setup routes
	app.setupRoutes()

	return app
}

// setupMiddleware configures the middleware stack.
func (a *App) setupMiddleware() {
	// Recovery middleware
	a.Fiber.Use(recover.New())

	// Logger middleware
	if a.Config.DevMode {
		a.Fiber.Use(logger.New())
	}

	// Compression
	a.Fiber.Use(compress.New(compress.Config{
		Level: compress.LevelDefault,
	}))

	// Security headers
	a.Fiber.Use(fiber.SecurityHeadersMiddleware())

	if len(a.Config.AllowedOrigins) > 0 {
		a.Fiber.Use(fiber.CORSMiddleware(a.Config.AllowedOrigins))
	}

	if a.Config.EnableCSRF {
		a.Fiber.Use(fiber.CSRFTokenMiddleware())
	}

	// SPA navigation middleware (must come before SPAMiddleware)
	if !a.Config.DisableSPA {
		a.Fiber.Use(fiber.SPANavigationMiddleware())
	}

	// SPA middleware
	spaConfig := fiber.DefaultConfig()
	spaConfig.DevMode = a.Config.DevMode
	spaConfig.RuntimeScript = a.Config.RuntimeScript
	a.Fiber.Use(fiber.SPAMiddleware(spaConfig))
}

// setupRoutes configures the routes.
func (a *App) setupRoutes() {
	// Serve main runtime script with specific middleware to support hashing and 'simple' toggle
	a.Fiber.Get(a.getRuntimePath(), fiber.RuntimeMiddleware(a.Config.SimpleRuntime))

	// Serve other runtime chunks from the embedded filesystem with long-term caching
	a.Fiber.Use("/_gospa/", func(c *fiberpkg.Ctx) error {
		c.Set("Cache-Control", "public, max-age=31536000, immutable")
		return c.Next()
	})
	a.Fiber.Use("/_gospa/", filesystem.New(filesystem.Config{
		Root: http.FS(embed.RuntimeFS()),
	}))

	// WebSocket endpoint (always registered since hub is always created)
	if a.Hub != nil {
		handlers := []fiberpkg.Handler{}
		if a.Config.WebSocketMiddleware != nil {
			handlers = append(handlers, a.Config.WebSocketMiddleware)
		}
		handlers = append(handlers, fiber.WebSocketHandler(fiber.WebSocketConfig{
			Hub: a.Hub,
		}))
		a.Fiber.Get(a.Config.WebSocketPath, handlers...)
	}

	// Remote Actions endpoint
	a.Fiber.Post(a.Config.RemotePrefix+"/:name", func(c *fiberpkg.Ctx) error {
		name := c.Params("name")
		fn, ok := routing.GetRemoteAction(name)
		if !ok {
			return c.Status(fiberpkg.StatusNotFound).JSON(fiberpkg.Map{
				"error": "Remote action not found",
			})
		}

		var input interface{}
		// Only parse if body is not empty
		if len(c.Body()) > 0 {
			if len(c.Body()) > a.Config.MaxRequestBodySize {
				return c.Status(fiberpkg.StatusRequestEntityTooLarge).JSON(fiberpkg.Map{
					"error": "Request body too large",
				})
			}
			if err := c.BodyParser(&input); err != nil {
				return c.Status(fiberpkg.StatusBadRequest).JSON(fiberpkg.Map{
					"error": "Invalid input JSON",
				})
			}
		}

		result, err := fn(c.Context(), input)
		if err != nil {
			// Log the actual error internally for debugging
			log.Printf("Remote action %q error: %v", name, err)
			// Return generic error to client to avoid information disclosure
			return c.Status(fiberpkg.StatusInternalServerError).JSON(fiberpkg.Map{
				"error": "Internal server error",
			})
		}

		return c.JSON(result)
	})

	// Static files
	if _, err := os.Stat(a.Config.StaticDir); err == nil {
		a.Fiber.Use(a.Config.StaticPrefix, filesystem.New(filesystem.Config{
			Root: http.Dir(a.Config.StaticDir),
		}))
		// Serve favicon from static dir if requested at root
		a.Fiber.Get("/favicon.ico", func(c *fiberpkg.Ctx) error {
			favPath := a.Config.StaticDir + "/favicon.ico"
			if _, err := os.Stat(favPath); err == nil {
				return c.SendFile(favPath)
			}
			return c.SendStatus(fiberpkg.StatusNoContent)
		})
	} else {
		// Prevent 404 errors for default favicon requests
		a.Fiber.Get("/favicon.ico", func(c *fiberpkg.Ctx) error {
			return c.SendStatus(fiberpkg.StatusNoContent)
		})
	}
}

// Scan scans the routes directory and builds the route tree.
func (a *App) Scan() error {
	return a.Router.Scan()
}

// RegisterRoutes registers all scanned routes with Fiber.
func (a *App) RegisterRoutes() error {
	if err := a.Scan(); err != nil {
		return err
	}

	// Register page routes
	for _, route := range a.Router.GetPages() {
		// Capture route for closure
		r := route
		a.Fiber.Get(r.Path, func(c *fiberpkg.Ctx) error {
			return a.renderRoute(c, r)
		})
	}

	return nil
}

// renderRoute renders a route with its layout chain.
func (a *App) renderRoute(c *fiberpkg.Ctx, route *routing.Route) error {
	// Check SSG cache
	opts := routing.GetRouteOptions(route.Path)
	cacheKey := c.Path()
	if a.Config.CacheTemplates && opts.Strategy == routing.StrategySSG {
		a.ssgCacheMu.RLock()
		if cached, ok := a.ssgCache[cacheKey]; ok {
			a.ssgCacheMu.RUnlock()
			c.Set("Content-Type", "text/html")
			return c.Send(cached)
		}
		a.ssgCacheMu.RUnlock()
	}

	// Get layout chain
	layouts := a.Router.ResolveLayoutChain(route)

	// Get params
	_, params := a.Router.Match(c.Path())

	// Create base context
	ctx := c.Context()

	// Build the page content from inside out
	var content templ.Component

	// Look up the page component in the registry
	pageFunc := routing.GetPage(route.Path)
	if pageFunc != nil {
		// Call the page component function with props
		props := map[string]interface{}{
			"path": c.Path(),
		}
		// Flatten params into props (e.g., props["id"] instead of props["params"]["id"])
		for k, v := range params {
			props[k] = v
		}
		content = pageFunc(props)
	} else {
		// Fallback to placeholder if no component registered
		content = templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			_, _ = fmt.Fprintf(w, `<div data-gospa-page="%s">Page: %s</div>`, route.Path, route.Path)
			return nil
		})
	}

	// Wrap with layouts from innermost to outermost
	for i := len(layouts) - 1; i >= 0; i-- {
		layout := layouts[i]

		// Look up the layout component in the registry
		layoutFunc := routing.GetLayout(layout.Path)
		if layoutFunc != nil {
			// Use the registered layout function
			props := map[string]interface{}{
				"path": c.Path(),
			}
			// Flatten params into props
			for k, v := range params {
				props[k] = v
			}
			content = layoutFunc(content, props)
		} else {
			// Fallback to wrapper div
			children := content
			content = templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
				_, _ = fmt.Fprintf(w, `<div data-gospa-layout="%s">`, layout.Path)
				if err := children.Render(ctx, w); err != nil {
					return err
				}
				_, _ = fmt.Fprint(w, `</div>`)
				return nil
			})
		}
	}

	// Render the full page
	c.Set("Content-Type", "text/html")

	// Look up the root layout
	rootLayoutFunc := routing.GetRootLayout()
	if rootLayoutFunc != nil {
		props := map[string]interface{}{
			"appName":          a.Config.AppName,
			"runtimePath":      a.getRuntimePath(),
			"path":             c.Path(),
			"debug":            a.Config.DevMode,
			"wsUrl":            a.getWSUrl(c),
			"hydrationMode":    a.Config.HydrationMode,
			"hydrationTimeout": a.Config.HydrationTimeout,
		}
		for k, v := range params {
			props[k] = v
		}

		wrappedContent := rootLayoutFunc(content, props)

		if a.Config.CacheTemplates && opts.Strategy == routing.StrategySSG {
			var buf bytes.Buffer
			if err := wrappedContent.Render(ctx, &buf); err != nil {
				log.Printf("Render error: %v", err)
			}
			a.ssgCacheMu.Lock()
			// FIFO eviction: cap cache to SSGCacheMaxEntries (default 500; -1 = unbounded)
			maxEntries := a.Config.SSGCacheMaxEntries
			if maxEntries == 0 {
				maxEntries = 500
			}
			if maxEntries > 0 && len(a.ssgCache) >= maxEntries && len(a.ssgCacheKeys) > 0 {
				oldest := a.ssgCacheKeys[0]
				a.ssgCacheKeys = a.ssgCacheKeys[1:]
				delete(a.ssgCache, oldest)
			}
			if _, exists := a.ssgCache[cacheKey]; !exists {
				a.ssgCacheKeys = append(a.ssgCacheKeys, cacheKey)
			}
			a.ssgCache[cacheKey] = buf.Bytes()
			a.ssgCacheMu.Unlock()
			return c.Send(buf.Bytes())
		}

		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			if err := wrappedContent.Render(ctx, w); err != nil {
				log.Printf("Streaming render error: %v", err)
			}
			w.Flush()
		})
		return nil
	}

	protocol := "ws://"
	if c.Secure() {
		protocol = "wss://"
	}
	wsUrl := protocol + string(c.Request().Host()) + a.Config.WebSocketPath
	runtimePath := a.getRuntimePath()
	appName := a.Config.AppName
	devMode := a.Config.DevMode

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		// Write HTML wrapper with main tag for SPA navigation
		_, _ = fmt.Fprint(w, `<!DOCTYPE html><html lang="en" data-gospa-auto><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>`)
		_, _ = fmt.Fprint(w, appName)
		_, _ = fmt.Fprint(w, `</title></head><body><div id="app" data-gospa-root><main>`)

		// Render content
		if err := content.Render(ctx, w); err != nil {
			log.Printf("Streaming render error: %v", err)
		}

		// Close wrapper
		_, _ = fmt.Fprint(w, `</main></div>`)

		// Inject runtime script
		_, _ = fmt.Fprintf(w, `<script src="%s" type="module"></script>`, runtimePath)

		_, _ = fmt.Fprintf(w, `<script type="module">
import * as runtime from '%s';
runtime.init({
	wsUrl: '%s',
	debug: %v,
	simpleRuntimeSVGs: %v,
	hydration: {
		mode: '%s',
		timeout: %d
	}
});
</script>`, runtimePath, wsUrl, devMode, a.Config.SimpleRuntimeSVGs, a.Config.HydrationMode, a.Config.HydrationTimeout)

		_, _ = fmt.Fprint(w, `</body></html>`)
		w.Flush()
	})

	return nil
}

// Run starts the application on the given address.
func (a *App) Run(addr string) error {
	// Scan routes if not already done
	if len(a.Router.GetRoutes()) == 0 {
		if err := a.Scan(); err != nil {
			return err
		}
	}

	// Register routes
	if err := a.RegisterRoutes(); err != nil {
		return err
	}

	// Start server
	log.Printf("GoSPA %s starting on %s", Version, addr)
	return a.Fiber.Listen(addr)
}

// RunTLS starts the application with TLS on the given address.
func (a *App) RunTLS(addr, certFile, keyFile string) error {
	// Scan routes if not already done
	if len(a.Router.GetRoutes()) == 0 {
		if err := a.Scan(); err != nil {
			return err
		}
	}

	// Register routes
	if err := a.RegisterRoutes(); err != nil {
		return err
	}

	// Start server with TLS
	log.Printf("GoSPA %s starting on %s (TLS)", Version, addr)
	return a.Fiber.ListenTLS(addr, certFile, keyFile)
}

// Shutdown gracefully shuts down the application.
func (a *App) Shutdown() error {
	return a.Fiber.Shutdown()
}

// Use adds a middleware to the application.
func (a *App) Use(middleware ...fiberpkg.Handler) {
	for _, m := range middleware {
		a.Fiber.Use(m)
	}
}

// Get adds a GET route.
func (a *App) Get(path string, handlers ...fiberpkg.Handler) {
	a.Fiber.Get(path, handlers...)
}

// Post adds a POST route.
func (a *App) Post(path string, handlers ...fiberpkg.Handler) {
	a.Fiber.Post(path, handlers...)
}

// Put adds a PUT route.
func (a *App) Put(path string, handlers ...fiberpkg.Handler) {
	a.Fiber.Put(path, handlers...)
}

// Delete adds a DELETE route.
func (a *App) Delete(path string, handlers ...fiberpkg.Handler) {
	a.Fiber.Delete(path, handlers...)
}

// Group creates a route group.
func (a *App) Group(prefix string, handlers ...fiberpkg.Handler) fiberpkg.Router {
	return a.Fiber.Group(prefix, handlers...)
}

// Static serves static files.
func (a *App) Static(prefix, root string) {
	a.Fiber.Static(prefix, root)
}

// GetHub returns the WebSocket hub.
func (a *App) GetHub() *fiber.WSHub {
	return a.Hub
}

// GetRouter returns the file-based router.
func (a *App) GetRouter() *routing.Router {
	return a.Router
}

// GetFiber returns the underlying Fiber app.
func (a *App) GetFiber() *fiberpkg.App {
	return a.Fiber
}

// Broadcast sends a message to all connected WebSocket clients.
func (a *App) Broadcast(message []byte) {
	if a.Hub != nil {
		a.Hub.Broadcast <- message
	}
}

// BroadcastState broadcasts a state update to all connected WebSocket clients.
func (a *App) BroadcastState(key string, value interface{}) error {
	return fiber.BroadcastState(a.Hub, key, value)
}
