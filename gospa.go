// Package gospa provides a modern SPA framework for Go with Fiber and Templ.
// It brings Svelte-like reactivity and state management to Go.
package gospa

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/aydenstechdungeon/gospa/embed"
	"github.com/aydenstechdungeon/gospa/fiber"
	"github.com/aydenstechdungeon/gospa/plugin"
	"github.com/aydenstechdungeon/gospa/routing"
	"github.com/aydenstechdungeon/gospa/routing/kit"
	"github.com/aydenstechdungeon/gospa/state"
	"github.com/aydenstechdungeon/gospa/store"
	json "github.com/goccy/go-json"
	fiberpkg "github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/logger"
	recovermw "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"
)

const islandsRoutePrefix = "/islands/"

func safeIslandTSPath(requestPath string) (string, int, error) {
	relPath := strings.TrimPrefix(requestPath, islandsRoutePrefix)
	if relPath == requestPath {
		return "", fiberpkg.StatusBadRequest, fmt.Errorf("invalid islands request path")
	}
	if relPath == "" || strings.ContainsRune(relPath, '\x00') {
		return "", fiberpkg.StatusBadRequest, fmt.Errorf("invalid islands request path")
	}

	cleanRel := filepath.Clean(filepath.FromSlash(relPath))
	if filepath.IsAbs(cleanRel) || cleanRel == "." || cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) {
		return "", fiberpkg.StatusForbidden, fmt.Errorf("forbidden islands path")
	}
	if filepath.Ext(cleanRel) != ".js" {
		return "", fiberpkg.StatusBadRequest, fmt.Errorf("invalid islands module extension")
	}

	targetRel := strings.TrimSuffix(cleanRel, ".js") + ".ts"
	baseAbs, err := filepath.Abs("generated")
	if err != nil {
		return "", fiberpkg.StatusInternalServerError, fmt.Errorf("failed to resolve islands base path")
	}
	targetAbs, err := filepath.Abs(filepath.Join(baseAbs, targetRel))
	if err != nil {
		return "", fiberpkg.StatusInternalServerError, fmt.Errorf("failed to resolve islands file path")
	}
	if !strings.HasPrefix(targetAbs, baseAbs+string(filepath.Separator)) {
		return "", fiberpkg.StatusForbidden, fmt.Errorf("forbidden islands path")
	}

	return targetAbs, 0, nil
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
	// pluginMiddleware stores middleware from runtime plugins.
	pluginMiddleware []fiberpkg.Handler
	// pluginTemplateFuncs stores template functions from plugins.
	pluginTemplateFuncs map[string]any
	// ssgCache stores pre-rendered SSG and ISR pages.
	ssgCache map[string]ssgEntry
	// ssgCacheKeys tracks insertion order for FIFO eviction.
	ssgCacheKeys []string
	// ssgCacheIndex maps a key to its position in ssgCacheKeys for O(1) existence checks.
	ssgCacheIndex map[string]struct{}
	// ssgCacheMu protects ssgCache, ssgCacheKeys, and ssgCacheIndex.
	ssgCacheMu sync.RWMutex
	// isrRevalidating guards against duplicate background revalidations.
	isrRevalidating sync.Map
	// isrSemaphore limits concurrent ISR background revalidations.
	isrSemaphore chan struct{}
	// isrSemOnce ensures semaphore is initialized once.
	isrSemOnce sync.Once
	// pprShellCache stores cached static shells for PPR pages.
	pprShellCache map[string]pprEntry
	// pprShellKeys tracks insertion order for PPR shell FIFO eviction.
	pprShellKeys []string
	// pprShellIndex maps a key to its existence in pprShellKeys for O(1) checks.
	pprShellIndex map[string]struct{}
	// pprShellMu protects pprShellCache, pprShellKeys, and pprShellIndex.
	pprShellMu sync.RWMutex
	// cacheIndexMu protects cacheTagIndex and cacheKeyIndex.
	cacheIndexMu sync.RWMutex
	// cacheTagIndex maps logical tags to cached route keys.
	cacheTagIndex map[string]map[string]struct{}
	// cacheKeyIndex maps logical keys to cached route keys.
	cacheKeyIndex map[string]map[string]struct{}
	// pprShellBuilding guards against duplicate PPR shell builds under concurrent load.
	pprShellBuilding sync.Map
	// cacheStatsMu protects route and slot cache metrics.
	cacheStatsMu sync.RWMutex
	// routeCacheStats tracks cache metrics by route path.
	routeCacheStats map[string]*routeCacheStats
	// slotCacheStats tracks dynamic slot render stats by "path#slot" key.
	slotCacheStats map[string]*slotCacheStat
	// ctx is the application-level context, canceled on Shutdown.
	ctx    context.Context
	cancel context.CancelFunc
	// startupErr stores configuration failures that should block server startup.
	startupErr error
}

var defaultApp *App
var defaultOnce sync.Once

// New creates a new GoSPA application with the given configuration.
func New(config Config) *App {
	applyDefaultConfig(&config)
	startupErr := validateAndLogConfig(&config)

	fiber.SetConnectionRateLimiter(config.WSConnBurst, config.WSConnRateLimit)
	state.SetNotificationQueueSize(config.NotificationBufferSize)

	// Load build manifest if available
	if len(config.BuildManifest) == 0 && config.ManifestPath != "" {
		if _, err := os.Stat(config.ManifestPath); err == nil {
			if data, err := os.ReadFile(config.ManifestPath); err == nil {
				var manifest map[string]string
				if err := json.Unmarshal(data, &manifest); err == nil {
					config.BuildManifest = manifest
					config.Logger.Info("Loaded build manifest", "path", config.ManifestPath, "entries", len(manifest))
				}
			}
		}
	}

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

	fiber.InitStores(config.Storage)

	var routerSource interface{}
	if config.RoutesFS != nil {
		routerSource = config.RoutesFS
	} else {
		routerSource = config.RoutesDir
	}
	router := routing.NewRouter(routerSource)

	fiberConfig := fiberpkg.Config{
		AppName:      config.AppName,
		ServerHeader: "GoSPA",
		BodyLimit:    config.MaxRequestBodySize,
	}
	if config.DevMode {
		config.Logger.Warn("DevMode is enabled — disable in production")
		fiberConfig.ServerHeader = "GoSPA/" + Version
	} else if config.PublicOrigin == "" {
		// CRITICAL: Production enforcement of PublicOrigin unless AllowInsecureWS is set
		if config.AllowInsecureWS || len(config.AllowPortsWithInsecureWS) > 0 {
			config.Logger.Warn("Warning: PublicOrigin is not set in production mode. insecure WebSocket will be used because AllowInsecureWS or AllowPortsWithInsecureWS is enabled.")
		} else {
			config.Logger.Error("CRITICAL: PublicOrigin must be set in production mode for secure WebSocket and absolute URL generation.")
		}
	}
	fiberApp := fiberpkg.New(fiberConfig)

	var hub *fiber.WSHub
	if config.EnableWebSocket {
		hub = fiber.NewWSHub(config.PubSub)
		go hub.Run()
	}

	stateMap := state.NewStateMap()
	for k, v := range config.DefaultState {
		r := state.NewRune(v)
		stateMap.Add(k, r)
	}

	app := &App{
		Config:              config,
		Router:              router,
		Fiber:               fiberApp,
		Hub:                 hub,
		StateMap:            stateMap,
		pluginTemplateFuncs: make(map[string]any),
		ssgCache:            make(map[string]ssgEntry),
		ssgCacheKeys:        make([]string, 0),
		ssgCacheIndex:       make(map[string]struct{}),
		pprShellCache:       make(map[string]pprEntry),
		pprShellKeys:        make([]string, 0),
		pprShellIndex:       make(map[string]struct{}),
		cacheTagIndex:       make(map[string]map[string]struct{}),
		cacheKeyIndex:       make(map[string]map[string]struct{}),
		routeCacheStats:     make(map[string]*routeCacheStats),
		slotCacheStats:      make(map[string]*slotCacheStat),
		startupErr:          startupErr,
	}
	app.ctx, app.cancel = context.WithCancel(context.Background())
	if startupErr != nil {
		app.Logger().Error("GoSPA startup validation failed", "err", startupErr)
	}

	app.setupMiddleware()

	defaultOnce.Do(func() {
		if defaultApp == nil {
			defaultApp = app
		}
	})

	return app
}

func applyDefaultConfig(config *Config) {
	if config.AppName == "" {
		config.AppName = "GoSPA Application"
	}
	if !config.EnableWebSocket && config.WebSocketPath == "" && config.DefaultState == nil {
		// Only re-enable if it looks like a zero-value Config was passed,
		// but don't override explicit disabling in MinimalConfig().
		// If it's already false and path is empty, we set defaults but keep it disabled.
		config.EnableWebSocket = true
	}
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

	switch {
	case config.SSGCacheMaxEntries == 0:
		config.SSGCacheMaxEntries = 500
	case config.SSGCacheMaxEntries < 0:
		config.SSGCacheMaxEntries = -1
	case config.SSGCacheMaxEntries > 10000:
		config.SSGCacheMaxEntries = 10000
	}

	if config.HydrationMode == "" {
		config.HydrationMode = "visible"
	}

	if config.WSMaxMessageSize == 0 {
		config.WSMaxMessageSize = 64 * 1024
	}
	if config.WSConnRateLimit == 0 {
		config.WSConnRateLimit = 1.5
	}
	if config.WSConnBurst == 0 {
		config.WSConnBurst = 15.0
	}
	if config.IslandsBundlePath == "" {
		config.IslandsBundlePath = "static/js/islands.js"
	}
}

func validateAndLogConfig(config *Config) error {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	var validationErr error

	// Validation: HydrationTimeout must be within 0-10s to prevent hanging or UI jank
	if config.HydrationTimeout < 0 {
		config.HydrationTimeout = 0
	} else if config.HydrationTimeout > 10000 {
		config.Logger.Warn("HydrationTimeout is too high (>10s). Capping to 10 seconds for UX safety.", "value", config.HydrationTimeout)
		config.HydrationTimeout = 10000
	}
	switch strings.ToLower(strings.TrimSpace(config.HydrationMode)) {
	case "", "visible", "lazy":
		config.HydrationMode = "visible"
	case "immediate", "manual", "idle":
		// Keep as-is.
	case "progressive":
		config.HydrationMode = "visible"
	default:
		config.Logger.Warn("Unknown HydrationMode, defaulting to visible progressive hydration", "mode", config.HydrationMode)
		config.HydrationMode = "visible"
	}

	// GOSPA_WS_INSECURE env var provides a quick override for development.
	// SECURITY: We block this override in production (DevMode: false) to prevent
	// accidental mixed-content exposure from leaked environment variables.
	if os.Getenv("GOSPA_WS_INSECURE") == "1" {
		if config.DevMode {
			config.AllowInsecureWS = true
		} else {
			config.Logger.Warn("GOSPA_WS_INSECURE=1 is ignored in production mode. Use AllowInsecureWS in config explicitly if required.")
		}
	}

	// SECURITY: Warn if RemoteActionMiddleware is missing in production.
	// We do not force a block here, leaving the security model in the developer's hands.
	if !config.DevMode && config.RemoteActionMiddleware == nil && !config.AllowUnauthenticatedRemoteActions {
		config.Logger.Warn("RemoteActionMiddleware is not set in production. Ensure remote actions are protected by another layer or are intentionally public.")
	}

	// SECURITY: Warn if DisableSanitization is enabled — this opens XSS vectors.
	if config.DisableSanitization {
		config.Logger.Warn("DisableSanitization is enabled — client-side HTML sanitization is OFF. This creates XSS vulnerabilities.")
	}

	routeOptions := routing.GetAllRouteOptions()
	for path, opts := range routeOptions {
		strategy := opts.Strategy
		if strategy == "" {
			strategy = config.DefaultRenderStrategy
		}
		if strategy == "" {
			strategy = routing.StrategySSR
		}
		needsTemplateCache := strategy == routing.StrategySSG || strategy == routing.StrategyISR || strategy == routing.StrategyPPR
		if needsTemplateCache && !config.CacheTemplates {
			validationErr = errors.Join(validationErr, fmt.Errorf("route %q uses %s but CacheTemplates=false; enable CacheTemplates or change strategy", path, strategy))
		}
		if strategy == routing.StrategySSG && config.SSGCacheTTL == 0 {
			config.Logger.Warn("SSG route caches forever because SSGCacheTTL=0", "path", path)
		}
	}

	if config.Prefork && isInMemoryStorage(config.Storage) {
		config.Logger.Warn("Prefork with in-memory cache/storage detected: render caches are process-local; use distributed Storage for seamless ISR/SSG/PPR")
	}

	if config.StrictProduction {
		if config.DevMode {
			validationErr = errors.Join(validationErr, fmt.Errorf("StrictProduction requires DevMode=false"))
		}
		if config.PublicOrigin == "" {
			validationErr = errors.Join(validationErr, fmt.Errorf("StrictProduction requires PublicOrigin"))
		}
		if config.AllowInsecureWS {
			validationErr = errors.Join(validationErr, fmt.Errorf("StrictProduction forbids AllowInsecureWS=true"))
		}
		if len(config.AllowedOrigins) == 0 {
			validationErr = errors.Join(validationErr, fmt.Errorf("StrictProduction requires AllowedOrigins to be set"))
		}
		if !strings.Contains(config.ContentSecurityPolicy, "{nonce}") {
			validationErr = errors.Join(validationErr, fmt.Errorf("StrictProduction requires CSP policy containing {nonce}"))
		}
		if config.DisableSanitization {
			validationErr = errors.Join(validationErr, fmt.Errorf("StrictProduction forbids DisableSanitization=true"))
		}
		if !config.AllowUnauthenticatedRemoteActions && config.RemoteActionMiddleware == nil {
			validationErr = errors.Join(validationErr, fmt.Errorf("StrictProduction requires RemoteActionMiddleware unless AllowUnauthenticatedRemoteActions=true"))
		}
		if config.ISRTimeout <= 0 {
			validationErr = errors.Join(validationErr, fmt.Errorf("StrictProduction requires ISRTimeout > 0"))
		}
	}

	return validationErr
}

func isInMemoryStorage(storage store.Storage) bool {
	if storage == nil {
		return true
	}
	t := reflect.TypeOf(storage)
	if t == nil {
		return true
	}
	return strings.Contains(strings.ToLower(t.String()), "memorystorage")
}

// setupRoutes configures core internal routes.
func (a *App) setupRoutes() {
	a.Fiber.Get(a.getRuntimePath(), fiber.RuntimeMiddleware(a.Config.RuntimeTier))

	a.Fiber.Use("/_gospa/", func(c fiberpkg.Ctx) error {
		c.Set("Cache-Control", "public, max-age=31536000, immutable")
		if strings.HasSuffix(c.Path(), ".js") {
			c.Set("Content-Type", "application/javascript")
		}
		return c.Next()
	})
	a.Fiber.Use("/_gospa/", static.New("", static.Config{
		FS:       embed.RuntimeFS(),
		Compress: true,
	}))

	// Serve dynamically compiled island modules with proper MIME type
	a.Fiber.Use(islandsRoutePrefix, func(c fiberpkg.Ctx) error {
		path := c.Path()
		if strings.HasSuffix(path, ".js") {
			c.Set("Content-Type", "application/javascript")
			tsPath, code, err := safeIslandTSPath(path)
			if err != nil {
				if code == fiberpkg.StatusForbidden || code == fiberpkg.StatusBadRequest {
					return c.Status(code).SendString(err.Error())
				}
				return c.Status(fiberpkg.StatusInternalServerError).SendString("internal islands path error")
			}
			if _, err := os.Stat(tsPath); err == nil {
				return c.SendFile(tsPath)
			}
		}
		return c.Next()
	})
	// Try to serve islands from static/islands or generated/ directory
	if _, err := os.Stat("static/islands"); err == nil {
		a.Fiber.Use("/islands/", static.New("static/islands", static.Config{
			Compress: true,
		}))
	} else if _, err := os.Stat("generated"); err == nil {
		a.Fiber.Use("/islands/", static.New("generated", static.Config{
			Compress: true,
		}))
	}

	if a.Hub != nil {
		handlers := []fiberpkg.Handler{
			fiber.SessionMiddleware(),
			fiber.WebSocketUpgradeMiddleware(),
		}
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
		hAny := make([]any, len(handlers))
		for i, h := range handlers {
			hAny[i] = h
		}
		a.Fiber.Get(a.Config.WebSocketPath, hAny[0], hAny[1:]...)
	}

	remoteHandlers := []fiberpkg.Handler{
		fiber.SessionMiddleware(),
		fiber.RemoteActionRateLimitMiddleware(),
	}
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
	remoteHandlers = append(remoteHandlers, a.handleRemoteAction)
	rhAny := make([]any, len(remoteHandlers))
	for i, h := range remoteHandlers {
		rhAny[i] = h
	}
	a.Fiber.Post(a.Config.RemotePrefix+"/:name", rhAny[0], rhAny[1:]...)

	a.Fiber.Post("/_gospa/invalidate", fiber.SessionMiddleware(), a.handleInvalidate)
	if a.Config.DevMode {
		a.Fiber.Get("/__gospa/cache", a.handleCacheStats)
	}
	a.Fiber.Get("/_gospa/poll", a.handleTransportPoll)

	if _, err := os.Stat(a.Config.StaticDir); err == nil {
		a.Fiber.Use(a.Config.StaticPrefix, static.New(a.Config.StaticDir, static.Config{
			Compress: true,
			ModifyResponse: func(c fiberpkg.Ctx) error {
				path := c.Path()
				switch {
				case strings.HasSuffix(path, ".js"), strings.HasSuffix(path, ".mjs"):
					c.Set("Content-Type", "application/javascript")
				case strings.HasSuffix(path, ".css"):
					c.Set("Content-Type", "text/css")
				case strings.HasSuffix(path, ".json"):
					c.Set("Content-Type", "application/json")
				case strings.HasSuffix(path, ".svg"):
					c.Set("Content-Type", "image/svg+xml")
				}
				return nil
			},
		}))
		a.Fiber.Get("/favicon.ico", func(c fiberpkg.Ctx) error {
			favPath := a.Config.StaticDir + "/favicon.ico"
			if _, err := os.Stat(favPath); err == nil {
				return c.SendFile(favPath)
			}
			return c.SendStatus(fiberpkg.StatusNoContent)
		})
	} else {
		a.Fiber.Get("/favicon.ico", func(c fiberpkg.Ctx) error {
			return c.SendStatus(fiberpkg.StatusNoContent)
		})
	}
}

func (a *App) handleRemoteAction(c fiberpkg.Ctx) error {
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
	if contentLength := c.Request().Header.ContentLength(); contentLength > a.Config.MaxRequestBodySize {
		return c.Status(fiberpkg.StatusRequestEntityTooLarge).JSON(fiberpkg.Map{
			"error": "Request body too large",
			"code":  "REQUEST_TOO_LARGE",
		})
	}

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
		var err error
		input, err = decodeRemoteActionBody(body)
		if err != nil {
			if errors.Is(err, ErrJSONTooDeep) {
				return c.Status(fiberpkg.StatusBadRequest).JSON(fiberpkg.Map{
					"error": "JSON nesting too deep",
					"code":  "JSON_TOO_DEEP",
				})
			}
			return c.Status(fiberpkg.StatusBadRequest).JSON(fiberpkg.Map{
				"error": "Invalid input JSON",
				"code":  "INVALID_JSON",
			})
		}
	}

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
		RequestID: c.Get("X-Request-Id"),
		SessionID: c.Get("X-Session-Id"),
		Headers:   headers,
	}

	result, err := fn(c.Context(), rc, input)
	if err != nil {
		a.Logger().Error("remote action error", "action", name, "err", err)

		response := fiberpkg.Map{
			"error": "Internal server error",
			"code":  "ACTION_FAILED",
		}

		// Include debug info in DevMode
		if a.Config.DevMode {
			response["debug"] = err.Error()
		}

		return c.Status(fiberpkg.StatusInternalServerError).JSON(response)
	}

	return c.JSON(fiberpkg.Map{
		"data": result,
		"code": "SUCCESS",
	})
}

func (a *App) handleInvalidate(c fiberpkg.Ctx) error {
	var payload struct {
		Path string `json:"path"`
		Tag  string `json:"tag"`
		Key  string `json:"key"`
		All  bool   `json:"all"`
	}
	if err := c.Bind().Body(&payload); err != nil {
		return c.Status(fiberpkg.StatusBadRequest).JSON(fiberpkg.Map{
			"error": "Invalid invalidation payload",
			"code":  "INVALID_INVALIDATION_PAYLOAD",
		})
	}

	invalidated := 0
	if payload.Path != "" {
		if a.Invalidate(payload.Path) > 0 {
			invalidated++
		}
	}
	if payload.Tag != "" {
		invalidated += a.InvalidateTag(payload.Tag)
	}
	if payload.Key != "" {
		invalidated += a.InvalidateKey(payload.Key)
	}
	if payload.All {
		invalidated += a.InvalidateAll()
	}

	return c.JSON(fiberpkg.Map{
		"ok":          true,
		"invalidated": invalidated,
	})
}

// UsePlugin registers a plugin with the application.
func (a *App) UsePlugin(p plugin.Plugin) error {
	if err := plugin.Register(p); err != nil {
		return err
	}
	if rp, ok := p.(plugin.RuntimePlugin); ok {
		_ = rp.Config()
		if mws := rp.Middlewares(); len(mws) > 0 {
			for _, mw := range mws {
				if handler, ok := any(mw).(func(fiberpkg.Ctx) error); ok {
					a.pluginMiddleware = append(a.pluginMiddleware, handler)
				} else if handler, ok := mw.(fiberpkg.Handler); ok {
					a.pluginMiddleware = append(a.pluginMiddleware, handler)
				}
			}
		}
		if funcs := rp.TemplateFuncs(); funcs != nil {
			for k, v := range funcs {
				a.pluginTemplateFuncs[k] = v
			}
		}
	}
	return nil
}

// UsePlugins registers multiple plugins with the application.
func (a *App) UsePlugins(plugins ...plugin.Plugin) error {
	for _, p := range plugins {
		if err := a.UsePlugin(p); err != nil {
			return err
		}
	}
	return nil
}

// GetPlugin retrieves a registered plugin by name.
func (a *App) GetPlugin(name string) (plugin.Plugin, bool) {
	p := plugin.GetPlugin(name)
	return p, p != nil
}

// ListPlugins returns information about all registered plugins.
func (a *App) ListPlugins() []plugin.Info {
	return plugin.GetAllPluginInfo()
}

// GetTemplateFuncs returns template functions registered by plugins.
func (a *App) GetTemplateFuncs() map[string]any {
	return a.pluginTemplateFuncs
}

func (a *App) applyPluginMiddleware() {
	for _, mw := range a.pluginMiddleware {
		a.Fiber.Use(mw)
	}
}

func (a *App) setupMiddleware() {
	// 1. Global Hooks (SvelteKit hooks.server.go style)
	for _, hook := range routing.GetHooks() {
		a.Fiber.Use(hook)
	}

	a.Fiber.Use(recovermw.New(recovermw.Config{
		EnableStackTrace: true,
	}))
	if a.Config.DevMode {
		a.Fiber.Use(logger.New())
	}
	a.Fiber.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))
	a.Fiber.Use(fiber.SecurityHeadersMiddleware(a.Config.ContentSecurityPolicy))
	if len(a.Config.AllowedOrigins) > 0 {
		a.Fiber.Use(fiber.CORSMiddleware(a.Config.AllowedOrigins))
	}
	if a.Config.EnableCSRF {
		a.Fiber.Use(fiber.CSRFSetTokenMiddleware())
		a.Fiber.Use(fiber.CSRFTokenMiddleware())
	}
	if !a.Config.DisableSPA {
		a.Fiber.Use(fiber.SPANavigationMiddleware())
	}
	preloadConfig := fiber.DefaultPreloadConfig()
	preloadConfig.RuntimeScript = a.getRuntimePath()
	preloadConfig.CSSLinks = a.Config.PreloadCSS
	preloadConfig.BuildManifest = a.Config.BuildManifest
	a.Fiber.Use(fiber.PreloadHeadersMiddleware(preloadConfig))

	spaConfig := fiber.DefaultConfig()
	spaConfig.DevMode = a.Config.DevMode
	spaConfig.RuntimeScript = a.Config.RuntimeScript
	spaConfig.EnableWebSocket = a.Config.EnableWebSocket
	spaConfig.WebSocketPath = a.Config.WebSocketPath
	spaConfig.ExpectCSPNonce = strings.Contains(a.Config.ContentSecurityPolicy, "{nonce}")
	spaConfig.StartupChecks = true
	spaConfig.BuildManifest = a.Config.BuildManifest
	a.Fiber.Use(fiber.SPAMiddleware(spaConfig))
}

// Scan scans the routes directory for page and layout components.
func (a *App) Scan() error {
	return a.Router.Scan()
}

// Logger returns the application logger.
func (a *App) Logger() *slog.Logger {
	if a.Config.Logger != nil {
		return a.Config.Logger
	}
	return slog.Default()
}

// Run starts the GoSPA application on the specified address.
func (a *App) Run(addr string) error {
	if a.startupErr != nil {
		return fmt.Errorf("gospa startup validation failed: %w", a.startupErr)
	}
	if err := plugin.TriggerHook(plugin.BeforeServe, map[string]interface{}{
		"fiber":  a.Fiber,
		"config": a.Config,
	}); err != nil {
		a.Logger().Error("plugin BeforeServe hook failed", "err", err)
	}
	a.applyPluginMiddleware()
	a.setupRoutes()
	if err := a.RegisterRoutes(); err != nil {
		return err
	}
	a.Logger().Info("starting GoSPA", "version", Version, "addr", addr)
	return a.Fiber.Listen(addr)
}

// RunTLS starts the GoSPA application on the specified address with TLS.
func (a *App) RunTLS(addr, certFile, keyFile string) error {
	if a.startupErr != nil {
		return fmt.Errorf("gospa startup validation failed: %w", a.startupErr)
	}
	if err := plugin.TriggerHook(plugin.BeforeServe, map[string]interface{}{
		"fiber":  a.Fiber,
		"config": a.Config,
	}); err != nil {
		a.Logger().Error("plugin BeforeServe hook failed", "err", err)
	}
	a.applyPluginMiddleware()
	a.setupRoutes()
	if err := a.RegisterRoutes(); err != nil {
		return err
	}
	a.Logger().Info("starting GoSPA (TLS)", "version", Version, "addr", addr)
	return a.Fiber.Listen(addr, fiberpkg.ListenConfig{
		CertFile:    certFile,
		CertKeyFile: keyFile,
	})
}

// Shutdown gracefully shuts down the GoSPA application.
func (a *App) Shutdown() error {
	if a.cancel != nil {
		a.cancel()
	}
	if err := plugin.TriggerHook(plugin.BeforePrune, nil); err != nil {
		a.Logger().Error("plugin BeforePrune hook failed", "err", err)
	}
	if a.Hub != nil {
		a.Hub.Close()
	}
	if closer, ok := a.Config.Storage.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			a.Logger().Error("Storage close failed", "err", err)
		}
	}
	err := a.Fiber.Shutdown()
	if err := plugin.TriggerHook(plugin.AfterPrune, nil); err != nil {
		a.Logger().Error("plugin AfterPrune hook failed", "err", err)
	}
	return err
}

// Context returns the application-level context.
func (a *App) Context() context.Context {
	if a.ctx == nil {
		return context.Background()
	}
	return a.ctx
}

// RegisterRoutes manually triggers route registration.
func (a *App) RegisterRoutes() error {
	if err := a.Scan(); err != nil {
		return err
	}
	for _, route := range a.Router.GetPages() {
		a.registerPageRoute(route)
	}
	return nil
}

func (a *App) registerPageRoute(r *routing.Route) {
	opts := routing.GetRouteOptions(r.Path)
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
	mws := a.Router.ResolveMiddlewareChain(r)
	for _, mwRoute := range mws {
		if fn := routing.GetMiddleware(mwRoute.Path); fn != nil {
			if mwHandler, ok := fn.(func(fiberpkg.Ctx) error); ok {
				handlers = append(handlers, mwHandler)
			} else if mwHandler, ok := fn.(fiberpkg.Handler); ok {
				handlers = append(handlers, mwHandler)
			}
		}
	}
	handlers = append(handlers, func(c fiberpkg.Ctx) error {
		return a.renderRoute(c, r, extractRouteParams(c, r))
	})
	a.Fiber.Get(r.Path, handlers[0], handlers[1:]...)

	// Register POST handler for form actions
	postHandlers := append([]any{}, handlers[:len(handlers)-1]...)
	postHandlers = append(postHandlers, func(c fiberpkg.Ctx) error {
		return a.handleFormAction(c, r)
	})
	if len(postHandlers) > 0 {
		a.Fiber.Post(r.Path, postHandlers[0], postHandlers[1:]...)
	}
}

func (a *App) handleFormAction(c fiberpkg.Ctx, r *routing.Route) error {
	actionName := c.Query("_action")
	if actionName == "" {
		actionName = "default"
	}

	action := routing.GetAction(r.Path, actionName)
	if action == nil {
		// If no specific action found, try "default" if it wasn't already checked
		if actionName != "default" {
			action = routing.GetAction(r.Path, "default")
		}
	}

	if action == nil {
		return c.Status(fiberpkg.StatusNotFound).SendString("Action not found")
	}

	baseCtx := &fiberLoadContext{c: c}
	lc := &helperLoadContext{LoadContext: baseCtx, parentData: nil}
	scope := kit.NewExecutionScope()
	scope.SetParentData(nil)
	var result interface{}
	var err error
	runErr := scope.Run(func() error {
		result, err = action(lc)
		return nil
	})
	if runErr != nil {
		err = runErr
	}
	responsePayload := routing.ActionResponse{
		Data: result,
		Code: "SUCCESS",
	}
	if actionResp, ok := result.(routing.ActionResponse); ok {
		responsePayload = actionResp
		if responsePayload.Code == "" {
			responsePayload.Code = "SUCCESS"
		}
	} else if actionResp, ok := result.(*routing.ActionResponse); ok && actionResp != nil {
		responsePayload = *actionResp
		if responsePayload.Code == "" {
			responsePayload.Code = "SUCCESS"
		}
	}

	applyRevalidation := func() {
		for _, path := range responsePayload.Revalidate {
			a.Invalidate(path)
		}
		for _, tag := range responsePayload.RevalidateTags {
			a.InvalidateTag(tag)
		}
		for _, key := range responsePayload.RevalidateKeys {
			a.InvalidateKey(key)
		}
	}

	// Check if AJAX (progressive enhancement)
	if c.Get("X-Gospa-Enhance") != "" {
		if redirectErr, ok := kit.AsRedirect(err); ok {
			responsePayload.Code = "REDIRECT"
			responsePayload.Redirect = &routing.ActionRedirect{
				To:     redirectErr.Location,
				Status: redirectErr.Status,
			}
			return c.JSON(responsePayload)
		}
		if failErr, ok := kit.AsFail(err); ok {
			responsePayload.Code = "FAIL"
			responsePayload.Data = failErr.Data
			if v := coerceActionValidation(failErr.Data); v != nil {
				responsePayload.Validation = v
			}
			return c.Status(failErr.Status).JSON(responsePayload)
		}
		if httpErr, ok := kit.AsError(err); ok {
			responsePayload.Code = "FAIL"
			responsePayload.Error = "Request failed"
			responsePayload.Data = httpErr.Body
			return c.Status(httpErr.Status).JSON(responsePayload)
		}
		if err != nil {
			responsePayload.Code = "FAIL"
			responsePayload.Error = "Internal server error"
			if a.Config.DevMode {
				responsePayload.Error = err.Error()
			}
			return c.Status(fiberpkg.StatusInternalServerError).JSON(responsePayload)
		}
		applyRevalidation()
		return c.JSON(responsePayload)
	}

	// Standard form submission: redirect back to the page
	if redirectErr, ok := kit.AsRedirect(err); ok {
		return c.Redirect().Status(redirectErr.Status).To(redirectErr.Location)
	}
	if err != nil {
		if failErr, ok := kit.AsFail(err); ok {
			fiber.SetFlash(c, "error", fmt.Sprintf("%v", failErr.Data))
			return c.Redirect().Status(fiberpkg.StatusSeeOther).To(c.Path())
		}
		if httpErr, ok := kit.AsError(err); ok {
			return a.renderError(c, httpErr.Status, fmt.Errorf("HTTP %d", httpErr.Status))
		}
		a.Logger().Error("Form action error", "path", r.Path, "action", actionName, "err", err)
		fiber.SetFlash(c, "error", err.Error())
		return c.Redirect().Status(fiberpkg.StatusSeeOther).To(c.Path())
	}
	applyRevalidation()
	if responsePayload.Redirect != nil && responsePayload.Redirect.To != "" {
		status := responsePayload.Redirect.Status
		if status == 0 {
			status = fiberpkg.StatusSeeOther
		}
		return c.Redirect().Status(status).To(responsePayload.Redirect.To)
	}

	return c.Redirect().Status(fiberpkg.StatusSeeOther).To(c.Path())
}

func coerceActionValidation(data interface{}) *routing.ActionValidationError {
	if data == nil {
		return nil
	}

	if v, ok := data.(routing.ActionValidationError); ok {
		return &v
	}
	if v, ok := data.(*routing.ActionValidationError); ok && v != nil {
		return v
	}

	switch typed := data.(type) {
	case map[string]interface{}:
		return validationFromGenericMap(typed)
	case map[string]string:
		return &routing.ActionValidationError{FieldErrors: typed}
	default:
		return nil
	}
}

func validationFromGenericMap(m map[string]interface{}) *routing.ActionValidationError {
	if len(m) == 0 {
		return nil
	}
	out := &routing.ActionValidationError{}
	if raw, ok := m["formError"].(string); ok && raw != "" {
		out.FormError = raw
	}
	if rawFieldErrors, ok := m["fieldErrors"].(map[string]interface{}); ok {
		fields := make(map[string]string, len(rawFieldErrors))
		for key, value := range rawFieldErrors {
			if msg, ok := value.(string); ok && msg != "" {
				fields[key] = msg
			}
		}
		if len(fields) > 0 {
			out.FieldErrors = fields
		}
	}
	if rawFieldErrors, ok := m["fieldErrors"].(map[string]string); ok && len(rawFieldErrors) > 0 {
		out.FieldErrors = rawFieldErrors
	}
	if out.FormError == "" && len(out.FieldErrors) == 0 {
		return nil
	}
	return out
}

// Get registers a GET route with the specified path and handlers.
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

// Post registers a POST route with the specified path and handlers.
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

// Put registers a PUT route with the specified path and handlers.
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

// Delete registers a DELETE route with the specified path and handlers.
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

// Group creates a new route group with the specified prefix and optional handlers.
func (a *App) Group(prefix string, handlers ...fiberpkg.Handler) fiberpkg.Router {
	hAny := make([]any, len(handlers))
	for i, h := range handlers {
		hAny[i] = h
	}
	return a.Fiber.Group(prefix, hAny...)
}

// Static registers a static directory with the specified prefix.
func (a *App) Static(prefix, root string) {
	a.Fiber.Use(prefix, static.New(root))
}

// GetHub returns the application's WebSocket hub.
func (a *App) GetHub() *fiber.WSHub {
	return a.Hub
}

// GetRouter returns the application's file-based router.
func (a *App) GetRouter() *routing.Router {
	return a.Router
}

// GetFiber returns the underlying Fiber application instance.
func (a *App) GetFiber() *fiberpkg.App {
	return a.Fiber
}

// Broadcast sends a message to all connected WebSocket clients.
func (a *App) Broadcast(message []byte) {
	if a.Hub != nil {
		a.Hub.Broadcast <- message
	}
}

// BroadcastState broadcasts a state update to all connected clients.
func (a *App) BroadcastState(key string, value interface{}) error {
	return fiber.BroadcastState(a.Hub, key, value)
}

// Computed adds a computed state variable to the application's global state.
// It automatically updates when its dependencies change and broadcasts the result to all clients.
func (a *App) Computed(key string, deps []string, fn func(map[string]interface{}) interface{}) *App {
	a.StateMap.AddComputed(key, deps, fn)
	return a
}

// Broadcast is a global convenience function to broadcast a message to all clients.
func Broadcast(message interface{}) error {
	if defaultApp == nil || defaultApp.Hub == nil {
		return fmt.Errorf("gospa app not initialized or websocket not enabled")
	}
	b, err := json.Marshal(message)
	if err != nil {
		return err
	}
	defaultApp.Hub.Broadcast <- b
	return nil
}

type fiberLoadContext struct {
	c fiberpkg.Ctx
}

func (f *fiberLoadContext) Param(key string) string {
	return f.c.Params(key)
}

func (f *fiberLoadContext) Params() map[string]string {
	if accessor, ok := any(f.c).(interface{ AllParams() map[string]string }); ok {
		all := accessor.AllParams()
		if all == nil {
			return map[string]string{}
		}
		out := make(map[string]string, len(all))
		for k, v := range all {
			out[k] = v
		}
		return out
	}
	return map[string]string{}
}

func (f *fiberLoadContext) Query(key string, defaultValue ...string) string {
	return f.c.Query(key, defaultValue...)
}

func (f *fiberLoadContext) QueryValues() map[string][]string {
	if accessor, ok := any(f.c).(interface{ Queries() map[string]string }); ok {
		raw := accessor.Queries()
		out := make(map[string][]string, len(raw))
		for k, v := range raw {
			out[k] = []string{v}
		}
		return out
	}
	return map[string][]string{}
}

func (f *fiberLoadContext) Header(key string) string {
	return f.c.Get(key)
}

func (f *fiberLoadContext) Headers() map[string]string {
	out := map[string]string{}
	for k, v := range f.c.Request().Header.All() {
		out[string(k)] = string(v)
	}
	return out
}

func (f *fiberLoadContext) SetHeader(key, value string) {
	if key == "" {
		return
	}
	f.c.Set(key, value)
}

func (f *fiberLoadContext) Cookie(key string) string {
	return f.c.Cookies(key)
}

func (f *fiberLoadContext) SetCookie(key, value string, maxAge int, path string, httpOnly, secure bool) {
	if key == "" {
		return
	}
	if path == "" {
		path = "/"
	}
	f.c.Cookie(&fiberpkg.Cookie{
		Name:     key,
		Value:    value,
		Path:     path,
		MaxAge:   maxAge,
		HTTPOnly: httpOnly,
		Secure:   secure,
	})
}

func (f *fiberLoadContext) FormValue(key string, defaultValue ...string) string {
	v := f.c.FormValue(key)
	if v == "" && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return v
}

func (f *fiberLoadContext) Method() string {
	return f.c.Method()
}

func (f *fiberLoadContext) Path() string {
	return f.c.Path()
}

func (f *fiberLoadContext) Local(key string) interface{} {
	return f.c.Locals(key)
}
