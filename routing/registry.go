// Package routing provides component registry for route handlers.
package routing

import (
	"sync"

	"github.com/a-h/templ"
)

// ComponentFunc is a function that returns a templ.Component.
type ComponentFunc func(props map[string]interface{}) templ.Component

// LayoutFunc is a function that returns a templ.Component for layouts.
type LayoutFunc func(children templ.Component, props map[string]interface{}) templ.Component

// RenderStrategy defines how a page is rendered.
type RenderStrategy string

const (
	StrategySSR RenderStrategy = "ssr"
	StrategySSG RenderStrategy = "ssg"
)

// RouteOptions holds page-level options like rendering strategy.
type RouteOptions struct {
	Strategy RenderStrategy
}

// Registry holds registered page and layout components.
type Registry struct {
	mu          sync.RWMutex
	pages       map[string]ComponentFunc
	pageOptions map[string]RouteOptions
	layouts     map[string]LayoutFunc
	rootLayout  LayoutFunc
}

// globalRegistry is the default global registry.
var globalRegistry = NewRegistry()

// NewRegistry creates a new component registry.
func NewRegistry() *Registry {
	return &Registry{
		pages:       make(map[string]ComponentFunc),
		pageOptions: make(map[string]RouteOptions),
		layouts:     make(map[string]LayoutFunc),
	}
}

// RegisterPage registers a page component for a route path (default to SSR).
func (r *Registry) RegisterPage(path string, fn ComponentFunc) {
	r.RegisterPageWithOptions(path, fn, RouteOptions{Strategy: StrategySSR})
}

// RegisterPageWithOptions registers a page component with specific options.
func (r *Registry) RegisterPageWithOptions(path string, fn ComponentFunc, opts RouteOptions) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pages[path] = fn
	r.pageOptions[path] = opts
}

// GetRouteOptions returns the route options for a path.
func (r *Registry) GetRouteOptions(path string) RouteOptions {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if opts, ok := r.pageOptions[path]; ok {
		return opts
	}
	return RouteOptions{Strategy: StrategySSR}
}

// RegisterLayout registers a layout component for a route path.
func (r *Registry) RegisterLayout(path string, fn LayoutFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.layouts[path] = fn
}

// GetPage returns the page component for a path.
func (r *Registry) GetPage(path string) ComponentFunc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.pages[path]
}

// GetLayout returns the layout component for a path.
func (r *Registry) GetLayout(path string) LayoutFunc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.layouts[path]
}

// RegisterRootLayout registers the root layout component.
func (r *Registry) RegisterRootLayout(fn LayoutFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rootLayout = fn
}

// GetRootLayout returns the root layout component.
func (r *Registry) GetRootLayout() LayoutFunc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.rootLayout
}

// HasPage checks if a page is registered for a path.
func (r *Registry) HasPage(path string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.pages[path]
	return ok
}

// HasLayout checks if a layout is registered for a path.
func (r *Registry) HasLayout(path string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.layouts[path]
	return ok
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

// GetPage returns the page component from the global registry.
func GetPage(path string) ComponentFunc {
	return globalRegistry.GetPage(path)
}

// GetLayout returns the layout component from the global registry.
func GetLayout(path string) LayoutFunc {
	return globalRegistry.GetLayout(path)
}

// RegisterRootLayout registers the root layout in the global registry.
func RegisterRootLayout(fn LayoutFunc) {
	globalRegistry.RegisterRootLayout(fn)
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

// GetGlobalRegistry returns the global registry.
func GetGlobalRegistry() *Registry {
	return globalRegistry
}
