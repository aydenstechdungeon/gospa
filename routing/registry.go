// Package routing provides component registry for route handlers.
package routing

import (
	"sync"
	"time"

	"github.com/a-h/templ"
	fiberpkg "github.com/gofiber/fiber/v3"
)

// ComponentFunc is a function that returns a templ.Component.
type ComponentFunc func(props map[string]interface{}) templ.Component

// LayoutFunc is a function that returns a templ.Component for layouts.
type LayoutFunc func(children templ.Component, props map[string]interface{}) templ.Component

// MiddlewareFunc represents a generic middleware handler.
// Typically this is downcast to fiber.Handler when registering routes.
type MiddlewareFunc interface{}

// RenderStrategy defines how a page is rendered.
type RenderStrategy string

const (
	// StrategySSR renders fresh on every request (default).
	StrategySSR RenderStrategy = "ssr"
	// StrategySSG renders once and caches forever (FIFO eviction via SSGCacheMaxEntries).
	StrategySSG RenderStrategy = "ssg"
	// StrategyISR renders once, then revalidates in the background after RevalidateAfter elapses
	// (stale-while-revalidate). Requires CacheTemplates: true.
	StrategyISR RenderStrategy = "isr"
	// StrategyPPR renders a static shell once and streams only named DynamicSlots
	// per-request. Requires CacheTemplates: true.
	StrategyPPR RenderStrategy = "ppr"
)

// RouteOptions holds page-level options like rendering strategy.
type RouteOptions struct {
	Strategy RenderStrategy

	// ISR: duration after which the cached page is considered stale.
	// On a stale request the old page is served immediately and a background
	// goroutine re-renders and updates the cache (stale-while-revalidate).
	// Zero means "always revalidate" which behaves identically to SSR.
	RevalidateAfter time.Duration

	// PPR: names of dynamic slots that are excluded from the cached static shell
	// and re-rendered per-request. Each name must match a slot registered with
	// RegisterSlot for this page path.
	DynamicSlots []string
	// DeferredSlots are slots that are rendered out-of-order after the initial page load.
	DeferredSlots []string

	// RuntimeTier specifies the minimum client runtime tier required for this route.
	RuntimeTier string

	// Optional per-route rate limiter config.
	RateLimit *RateLimitOptions
}

// RateLimitOptions holds configuration for per-route rate limiters.
type RateLimitOptions struct {
	MaxRequests int
	Window      time.Duration
	Message     string
}

// SlotFunc returns a templ.Component for a named PPR dynamic slot.
type SlotFunc func(props map[string]interface{}) templ.Component

// LoadContext provides access to request data for server-side Load functions.
type LoadContext interface {
	Param(key string) string
	Query(key string, defaultValue ...string) string
	Header(key string) string
	Cookie(key string) string
	FormValue(key string, defaultValue ...string) string
	Method() string
	Path() string
}

// LoadFunc is a function that returns data for a page or layout.
type LoadFunc func(c LoadContext) (map[string]interface{}, error)

// ActionFunc is a function that handles a form action.
type ActionFunc func(c LoadContext) (interface{}, error)

// ActionRedirect describes a redirect response for progressive-enhanced forms.
type ActionRedirect struct {
	To     string `json:"to"`
	Status int    `json:"status,omitempty"`
}

// ActionValidationError contains form and field-level validation details.
type ActionValidationError struct {
	FieldErrors map[string]string `json:"fieldErrors,omitempty"`
	FormError   string            `json:"formError,omitempty"`
}

// ActionResponse is the structured response contract for form actions.
type ActionResponse struct {
	Data           interface{}            `json:"data,omitempty"`
	Code           string                 `json:"code,omitempty"`
	Redirect       *ActionRedirect        `json:"redirect,omitempty"`
	Validation     *ActionValidationError `json:"validation,omitempty"`
	Revalidate     []string               `json:"revalidate,omitempty"`
	RevalidateTags []string               `json:"revalidateTags,omitempty"`
	RevalidateKeys []string               `json:"revalidateKeys,omitempty"`
}

// HookFunc is a function that handles a server-side hook (middleware).
type HookFunc func(c fiberpkg.Ctx) error

// Registry holds registered page and layout components.
type Registry struct {
	pagesMu     sync.RWMutex
	pages       map[string]ComponentFunc
	pageOptions map[string]RouteOptions

	layoutsMu sync.RWMutex
	layouts   map[string]LayoutFunc

	errorsMu sync.RWMutex
	errors   map[string]ComponentFunc

	middlewaresMu sync.RWMutex
	middlewares   map[string]MiddlewareFunc

	loadingsMu sync.RWMutex
	loadings   map[string]ComponentFunc

	rootLayoutMu sync.RWMutex
	rootLayout   LayoutFunc

	loadFuncsMu sync.RWMutex
	loadFuncs   map[string]LoadFunc

	layoutLoaderMu sync.RWMutex
	layoutLoader   map[string]LoadFunc

	actionsMu sync.RWMutex
	actions   map[string]map[string]ActionFunc

	hooksMu sync.RWMutex
	hooks   []HookFunc

	slotsMu sync.RWMutex
	// slots maps pagePath → slotName → SlotFunc for PPR.
	slots map[string]map[string]SlotFunc

	layoutTiersMu sync.RWMutex
	// layoutTiers maps layoutPath → RuntimeTier
	layoutTiers map[string]string
}

// globalRegistry is the default global registry.
var globalRegistry = NewRegistry()

// NewRegistry creates a new component registry.
func NewRegistry() *Registry {
	return &Registry{
		pages:        make(map[string]ComponentFunc),
		pageOptions:  make(map[string]RouteOptions),
		layouts:      make(map[string]LayoutFunc),
		errors:       make(map[string]ComponentFunc),
		middlewares:  make(map[string]MiddlewareFunc),
		loadings:     make(map[string]ComponentFunc),
		loadFuncs:    make(map[string]LoadFunc),
		layoutLoader: make(map[string]LoadFunc),
		actions:      make(map[string]map[string]ActionFunc),
		hooks:        make([]HookFunc, 0),
		slots:        make(map[string]map[string]SlotFunc),
		layoutTiers:  make(map[string]string),
	}
}

// RegisterAction registers an action for a page path.
func (r *Registry) RegisterAction(pagePath, actionName string, fn ActionFunc) {
	r.actionsMu.Lock()
	defer r.actionsMu.Unlock()
	if r.actions[pagePath] == nil {
		r.actions[pagePath] = make(map[string]ActionFunc)
	}
	r.actions[pagePath][actionName] = fn
}

// GetActions returns all actions for a page path.
func (r *Registry) GetActions(pagePath string) map[string]ActionFunc {
	r.actionsMu.RLock()
	defer r.actionsMu.RUnlock()
	return r.actions[pagePath]
}

// GetAction returns a specific action for a page path.
func (r *Registry) GetAction(pagePath, actionName string) ActionFunc {
	r.actionsMu.RLock()
	defer r.actionsMu.RUnlock()
	if actions, ok := r.actions[pagePath]; ok {
		return actions[actionName]
	}
	return nil
}

// Global registration functions

// RegisterAction registers a page action in the global registry.
func RegisterAction(pagePath, actionName string, fn ActionFunc) {
	globalRegistry.RegisterAction(pagePath, actionName, fn)
}

// GetActions returns all actions for a page path from the global registry.
func GetActions(pagePath string) map[string]ActionFunc {
	return globalRegistry.GetActions(pagePath)
}

// GetAction returns a specific action for a page path from the global registry.
func GetAction(pagePath, actionName string) ActionFunc {
	return globalRegistry.GetAction(pagePath, actionName)
}

// RegisterHook registers a global server-side hook (middleware).
func (r *Registry) RegisterHook(fn HookFunc) {
	r.hooksMu.Lock()
	defer r.hooksMu.Unlock()
	r.hooks = append(r.hooks, fn)
}

// GetHooks returns all registered global hooks.
func (r *Registry) GetHooks() []HookFunc {
	r.hooksMu.RLock()
	defer r.hooksMu.RUnlock()
	return r.hooks
}

// Global registration functions

// RegisterHook registers a global server-side hook in the global registry.
func RegisterHook(fn HookFunc) {
	globalRegistry.RegisterHook(fn)
}

// GetHooks returns all registered global hooks from the global registry.
func GetHooks() []HookFunc {
	return globalRegistry.GetHooks()
}

// RegisterLoad registers a load function for a route path.
func (r *Registry) RegisterLoad(path string, fn LoadFunc) {
	r.loadFuncsMu.Lock()
	defer r.loadFuncsMu.Unlock()
	r.loadFuncs[path] = fn
}

// GetLoad returns the load function for a path.
func (r *Registry) GetLoad(path string) LoadFunc {
	r.loadFuncsMu.RLock()
	defer r.loadFuncsMu.RUnlock()
	return r.loadFuncs[path]
}

// RegisterLayoutLoad registers a load function for a layout path.
func (r *Registry) RegisterLayoutLoad(path string, fn LoadFunc) {
	r.layoutLoaderMu.Lock()
	defer r.layoutLoaderMu.Unlock()
	r.layoutLoader[path] = fn
}

// GetLayoutLoad returns the load function for a layout path.
func (r *Registry) GetLayoutLoad(path string) LoadFunc {
	r.layoutLoaderMu.RLock()
	defer r.layoutLoaderMu.RUnlock()
	return r.layoutLoader[path]
}

// RegisterPage registers a page component for a route path (default to SSR).
func (r *Registry) RegisterPage(path string, fn ComponentFunc) {
	r.RegisterPageWithOptions(path, fn, RouteOptions{Strategy: StrategySSR})
}

// RegisterPageWithOptions registers a page component with specific options.
func (r *Registry) RegisterPageWithOptions(path string, fn ComponentFunc, opts RouteOptions) {
	r.pagesMu.Lock()
	defer r.pagesMu.Unlock()
	r.pages[path] = fn
	r.pageOptions[path] = opts
}

// GetRouteOptions returns the route options for a path.
func (r *Registry) GetRouteOptions(path string) RouteOptions {
	r.pagesMu.RLock()
	defer r.pagesMu.RUnlock()
	if opts, ok := r.pageOptions[path]; ok {
		return opts
	}
	return RouteOptions{Strategy: StrategySSR}
}

// RegisterLayout registers a layout component for a route path.
func (r *Registry) RegisterLayout(path string, fn LayoutFunc) {
	r.RegisterLayoutWithOptions(path, fn, "")
}

// RegisterLayoutWithOptions registers a layout component with a specific runtime tier.
func (r *Registry) RegisterLayoutWithOptions(path string, fn LayoutFunc, tier string) {
	r.layoutsMu.Lock()
	r.layoutTiersMu.Lock()
	defer r.layoutsMu.Unlock()
	defer r.layoutTiersMu.Unlock()
	r.layouts[path] = fn
	r.layoutTiers[path] = tier
}

// GetLayoutTier returns the runtime tier for a layout path.
func (r *Registry) GetLayoutTier(path string) string {
	r.layoutTiersMu.RLock()
	defer r.layoutTiersMu.RUnlock()
	return r.layoutTiers[path]
}

// RegisterMiddleware registers a middleware function for a route path.
func (r *Registry) RegisterMiddleware(path string, fn MiddlewareFunc) {
	r.middlewaresMu.Lock()
	defer r.middlewaresMu.Unlock()
	r.middlewares[path] = fn
}

// RegisterLoading registers a loading component for a route path.
func (r *Registry) RegisterLoading(path string, fn ComponentFunc) {
	r.loadingsMu.Lock()
	defer r.loadingsMu.Unlock()
	r.loadings[path] = fn
}

// RegisterError registers an error component for a route path.
func (r *Registry) RegisterError(path string, fn ComponentFunc) {
	r.errorsMu.Lock()
	defer r.errorsMu.Unlock()
	r.errors[path] = fn
}

// GetPage returns the page component for a path.
func (r *Registry) GetPage(path string) ComponentFunc {
	r.pagesMu.RLock()
	defer r.pagesMu.RUnlock()
	return r.pages[path]
}

// GetLayout returns the layout component for a path.
func (r *Registry) GetLayout(path string) LayoutFunc {
	r.layoutsMu.RLock()
	defer r.layoutsMu.RUnlock()
	return r.layouts[path]
}

// GetMiddleware returns the middleware function for a path.
func (r *Registry) GetMiddleware(path string) MiddlewareFunc {
	r.middlewaresMu.RLock()
	defer r.middlewaresMu.RUnlock()
	return r.middlewares[path]
}

// GetLoading returns the loading component for a path.
func (r *Registry) GetLoading(path string) ComponentFunc {
	r.loadingsMu.RLock()
	defer r.loadingsMu.RUnlock()
	return r.loadings[path]
}

// GetError returns the error component for a path.
func (r *Registry) GetError(path string) ComponentFunc {
	r.errorsMu.RLock()
	defer r.errorsMu.RUnlock()
	return r.errors[path]
}

// RegisterRootLayout registers the root layout component with a runtime tier.
func (r *Registry) RegisterRootLayout(fn LayoutFunc, tier string) {
	r.rootLayoutMu.Lock()
	r.layoutTiersMu.Lock()
	defer r.rootLayoutMu.Unlock()
	defer r.layoutTiersMu.Unlock()
	r.rootLayout = fn
	r.layoutTiers[""] = tier
}

// GetRootLayout returns the root layout component.
func (r *Registry) GetRootLayout() LayoutFunc {
	r.rootLayoutMu.RLock()
	defer r.rootLayoutMu.RUnlock()
	return r.rootLayout
}

// HasPage checks if a page is registered for a path.
func (r *Registry) HasPage(path string) bool {
	r.pagesMu.RLock()
	defer r.pagesMu.RUnlock()
	_, ok := r.pages[path]
	return ok
}

// HasLayout checks if a layout is registered for a path.
func (r *Registry) HasLayout(path string) bool {
	r.layoutsMu.RLock()
	defer r.layoutsMu.RUnlock()
	_, ok := r.layouts[path]
	return ok
}

// RegisterSlot registers a PPR dynamic slot component for a page path.
func (r *Registry) RegisterSlot(pagePath, slotName string, fn SlotFunc) {
	r.slotsMu.Lock()
	defer r.slotsMu.Unlock()
	if r.slots[pagePath] == nil {
		r.slots[pagePath] = make(map[string]SlotFunc)
	}
	r.slots[pagePath][slotName] = fn
}

// GetSlot returns the SlotFunc for a named PPR slot on a page path.
func (r *Registry) GetSlot(pagePath, slotName string) SlotFunc {
	r.slotsMu.RLock()
	defer r.slotsMu.RUnlock()
	if m := r.slots[pagePath]; m != nil {
		return m[slotName]
	}
	return nil
}

// Global registry functions

// RegisterPage registers a page component in the global registry (default SSR).
func RegisterPage(path string, fn ComponentFunc) {
	globalRegistry.RegisterPage(path, fn)
}

// RegisterPageWithOptions registers a page in the global registry with options.
func RegisterPageWithOptions(path string, fn ComponentFunc, opts RouteOptions) {
	globalRegistry.RegisterPageWithOptions(path, fn, opts)
}

// GetRouteOptions returns route options from the global registry.
func GetRouteOptions(path string) RouteOptions {
	return globalRegistry.GetRouteOptions(path)
}

// RegisterLayout registers a layout component in the global registry.
func RegisterLayout(path string, fn LayoutFunc) {
	globalRegistry.RegisterLayout(path, fn)
}

// RegisterLayoutWithOptions registers a layout component with options in the global registry.
func RegisterLayoutWithOptions(path string, fn LayoutFunc, tier string) {
	globalRegistry.RegisterLayoutWithOptions(path, fn, tier)
}

// GetLayoutTier returns the runtime tier for a layout from the global registry.
func GetLayoutTier(path string) string {
	return globalRegistry.GetLayoutTier(path)
}

// RegisterMiddleware registers a middleware function in the global registry.
func RegisterMiddleware(path string, fn MiddlewareFunc) {
	globalRegistry.RegisterMiddleware(path, fn)
}

// RegisterLoading registers a loading component in the global registry.
func RegisterLoading(path string, fn ComponentFunc) {
	globalRegistry.RegisterLoading(path, fn)
}

// RegisterError registers an error component in the global registry.
func RegisterError(path string, fn ComponentFunc) {
	globalRegistry.RegisterError(path, fn)
}

// GetPage returns the page component from the global registry.
func GetPage(path string) ComponentFunc {
	return globalRegistry.GetPage(path)
}

// GetLayout returns the layout component from the global registry.
func GetLayout(path string) LayoutFunc {
	return globalRegistry.GetLayout(path)
}

// GetMiddleware returns the middleware function from the global registry.
func GetMiddleware(path string) MiddlewareFunc {
	return globalRegistry.GetMiddleware(path)
}

// GetLoading returns the loading component from the global registry.

// GetLoading returns the loading component from the global registry.
func GetLoading(path string) ComponentFunc {
	return globalRegistry.GetLoading(path)
}

// GetError returns the error component from the global registry.
func GetError(path string) ComponentFunc {
	return globalRegistry.GetError(path)
}

// RegisterRootLayout registers the root layout in the global registry with a runtime tier.
func RegisterRootLayout(fn LayoutFunc, tier string) {
	globalRegistry.RegisterRootLayout(fn, tier)
}

// GetRootLayout returns the root layout from the global registry.
func GetRootLayout() LayoutFunc {
	return globalRegistry.GetRootLayout()
}

// HasPage checks if a page is registered in the global registry.
func HasPage(path string) bool {
	return globalRegistry.HasPage(path)
}

// HasLayout checks if a layout is registered in the global registry.
func HasLayout(path string) bool {
	return globalRegistry.HasLayout(path)
}

// RegisterSlot registers a PPR dynamic slot in the global registry.
func RegisterSlot(pagePath, slotName string, fn SlotFunc) {
	globalRegistry.RegisterSlot(pagePath, slotName, fn)
}

// GetSlot returns a PPR slot from the global registry.
func GetSlot(pagePath, slotName string) SlotFunc {
	return globalRegistry.GetSlot(pagePath, slotName)
}

// RegisterLoad registers a load function in the global registry.
func RegisterLoad(path string, fn LoadFunc) {
	globalRegistry.RegisterLoad(path, fn)
}

// GetLoad returns a load function from the global registry.
func GetLoad(path string) LoadFunc {
	return globalRegistry.GetLoad(path)
}

// RegisterLayoutLoad registers a layout load function in the global registry.
func RegisterLayoutLoad(path string, fn LoadFunc) {
	globalRegistry.RegisterLayoutLoad(path, fn)
}

// GetLayoutLoad returns a layout load function from the global registry.
func GetLayoutLoad(path string) LoadFunc {
	return globalRegistry.GetLayoutLoad(path)
}

// GetGlobalRegistry returns the global registry.
func GetGlobalRegistry() *Registry {
	return globalRegistry
}
