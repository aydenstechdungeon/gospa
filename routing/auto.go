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
)

// RouteType represents the type of route.
type RouteType int

const (
	RouteTypePage RouteType = iota
	RouteTypeLayout
	RouteTypeError
	RouteTypeAPI
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
}

// Router manages all routes.
type Router struct {
	routes []*Route
	fs     fs.FS
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
		routes: make([]*Route, 0),
		fs:     fileSystem,
	}
}

// Scan scans the routes directory and builds the route tree.
func (r *Router) Scan() error {
	// Walk the routes filesystem
	err := fs.WalkDir(r.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process .templ files
		if !strings.HasSuffix(path, ".templ") {
			return nil
		}

		// Parse the route. path is already relative to the fs root.
		route, err := r.parseRoute(path)
		if err != nil {
			return fmt.Errorf("failed to parse route %s: %w", path, err)
		}

		r.routes = append(r.routes, route)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan routes: %w", err)
	}

	// Sort routes by priority
	sort.Slice(r.routes, func(i, j int) bool {
		return r.routes[i].Priority < r.routes[j].Priority
	})

	// Build layout hierarchy
	r.buildLayoutHierarchy()

	return nil
}

// parseRoute parses a file path into a Route.
func (r *Router) parseRoute(relPath string) (*Route, error) {
	// Normalize path separators (fs.FS uses slash, but just in case)
	relPath = filepath.ToSlash(relPath)

	// Determine route type
	routeType := RouteTypePage
	fileName := filepath.Base(relPath)

	switch {
	case fileName == "page.templ":
		routeType = RouteTypePage
	case fileName == "layout.templ":
		routeType = RouteTypeLayout
	case fileName == "error.templ":
		routeType = RouteTypeError
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
		Path:       urlPath,
		File:       relPath,
		Type:       routeType,
		Params:     params,
		IsDynamic:  isDynamic,
		IsCatchAll: isCatchAll,
		Priority:   priority,
		Children:   make([]*Route, 0),
	}, nil
}

// filePathToURLPath converts a file path to a URL path pattern.
func (r *Router) filePathToURLPath(relPath string, routeType RouteType) string {
	// Remove file extension
	path := strings.TrimSuffix(relPath, filepath.Ext(relPath))

	// Handle different route types
	// Check for exact matches (root level) and path suffixes
	switch {
	case path == "page" || strings.HasSuffix(path, "/page"):
		// Root page.templ -> /, nested page.templ -> parent path
		if path == "page" {
			path = ""
		} else {
			path = strings.TrimSuffix(path, "page")
		}
	case path == "layout" || strings.HasSuffix(path, "/layout"):
		if path == "layout" {
			path = ""
		} else {
			path = strings.TrimSuffix(path, "layout")
		}
	case path == "error" || strings.HasSuffix(path, "/error"):
		if path == "error" {
			path = ""
		} else {
			path = strings.TrimSuffix(path, "error")
		}
	case strings.Contains(path, "+server"):
		// API route: remove +server suffix
		idx := strings.Index(path, "+server")
		path = path[:idx]
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
			result = append(result, "*"+param)
			continue
		}

		// Check for optional [[param]]
		if strings.HasPrefix(seg, "[[") && strings.HasSuffix(seg, "]]") {
			param := seg[2 : len(seg)-2]
			result = append(result, ":"+param)
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
			params = append(params, seg[1:])
			isDynamic = true
			isCatchAll = true
			continue
		}

		// Dynamic parameter
		if strings.HasPrefix(seg, ":") {
			params = append(params, seg[1:])
			isDynamic = true
			continue
		}
	}

	return params, isDynamic, isCatchAll
}

// calculatePriority calculates route priority for matching.
// Lower values = higher priority.
func calculatePriority(path string, isDynamic bool, isCatchAll bool) int {
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
	for _, route := range r.routes {
		if route.Type != RouteTypePage {
			continue
		}

		if params, ok := r.matchRoute(route.Path, urlPath); ok {
			return route, params
		}
	}
	return nil, nil
}

// matchRoute checks if a route pattern matches a URL path.
func (r *Router) matchRoute(pattern, path string) (map[string]string, bool) {
	patternSegs := strings.Split(strings.Trim(pattern, "/"), "/")
	pathSegs := strings.Split(strings.Trim(path, "/"), "/")

	params := make(map[string]string)

	// Handle catch-all
	if len(patternSegs) > 0 && strings.HasPrefix(patternSegs[len(patternSegs)-1], "*") {
		// Check prefix match
		prefixSegs := patternSegs[:len(patternSegs)-1]
		if len(pathSegs) < len(prefixSegs) {
			return nil, false
		}

		// Match prefix segments
		for i, seg := range prefixSegs {
			if seg == "" {
				continue
			}
			if !r.matchSegment(seg, pathSegs[i], params) {
				return nil, false
			}
		}

		// Capture remaining as catch-all param
		paramName := patternSegs[len(patternSegs)-1][1:]
		remaining := strings.Join(pathSegs[len(prefixSegs):], "/")
		params[paramName] = remaining

		return params, true
	}

	// Exact segment match
	if len(patternSegs) != len(pathSegs) {
		return nil, false
	}

	for i, seg := range patternSegs {
		if seg == "" && pathSegs[i] == "" {
			continue
		}
		if !r.matchSegment(seg, pathSegs[i], params) {
			return nil, false
		}
	}

	return params, true
}

// matchSegment matches a single path segment.
func (r *Router) matchSegment(pattern, value string, params map[string]string) bool {
	// Dynamic parameter
	if strings.HasPrefix(pattern, ":") {
		paramName := pattern[1:]
		params[paramName] = value
		return true
	}

	// Static match
	return pattern == value
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

// RouteRegex returns a regex pattern for the route.
func (r *Route) RouteRegex() *regexp.Regexp {
	pattern := r.Path

	// Escape special regex characters
	pattern = regexp.QuoteMeta(pattern)

	// Replace :param with capture group
	paramPattern := `([^/]+)`
	pattern = regexp.MustCompile(`:[a-zA-Z_][a-zA-Z0-9_]*`).ReplaceAllString(pattern, paramPattern)

	// Replace *param with capture group for catch-all
	catchAllPattern := `(.*)`
	pattern = regexp.MustCompile(`\*[a-zA-Z_][a-zA-Z0-9_]*`).ReplaceAllString(pattern, catchAllPattern)

	// Anchor the pattern
	pattern = "^" + pattern + "$"

	return regexp.MustCompile(pattern)
}

// String returns a string representation of the route.
func (r *Route) String() string {
	return fmt.Sprintf("Route{Path: %s, File: %s, Type: %v, Params: %v}",
		r.Path, r.File, r.Type, r.Params)
}

// ResolveLayoutChain resolves the complete layout chain for a matched route.
// It returns all layouts from root to the nearest parent, ordered root-first.
func (r *Router) ResolveLayoutChain(route *Route) []*Route {
	if route == nil {
		return nil
	}

	chain := make([]*Route, 0)

	// Collect all layouts
	layouts := make(map[string]*Route)
	for _, rt := range r.routes {
		if rt.Type == RouteTypeLayout {
			layouts[rt.Path] = rt
		}
	}

	// Walk up the path hierarchy collecting layouts
	path := route.Path

	// Check for layout at current path (if it's a page or error)
	if route.Type == RouteTypePage || route.Type == RouteTypeError {
		if layout, ok := layouts[path]; ok {
			chain = append([]*Route{layout}, chain...)
		}
	}

	for {
		// Check for layout at parent path
		parent := parentDir(path)
		if parent == path {
			break
		}

		if layout, ok := layouts[parent]; ok {
			chain = append([]*Route{layout}, chain...)
		}

		path = parent
	}

	// Check for root layout
	if layout, ok := layouts["/"]; ok {
		if len(chain) == 0 || chain[0].Path != "/" {
			chain = append([]*Route{layout}, chain...)
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
	// Collect all error routes
	errors := make(map[string]*Route)
	for _, rt := range r.routes {
		if rt.Type == RouteTypeError {
			errors[rt.Path] = rt
		}
	}

	// Walk up the path hierarchy
	current := path
	for {
		if errRoute, ok := errors[current]; ok {
			return errRoute
		}

		parent := parentDir(current)
		if parent == current {
			break
		}
		current = parent
	}

	// Check for root error
	if errRoute, ok := errors["/"]; ok {
		return errRoute
	}

	return nil
}
