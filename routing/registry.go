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

// Registry holds registered page and layout components.
type Registry struct {
	mu         sync.RWMutex
	pages      map[string]ComponentFunc
	layouts    map[string]LayoutFunc
	rootLayout LayoutFunc
}

// globalRegistry is the default global registry.
var globalRegistry = NewRegistry()

// NewRegistry creates a new component registry.
func NewRegistry() *Registry {
	return &Registry{
		pages:   make(map[string]ComponentFunc),
		layouts: make(map[string]LayoutFunc),
	}
}

// RegisterPage registers a page component for a route path.
func (r *Registry) RegisterPage(path string, fn ComponentFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pages[path] = fn
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

// RegisterPage registers a page component in the global registry.
func RegisterPage(path string, fn ComponentFunc) {
	globalRegistry.RegisterPage(path, fn)
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
