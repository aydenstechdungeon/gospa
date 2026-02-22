package routing

import (
	"fmt"
	"sync"
)

// LayoutChain represents a chain of nested layouts.
type LayoutChain struct {
	// Layouts are ordered from root to leaf
	Layouts []*Route
	// Page is the final page route
	Page *Route
	// Error is the error route for this chain
	Error *Route
}

// LayoutResolver resolves layout hierarchies for routes.
type LayoutResolver struct {
	mu      sync.RWMutex
	layouts map[string]*Route
	errors  map[string]*Route
}

// NewLayoutResolver creates a new layout resolver.
func NewLayoutResolver() *LayoutResolver {
	return &LayoutResolver{
		layouts: make(map[string]*Route),
		errors:  make(map[string]*Route),
	}
}

// RegisterLayout registers a layout route.
func (lr *LayoutResolver) RegisterLayout(route *Route) {
	lr.mu.Lock()
	defer lr.mu.Unlock()
	lr.layouts[route.Path] = route
}

// RegisterError registers an error route.
func (lr *LayoutResolver) RegisterError(route *Route) {
	lr.mu.Lock()
	defer lr.mu.Unlock()
	lr.errors[route.Path] = route
}

// ResolveChain resolves the layout chain for a page route.
func (lr *LayoutResolver) ResolveChain(page *Route) *LayoutChain {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	chain := &LayoutChain{
		Page:    page,
		Layouts: make([]*Route, 0),
	}

	// Collect all parent layouts
	path := page.Path
	for {
		// Check for layout at current path
		if layout, ok := lr.layouts[path]; ok {
			chain.Layouts = append([]*Route{layout}, chain.Layouts...)
		}

		// Move to parent path
		parent := parentPath(path)
		if parent == path {
			break
		}
		path = parent
	}

	// Check for root layout
	if layout, ok := lr.layouts["/"]; ok {
		// Check if root layout is already included
		if len(chain.Layouts) == 0 || chain.Layouts[0].Path != "/" {
			chain.Layouts = append([]*Route{layout}, chain.Layouts...)
		}
	}

	// Find error route
	chain.Error = lr.findError(page.Path)

	return chain
}

// findError finds the nearest error route for a page.
func (lr *LayoutResolver) findError(pagePath string) *Route {
	path := pagePath
	for {
		if errRoute, ok := lr.errors[path]; ok {
			return errRoute
		}

		parent := parentPath(path)
		if parent == path {
			break
		}
		path = parent
	}

	// Check for root error
	if errRoute, ok := lr.errors["/"]; ok {
		return errRoute
	}

	return nil
}

// parentPath returns the parent path of a URL path.
func parentPath(path string) string {
	if path == "/" || path == "" {
		return "/"
	}

	// Remove trailing slash
	path = trimSuffix(path, "/")

	// Find last slash
	lastSlash := lastIndexOf(path, "/")
	if lastSlash <= 0 {
		return "/"
	}

	return path[:lastSlash]
}

// trimSuffix removes a suffix from a string.
func trimSuffix(s, suffix string) string {
	if len(suffix) > 0 && len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}

// lastIndexOf returns the last index of a substring.
func lastIndexOf(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// LayoutTree represents a tree of layouts.
type LayoutTree struct {
	Root *LayoutNode
}

// LayoutNode represents a node in the layout tree.
type LayoutNode struct {
	Route    *Route
	Children []*LayoutNode
	Parent   *LayoutNode
}

// NewLayoutTree creates a new layout tree from routes.
func NewLayoutTree(routes []*Route) *LayoutTree {
	tree := &LayoutTree{
		Root: &LayoutNode{
			Route:    &Route{Path: "/", Type: RouteTypeLayout},
			Children: make([]*LayoutNode, 0),
		},
	}

	// Collect layouts
	layouts := make(map[string]*Route)
	for _, route := range routes {
		if route.Type == RouteTypeLayout {
			layouts[route.Path] = route
		}
	}

	// Build tree
	tree.buildTree(tree.Root, layouts)

	return tree
}

// buildTree recursively builds the layout tree.
func (t *LayoutTree) buildTree(node *LayoutNode, layouts map[string]*Route) {
	// Find all direct children
	for path, route := range layouts {
		if isDirectChild(node.Route.Path, path) {
			child := &LayoutNode{
				Route:    route,
				Children: make([]*LayoutNode, 0),
				Parent:   node,
			}
			node.Children = append(node.Children, child)

			// Remove from map to avoid reprocessing
			delete(layouts, path)

			// Recursively build subtree
			t.buildTree(child, layouts)
		}
	}
}

// isDirectChild checks if childPath is a direct child of parentPath.
func isDirectChild(parentPath, childPath string) bool {
	if parentPath == "/" {
		// Root's direct children have no other slashes
		trimmed := trimPrefix(childPath, "/")
		return trimmed != "" && !contains(trimmed, "/")
	}

	// Check if child starts with parent
	if !hasPrefix(childPath, parentPath+"/") {
		return false
	}

	// Check if there are no more path segments
	remaining := trimPrefix(childPath, parentPath+"/")
	return !contains(remaining, "/")
}

// trimPrefix removes a prefix from a string.
func trimPrefix(s, prefix string) string {
	if len(prefix) > 0 && len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// hasPrefix checks if a string has a prefix.
func hasPrefix(s, prefix string) bool {
	if len(prefix) > len(s) {
		return false
	}
	return s[:len(prefix)] == prefix
}

// FindLayoutChain finds the layout chain for a path.
func (t *LayoutTree) FindLayoutChain(path string) []*Route {
	chain := make([]*Route, 0)
	t.findChain(t.Root, path, &chain)
	return chain
}

// findChain recursively finds the layout chain.
func (t *LayoutTree) findChain(node *LayoutNode, path string, chain *[]*Route) {
	// Add current node to chain
	if node.Route != nil && node.Route.Path != "/" {
		*chain = append(*chain, node.Route)
	}

	// Find matching child
	for _, child := range node.Children {
		if isParentPath(child.Route.Path, path) || child.Route.Path == path {
			t.findChain(child, path, chain)
			break
		}
	}
}

// isParentPath checks if parentPath is a parent of childPath.
func isParentPath(parentPath, childPath string) bool {
	if parentPath == "/" {
		return true
	}
	return hasPrefix(childPath, parentPath+"/")
}

// LayoutData represents data passed through layouts.
type LayoutData struct {
	// Data is the layout data
	Data map[string]interface{}
	// Children is nested content
	Children interface{}
}

// NewLayoutData creates new layout data.
func NewLayoutData() *LayoutData {
	return &LayoutData{
		Data: make(map[string]interface{}),
	}
}

// Set sets a value in layout data.
func (ld *LayoutData) Set(key string, value interface{}) {
	ld.Data[key] = value
}

// Get gets a value from layout data.
func (ld *LayoutData) Get(key string) (interface{}, bool) {
	val, ok := ld.Data[key]
	return val, ok
}

// Merge merges another layout data into this one.
func (ld *LayoutData) Merge(other *LayoutData) {
	if other == nil {
		return
	}
	for k, v := range other.Data {
		ld.Data[k] = v
	}
}

// LayoutContext provides context for layout rendering.
type LayoutContext struct {
	// Path is the current URL path
	Path string
	// Params are the route parameters
	Params map[string]string
	// Data is the layout data chain
	Data []*LayoutData
	// Depth is the current layout depth
	Depth int
}

// NewLayoutContext creates a new layout context.
func NewLayoutContext(path string, params map[string]string) *LayoutContext {
	return &LayoutContext{
		Path:   path,
		Params: params,
		Data:   make([]*LayoutData, 0),
	}
}

// PushData pushes new layout data onto the chain.
func (lc *LayoutContext) PushData(data *LayoutData) {
	lc.Data = append(lc.Data, data)
	lc.Depth++
}

// PopData pops layout data from the chain.
func (lc *LayoutContext) PopData() *LayoutData {
	if len(lc.Data) == 0 {
		return nil
	}
	data := lc.Data[len(lc.Data)-1]
	lc.Data = lc.Data[:len(lc.Data)-1]
	lc.Depth--
	return data
}

// CurrentData returns the current layout data.
func (lc *LayoutContext) CurrentData() *LayoutData {
	if len(lc.Data) == 0 {
		return nil
	}
	return lc.Data[len(lc.Data)-1]
}

// String returns a string representation of the layout chain.
func (lc *LayoutChain) String() string {
	paths := make([]string, 0, len(lc.Layouts)+1)
	for _, l := range lc.Layouts {
		paths = append(paths, l.Path)
	}
	if lc.Page != nil {
		paths = append(paths, lc.Page.Path)
	}
	return fmt.Sprintf("LayoutChain{Layouts: %v, Page: %v}",
		paths, lc.Page != nil)
}
