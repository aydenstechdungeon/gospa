// Package routing provides file-based routing similar to SvelteKit.
// It scans .templ files in a routes directory and maps them to URLs.
package routing

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// RouteType represents the type of route.
type RouteType int

const (
	// RouteTypePage represents a page route.
	RouteTypePage RouteType = iota
	// RouteTypeLayout represents a layout component route
	RouteTypeLayout
	// RouteTypeError represents an error component route
	RouteTypeError
	// RouteTypeAPI represents an API component route
	RouteTypeAPI
	// RouteTypeMiddleware represents a middleware route
	RouteTypeMiddleware
	// RouteTypeLoading represents a loading component route
	RouteTypeLoading
)

// Route represents a parsed route from the filesystem.
type Route struct {
	// Path is the URL path pattern (e.g., /blog/:id)
	Path string
	// File is the absolute path to the .templ file
	File string
	// Type indicates the route type
	Type RouteType
	// Params are the extracted parameter names
	Params []string
	// IsDynamic indicates if the route has dynamic segments
	IsDynamic bool
	// IsCatchAll indicates if the route has a catch-all segment
	IsCatchAll bool
	// Priority is used for route matching order
	Priority int
	// Children are nested routes
	Children []*Route
	// Layout is the parent layout route
	Layout *Route
	// Middleware is the middleware chain for this route
	Middleware []string
	// regexCache stores the compiled regex pattern for this route (computed once)
	regexCache *regexp.Regexp
	// regexOnce ensures regex is compiled only once
	regexOnce sync.Once
	// matchSegments stores precompiled route segments for dynamic matching.
	matchSegments []routeSegment
}

type routeSegmentKind uint8

const (
	segmentStatic routeSegmentKind = iota
	segmentParam
	segmentOptionalParam
	segmentCatchAll
	segmentOptionalCatchAll
)

type routeSegment struct {
	kind  routeSegmentKind
	value string
}

// Router manages all routes.
type Router struct {
	routes          []*Route
	fs              fs.FS
	layoutIndex     map[string]*Route
	middlewareIndex map[string]*Route
	errorRouteIndex map[string]*Route
	staticPageIndex map[string]*Route
	dynamicRoutes   []*Route
}

// NewRouter creates a new router with the given routes directory or filesystem.
// You can pass a string (directory path) or fs.FS.
func NewRouter(routesSource interface{}) *Router {
	var fileSystem fs.FS

	switch src := routesSource.(type) {
	case string:
		fileSystem = os.DirFS(src)
	case fs.FS:
		fileSystem = src
	default:
		// Fallback
		fileSystem = os.DirFS("./routes")
	}

	return &Router{
		routes:          make([]*Route, 0),
		fs:              fileSystem,
		layoutIndex:     make(map[string]*Route),
		middlewareIndex: make(map[string]*Route),
		errorRouteIndex: make(map[string]*Route),
		staticPageIndex: make(map[string]*Route),
		dynamicRoutes:   make([]*Route, 0),
	}
}

// Scan scans the routes directory and builds the route tree.
func (r *Router) Scan() error {
	// Reset previously discovered routes so repeated Scan calls are idempotent.
	r.routes = r.routes[:0]

	type routeKey struct {
		path  string
		rType RouteType
	}
	bestRoutes := make(map[routeKey]*Route)

	// Walk the routes filesystem
	err := fs.WalkDir(r.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process .templ, .gospa files, _middleware.go and +middleware.go
		base := filepath.Base(path)
		if !strings.HasSuffix(path, ".templ") && !strings.HasSuffix(path, ".gospa") &&
			base != "_middleware.go" && base != "+middleware.go" && base != "+server.go" {
			return nil
		}

		// Parse the route. path is already relative to the fs root.
		route, err := r.parseRoute(path)
		if err != nil {
			return fmt.Errorf("failed to parse route %s: %w", path, err)
		}

		key := routeKey{path: route.Path, rType: route.Type}
		existing, ok := bestRoutes[key]
		if !ok {
			bestRoutes[key] = route
			return nil
		}

		// Prioritization logic: + prefix wins
		currentBase := filepath.Base(route.File)
		existingBase := filepath.Base(existing.File)

		currentIsPlus := strings.HasPrefix(currentBase, "+")
		existingIsPlus := strings.HasPrefix(existingBase, "+")

		if currentIsPlus && !existingIsPlus {
			bestRoutes[key] = route
		}
		// If both are plus or both are not plus, we keep the first one found (usually not an issue if follow naming)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan routes: %w", err)
	}

	// Collect best routes
	for _, route := range bestRoutes {
		r.routes = append(r.routes, route)
	}

	// Sort routes by priority
	sort.Slice(r.routes, func(i, j int) bool {
		return r.routes[i].Priority < r.routes[j].Priority
	})

	// Build layout hierarchy
	r.buildLayoutHierarchy()
	r.rebuildIndexes()

	return nil
}

// parseRoute parses a file path into a Route.
func (r *Router) parseRoute(relPath string) (*Route, error) {
	// Normalize path separators (fs.FS uses slash, but just in case)
	relPath = filepath.ToSlash(relPath)

	// Determine route type
	routeType := RouteTypePage
	fileName := filepath.Base(relPath)
	cleanFileName := strings.TrimPrefix(fileName, "+")

	switch {
	case cleanFileName == "page.templ" || cleanFileName == "page.gospa":
		routeType = RouteTypePage
	case cleanFileName == "layout.templ" || cleanFileName == "layout.gospa":
		routeType = RouteTypeLayout
	case cleanFileName == "error.templ" || cleanFileName == "error.gospa" ||
		cleanFileName == "_error.templ" || cleanFileName == "_error.gospa":
		routeType = RouteTypeError
	case fileName == "_middleware.go" || fileName == "+middleware.go":
		routeType = RouteTypeMiddleware
	case cleanFileName == "_loading.templ" || cleanFileName == "loading.templ" || cleanFileName == "_loading.gospa" || cleanFileName == "loading.gospa":
		routeType = RouteTypeLoading
	case strings.HasSuffix(fileName, "+server.go"):
		routeType = RouteTypeAPI
	}

	// Convert file path to URL path
	urlPath := r.filePathToURLPath(relPath, routeType)

	// Extract parameters
	params, isDynamic, isCatchAll := extractParams(urlPath)

	// Calculate priority (lower = higher priority)
	priority := calculatePriority(urlPath, isDynamic, isCatchAll)

	return &Route{
		Path:          urlPath,
		File:          relPath,
		Type:          routeType,
		Params:        params,
		IsDynamic:     isDynamic,
		IsCatchAll:    isCatchAll,
		Priority:      priority,
		Children:      make([]*Route, 0),
		matchSegments: compileRouteSegments(urlPath),
	}, nil
}

// filePathToURLPath converts a file path to a URL path pattern.
func (r *Router) filePathToURLPath(relPath string, _ RouteType) string {
	// Remove file extension
	path := strings.TrimSuffix(relPath, filepath.Ext(relPath))

	// Handle different route types
	// Check for exact matches (root level) and path suffixes
	fileName := filepath.Base(path)
	cleanFileName := strings.TrimPrefix(fileName, "+")
	dirPath := filepath.Dir(path)

	switch {
	case cleanFileName == "page":
		// Root +page.templ -> /, nested +page.templ -> parent path
		if dirPath == "." {
			path = ""
		} else {
			path = dirPath
		}
	case cleanFileName == "layout":
		if dirPath == "." {
			path = ""
		} else {
			path = dirPath
		}
	case cleanFileName == "error":
		if dirPath == "." {
			path = ""
		} else {
			path = dirPath
		}
	case cleanFileName == "_error":
		if dirPath == "." {
			path = ""
		} else {
			path = dirPath
		}
	case fileName == "_middleware" || fileName == "+middleware":
		if dirPath == "." {
			path = ""
		} else {
			path = dirPath
		}
	case cleanFileName == "_loading" || cleanFileName == "loading":
		if dirPath == "." {
			path = ""
		} else {
			path = dirPath
		}
	case strings.HasSuffix(fileName, "+server"):
		// API route: remove +server suffix
		if dirPath == "." {
			path = ""
		} else {
			path = dirPath
		}
	}

	// Clean the path
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		path = "/"
	}

	// Convert [param] to :param and [...rest] to *rest
	path = convertDynamicSegments(path)

	// Ensure leading slash
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return path
}

// convertDynamicSegments converts [param] to :param, [...rest] to *rest,
// _param to :param (underscore prefix convention for Go-compatible directory names),
// and strips (name) route groups (parentheses convention for organizational folders
// that don't affect the URL path).
func convertDynamicSegments(path string) string {
	segments := strings.Split(path, "/")
	result := make([]string, 0, len(segments))

	for _, seg := range segments {
		if seg == "" {
			continue
		}

		// Check for route group (name) - strip from path entirely
		// Route groups organize routes without affecting the URL
		if strings.HasPrefix(seg, "(") && strings.HasSuffix(seg, ")") {
			// Skip this segment - it's a route group
			continue
		}

		// Check for catch-all [...rest]
		if strings.HasPrefix(seg, "[...") && strings.HasSuffix(seg, "]") {
			param := seg[4 : len(seg)-1]
			result = append(result, "*"+param)
			continue
		}

		// Check for optional catch-all [[...rest]]
		if strings.HasPrefix(seg, "[[...") && strings.HasSuffix(seg, "]]") {
			param := seg[5 : len(seg)-2]
			result = append(result, "*?"+param)
			continue
		}

		// Check for optional [[param]]
		if strings.HasPrefix(seg, "[[") && strings.HasSuffix(seg, "]]") {
			param := seg[2 : len(seg)-2]
			result = append(result, ":?"+param)
			continue
		}

		// Check for dynamic [param]
		if strings.HasPrefix(seg, "[") && strings.HasSuffix(seg, "]") {
			param := seg[1 : len(seg)-1]
			result = append(result, ":"+param)
			continue
		}

		// Check for underscore prefix _param (Go-compatible dynamic segment naming)
		// This allows directory names like _id instead of [id] since Go module paths
		// cannot contain brackets.
		if strings.HasPrefix(seg, "_") && len(seg) > 1 {
			param := seg[1:]
			result = append(result, ":"+param)
			continue
		}

		// Static segment
		result = append(result, seg)
	}

	return "/" + strings.Join(result, "/")
}

// extractParams extracts parameter names from a URL path.
func extractParams(path string) (params []string, isDynamic bool, isCatchAll bool) {
	segments := strings.Split(path, "/")

	for _, seg := range segments {
		if seg == "" {
			continue
		}

		// Catch-all parameter
		if strings.HasPrefix(seg, "*") {
			if strings.HasPrefix(seg, "*?") {
				params = append(params, seg[2:])
			} else {
				params = append(params, seg[1:])
			}
			isDynamic = true
			isCatchAll = true
			continue
		}

		// Dynamic parameter
		if strings.HasPrefix(seg, ":") {
			if strings.HasPrefix(seg, ":?") {
				params = append(params, seg[2:])
			} else {
				params = append(params, seg[1:])
			}
			isDynamic = true
			continue
		}
	}

	return params, isDynamic, isCatchAll
}

// calculatePriority calculates route priority for matching.
// Lower values = higher priority.
func calculatePriority(path string, _ bool, _ bool) int {
	segments := strings.Split(path, "/")
	priority := 0

	for i, seg := range segments {
		if seg == "" {
			continue
		}

		// Catch-all has lowest priority
		if strings.HasPrefix(seg, "*") {
			priority += 1000 + i
			continue
		}

		// Optional dynamic segments are less specific than required dynamics
		if strings.HasPrefix(seg, ":?") {
			priority += 150 + i
			continue
		}

		// Dynamic segments have lower priority
		if strings.HasPrefix(seg, ":") {
			priority += 100 + i
			continue
		}

		// Static segments have highest priority
		priority += i
	}

	return priority
}

// buildLayoutHierarchy builds the layout hierarchy for routes.
func (r *Router) buildLayoutHierarchy() {
	// Collect all layouts
	layouts := make(map[string]*Route)
	for _, route := range r.routes {
		if route.Type == RouteTypeLayout {
			layouts[route.Path] = route
		}
	}

	// Assign layouts to pages
	for _, route := range r.routes {
		if route.Type == RouteTypePage || route.Type == RouteTypeError {
			route.Layout = r.findLayout(route.Path, layouts)
		}
	}
}

// findLayout finds the nearest parent layout for a path.
func (r *Router) findLayout(path string, layouts map[string]*Route) *Route {
	// Walk up the path hierarchy
	dir := filepath.Dir(path)
	for dir != "/" && dir != "." {
		if layout, ok := layouts[dir]; ok {
			return layout
		}
		dir = filepath.Dir(dir)
	}

	// Check for root layout
	if layout, ok := layouts["/"]; ok {
		return layout
	}

	return nil
}

// Match matches a URL path to a route.
func (r *Router) Match(urlPath string) (*Route, map[string]string) {
	// Normalize path for lookup
	urlPath = strings.TrimSuffix(urlPath, "/")
	if urlPath == "" {
		urlPath = "/"
	}
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}

	// 1. Check static routes first (O(1))
	if route, ok := r.staticPageIndex[urlPath]; ok {
		return route, make(map[string]string)
	}

	pathSegs := splitPathSegments(urlPath)

	// 2. Check dynamic routes (O(D) where D is number of dynamic routes)
	for _, route := range r.dynamicRoutes {
		if params, ok := matchRouteSegments(route.matchSegments, pathSegs); ok {
			return route, params
		}
	}
	return nil, nil
}

// matchRoute checks if a route pattern matches a URL path.
// Kept for compatibility with existing tests/callers.
func (r *Router) matchRoute(pattern, path string) (map[string]string, bool) {
	return matchRouteSegments(compileRouteSegments(pattern), splitPathSegments(path))
}

func splitPathSegments(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "/")
}

func compileRouteSegments(pattern string) []routeSegment {
	parts := splitPathSegments(pattern)
	segments := make([]routeSegment, 0, len(parts))
	for _, part := range parts {
		switch {
		case strings.HasPrefix(part, "*?"):
			segments = append(segments, routeSegment{kind: segmentOptionalCatchAll, value: part[2:]})
		case strings.HasPrefix(part, "*"):
			segments = append(segments, routeSegment{kind: segmentCatchAll, value: part[1:]})
		case strings.HasPrefix(part, ":?"):
			segments = append(segments, routeSegment{kind: segmentOptionalParam, value: part[2:]})
		case strings.HasPrefix(part, ":"):
			segments = append(segments, routeSegment{kind: segmentParam, value: part[1:]})
		default:
			segments = append(segments, routeSegment{kind: segmentStatic, value: part})
		}
	}
	return segments
}

func matchRouteSegments(pattern []routeSegment, pathSegs []string) (map[string]string, bool) {
	cloneParams := func(src map[string]string) map[string]string {
		if src == nil {
			return make(map[string]string)
		}
		out := make(map[string]string, len(src)+1)
		for k, v := range src {
			out[k] = v
		}
		return out
	}

	var walk func(i, j int, params map[string]string) (map[string]string, bool)
	walk = func(i, j int, params map[string]string) (map[string]string, bool) {
		if i == len(pattern) {
			if j == len(pathSegs) {
				if params == nil {
					return map[string]string{}, true
				}
				return params, true
			}
			return nil, false
		}

		seg := pattern[i]
		switch seg.kind {
		case segmentStatic:
			if j >= len(pathSegs) || seg.value != pathSegs[j] {
				return nil, false
			}
			return walk(i+1, j+1, params)

		case segmentParam:
			if j >= len(pathSegs) {
				return nil, false
			}
			next := cloneParams(params)
			next[seg.value] = pathSegs[j]
			return walk(i+1, j+1, next)

		case segmentOptionalParam:
			// Try consuming first; fall back to omission when suffix segments require it.
			if j < len(pathSegs) {
				withValue := cloneParams(params)
				withValue[seg.value] = pathSegs[j]
				if out, ok := walk(i+1, j+1, withValue); ok {
					return out, true
				}
			}
			withoutValue := cloneParams(params)
			withoutValue[seg.value] = ""
			return walk(i+1, j, withoutValue)

		case segmentCatchAll:
			// Required catch-all must capture at least one segment.
			if j >= len(pathSegs) {
				return nil, false
			}
			// Greedy but backtracking, so suffix segments can still match.
			for k := len(pathSegs); k > j; k-- {
				next := cloneParams(params)
				next[seg.value] = strings.Join(pathSegs[j:k], "/")
				if out, ok := walk(i+1, k, next); ok {
					return out, true
				}
			}
			return nil, false

		case segmentOptionalCatchAll:
			// Greedy with backtracking; may also capture empty.
			for k := len(pathSegs); k >= j; k-- {
				next := cloneParams(params)
				next[seg.value] = strings.Join(pathSegs[j:k], "/")
				if out, ok := walk(i+1, k, next); ok {
					return out, true
				}
			}
			return nil, false
		}

		return nil, false
	}

	return walk(0, 0, nil)
}

// GetRoutes returns all routes.
func (r *Router) GetRoutes() []*Route {
	return r.routes
}

// GetPages returns all page routes.
func (r *Router) GetPages() []*Route {
	pages := make([]*Route, 0)
	for _, route := range r.routes {
		if route.Type == RouteTypePage {
			pages = append(pages, route)
		}
	}
	return pages
}

// GetLayouts returns all layout routes.
func (r *Router) GetLayouts() []*Route {
	layouts := make([]*Route, 0)
	for _, route := range r.routes {
		if route.Type == RouteTypeLayout {
			layouts = append(layouts, route)
		}
	}
	return layouts
}

// GetMiddlewares returns all middleware routes.
func (r *Router) GetMiddlewares() []*Route {
	mws := make([]*Route, 0)
	for _, route := range r.routes {
		if route.Type == RouteTypeMiddleware {
			mws = append(mws, route)
		}
	}
	return mws
}

// GetLoadings returns all loading routes.
func (r *Router) GetLoadings() []*Route {
	loadings := make([]*Route, 0)
	for _, route := range r.routes {
		if route.Type == RouteTypeLoading {
			loadings = append(loadings, route)
		}
	}
	return loadings
}

// RouteRegex returns a regex pattern for the route.
// The regex is compiled once and cached for performance.
func (r *Route) RouteRegex() *regexp.Regexp {
	r.regexOnce.Do(func() {
		pattern := r.Path

		// Escape special regex characters
		pattern = regexp.QuoteMeta(pattern)

		// Replace optional :?param with optional single-segment capture group
		optionalParamPattern := `(?:/([^/]+))?`
		pattern = regexp.MustCompile(`/:\?[a-zA-Z_][a-zA-Z0-9_]*`).ReplaceAllString(pattern, optionalParamPattern)

		// Replace :param with capture group
		paramPattern := `([^/]+)`
		pattern = regexp.MustCompile(`:[a-zA-Z_][a-zA-Z0-9_]*`).ReplaceAllString(pattern, paramPattern)

		// Replace optional *?param with optional catch-all capture group
		optionalCatchAllPattern := `(?:/(.*))?`
		pattern = regexp.MustCompile(`/\*\?[a-zA-Z_][a-zA-Z0-9_]*`).ReplaceAllString(pattern, optionalCatchAllPattern)

		// Replace *param with capture group for catch-all
		catchAllPattern := `(.*)`
		pattern = regexp.MustCompile(`\*[a-zA-Z_][a-zA-Z0-9_]*`).ReplaceAllString(pattern, catchAllPattern)

		// Anchor the pattern
		pattern = "^" + pattern + "$"

		r.regexCache = regexp.MustCompile(pattern)
	})
	return r.regexCache
}

// String returns a string representation of the route.
func (r *Route) String() string {
	return fmt.Sprintf("Route{Path: %s, File: %s, Type: %v, Params: %v}",
		r.Path, r.File, r.Type, r.Params)
}

// ResolveLayoutChain resolves the complete layout chain for a matched route.
// It returns all layouts from root to the nearest parent, ordered root-first.
//
// When the filesystem scan (Scan) found layout files those entries take
// priority.  If the router was initialised without a routes directory, or the
// directory does not contain .templ source files (e.g. a production binary
// deployed without source), the global registry is consulted as a fallback so
// that layouts registered via generated init() code are still applied.
func (r *Router) ResolveLayoutChain(route *Route) []*Route {
	if route == nil {
		return nil
	}

	chain := make([]*Route, 0)

	// synthRoute creates a synthetic *Route for a layout that exists only in
	// the global registry (no corresponding .templ file on disk).
	synthRoute := func(p string) *Route {
		return &Route{Path: p, Type: RouteTypeLayout}
	}

	// Walk up the path hierarchy collecting layouts
	path := route.Path

	// Check for layout at current path (if it's a page or error)
	if route.Type == RouteTypePage || route.Type == RouteTypeError {
		if layout, ok := r.layoutIndex[path]; ok {
			chain = append([]*Route{layout}, chain...)
		} else if HasLayout(path) {
			// Fallback: layout registered in global registry but not on disk.
			chain = append([]*Route{synthRoute(path)}, chain...)
		}
	}

	for {
		// Check for layout at parent path
		parent := parentDir(path)
		if parent == path {
			break
		}

		if layout, ok := r.layoutIndex[parent]; ok {
			chain = append([]*Route{layout}, chain...)
		} else if HasLayout(parent) {
			// Fallback: layout registered in global registry but not on disk.
			chain = append([]*Route{synthRoute(parent)}, chain...)
		}

		path = parent
	}

	// Check for root layout ("/") via filesystem index.
	// Note: the root_layout registered via RegisterRootLayout is handled
	// separately in render.go via GetRootLayout(); we only include a "/"
	// entry here when an explicit layout.templ lives at the routes root.
	if layout, ok := r.layoutIndex["/"]; ok {
		if len(chain) == 0 || chain[0].Path != "/" {
			chain = append([]*Route{layout}, chain...)
		}
	} else if HasLayout("/") {
		if len(chain) == 0 || chain[0].Path != "/" {
			chain = append([]*Route{synthRoute("/")}, chain...)
		}
	}

	return chain
}

// ResolveMiddlewareChain resolves the complete middleware chain for a matched route.
// It returns all middlewares from root to the nearest parent, ordered root-first.
func (r *Router) ResolveMiddlewareChain(route *Route) []*Route {
	if route == nil {
		return nil
	}

	chain := make([]*Route, 0)

	path := route.Path
	if route.Type == RouteTypePage || route.Type == RouteTypeError {
		if mw, ok := r.middlewareIndex[path]; ok {
			chain = append([]*Route{mw}, chain...)
		}
	}

	for {
		parent := parentDir(path)
		if parent == path {
			break
		}
		if mw, ok := r.middlewareIndex[parent]; ok {
			chain = append([]*Route{mw}, chain...)
		}
		path = parent
	}

	// Check for root middleware
	if mw, ok := r.middlewareIndex["/"]; ok {
		if len(chain) == 0 || chain[0].Path != "/" {
			chain = append([]*Route{mw}, chain...)
		}
	}

	return chain
}

// parentDir returns the parent directory of a path.
func parentDir(path string) string {
	if path == "/" || path == "" {
		return "/"
	}

	// Remove trailing slash
	if path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	// Find last slash
	lastSlash := -1
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			lastSlash = i
			break
		}
	}

	if lastSlash <= 0 {
		return "/"
	}

	return path[:lastSlash]
}

// MatchWithLayout matches a URL path and returns both the route and its layout chain.
func (r *Router) MatchWithLayout(urlPath string) (*Route, []*Route, map[string]string) {
	route, params := r.Match(urlPath)
	if route == nil {
		return nil, nil, nil
	}

	layouts := r.ResolveLayoutChain(route)
	return route, layouts, params
}

// GetErrorRoute returns the nearest error route for a given path.
func (r *Router) GetErrorRoute(path string) *Route {
	// Walk up the path hierarchy
	current := path
	for {
		if errRoute, ok := r.errorRouteIndex[current]; ok {
			return errRoute
		}

		parent := parentDir(current)
		if parent == current {
			break
		}
		current = parent
	}

	// Check for root error
	if errRoute, ok := r.errorRouteIndex["/"]; ok {
		return errRoute
	}

	return nil
}

func (r *Router) rebuildIndexes() {
	r.layoutIndex = make(map[string]*Route)
	r.middlewareIndex = make(map[string]*Route)
	r.errorRouteIndex = make(map[string]*Route)
	r.staticPageIndex = make(map[string]*Route)
	r.dynamicRoutes = make([]*Route, 0)

	for _, rt := range r.routes {
		switch rt.Type {
		case RouteTypePage:
			if rt.IsDynamic || rt.IsCatchAll {
				r.dynamicRoutes = append(r.dynamicRoutes, rt)
			} else {
				r.staticPageIndex[rt.Path] = rt
			}
		case RouteTypeLayout:
			r.layoutIndex[rt.Path] = rt
		case RouteTypeMiddleware:
			r.middlewareIndex[rt.Path] = rt
		case RouteTypeError:
			r.errorRouteIndex[rt.Path] = rt
		}
	}
}
