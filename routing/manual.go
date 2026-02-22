package routing

import (
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
)

// Handler is a generic HTTP handler function.
type Handler func(c *fiber.Ctx) error

// Middleware is a Fiber middleware function.
type Middleware func(c *fiber.Ctx) error

// ManualRoute represents a manually registered route.
type ManualRoute struct {
	// Path is the URL path pattern
	Path string
	// Method is the HTTP method (GET, POST, etc.)
	Method string
	// Handler is the route handler
	Handler Handler
	// Component is the Templ component (for page routes)
	Component templ.Component
	// Middleware is the route-specific middleware
	Middleware []Middleware
	// Params are the parameter names
	Params []string
	// IsDynamic indicates if the route has dynamic segments
	IsDynamic bool
	// Priority is used for route matching order
	Priority int
	// Meta contains route metadata
	Meta map[string]interface{}
}

// RouteGroup represents a group of routes with shared configuration.
type RouteGroup struct {
	// Prefix is the URL prefix for all routes in the group
	Prefix string
	// Middleware is applied to all routes in the group
	Middleware []Middleware
	// Routes are the routes in this group
	Routes []*ManualRoute
	// Groups are nested route groups
	Groups []*RouteGroup
	// Parent is the parent group (nil for root)
	Parent *RouteGroup
	// Meta contains group metadata
	Meta map[string]interface{}
	mu   sync.RWMutex
}

// ManualRouter manages manually registered routes.
type ManualRouter struct {
	routes     []*ManualRoute
	groups     []*RouteGroup
	mu         sync.RWMutex
	autoRouter *Router
}

// NewManualRouter creates a new manual router.
func NewManualRouter() *ManualRouter {
	return &ManualRouter{
		routes: make([]*ManualRoute, 0),
		groups: make([]*RouteGroup, 0),
	}
}

// SetAutoRouter sets the auto router for hybrid routing.
func (mr *ManualRouter) SetAutoRouter(router *Router) {
	mr.autoRouter = router
}

// RegisterRoute registers a manual route.
func (mr *ManualRouter) RegisterRoute(method, path string, handler Handler, middleware ...Middleware) *ManualRoute {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	params, isDynamic, _ := extractParams(path)
	priority := calculatePriority(path, isDynamic, false)

	route := &ManualRoute{
		Path:       path,
		Method:     method,
		Handler:    handler,
		Middleware: middleware,
		Params:     params,
		IsDynamic:  isDynamic,
		Priority:   priority,
		Meta:       make(map[string]interface{}),
	}

	mr.routes = append(mr.routes, route)
	mr.sortRoutes()

	return route
}

// RegisterComponent registers a Templ component as a page route.
func (mr *ManualRouter) RegisterComponent(path string, component templ.Component, middleware ...Middleware) *ManualRoute {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	params, isDynamic, _ := extractParams(path)
	priority := calculatePriority(path, isDynamic, false)

	route := &ManualRoute{
		Path:       path,
		Method:     http.MethodGet,
		Component:  component,
		Middleware: middleware,
		Params:     params,
		IsDynamic:  isDynamic,
		Priority:   priority,
		Meta:       make(map[string]interface{}),
	}

	mr.routes = append(mr.routes, route)
	mr.sortRoutes()

	return route
}

// GET registers a GET route.
func (mr *ManualRouter) GET(path string, handler Handler, middleware ...Middleware) *ManualRoute {
	return mr.RegisterRoute(http.MethodGet, path, handler, middleware...)
}

// POST registers a POST route.
func (mr *ManualRouter) POST(path string, handler Handler, middleware ...Middleware) *ManualRoute {
	return mr.RegisterRoute(http.MethodPost, path, handler, middleware...)
}

// PUT registers a PUT route.
func (mr *ManualRouter) PUT(path string, handler Handler, middleware ...Middleware) *ManualRoute {
	return mr.RegisterRoute(http.MethodPut, path, handler, middleware...)
}

// DELETE registers a DELETE route.
func (mr *ManualRouter) DELETE(path string, handler Handler, middleware ...Middleware) *ManualRoute {
	return mr.RegisterRoute(http.MethodDelete, path, handler, middleware...)
}

// PATCH registers a PATCH route.
func (mr *ManualRouter) PATCH(path string, handler Handler, middleware ...Middleware) *ManualRoute {
	return mr.RegisterRoute(http.MethodPatch, path, handler, middleware...)
}

// OPTIONS registers an OPTIONS route.
func (mr *ManualRouter) OPTIONS(path string, handler Handler, middleware ...Middleware) *ManualRoute {
	return mr.RegisterRoute(http.MethodOptions, path, handler, middleware...)
}

// HEAD registers a HEAD route.
func (mr *ManualRouter) HEAD(path string, handler Handler, middleware ...Middleware) *ManualRoute {
	return mr.RegisterRoute(http.MethodHead, path, handler, middleware...)
}

// Page registers a Templ component as a page.
func (mr *ManualRouter) Page(path string, component templ.Component, middleware ...Middleware) *ManualRoute {
	return mr.RegisterComponent(path, component, middleware...)
}

// Group creates a new route group.
func (mr *ManualRouter) Group(prefix string, middleware ...Middleware) *RouteGroup {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	group := &RouteGroup{
		Prefix:     prefix,
		Middleware: middleware,
		Routes:     make([]*ManualRoute, 0),
		Groups:     make([]*RouteGroup, 0),
		Meta:       make(map[string]interface{}),
	}

	mr.groups = append(mr.groups, group)
	return group
}

// sortRoutes sorts routes by priority.
func (mr *ManualRouter) sortRoutes() {
	sort.Slice(mr.routes, func(i, j int) bool {
		return mr.routes[i].Priority < mr.routes[j].Priority
	})
}

// GetRoutes returns all manual routes.
func (mr *ManualRouter) GetRoutes() []*ManualRoute {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	routes := make([]*ManualRoute, len(mr.routes))
	copy(routes, mr.routes)
	return routes
}

// Match matches a URL path and method to a route.
func (mr *ManualRouter) Match(method, path string) (*ManualRoute, Params) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	for _, route := range mr.routes {
		if route.Method != method && route.Method != "*" {
			continue
		}

		extractor := NewParamExtractor(route.Path)
		if params, ok := extractor.Extract(path); ok {
			return route, params
		}
	}

	return nil, nil
}

// MatchAll returns all routes matching a path (for any method).
func (mr *ManualRouter) MatchAll(path string) []*ManualRoute {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	matches := make([]*ManualRoute, 0)
	for _, route := range mr.routes {
		extractor := NewParamExtractor(route.Path)
		if extractor.Match(path) {
			matches = append(matches, route)
		}
	}
	return matches
}

// RegisterToFiber registers all routes with a Fiber app.
func (mr *ManualRouter) RegisterToFiber(app *fiber.App) error {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	for _, route := range mr.routes {
		if err := mr.registerRouteToFiber(app, route); err != nil {
			return err
		}
	}

	for _, group := range mr.groups {
		if err := mr.registerGroupToFiber(app, group); err != nil {
			return err
		}
	}

	return nil
}

// registerRouteToFiber registers a single route with Fiber.
func (mr *ManualRouter) registerRouteToFiber(app *fiber.App, route *ManualRoute) error {
	handler := route.Handler

	// Wrap with middleware
	for i := len(route.Middleware) - 1; i >= 0; i-- {
		mw := route.Middleware[i]
		h := handler
		handler = func(c *fiber.Ctx) error {
			if err := mw(c); err != nil {
				return err
			}
			return h(c)
		}
	}

	// Register with Fiber
	switch route.Method {
	case http.MethodGet:
		app.Get(route.Path, handler)
	case http.MethodPost:
		app.Post(route.Path, handler)
	case http.MethodPut:
		app.Put(route.Path, handler)
	case http.MethodDelete:
		app.Delete(route.Path, handler)
	case http.MethodPatch:
		app.Patch(route.Path, handler)
	case http.MethodOptions:
		app.Options(route.Path, handler)
	case http.MethodHead:
		app.Head(route.Path, handler)
	case "*":
		app.All(route.Path, handler)
	default:
		return fmt.Errorf("unsupported HTTP method: %s", route.Method)
	}

	return nil
}

// registerGroupToFiber registers a route group with Fiber.
func (mr *ManualRouter) registerGroupToFiber(app *fiber.App, group *RouteGroup) error {
	// Convert Middleware to fiber.Handler
	fiberMiddleware := make([]fiber.Handler, len(group.Middleware))
	for i, mw := range group.Middleware {
		fiberMiddleware[i] = mw
	}

	// Create Fiber group
	fg := app.Group(group.Prefix, fiberMiddleware...)

	// Register routes
	for _, route := range group.Routes {
		handler := route.Handler

		// Wrap with route middleware
		for i := len(route.Middleware) - 1; i >= 0; i-- {
			mw := route.Middleware[i]
			h := handler
			handler = func(c *fiber.Ctx) error {
				if err := mw(c); err != nil {
					return err
				}
				return h(c)
			}
		}

		// Register with Fiber group
		switch route.Method {
		case http.MethodGet:
			fg.Get(route.Path, handler)
		case http.MethodPost:
			fg.Post(route.Path, handler)
		case http.MethodPut:
			fg.Put(route.Path, handler)
		case http.MethodDelete:
			fg.Delete(route.Path, handler)
		case http.MethodPatch:
			fg.Patch(route.Path, handler)
		case http.MethodOptions:
			fg.Options(route.Path, handler)
		case http.MethodHead:
			fg.Head(route.Path, handler)
		case "*":
			fg.All(route.Path, handler)
		}
	}

	// Register nested groups
	for _, nestedGroup := range group.Groups {
		if err := mr.registerNestedGroupToFiber(fg, nestedGroup); err != nil {
			return err
		}
	}

	return nil
}

// registerNestedGroupToFiber registers a nested group with a Fiber group.
func (mr *ManualRouter) registerNestedGroupToFiber(fg fiber.Router, group *RouteGroup) error {
	// Convert Middleware to fiber.Handler
	fiberMiddleware := make([]fiber.Handler, len(group.Middleware))
	for i, mw := range group.Middleware {
		fiberMiddleware[i] = mw
	}

	// Create nested Fiber group
	nfg := fg.Group(group.Prefix, fiberMiddleware...)

	// Register routes
	for _, route := range group.Routes {
		handler := route.Handler

		for i := len(route.Middleware) - 1; i >= 0; i-- {
			mw := route.Middleware[i]
			h := handler
			handler = func(c *fiber.Ctx) error {
				if err := mw(c); err != nil {
					return err
				}
				return h(c)
			}
		}

		switch route.Method {
		case http.MethodGet:
			nfg.Get(route.Path, handler)
		case http.MethodPost:
			nfg.Post(route.Path, handler)
		case http.MethodPut:
			nfg.Put(route.Path, handler)
		case http.MethodDelete:
			nfg.Delete(route.Path, handler)
		case http.MethodPatch:
			nfg.Patch(route.Path, handler)
		case http.MethodOptions:
			nfg.Options(route.Path, handler)
		case http.MethodHead:
			nfg.Head(route.Path, handler)
		case "*":
			nfg.All(route.Path, handler)
		}
	}

	// Register nested groups
	for _, nestedGroup := range group.Groups {
		if err := mr.registerNestedGroupToFiber(nfg, nestedGroup); err != nil {
			return err
		}
	}

	return nil
}

// RouteGroup methods

// GET registers a GET route in the group.
func (g *RouteGroup) GET(path string, handler Handler, middleware ...Middleware) *ManualRoute {
	return g.RegisterRoute(http.MethodGet, path, handler, middleware...)
}

// POST registers a POST route in the group.
func (g *RouteGroup) POST(path string, handler Handler, middleware ...Middleware) *ManualRoute {
	return g.RegisterRoute(http.MethodPost, path, handler, middleware...)
}

// PUT registers a PUT route in the group.
func (g *RouteGroup) PUT(path string, handler Handler, middleware ...Middleware) *ManualRoute {
	return g.RegisterRoute(http.MethodPut, path, handler, middleware...)
}

// DELETE registers a DELETE route in the group.
func (g *RouteGroup) DELETE(path string, handler Handler, middleware ...Middleware) *ManualRoute {
	return g.RegisterRoute(http.MethodDelete, path, handler, middleware...)
}

// PATCH registers a PATCH route in the group.
func (g *RouteGroup) PATCH(path string, handler Handler, middleware ...Middleware) *ManualRoute {
	return g.RegisterRoute(http.MethodPatch, path, handler, middleware...)
}

// RegisterRoute registers a route in the group.
func (g *RouteGroup) RegisterRoute(method, path string, handler Handler, middleware ...Middleware) *ManualRoute {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Combine group middleware with route middleware
	allMiddleware := make([]Middleware, 0, len(g.Middleware)+len(middleware))
	allMiddleware = append(allMiddleware, g.Middleware...)
	allMiddleware = append(allMiddleware, middleware...)

	// Combine prefix with path
	fullPath := g.Prefix + path

	params, isDynamic, _ := extractParams(fullPath)
	priority := calculatePriority(fullPath, isDynamic, false)

	route := &ManualRoute{
		Path:       fullPath,
		Method:     method,
		Handler:    handler,
		Middleware: allMiddleware,
		Params:     params,
		IsDynamic:  isDynamic,
		Priority:   priority,
		Meta:       make(map[string]interface{}),
	}

	g.Routes = append(g.Routes, route)
	return route
}

// Page registers a Templ component as a page in the group.
func (g *RouteGroup) Page(path string, component templ.Component, middleware ...Middleware) *ManualRoute {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Combine group middleware with route middleware
	allMiddleware := make([]Middleware, 0, len(g.Middleware)+len(middleware))
	allMiddleware = append(allMiddleware, g.Middleware...)
	allMiddleware = append(allMiddleware, middleware...)

	// Combine prefix with path
	fullPath := g.Prefix + path

	params, isDynamic, _ := extractParams(fullPath)
	priority := calculatePriority(fullPath, isDynamic, false)

	route := &ManualRoute{
		Path:       fullPath,
		Method:     http.MethodGet,
		Component:  component,
		Middleware: allMiddleware,
		Params:     params,
		IsDynamic:  isDynamic,
		Priority:   priority,
		Meta:       make(map[string]interface{}),
	}

	g.Routes = append(g.Routes, route)
	return route
}

// Group creates a nested route group.
func (g *RouteGroup) Group(prefix string, middleware ...Middleware) *RouteGroup {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Combine parent middleware with new middleware
	allMiddleware := make([]Middleware, 0, len(g.Middleware)+len(middleware))
	allMiddleware = append(allMiddleware, g.Middleware...)
	allMiddleware = append(allMiddleware, middleware...)

	group := &RouteGroup{
		Prefix:     g.Prefix + prefix,
		Middleware: allMiddleware,
		Routes:     make([]*ManualRoute, 0),
		Groups:     make([]*RouteGroup, 0),
		Parent:     g,
		Meta:       make(map[string]interface{}),
	}

	g.Groups = append(g.Groups, group)
	return group
}

// Use adds middleware to the group.
func (g *RouteGroup) Use(middleware ...Middleware) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Middleware = append(g.Middleware, middleware...)
}

// SetMeta sets metadata on the route.
func (r *ManualRoute) SetMeta(key string, value interface{}) *ManualRoute {
	r.Meta[key] = value
	return r
}

// GetMeta gets metadata from the route.
func (r *ManualRoute) GetMeta(key string) (interface{}, bool) {
	val, ok := r.Meta[key]
	return val, ok
}

// Name sets a name for the route.
func (r *ManualRoute) Name(name string) *ManualRoute {
	r.Meta["name"] = name
	return r
}

// GetName gets the route name.
func (r *ManualRoute) GetName() string {
	if name, ok := r.Meta["name"].(string); ok {
		return name
	}
	return ""
}

// String returns a string representation of the route.
func (r *ManualRoute) String() string {
	return fmt.Sprintf("ManualRoute{Method: %s, Path: %s, Params: %v}",
		r.Method, r.Path, r.Params)
}
