// Package gospa provides a modern SPA framework for Go with Fiber and Templ.
// It brings Svelte-like reactivity and state management to Go.
package gospa

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

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

// Version is the current version of GoSPA.
const Version = "0.1.0"

// Config holds the application configuration.
type Config struct {
	// RoutesDir is the directory containing route files.
	RoutesDir string
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
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		RoutesDir:       "./routes",
		DevMode:         false,
		RuntimeScript:   "/_gospa/runtime.js",
		StaticDir:       "./static",
		StaticPrefix:    "/static",
		AppName:         "GoSPA App",
		DefaultState:    make(map[string]interface{}),
		EnableWebSocket: true,
		WebSocketPath:   "/_gospa/ws",
	}
}

// getRuntimePath returns the versioned path for the runtime script.
func (a *App) getRuntimePath() string {
	if a.Config.RuntimeScript != "/_gospa/runtime.js" && a.Config.RuntimeScript != "" {
		return a.Config.RuntimeScript
	}
	// Try to get hash from content for better cache busting
	if h, err := embed.RuntimeHash(); err == nil {
		return fmt.Sprintf("/_gospa/runtime.%s.js", h)
	}
	// Fallback to version-based hash
	h := fmt.Sprintf("%x", sha256.Sum256([]byte(Version)))
	return fmt.Sprintf("/_gospa/runtime.%s.js", h[:8])
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

	// Create router
	router := routing.NewRouter(config.RoutesDir)

	// Create Fiber app
	fiberConfig := fiberpkg.Config{
		AppName: config.AppName,
	}
	if config.DevMode {
		fiberConfig.EnablePrintRoutes = true
	}
	fiberApp := fiberpkg.New(fiberConfig)

	// Create WebSocket hub
	var hub *fiber.WSHub
	if config.EnableWebSocket {
		hub = fiber.NewWSHub()
		go hub.Run()
	}

	// Create state map
	stateMap := state.NewStateMap()
	for k, v := range config.DefaultState {
		r := state.NewRune(v)
		stateMap.Add(k, r)
	}

	app := &App{
		Config:   config,
		Router:   router,
		Fiber:    fiberApp,
		Hub:      hub,
		StateMap: stateMap,
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
		Level: compress.LevelBestSpeed,
	}))

	// Security headers
	a.Fiber.Use(fiber.SecurityHeadersMiddleware())

	// SPA navigation middleware (must come before SPAMiddleware)
	a.Fiber.Use(fiber.SPANavigationMiddleware())

	// SPA middleware
	spaConfig := fiber.DefaultConfig()
	spaConfig.DevMode = a.Config.DevMode
	spaConfig.RuntimeScript = a.Config.RuntimeScript
	a.Fiber.Use(fiber.SPAMiddleware(spaConfig))
}

// setupRoutes configures the routes.
func (a *App) setupRoutes() {
	// Runtime script
	a.Fiber.Get(a.getRuntimePath(), fiber.RuntimeMiddleware())

	// WebSocket endpoint
	if a.Config.EnableWebSocket && a.Hub != nil {
		handlers := []fiberpkg.Handler{}
		if a.Config.WebSocketMiddleware != nil {
			handlers = append(handlers, a.Config.WebSocketMiddleware)
		}
		handlers = append(handlers, fiber.WebSocketHandler(fiber.WebSocketConfig{
			Hub: a.Hub,
		}))
		a.Fiber.Get(a.Config.WebSocketPath, handlers...)
	}

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
			"appName":     a.Config.AppName,
			"runtimePath": a.getRuntimePath(),
			"path":        c.Path(),
		}
		for k, v := range params {
			props[k] = v
		}

		wrappedContent := rootLayoutFunc(content, props)
		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			if err := wrappedContent.Render(ctx, w); err != nil {
				log.Printf("Streaming render error: %v", err)
			}
			w.Flush()
		})
		return nil
	}

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		// Write HTML wrapper with main tag for SPA navigation
		_, _ = fmt.Fprint(w, `<!DOCTYPE html><html lang="en" data-gospa-auto><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>`)
		_, _ = fmt.Fprint(w, a.Config.AppName)
		_, _ = fmt.Fprint(w, `</title></head><body><div id="app" data-gospa-root><main>`)

		// Render content
		if err := content.Render(ctx, w); err != nil {
			log.Printf("Streaming render error: %v", err)
		}

		// Close wrapper
		_, _ = fmt.Fprint(w, `</main></div>`)

		// Inject runtime script
		_, _ = fmt.Fprintf(w, `<script src="%s" type="module"></script>`, a.getRuntimePath())

		protocol := "ws://"
		if c.Secure() {
			protocol = "wss://"
		}
		wsUrl := protocol + string(c.Request().Host()) + a.Config.WebSocketPath

		_, _ = fmt.Fprintf(w, `<script type="module">
import runtime from '%s';
runtime.init({
	wsUrl: '%s',
	debug: %v
});
</script>`, a.getRuntimePath(), wsUrl, a.Config.DevMode)

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
