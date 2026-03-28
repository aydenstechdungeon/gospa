// Package gospa provides a modern SPA framework for Go with Fiber and Templ.
// It brings Svelte-like reactivity and state management to Go.
package gospa

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/aydenstechdungeon/gospa/embed"
	"github.com/aydenstechdungeon/gospa/fiber"
	"github.com/aydenstechdungeon/gospa/plugin"
	"github.com/aydenstechdungeon/gospa/routing"
	"github.com/aydenstechdungeon/gospa/state"
	"github.com/aydenstechdungeon/gospa/store"
	json "github.com/goccy/go-json"
	fiberpkg "github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/logger"
	recovermw "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"
)

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
	// pprShellBuilding guards against duplicate PPR shell builds under concurrent load.
	pprShellBuilding sync.Map
}

var defaultApp *App

// New creates a new GoSPA application with the given configuration.
func New(config Config) *App {
	if config.AppName == "" {
		config.AppName = "GoSPA Application"
	}
	if !config.EnableWebSocket && config.WebSocketPath == "" {
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
		config.HydrationMode = "immediate"
	}
	// HydrationTimeout validation: must be within 0-10s to prevent hanging or UI jank
	if config.HydrationTimeout < 0 {
		config.HydrationTimeout = 0
	} else if config.HydrationTimeout > 10000 {
		config.Logger.Warn("HydrationTimeout is too high (>10s). Capping to 10 seconds for UX safety.", "value", config.HydrationTimeout)
		config.HydrationTimeout = 10000
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
	// SECURITY FIX: GOSPA_WS_INSECURE must only be honoured in dev mode.
	// In production, this env var is intentionally ignored to prevent accidental
	// misconfiguration that would allow plaintext ws:// connections in production.
	if !config.AllowInsecureWS && os.Getenv("GOSPA_WS_INSECURE") == "1" {
		if config.DevMode {
			config.AllowInsecureWS = true
		} else {
			config.Logger.Warn("GOSPA_WS_INSECURE=1 is set but is ignored because DevMode is false. This override only applies in development environments.")
		}
	}

	fiber.SetConnectionRateLimiter(config.WSConnBurst, config.WSConnRateLimit)
	state.SetNotificationQueueSize(config.NotificationBufferSize)

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
	}

	app.setupMiddleware()

	if defaultApp == nil {
		defaultApp = app
	}

	return app
}

// setupRoutes configures core internal routes.
func (a *App) setupRoutes() {
	a.Fiber.Get(a.getRuntimePath(), fiber.RuntimeMiddleware(a.Config.SimpleRuntime))

	a.Fiber.Use("/_gospa/", func(c fiberpkg.Ctx) error {
		c.Set("Cache-Control", "public, max-age=31536000, immutable")
		return c.Next()
	})
	a.Fiber.Use("/_gospa/", static.New("", static.Config{
		FS: embed.RuntimeFS(),
	}))

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

	if _, err := os.Stat(a.Config.StaticDir); err == nil {
		a.Fiber.Use(a.Config.StaticPrefix, static.New(a.Config.StaticDir))
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
		return c.Status(fiberpkg.StatusInternalServerError).JSON(fiberpkg.Map{
			"error": "Internal server error",
			"code":  "ACTION_FAILED",
		})
	}

	return c.JSON(fiberpkg.Map{
		"data": result,
		"code": "SUCCESS",
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
	a.Fiber.Use(fiber.PreloadHeadersMiddleware(preloadConfig))

	spaConfig := fiber.DefaultConfig()
	spaConfig.DevMode = a.Config.DevMode
	spaConfig.RuntimeScript = a.Config.RuntimeScript
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
	if err := plugin.TriggerHook(plugin.BeforePrune, nil); err != nil {
		a.Logger().Error("plugin BeforePrune hook failed", "err", err)
	}
	if a.Hub != nil {
		a.Hub.Close()
	}
	if closer, ok := a.Config.Storage.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
	err := a.Fiber.Shutdown()
	if err := plugin.TriggerHook(plugin.AfterPrune, nil); err != nil {
		a.Logger().Error("plugin AfterPrune hook failed", "err", err)
	}
	return err
}

// RegisterRoutes manually triggers route registration.
func (a *App) RegisterRoutes() error {
	if err := a.Scan(); err != nil {
		return err
	}
	for _, route := range a.Router.GetPages() {
		r := route
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
			return a.renderRoute(c, r)
		})
		a.Fiber.Get(r.Path, handlers[0], handlers[1:]...)
	}
	return nil
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
