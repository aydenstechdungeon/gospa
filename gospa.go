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
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	json "github.com/goccy/go-json"

	templ "github.com/a-h/templ"
	"github.com/aydenstechdungeon/gospa/embed"
	"github.com/aydenstechdungeon/gospa/fiber"
	"github.com/aydenstechdungeon/gospa/routing"
	"github.com/aydenstechdungeon/gospa/state"
	"github.com/aydenstechdungeon/gospa/store"
	templpkg "github.com/aydenstechdungeon/gospa/templ"
	fiberpkg "github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/logger"
	recovermw "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/valyala/fasthttp"
)

// StateSerializerFunc defines a function for state serialization
type StateSerializerFunc func(interface{}) ([]byte, error)

// StateDeserializerFunc defines a function for state deserialization
type StateDeserializerFunc func([]byte, interface{}) error

// Version is the current version of GoSPA.
const Version = "0.1.27"

// Serialization formats
const (
	SerializationJSON    = "json"
	SerializationMsgPack = "msgpack"
)

// NavigationSpeculativePrefetchingConfig configures speculative prefetching
type NavigationSpeculativePrefetchingConfig struct {
	Enabled        *bool `json:"enabled,omitempty"`
	TTL            *int  `json:"ttl,omitempty"`
	HoverDelay     *int  `json:"hoverDelay,omitempty"`
	ViewportMargin *int  `json:"viewportMargin,omitempty"`
}

// NavigationURLParsingCacheConfig configures the URL parsing cache
type NavigationURLParsingCacheConfig struct {
	Enabled *bool `json:"enabled,omitempty"`
	MaxSize *int  `json:"maxSize,omitempty"`
	TTL     *int  `json:"ttl,omitempty"`
}

// NavigationIdleCallbackBatchUpdatesConfig configures idle callback batching
type NavigationIdleCallbackBatchUpdatesConfig struct {
	Enabled             *bool `json:"enabled,omitempty"`
	FallbackToMicrotask *bool `json:"fallbackToMicrotask,omitempty"`
}

// NavigationLazyRuntimeInitializationConfig configures lazy runtime init
type NavigationLazyRuntimeInitializationConfig struct {
	Enabled       *bool `json:"enabled,omitempty"`
	DeferBindings *bool `json:"deferBindings,omitempty"`
}

// NavigationServiceWorkerCachingConfig configures service worker caching
type NavigationServiceWorkerCachingConfig struct {
	Enabled   *bool  `json:"enabled,omitempty"`
	CacheName string `json:"cacheName,omitempty"`
	Path      string `json:"path,omitempty"`
}

// NavigationViewTransitionsConfig configures view transitions
type NavigationViewTransitionsConfig struct {
	Enabled           *bool `json:"enabled,omitempty"`
	FallbackToClassic *bool `json:"fallbackToClassic,omitempty"`
}

// NavigationOptions configures client-side navigation
type NavigationOptions struct {
	SpeculativePrefetching         *NavigationSpeculativePrefetchingConfig    `json:"speculativePrefetching,omitempty"`
	URLParsingCache                *NavigationURLParsingCacheConfig           `json:"urlParsingCache,omitempty"`
	IdleCallbackBatchUpdates       *NavigationIdleCallbackBatchUpdatesConfig  `json:"idleCallbackBatchUpdates,omitempty"`
	LazyRuntimeInitialization      *NavigationLazyRuntimeInitializationConfig `json:"lazyRuntimeInitialization,omitempty"`
	ServiceWorkerNavigationCaching *NavigationServiceWorkerCachingConfig      `json:"serviceWorkerNavigationCaching,omitempty"`
	ViewTransitions                *NavigationViewTransitionsConfig           `json:"viewTransitions,omitempty"`
}

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
	// Logger is the structured logger. Defaults to slog.Default().
	Logger *slog.Logger

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
	// DisableSanitization disables client-side HTML sanitization for SPA navigation.
	// When enabled, GoSPA trusts server-rendered HTML without DOMPurify filtering.
	// This provides a SvelteKit-like experience but requires careful handling of
	// user-generated content. Use with caution - only for trusted content.
	DisableSanitization bool

	// WebSocket Options — these values are passed directly to the client runtime's init() call.
	// Defaults: WSReconnectDelay=1s, WSMaxReconnect=10, WSHeartbeat=30s.
	WSReconnectDelay time.Duration // Initial reconnect delay (default 1s)
	WSMaxReconnect   int           // Max reconnect attempts (default 10)
	WSHeartbeat      time.Duration // Heartbeat ping interval (default 30s)

	// WSMaxMessageSize limits the maximum payload size for WebSocket messages (default 64KB).
	WSMaxMessageSize int
	// WSConnRateLimit sets the refilling rate in connections per second for WebSocket upgrades (default 1.5).
	WSConnRateLimit float64
	// WSConnBurst sets the burst capacity for WebSocket connection upgrades (default 15.0).
	WSConnBurst float64

	// Hydration Options
	// HydrationMode controls when components become interactive.
	// Supported values: "immediate" | "lazy" | "visible" | "idle" (default: "immediate").
	HydrationMode    string
	HydrationTimeout int // ms before force hydrate (used with "visible" and "idle" modes)

	// Serialization Options
	// SerializationFormat sets the underlying format for all WebSocket communications.
	// Supported values: "json" (default, using goccy/go-json) | "msgpack".
	SerializationFormat string
	// StateSerializer overrides the default state serialization for outbound WebSocket payloads.
	// StateDeserializer overrides the default state deserialization for inbound WebSocket payloads.
	StateSerializer   StateSerializerFunc
	StateDeserializer StateDeserializerFunc

	// Routing Options
	DisableSPA bool // Disable SPA navigation completely

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
	// RemoteActionMiddleware allows injecting authorization/tenant checks before remote action handlers.
	RemoteActionMiddleware fiberpkg.Handler
	// AllowUnauthenticatedRemoteActions disables the production safety guard that blocks
	// remote actions when no RemoteActionMiddleware is configured.
	// Default false (secure-by-default).
	AllowUnauthenticatedRemoteActions bool

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

	// NavigationOptions configures optional client-side navigation behavior and performance optimizations.
	NavigationOptions NavigationOptions
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		RoutesDir:           "./routes",
		DevMode:             false,
		RuntimeScript:       "/_gospa/runtime.js",
		StaticDir:           "./static",
		StaticPrefix:        "/static",
		AppName:             "GoSPA App",
		DefaultState:        make(map[string]interface{}),
		EnableWebSocket:     true,
		WebSocketPath:       "/_gospa/ws",
		RemotePrefix:        "/_gospa/remote",
		MaxRequestBodySize:  4 * 1024 * 1024, // Default 4MB
		SerializationFormat: SerializationJSON,
		EnableCSRF:          true,
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
func (a *App) getWSUrl(c fiberpkg.Ctx) string {
	protocol := "ws://"
	// Check Protocol() which respects Fiber's ProxyHeader config,
	// and fallback to checking X-Forwarded-Proto explicitly for common proxy setups.
	if c.Protocol() == "https" || strings.ToLower(c.Get("X-Forwarded-Proto")) == "https" {
		protocol = "wss://"
	}
	return protocol + string(c.Request().Host()) + a.Config.WebSocketPath
}

// ssgEntry holds a cached HTML page and when it was generated.
type ssgEntry struct {
	html      []byte
	createdAt time.Time
}

type pprEntry struct {
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
		createdAt: time.Unix(0, int64(createdAtNano)), //nolint:gosec
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
	pprShellCache map[string]pprEntry
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
	switch {
	case config.SSGCacheMaxEntries == 0:
		config.SSGCacheMaxEntries = 500
	case config.SSGCacheMaxEntries < 0:
		// Normalize all negative values to -1 for "unlimited"
		config.SSGCacheMaxEntries = -1
	case config.SSGCacheMaxEntries > 10000:
		config.SSGCacheMaxEntries = 10000
	}

	// Default hydration mode: "immediate" means components hydrate as soon as the
	// runtime is ready. Other valid values: "lazy", "visible", "idle".
	if config.HydrationMode == "" {
		config.HydrationMode = "immediate"
	}

	if config.WSMaxMessageSize == 0 {
		config.WSMaxMessageSize = 64 * 1024
	}

	if config.WSConnRateLimit == 0 {
		config.WSConnRateLimit = 1.5
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.WSConnBurst == 0 {
		config.WSConnBurst = 15.0
	}
	// Configure global rate limiter
	fiber.SetConnectionRateLimiter(config.WSConnBurst, config.WSConnRateLimit)

	if config.Storage == nil {
		if config.Prefork {
			config.Logger.Warn("Prefork enabled with in-memory Storage: sessions will NOT be shared between processes")
		}
		config.Storage = store.NewMemoryStorage()
	}
	if config.PubSub == nil {
		if config.Prefork {
			config.Logger.Warn("Prefork enabled with in-memory PubSub: WebSocket broadcasts will NOT work across processes")
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
		ServerHeader: "GoSPA",
	}
	if config.DevMode {
		config.Logger.Warn("DevMode is enabled — disable in production")
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
		pprShellCache: make(map[string]pprEntry),
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
	a.Fiber.Use(recovermw.New(recovermw.Config{
		EnableStackTrace: true,
	}))

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

	// Preload critical assets before reaching the SPA middleware to ensure headers are set on HTML
	preloadConfig := fiber.DefaultPreloadConfig()
	// Sync with dynamic runtime path
	preloadConfig.RuntimeScript = a.getRuntimePath()
	a.Fiber.Use(fiber.PreloadHeadersMiddleware(preloadConfig))

	// SPA middleware
	spaConfig := fiber.DefaultConfig()
	spaConfig.DevMode = a.Config.DevMode
	spaConfig.RuntimeScript = a.Config.RuntimeScript
	a.Fiber.Use(fiber.SPAMiddleware(spaConfig))

	// DevMode error overlay
	if a.Config.DevMode {
		overlay := fiber.NewErrorOverlay(fiber.DefaultErrorOverlayConfig())
		a.Fiber.Use(func(c fiberpkg.Ctx) error {
			err := c.Next()
			if err == nil {
				return nil
			}
			accept := c.Get("Accept")
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
	a.Fiber.Use("/_gospa/", func(c fiberpkg.Ctx) error {
		c.Set("Cache-Control", "public, max-age=31536000, immutable")
		return c.Next()
	})
	a.Fiber.Use("/_gospa/", static.New("", static.Config{
		FS: embed.RuntimeFS(),
	}))

	// WebSocket endpoint (always registered since hub is always created)
	if a.Hub != nil {
		handlers := []fiberpkg.Handler{}
		// SECURITY: Always enforce per-IP rate limiting before the WebSocket upgrade.
		// This prevents a flood of connections from exhausting the hub's Clients map
		// before any application-level auth/middleware has a chance to run.
		handlers = append(handlers, fiber.WebSocketUpgradeMiddleware())
		if a.Config.WebSocketMiddleware != nil {
			handlers = append(handlers, a.Config.WebSocketMiddleware)
		}
		handlers = append(handlers, fiber.WebSocketHandler(fiber.WebSocketConfig{
			Hub:                 a.Hub,
			CompressState:       a.Config.CompressState,
			StateDiffing:        a.Config.StateDiffing,
			Serializer:          a.Config.StateSerializer,
			Deserializer:        a.Config.StateDeserializer,
			SerializationFormat: a.Config.SerializationFormat,
			WSMaxMessageSize:    a.Config.WSMaxMessageSize,
		}))
		// In Fiber v3 Get/Post/etc. take (path, handler any, ...any), not (path, ...Handler).
		// Build a []any from our handlers slice and spread.
		hAny := make([]any, len(handlers))
		for i, h := range handlers {
			hAny[i] = h
		}
		a.Fiber.Get(a.Config.WebSocketPath, hAny[0], hAny[1:]...)
	}

	// Remote Actions endpoint
	remoteHandlers := []fiberpkg.Handler{fiber.RemoteActionRateLimitMiddleware()}
	if !a.Config.DevMode && a.Config.RemoteActionMiddleware == nil && !a.Config.AllowUnauthenticatedRemoteActions {
		remoteHandlers = append(remoteHandlers, func(c fiberpkg.Ctx) error {
			return c.Status(fiberpkg.StatusUnauthorized).JSON(fiberpkg.Map{
				"error": "Remote actions require RemoteActionMiddleware in production",
				"code":  "REMOTE_AUTH_REQUIRED",
			})
		})
	}
	if a.Config.RemoteActionMiddleware != nil {
		remoteHandlers = append(remoteHandlers, a.Config.RemoteActionMiddleware)
	}
	remoteHandlers = append(remoteHandlers, func(c fiberpkg.Ctx) error {
		name := c.Params("name")
		if len(name) > 256 {
			return c.Status(fiberpkg.StatusBadRequest).JSON(fiberpkg.Map{
				"error": "Action name too long",
				"code":  "INVALID_ACTION_NAME",
			})
		}
		fn, ok := routing.GetRemoteAction(name)
		if !ok {
			return c.Status(fiberpkg.StatusNotFound).JSON(fiberpkg.Map{
				"error": "Remote action not found",
				"code":  "ACTION_NOT_FOUND",
			})
		}

		var input interface{}
		// Check request body size before materializing it.
		// ContentLength might be -1 for chunked transfer, which Fiber will handle via BodyLimit.
		if contentLength := c.Request().Header.ContentLength(); contentLength > a.Config.MaxRequestBodySize {
			return c.Status(fiberpkg.StatusRequestEntityTooLarge).JSON(fiberpkg.Map{
				"error": "Request body too large",
				"code":  "REQUEST_TOO_LARGE",
			})
		}

		// Only parse if body is not empty
		if body := c.Body(); len(body) > 0 {
			if !strings.Contains(c.Get("Content-Type"), "application/json") {
				return c.Status(fiberpkg.StatusUnsupportedMediaType).JSON(fiberpkg.Map{
					"error": "Unsupported Media Type: expected application/json",
					"code":  "INVALID_CONTENT_TYPE",
				})
			}
			if len(body) > a.Config.MaxRequestBodySize {
				return c.Status(fiberpkg.StatusRequestEntityTooLarge).JSON(fiberpkg.Map{
					"error": "Request body too large",
					"code":  "REQUEST_TOO_LARGE",
				})
			}
			if err := json.Unmarshal(c.Body(), &input); err != nil {
				return c.Status(fiberpkg.StatusBadRequest).JSON(fiberpkg.Map{
					"error": "Invalid input JSON",
					"code":  "INVALID_JSON",
				})
			}
		}

		// Extract tracing headers directly without copying the lock value
		headers := make(map[string]string, 4)
		if requestID := string(c.Request().Header.Peek("X-Request-Id")); requestID != "" {
			headers["X-Request-Id"] = requestID
		}
		if traceParent := string(c.Request().Header.Peek("Traceparent")); traceParent != "" {
			headers["Traceparent"] = traceParent
		}
		if traceState := string(c.Request().Header.Peek("Tracestate")); traceState != "" {
			headers["Tracestate"] = traceState
		}
		if b3 := string(c.Request().Header.Peek("B3")); b3 != "" {
			headers["B3"] = b3
		}

		rc := routing.RemoteContext{
			IP:        c.IP(),
			UserAgent: string(c.Request().Header.UserAgent()),
			RequestID: c.GetRespHeader("X-Request-Id"),
			SessionID: c.Get("X-Session-Id"),
			Headers:   headers,
		}

		result, err := fn(c.Context(), rc, input)
		if err != nil {
			// Log the actual error internally
			a.Logger().Error("remote action error", "action", name, "err", err)
			// SECURITY: Return a generic message to prevent internal detail leakage
			// (DB strings, file paths, stack info, etc. must not reach the browser)
			return c.Status(fiberpkg.StatusInternalServerError).JSON(fiberpkg.Map{
				"error": "Internal server error",
				"code":  "ACTION_FAILED",
			})
		}

		return c.JSON(fiberpkg.Map{
			"data": result,
			"code": "SUCCESS",
		})
	})
	// Convert remoteHandlers ([]Handler) to []any for v3 routing
	rhAny := make([]any, len(remoteHandlers))
	for i, h := range remoteHandlers {
		rhAny[i] = h
	}
	a.Fiber.Post(a.Config.RemotePrefix+"/:name", rhAny[0], rhAny[1:]...)

	// Static files
	if _, err := os.Stat(a.Config.StaticDir); err == nil {
		a.Fiber.Use(a.Config.StaticPrefix, static.New(a.Config.StaticDir))
		// Serve favicon from static dir if requested at root
		a.Fiber.Get("/favicon.ico", func(c fiberpkg.Ctx) error {
			favPath := a.Config.StaticDir + "/favicon.ico"
			if _, err := os.Stat(favPath); err == nil {
				return c.SendFile(favPath)
			}
			return c.SendStatus(fiberpkg.StatusNoContent)
		})
	} else {
		// Prevent 404 errors for default favicon requests
		a.Fiber.Get("/favicon.ico", func(c fiberpkg.Ctx) error {
			return c.SendStatus(fiberpkg.StatusNoContent)
		})
	}
}

// Scan scans the routes directory and builds the route tree.
func (a *App) Scan() error {
	return a.Router.Scan()
}

// Logger returns the configured logger.
func (a *App) Logger() *slog.Logger {
	if a.Config.Logger != nil {
		return a.Config.Logger
	}
	return slog.Default()
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
		opts := routing.GetRouteOptions(r.Path)

		// Setup route-specific rate limiter if configured
		var handlers []any
		if opts.RateLimit != nil {
			rl := fiber.NewConnectionRateLimiter(a.Config.Storage)
			windowSecs := opts.RateLimit.Window.Seconds()
			if windowSecs <= 0 {
				windowSecs = 1
			}
			rl.SetLimits(float64(opts.RateLimit.MaxRequests), float64(opts.RateLimit.MaxRequests)/windowSecs)
			msg := opts.RateLimit.Message
			if msg == "" {
				msg = "Too many requests"
			}

			handlers = append(handlers, func(c fiberpkg.Ctx) error {
				if !rl.Allow(c.IP()) {
					return c.Status(fiberpkg.StatusTooManyRequests).SendString(msg)
				}
				return c.Next()
			})
		}

		// Apply middlewares globally registered for this route
		mws := a.Router.ResolveMiddlewareChain(r)
		for _, mwRoute := range mws {
			if fn := routing.GetMiddleware(mwRoute.Path); fn != nil {
				if mwHandler, ok := fn.(func(fiberpkg.Ctx) error); ok {
					handlers = append(handlers, mwHandler)
				} else if mwHandler, ok := fn.(fiberpkg.Handler); ok {
					handlers = append(handlers, mwHandler)
				} else {
					a.Logger().Error("skipping invalid middleware signature", "path", mwRoute.Path)
				}
			}
		}

		handlers = append(handlers, func(c fiberpkg.Ctx) error {
			return a.renderRoute(c, r)
		})

		a.Fiber.Get(r.Path, handlers[0], handlers[1:]...)
	}

	return nil
}

// renderError renders the appropriate error boundary for a path.
func (a *App) renderError(c fiberpkg.Ctx, statusCode int, errToDisplay error) error {
	path := c.Path()
	errRoute := a.Router.GetErrorRoute(path)
	if errRoute == nil {
		return c.Status(statusCode).SendString(errToDisplay.Error())
	}

	errCompFn := routing.GetError(errRoute.Path)
	if errCompFn == nil {
		// Fallback to basic string if error comp isn't registered
		return c.Status(statusCode).SendString(errToDisplay.Error())
	}

	props := map[string]interface{}{
		"error": errToDisplay.Error(),
		"code":  statusCode,
		"path":  path,
	}

	content := errCompFn(props)
	params := make(map[string]string)

	// Apply layouts to the error content
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

// renderRoute renders a route with its layout chain.
func (a *App) renderRoute(c fiberpkg.Ctx, route *routing.Route) error {
	cacheKey := c.Path()
	// SECURITY (P1): Removed query string from cache key to prevent cache-busting DoS attacks.
	// An attacker spamming /path?random=1..n would fill the FIFO cache with junk keys,
	// evicting all legitimate cached entries and forcing full SSR.
	// Since static shells are identical regardless of query parameters, we only cache by path.

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
			// Check TTL on access to avoid leaky goroutines
			if a.Config.SSGCacheTTL > 0 && time.Since(entry.createdAt) >= a.Config.SSGCacheTTL {
				hit = false
			}
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
			// Check global TTL on access
			if a.Config.SSGCacheTTL > 0 && time.Since(entry.createdAt) >= a.Config.SSGCacheTTL {
				hit = false
			}
		}

		if hit {
			age := time.Since(entry.createdAt)
			if ttl > 0 && age >= ttl {
				// Stale: serve cached, kick off background revalidation.
				if _, alreadyRunning := a.isrRevalidating.LoadOrStore(cacheKey, true); !alreadyRunning {
					// Capture values for goroutine closure.
					routeSnap := route
					//nolint:gosec // Background revalidation intentionally detaches from request context
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
						// Use context.Background() because the request context will be cancelled after the response is sent
						freshHTML, err := a.buildPageHTML(context.Background(), routeSnap, nil)
						if err != nil {
							a.Logger().Error("ISR background render error", "path", cacheKey, "err", err)
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
			p, hit := a.pprShellCache[cacheKey]
			if hit {
				// Check TTL on access
				if a.Config.SSGCacheTTL <= 0 || time.Since(p.createdAt) < a.Config.SSGCacheTTL {
					shell = p.html
					shellHit = true
				}
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
				a.Logger().Error("SSG render error", "err", err)
				return a.renderError(c, fiberpkg.StatusInternalServerError, err)
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
				a.Logger().Error("ISR render error", "err", err)
				return a.renderError(c, fiberpkg.StatusInternalServerError, err)
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

				// 3e: If a loading component exists, use it as the PPR shell content
				shellContent := wrappedContent
				if loadingFn := routing.GetLoading(route.Path); loadingFn != nil {
					props := map[string]interface{}{}
					ld := loadingFn(props)
					ld = a.wrapWithLayouts(ld, layouts, params, c.Path())
					rootProps := a.buildRootLayoutProps(c, params)
					shellContent = rootLayoutFunc(ld, rootProps)
				}

				var shellBuf bytes.Buffer
				if err := shellContent.Render(shellCtx, &shellBuf); err != nil {
					a.Logger().Error("PPR shell render error", "err", err)
					return a.renderError(c, fiberpkg.StatusInternalServerError, err)
				}
				a.storePprShell(cacheKey, shellBuf.Bytes())
				result, err := a.applyPPRSlots(route, shellBuf.Bytes(), c.Path(), opts)
				if err != nil {
					a.Logger().Error("PPR slot error", "err", err)
					return a.renderError(c, fiberpkg.StatusInternalServerError, err)
				}
				c.Set("Cache-Control", "no-store")
				return c.Send(result)
			}
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
				p, hit := a.pprShellCache[cacheKey]
				if hit {
					if a.Config.SSGCacheTTL <= 0 || time.Since(p.createdAt) < a.Config.SSGCacheTTL {
						shellHTML = p.html
						shellOk = true
					}
				}
				a.pprShellMu.RUnlock()
			}
			if shellOk {
				result, err := a.applyPPRSlots(route, shellHTML, c.Path(), opts)
				if err != nil {
					a.Logger().Error("PPR slot error", "err", err)
					return a.renderError(c, fiberpkg.StatusInternalServerError, err)
				}
				c.Set("Cache-Control", "no-store")
				return c.Send(result)
			}
			// Shell still not ready, fall back to SSR (render inline)
			var fallbackBuf bytes.Buffer
			if err := wrappedContent.Render(c.Context(), &fallbackBuf); err != nil {
				a.Logger().Error("PPR fallback render error", "err", err)
				return a.renderError(c, fiberpkg.StatusInternalServerError, err)
			}
			c.Set("Cache-Control", "no-store")
			return c.Send(fallbackBuf.Bytes())
		}

		// ─── SSR (default) ────────────────────────────────────────────────
		errRoute := a.Router.GetErrorRoute(c.Path())
		if errRoute != nil {
			// If error boundary exists, we must buffer instead of streaming to catch panics/errors
			var buf bytes.Buffer
			err := func() (err error) {
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("panic during render: %v", r)
					}
				}()
				return wrappedContent.Render(ctx, &buf)
			}()

			if err != nil {
				a.Logger().Error("SSR render error (buffered)", "err", err)
				return a.renderError(c, fiberpkg.StatusInternalServerError, err)
			}

			c.Set("Cache-Control", "no-store")
			return c.Send(buf.Bytes())
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

	// No root layout registered — minimal fallback HTML wrapper.
	wsURL := a.getWSUrl(c)
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
	c.Response().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		_, _ = fmt.Fprint(w, `<!DOCTYPE html><html lang="en" data-gospa-auto><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>`)
		_, _ = fmt.Fprint(w, appName)
		_, _ = fmt.Fprint(w, `</title></head><body><div id="app" data-gospa-root><main>`)
		if err := content.Render(ctx, w); err != nil {
			a.Logger().Error("streaming render error", "err", err)
		}
		_, _ = fmt.Fprint(w, `</main></div>`)
		_, _ = fmt.Fprintf(w, `<script src="%s" type="module"></script>`, runtimePath)
		_, _ = fmt.Fprintf(w, `<script type="module">
import * as runtime from '%s';
window.__GOSPA_CONFIG__ = {
	navigationOptions: %s,
};
runtime.init({
	wsUrl: '%s',
	debug: %v,
	simpleRuntimeSVGs: %v,
	disableSanitization: %v,
	wsReconnectDelay: %d,
	wsMaxReconnect: %d,
	wsHeartbeat: %d,
	hydration: {
		mode: '%s',
		timeout: %d
	}
});
</script>`, jsEscape(runtimePath), toJS(a.Config.NavigationOptions), jsEscape(wsURL), devMode, a.Config.SimpleRuntimeSVGs, a.Config.DisableSanitization, wsReconnectDelay, wsMaxReconnect, wsHeartbeat, jsEscape(a.Config.HydrationMode), a.Config.HydrationTimeout)
		_, _ = fmt.Fprint(w, `</body></html>`)
		_ = w.Flush()
	}))
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
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
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
func (a *App) buildRootLayoutProps(c fiberpkg.Ctx, params map[string]string) map[string]interface{} {
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
		// Batch evict 10% of entries to avoid O(n) slice shift on every insert (P2 fix)
		evictCount := maxEntries / 10
		if evictCount < 1 {
			evictCount = 1
		}
		if evictCount > len(a.ssgCacheKeys) {
			evictCount = len(a.ssgCacheKeys)
		}
		for i := 0; i < evictCount; i++ {
			oldest := a.ssgCacheKeys[i]
			delete(a.ssgCache, oldest)
		}
		a.ssgCacheKeys = append([]string(nil), a.ssgCacheKeys[evictCount:]...)
	}
	if _, exists := a.ssgCache[key]; !exists {
		a.ssgCacheKeys = append(a.ssgCacheKeys, key)
	}
	a.ssgCache[key] = ssgEntry{html: html, createdAt: time.Now()}
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
		// Batch evict 10% of entries to avoid O(n) slice shift on every insert (P2 fix)
		evictCount := maxEntries / 10
		if evictCount < 1 {
			evictCount = 1
		}
		if evictCount > len(a.pprShellKeys) {
			evictCount = len(a.pprShellKeys)
		}
		for i := 0; i < evictCount; i++ {
			oldest := a.pprShellKeys[i]
			delete(a.pprShellCache, oldest)
		}
		a.pprShellKeys = append([]string(nil), a.pprShellKeys[evictCount:]...)
	}
	if _, exists := a.pprShellCache[key]; !exists {
		a.pprShellKeys = append(a.pprShellKeys, key)
	}
	a.pprShellCache[key] = pprEntry{html: shell, createdAt: time.Now()}
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

func toJS(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(b)
}

// jsEscape escapes a string for safe interpolation inside a JavaScript
// single-quoted string literal (e.g. '...'). It prevents script injection
// via Config values like AppName or WebSocketPath that are written directly
// into inline <script> blocks.
func jsEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	return s
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
	a.Logger().Info("starting GoSPA", "version", Version, "addr", addr)
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
	a.Logger().Info("starting GoSPA (TLS)", "version", Version, "addr", addr)
	return a.Fiber.Listen(addr, fiberpkg.ListenConfig{
		CertFile:    certFile,
		CertKeyFile: keyFile,
	})
}

// Shutdown gracefully shuts down the application.
func (a *App) Shutdown() error {
	if a.Hub != nil {
		a.Hub.Close()
	}
	if closer, ok := a.Config.Storage.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
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
	if len(handlers) == 0 {
		return
	}
	hAny := make([]any, len(handlers)-1)
	for i, h := range handlers[1:] {
		hAny[i] = h
	}
	a.Fiber.Get(path, handlers[0], hAny...)
}

// Post adds a POST route.
func (a *App) Post(path string, handlers ...fiberpkg.Handler) {
	if len(handlers) == 0 {
		return
	}
	hAny := make([]any, len(handlers)-1)
	for i, h := range handlers[1:] {
		hAny[i] = h
	}
	a.Fiber.Post(path, handlers[0], hAny...)
}

// Put adds a PUT route.
func (a *App) Put(path string, handlers ...fiberpkg.Handler) {
	if len(handlers) == 0 {
		return
	}
	hAny := make([]any, len(handlers)-1)
	for i, h := range handlers[1:] {
		hAny[i] = h
	}
	a.Fiber.Put(path, handlers[0], hAny...)
}

// Delete adds a DELETE route.
func (a *App) Delete(path string, handlers ...fiberpkg.Handler) {
	if len(handlers) == 0 {
		return
	}
	hAny := make([]any, len(handlers)-1)
	for i, h := range handlers[1:] {
		hAny[i] = h
	}
	a.Fiber.Delete(path, handlers[0], hAny...)
}

// Group creates a route group.
func (a *App) Group(prefix string, handlers ...fiberpkg.Handler) fiberpkg.Router {
	hAny := make([]any, len(handlers))
	for i, h := range handlers {
		hAny[i] = h
	}
	return a.Fiber.Group(prefix, hAny...)
}

// Static serves static files via the static middleware.
func (a *App) Static(prefix, root string) {
	a.Fiber.Use(prefix, static.New(root))
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
