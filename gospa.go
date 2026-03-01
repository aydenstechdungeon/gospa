// Package gospa provides a modern SPA framework for Go with Fiber and Templ.
// It brings Svelte-like reactivity and state management to Go.
package gospa

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	templ "github.com/a-h/templ"
	"github.com/aydenstechdungeon/gospa/embed"
	"github.com/aydenstechdungeon/gospa/fiber"
	"github.com/aydenstechdungeon/gospa/routing"
	"github.com/aydenstechdungeon/gospa/state"
	"github.com/aydenstechdungeon/gospa/store"
	templpkg "github.com/aydenstechdungeon/gospa/templ"
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
const Version = "0.1.5"

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
	// CompressState enables gzip compression of outbound WebSocket state payloads.
	// The client receives a { type:"compressed", data: "<base64>", compressed: true }
	// envelope and must decompress using the DecompressionStream browser API.
	CompressState bool
	// StateDiffing enables delta-only "patch" WebSocket messages for state syncs.
	// Only changed state keys are transmitted after the initial full snapshot.
	StateDiffing   bool
	CacheTemplates bool // Cache compiled templates (SSG only)
	SimpleRuntime  bool // Use lightweight runtime without DOMPurify (~6KB smaller)
	// SimpleRuntimeSVGs allows SVG elements in the simple runtime sanitizer.
	// WARNING: Only enable if your content is fully trusted and never user-generated.
	SimpleRuntimeSVGs bool

	// WebSocket Options — these values are passed directly to the client runtime's init() call.
	// Defaults: WSReconnectDelay=1s, WSMaxReconnect=10, WSHeartbeat=30s.
	WSReconnectDelay time.Duration // Initial reconnect delay (default 1s)
	WSMaxReconnect   int           // Max reconnect attempts (default 10)
	WSHeartbeat      time.Duration // Heartbeat ping interval (default 30s)

	// WSMaxMessageSize limits the maximum payload size for WebSocket messages (default 64KB).
	WSMaxMessageSize int
	// WSConnRateLimit sets the refilling rate in connections per second for WebSocket upgrades (default 0.2).
	WSConnRateLimit float64
	// WSConnBurst sets the burst capacity for WebSocket connection upgrades (default 5.0).
	WSConnBurst float64

	// Hydration Options
	// HydrationMode controls when components become interactive.
	// Supported values: "immediate" | "lazy" | "visible" | "idle" (default: "immediate").
	HydrationMode    string
	HydrationTimeout int // ms before force hydrate (used with "visible" and "idle" modes)

	// Serialization Options
	// StateSerializer overrides JSON for outbound WebSocket state serialization.
	// StateDeserializer overrides JSON for inbound WebSocket state deserialization.
	StateSerializer   StateSerializerFunc
	StateDeserializer StateDeserializerFunc

	// Routing Options
	DisableSPA bool // Disable SPA navigation completely
	// NOTE: SSR (global SSR mode) is planned but not yet implemented — currently all pages are SSR by default.
	SSR bool

	// Rendering Strategy Defaults
	// DefaultRenderStrategy sets the fallback strategy for pages that do not
	// explicitly call RegisterPageWithOptions. Defaults to StrategySSR.
	DefaultRenderStrategy routing.RenderStrategy
	// DefaultRevalidateAfter is the ISR TTL used when a page uses StrategyISR
	// but does not set RouteOptions.RevalidateAfter. Zero means revalidate every request.
	DefaultRevalidateAfter time.Duration

	// Remote Action Options
	MaxRequestBodySize int    // Maximum allowed size for remote action request bodies
	RemotePrefix       string // Prefix for remote action endpoints (default "/_gospa/remote")

	// Security Options
	AllowedOrigins []string // Allowed CORS origins
	EnableCSRF     bool     // Enable automatic CSRF protection (requires CSRFSetTokenMiddleware + CSRFTokenMiddleware)
	// SSGCacheMaxEntries caps the SSG/ISR/PPR page cache size. Oldest entries are evicted when full.
	// Default: 500. Set to -1 to disable eviction (unbounded, not recommended in production).
	SSGCacheMaxEntries int
	// SSGCacheTTL sets an expiration time for SSG cache entries. If zero, they never expire.
	SSGCacheTTL time.Duration

	// Prefork enables Fiber's prefork mode.
	// WARNING: If enabled without an external Storage/PubSub backend like Redis, in-memory state and WebSockets will be isolated per-process.
	Prefork bool

	// Storage defines the external storage backend for sessions and state. Defaults to in-memory.
	Storage store.Storage

	// PubSub defines the messaging backend for multi-process broadcasting. Defaults to in-memory.
	PubSub store.PubSub
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

// ssgEntry holds a cached HTML page and when it was generated.
type ssgEntry struct {
	html      []byte
	createdAt time.Time
}

// encodeSsgEntry encodes an SSG entry into bytes for external storage.
func encodeSsgEntry(entry ssgEntry) []byte {
	buf := make([]byte, 8+len(entry.html))
	binary.LittleEndian.PutUint64(buf[0:8], uint64(entry.createdAt.UnixNano()))
	copy(buf[8:], entry.html)
	return buf
}

// decodeSsgEntry decodes bytes into an SSG entry.
func decodeSsgEntry(data []byte) (ssgEntry, bool) {
	if len(data) < 8 {
		return ssgEntry{}, false
	}
	createdAtNano := binary.LittleEndian.Uint64(data[0:8])
	return ssgEntry{
		html:      data[8:],
		createdAt: time.Unix(0, int64(createdAtNano)),
	}, true
}

// isrSemaphore limits concurrent ISR background revalidations.
var isrSemaphore = make(chan struct{}, 10)

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
	// ssgCache stores pre-rendered SSG and ISR pages.
	ssgCache map[string]ssgEntry
	// ssgCacheKeys tracks insertion order for FIFO eviction.
	ssgCacheKeys []string
	// ssgCacheMu protects ssgCache and ssgCacheKeys.
	ssgCacheMu sync.RWMutex
	// isrRevalidating guards against duplicate background revalidations.
	isrRevalidating sync.Map
	// pprShellCache stores cached static shells for PPR pages.
	pprShellCache map[string][]byte
	// pprShellKeys tracks insertion order for PPR shell FIFO eviction.
	pprShellKeys []string
	// pprShellMu protects pprShellCache and pprShellKeys.
	pprShellMu sync.RWMutex
	// pprShellBuilding guards against duplicate PPR shell builds under concurrent load.
	pprShellBuilding sync.Map
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

	// SSGCacheMaxEntries controls SSG/ISR/PPR cache size:
	//   - 0 or unset: use default of 500 entries (recommended for most apps)
	//   - -1: unlimited entries (no eviction) - NOT recommended in production
	//   - 1-10000: use specified value (values >10000 are capped at 10000)
	// Note: There is no "disable cache" option. To disable, use SSR strategy instead.
	if config.SSGCacheMaxEntries == 0 {
		config.SSGCacheMaxEntries = 500
	} else if config.SSGCacheMaxEntries < 0 {
		// Normalize all negative values to -1 for "unlimited"
		config.SSGCacheMaxEntries = -1
	} else if config.SSGCacheMaxEntries > 10000 {
		config.SSGCacheMaxEntries = 10000
	}

	if config.WSMaxMessageSize == 0 {
		config.WSMaxMessageSize = 64 * 1024
	}
	if config.WSConnRateLimit == 0 {
		config.WSConnRateLimit = 0.2
	}
	if config.WSConnBurst == 0 {
		config.WSConnBurst = 5.0
	}
	// Configure global rate limiter
	fiber.SetConnectionRateLimiter(config.WSConnBurst, config.WSConnRateLimit)

	if config.Storage == nil {
		if config.Prefork {
			log.Println("WARNING: Prefork is enabled with in-memory Storage. Sessions and State will be isolated per process!")
		}
		config.Storage = store.NewMemoryStorage()
	}
	if config.PubSub == nil {
		if config.Prefork {
			log.Println("WARNING: Prefork is enabled with in-memory PubSub. WebSocket broadcasts will be isolated per process!")
		}
		config.PubSub = store.NewMemoryPubSub()
	}

	// Update global stores (Optional: This is a side effect but keeps existing semantics working)
	fiber.InitStores(config.Storage)

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
		AppName:      config.AppName,
		Prefork:      config.Prefork,
		ServerHeader: "GoSPA",
	}
	if config.DevMode {
		log.Println("WARNING: DevMode is enabled. This exposes detailed stack traces and should be disabled in production.")
		fiberConfig.EnablePrintRoutes = true
		fiberConfig.ServerHeader = "GoSPA/" + Version
	}
	fiberApp := fiberpkg.New(fiberConfig)

	// Create WebSocket hub (always enabled by default - WebSocket is a core feature)
	// Note: Go can't distinguish between "unset" and "explicitly false" for bools,
	// so we always create the hub. Users who don't want WebSocket can simply not use it.
	hub := fiber.NewWSHub(config.PubSub)
	go hub.Run()

	// Create state map
	stateMap := state.NewStateMap()
	for k, v := range config.DefaultState {
		r := state.NewRune(v)
		stateMap.Add(k, r)
	}

	app := &App{
		Config:        config,
		Router:        router,
		Fiber:         fiberApp,
		Hub:           hub,
		StateMap:      stateMap,
		ssgCache:      make(map[string]ssgEntry),
		ssgCacheKeys:  make([]string, 0),
		pprShellCache: make(map[string][]byte),
		pprShellKeys:  make([]string, 0),
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
		// CSRF token setter must come BEFORE protection middleware
		// to ensure tokens are set on GET/HEAD requests before POST validation
		a.Fiber.Use(fiber.CSRFSetTokenMiddleware())
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

	// DevMode error overlay: for HTML-accepting requests that error, render the
	// rich HTML overlay instead of a plain JSON error.
	if a.Config.DevMode {
		overlay := fiber.NewErrorOverlay(fiber.DefaultErrorOverlayConfig())
		a.Fiber.Use(func(c *fiberpkg.Ctx) error {
			err := c.Next()
			if err == nil {
				return nil
			}
			accept := string(c.Request().Header.Peek("Accept"))
			if strings.Contains(accept, "text/html") {
				overlayHTML := overlay.RenderOverlay(err, nil)
				c.Status(fiberpkg.StatusInternalServerError)
				c.Set("Content-Type", "text/html; charset=utf-8")
				return c.SendString(overlayHTML)
			}
			return err
		})
	}
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
			Hub:              a.Hub,
			CompressState:    a.Config.CompressState,
			StateDiffing:     a.Config.StateDiffing,
			Serializer:       a.Config.StateSerializer,
			Deserializer:     a.Config.StateDeserializer,
			WSMaxMessageSize: a.Config.WSMaxMessageSize,
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
				"code":  "ACTION_NOT_FOUND",
			})
		}

		var input interface{}
		// Only parse if body is not empty
		if len(c.Body()) > 0 {
			if !strings.Contains(c.Get("Content-Type"), "application/json") {
				return c.Status(fiberpkg.StatusUnsupportedMediaType).JSON(fiberpkg.Map{
					"error": "Unsupported Media Type: expected application/json",
					"code":  "INVALID_CONTENT_TYPE",
				})
			}
			if len(c.Body()) > a.Config.MaxRequestBodySize {
				return c.Status(fiberpkg.StatusRequestEntityTooLarge).JSON(fiberpkg.Map{
					"error": "Request body too large",
					"code":  "REQUEST_TOO_LARGE",
				})
			}
			if err := c.BodyParser(&input); err != nil {
				return c.Status(fiberpkg.StatusBadRequest).JSON(fiberpkg.Map{
					"error": "Invalid input JSON",
					"code":  "INVALID_JSON",
				})
			}
		}

		result, err := fn(c.Context(), input)
		if err != nil {
			// Log the actual error internally for debugging
			log.Printf("Remote action %q error: %v", name, err)
			// Return error with code for programmatic handling
			return c.Status(fiberpkg.StatusInternalServerError).JSON(fiberpkg.Map{
				"error": err.Error(),
				"code":  "ACTION_FAILED",
			})
		}

		return c.JSON(fiberpkg.Map{
			"data": result,
			"code": "SUCCESS",
		})
	})

	// Static files
	if _, err := os.Stat(a.Config.StaticDir); err == nil {
		a.Fiber.Use(a.Config.StaticPrefix, filesystem.New(filesystem.Config{
			Root:   http.Dir(a.Config.StaticDir),
			MaxAge: 31536000,
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
	cacheKey := c.Path()
	if query := string(c.Request().URI().QueryString()); len(query) > 0 {
		cacheKey += "?" + query
	}
	opts := routing.GetRouteOptions(route.Path)

	// Resolve effective strategy (per-page opts override config default).
	effStrategy := opts.Strategy
	if effStrategy == "" {
		effStrategy = a.Config.DefaultRenderStrategy
	}
	if effStrategy == "" {
		effStrategy = routing.StrategySSR
	}

	// Quick SSG cache check — serve without building layout chain.
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

		if hit {
			c.Set("Content-Type", "text/html")
			c.Set("Cache-Control", "public, max-age=31536000, immutable")
			return c.Send(entry.html)
		}
	}

	// Quick ISR cache check — serve from cache (possibly stale) without rebuild.
	if a.Config.CacheTemplates && effStrategy == routing.StrategyISR {
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

		if hit {
			age := time.Since(entry.createdAt)
			if ttl > 0 && age >= ttl {
				// Stale: serve cached, kick off background revalidation.
				if _, alreadyRunning := a.isrRevalidating.LoadOrStore(cacheKey, true); !alreadyRunning {
					// Capture values for goroutine closure.
					routeSnap := route
					go func() {
						defer a.isrRevalidating.Delete(cacheKey)
						// Limit concurrent revalidations with semaphore
						select {
						case isrSemaphore <- struct{}{}:
							defer func() { <-isrSemaphore }()
						default:
							// Too many concurrent revalidations, skip this one
							return
						}
						freshHTML, err := a.buildPageHTML(context.Background(), routeSnap, nil)
						if err != nil {
							log.Printf("ISR background render error for %s: %v", cacheKey, err)
							return
						}
						a.storeSsgEntry(cacheKey, freshHTML)
					}()
				}
			}
			c.Set("Content-Type", "text/html")
			c.Set("Cache-Control", fmt.Sprintf("public, s-maxage=%d, stale-while-revalidate=%d", ttlSec, ttlSec))
			return c.Send(entry.html)
		}
	}

	// Quick PPR shell check.
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
			shell, shellHit = a.pprShellCache[cacheKey]
			a.pprShellMu.RUnlock()
		}

		if shellHit {
			result, err := a.applyPPRSlots(route, shell, c.Path(), opts)
			if err != nil {
				log.Printf("PPR slot error: %v", err)
			}
			c.Set("Content-Type", "text/html")
			c.Set("Cache-Control", "no-store")
			return c.Send(result)
		}
	}

	// ── Build layout chain & full component tree ─────────────────────────
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

		// ─── SSG ─────────────────────────────────────────────────────────
		if a.Config.CacheTemplates && effStrategy == routing.StrategySSG {
			var buf bytes.Buffer
			if err := wrappedContent.Render(ctx, &buf); err != nil {
				log.Printf("SSG render error: %v", err)
				return c.Status(fiberpkg.StatusInternalServerError).SendString("Render error")
			}
			a.storeSsgEntry(cacheKey, buf.Bytes())
			c.Set("Cache-Control", "public, max-age=31536000, immutable")
			return c.Send(buf.Bytes())
		}

		// ─── ISR (cache miss) ─────────────────────────────────────────────
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
				log.Printf("ISR render error: %v", err)
				return c.Status(fiberpkg.StatusInternalServerError).SendString("Render error")
			}
			a.storeSsgEntry(cacheKey, buf.Bytes())
			c.Set("Cache-Control", fmt.Sprintf("public, s-maxage=%d, stale-while-revalidate=%d", ttlSec, ttlSec))
			return c.Send(buf.Bytes())
		}

		// ─── PPR (shell miss) ─────────────────────────────────────────────
		if a.Config.CacheTemplates && effStrategy == routing.StrategyPPR {
			// Use sync.Map for deduplication to prevent thundering herd
			done := make(chan struct{})
			actual, loaded := a.pprShellBuilding.LoadOrStore(cacheKey, done)
			if !loaded {
				defer func() {
					close(done)
					a.pprShellBuilding.Delete(cacheKey)
				}()
				shellCtx := templpkg.WithPPRShellBuild(context.Background())
				var shellBuf bytes.Buffer
				if err := wrappedContent.Render(shellCtx, &shellBuf); err != nil {
					log.Printf("PPR shell render error: %v", err)
					return c.Status(fiberpkg.StatusInternalServerError).SendString("Render error")
				}
				a.storePprShell(cacheKey, shellBuf.Bytes())
				result, err := a.applyPPRSlots(route, shellBuf.Bytes(), c.Path(), opts)
				if err != nil {
					log.Printf("PPR slot error: %v", err)
					return c.Status(fiberpkg.StatusInternalServerError).SendString("Render error")
				}
				c.Set("Cache-Control", "no-store")
				return c.Send(result)
			} else {
				// Another goroutine is building; wait for it to finish
				waitChan := actual.(chan struct{})
				<-waitChan

				var shellHTML []byte
				var shellOk bool
				if a.Config.Storage != nil && !a.Config.Prefork {
					if data, err := a.Config.Storage.Get("gospa:ppr:" + cacheKey); err == nil {
						shellHTML = data
						shellOk = true
					}
				} else {
					a.pprShellMu.RLock()
					shellHTML, shellOk = a.pprShellCache[cacheKey]
					a.pprShellMu.RUnlock()
				}
				if shellOk {
					result, err := a.applyPPRSlots(route, shellHTML, c.Path(), opts)
					if err != nil {
						log.Printf("PPR slot error: %v", err)
						return c.Status(fiberpkg.StatusInternalServerError).SendString("Render error")
					}
					c.Set("Cache-Control", "no-store")
					return c.Send(result)
				}
				// Shell still not ready, fall back to SSR (render inline)
				var fallbackBuf bytes.Buffer
				if err := wrappedContent.Render(c.Context(), &fallbackBuf); err != nil {
					log.Printf("PPR fallback render error: %v", err)
					return c.Status(fiberpkg.StatusInternalServerError).SendString("Render error")
				}
				c.Set("Cache-Control", "no-store")
				return c.Send(fallbackBuf.Bytes())
			}
		}

		// ─── SSR (default) ────────────────────────────────────────────────
		c.Set("Cache-Control", "no-store")
		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			if err := wrappedContent.Render(ctx, w); err != nil {
				log.Printf("Streaming render error: %v", err)
			}
			w.Flush()
		})
		return nil
	}

	// No root layout registered — minimal fallback HTML wrapper.
	protocol := "ws://"
	if c.Secure() {
		protocol = "wss://"
	}
	wsUrl := protocol + string(c.Request().Host()) + a.Config.WebSocketPath
	runtimePath := a.getRuntimePath()
	appName := a.Config.AppName
	devMode := a.Config.DevMode

	wsReconnectDelay := int(a.Config.WSReconnectDelay.Milliseconds())
	if wsReconnectDelay <= 0 {
		wsReconnectDelay = 1000
	}
	wsMaxReconnect := a.Config.WSMaxReconnect
	if wsMaxReconnect <= 0 {
		wsMaxReconnect = 10
	}
	wsHeartbeat := int(a.Config.WSHeartbeat.Milliseconds())
	if wsHeartbeat <= 0 {
		wsHeartbeat = 30000
	}

	c.Set("Cache-Control", "no-store")
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		_, _ = fmt.Fprint(w, `<!DOCTYPE html><html lang="en" data-gospa-auto><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>`)
		_, _ = fmt.Fprint(w, appName)
		_, _ = fmt.Fprint(w, `</title></head><body><div id="app" data-gospa-root><main>`)
		if err := content.Render(ctx, w); err != nil {
			log.Printf("Streaming render error: %v", err)
		}
		_, _ = fmt.Fprint(w, `</main></div>`)
		_, _ = fmt.Fprintf(w, `<script src="%s" type="module"></script>`, runtimePath)
		_, _ = fmt.Fprintf(w, `<script type="module">
import * as runtime from '%s';
runtime.init({
	wsUrl: '%s',
	debug: %v,
	simpleRuntimeSVGs: %v,
	wsReconnectDelay: %d,
	wsMaxReconnect: %d,
	wsHeartbeat: %d,
	hydration: {
		mode: '%s',
		timeout: %d
	}
});
</script>`, runtimePath, wsUrl, devMode, a.Config.SimpleRuntimeSVGs, wsReconnectDelay, wsMaxReconnect, wsHeartbeat, a.Config.HydrationMode, a.Config.HydrationTimeout)
		_, _ = fmt.Fprint(w, `</body></html>`)
		w.Flush()
	})
	return nil
}

// buildPageContent builds the innermost page templ.Component for a route.
func (a *App) buildPageContent(route *routing.Route, params map[string]string, path string) templ.Component {
	pageFunc := routing.GetPage(route.Path)
	if pageFunc != nil {
		props := map[string]interface{}{"path": path}
		for k, v := range params {
			props[k] = v
		}
		return pageFunc(props)
	}
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, _ = fmt.Fprintf(w, `<div data-gospa-page="%s">Page: %s</div>`, route.Path, route.Path)
		return nil
	})
}

// wrapWithLayouts wraps a component with every layout in the chain (innermost → outermost).
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

// buildRootLayoutProps assembles the props map expected by the root layout function.
func (a *App) buildRootLayoutProps(c *fiberpkg.Ctx, params map[string]string) map[string]interface{} {
	wsRD := int(a.Config.WSReconnectDelay.Milliseconds())
	if wsRD <= 0 {
		wsRD = 1000
	}
	wsMR := a.Config.WSMaxReconnect
	if wsMR <= 0 {
		wsMR = 10
	}
	wsHB := int(a.Config.WSHeartbeat.Milliseconds())
	if wsHB <= 0 {
		wsHB = 30000
	}
	props := map[string]interface{}{
		"appName":          a.Config.AppName,
		"runtimePath":      a.getRuntimePath(),
		"path":             c.Path(),
		"debug":            a.Config.DevMode,
		"wsUrl":            a.getWSUrl(c),
		"hydrationMode":    a.Config.HydrationMode,
		"hydrationTimeout": a.Config.HydrationTimeout,
		"wsReconnectDelay": wsRD,
		"wsMaxReconnect":   wsMR,
		"wsHeartbeat":      wsHB,
	}
	for k, v := range params {
		props[k] = v
	}
	return props
}

// buildPageHTML renders a complete page to bytes using a background context.
// It is called by the ISR background revalidation goroutine.
// params may be nil for pages without dynamic segments.
func (a *App) buildPageHTML(ctx context.Context, route *routing.Route, params map[string]string) ([]byte, error) {
	layouts := a.Router.ResolveLayoutChain(route)
	if params == nil {
		params = map[string]string{}
	}

	// Build a synthetic path (no query string needed for revalidation).
	path := route.Path

	content := a.buildPageContent(route, params, path)
	content = a.wrapWithLayouts(content, layouts, params, path)

	rootLayoutFunc := routing.GetRootLayout()
	if rootLayoutFunc == nil {
		// No root layout: render just the inner content.
		var buf bytes.Buffer
		if err := content.Render(ctx, &buf); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	// Build minimal root props (no live Fiber ctx available in background goroutine).
	wsRD := int(a.Config.WSReconnectDelay.Milliseconds())
	if wsRD <= 0 {
		wsRD = 1000
	}
	wsMR := a.Config.WSMaxReconnect
	if wsMR <= 0 {
		wsMR = 10
	}
	wsHB := int(a.Config.WSHeartbeat.Milliseconds())
	if wsHB <= 0 {
		wsHB = 30000
	}
	rootProps := map[string]interface{}{
		"appName":          a.Config.AppName,
		"runtimePath":      a.getRuntimePath(),
		"path":             path,
		"debug":            false,
		"wsUrl":            a.Config.WebSocketPath,
		"hydrationMode":    a.Config.HydrationMode,
		"hydrationTimeout": a.Config.HydrationTimeout,
		"wsReconnectDelay": wsRD,
		"wsMaxReconnect":   wsMR,
		"wsHeartbeat":      wsHB,
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

// storeSsgEntry writes html into the shared ssgCache with FIFO eviction.
func (a *App) storeSsgEntry(key string, html []byte) {
	if a.Config.Storage != nil && !a.Config.Prefork {
		entry := ssgEntry{html: html, createdAt: time.Now()}
		_ = a.Config.Storage.Set("gospa:ssg:"+key, encodeSsgEntry(entry), 0)
		return
	}

	a.ssgCacheMu.Lock()
	defer a.ssgCacheMu.Unlock()
	maxEntries := a.Config.SSGCacheMaxEntries
	if maxEntries == 0 {
		maxEntries = 500
	}
	// Only evict if maxEntries > 0 (-1 means unlimited)
	if maxEntries > 0 && len(a.ssgCache) >= maxEntries && len(a.ssgCacheKeys) > 0 {
		oldest := a.ssgCacheKeys[0]
		a.ssgCacheKeys = a.ssgCacheKeys[1:]
		delete(a.ssgCache, oldest)
	}
	if _, exists := a.ssgCache[key]; !exists {
		a.ssgCacheKeys = append(a.ssgCacheKeys, key)
	}
	a.ssgCache[key] = ssgEntry{html: html, createdAt: time.Now()}

	if a.Config.SSGCacheTTL > 0 {
		time.AfterFunc(a.Config.SSGCacheTTL, func() {
			a.ssgCacheMu.Lock()
			defer a.ssgCacheMu.Unlock()
			delete(a.ssgCache, key)
		})
	}
}

// storePprShell writes a PPR static shell into the pprShellCache with FIFO eviction.
func (a *App) storePprShell(key string, shell []byte) {
	if a.Config.Storage != nil && !a.Config.Prefork {
		_ = a.Config.Storage.Set("gospa:ppr:"+key, shell, 0)
		return
	}

	a.pprShellMu.Lock()
	defer a.pprShellMu.Unlock()
	maxEntries := a.Config.SSGCacheMaxEntries
	if maxEntries == 0 {
		maxEntries = 500
	}
	// Only evict if maxEntries > 0 (-1 means unlimited)
	if maxEntries > 0 && len(a.pprShellCache) >= maxEntries && len(a.pprShellKeys) > 0 {
		oldest := a.pprShellKeys[0]
		a.pprShellKeys = a.pprShellKeys[1:]
		delete(a.pprShellCache, oldest)
	}
	if _, exists := a.pprShellCache[key]; !exists {
		a.pprShellKeys = append(a.pprShellKeys, key)
	}
	a.pprShellCache[key] = shell
}

// applyPPRSlots renders each named dynamic slot and splices it into the static
// shell by replacing <!--gospa-slot:name--> placeholders.
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
			log.Printf("PPR slot %q render error: %v", slotName, err)
			continue
		}
		placeholder := []byte(templpkg.SlotPlaceholder(slotName))
		open := []byte(fmt.Sprintf(`<div data-gospa-slot="%s">`, slotName))
		close := []byte(`</div>`)
		replacement := append(open, append(slotBuf.Bytes(), close...)...)
		result = bytes.ReplaceAll(result, placeholder, replacement)
	}
	return result, nil
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
