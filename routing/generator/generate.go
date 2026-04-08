// Package generator provides code generation for automatic route registration.
package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	rePkgName      = regexp.MustCompile(`[^a-zA-Z0-9]+`)
	reDynamicParam = regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)
)

// RouteInfo holds information about a discovered route.
type RouteInfo struct {
	FilePath     string      // Relative path to .templ file
	URLPath      string      // URL path (e.g., /blog/:id)
	ComponentFn  string      // Component function name (e.g., BlogPage)
	IsLayout     bool        // True if this is a layout file
	IsDynamic    bool        // True if route has dynamic segments
	DynamicParam string      // The dynamic parameter name if any
	Params       []FuncParam // Function parameters parsed from _templ.go
	RouteParams  []string    // Dynamic route parameters extracted from URL path (e.g., ["id"] from /blog/:id)
	PackageName  string      // Package name for this route (e.g., "routes" or "blog")
	ImportPath   string      // Import path for subdirectory packages
	HasLoader    bool        // True if this route has a server-side Load function
	HasActions   bool        // True if this route has server-side form actions
	Actions      []string    // List of action names discovered in Actions map
	RuntimeTier  string      // Client runtime tier needed by this component
}

// FuncParam represents a function parameter.
type FuncParam struct {
	Name string
	Type string
}

// Generate scans the routes directory and generates registration code.
func Generate(routesDir string) error {
	// Output file path
	outputPath := filepath.Join(routesDir, "generated_routes.go")

	// Scan for .templ files
	routes, err := scanRoutes(routesDir)
	if err != nil {
		return fmt.Errorf("scanning routes: %w", err)
	}

	// Check for hooks.server.go
	hasHooks := false
	hooksPath := filepath.Join(routesDir, "hooks.server.go")
	if _, err := os.Stat(hooksPath); err == nil {
		if hasHandleFunction(hooksPath) {
			hasHooks = true
		}
	}

	// Generate code
	code, err := generateCode(routes, routesDir, hasHooks)
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, []byte(code), 0600); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	// Generate TypeScript definitions
	if err := GenerateTypeScriptDefinitions(routesDir); err != nil {
		fmt.Printf("Warning: failed to generate TypeScript definitions: %v\n", err)
	}

	// Generate type-safe route helpers
	modulePath, _ := getModuleInfo(routesDir)
	routeGen := NewRouteTypeScriptGenerator(routes, modulePath)
	generatedDir := filepath.Join(routesDir, "..", "generated")
	if err := os.MkdirAll(generatedDir, 0750); err != nil {
		fmt.Printf("Warning: failed to create generated directory: %v\n", err)
	} else {
		if err := routeGen.GenerateRoutesFile(generatedDir); err != nil {
			fmt.Printf("Warning: failed to generate route helpers: %v\n", err)
		}

		// Generate Remote Action helpers
		actionGen := NewActionTypeScriptGenerator()
		// Scan from module root to find all actions
		_, moduleRoot := getModuleInfo(routesDir)
		if err := actionGen.ScanCodebase(moduleRoot); err == nil {
			if err := actionGen.GenerateActionsFile(generatedDir); err != nil {
				fmt.Printf("Warning: failed to generate action helpers: %v\n", err)
			}
		}
	}

	fmt.Printf("Generated %s with %d routes\n", outputPath, len(routes))
	return nil
}

// scanRoutes scans the routes directory for .templ files.
func scanRoutes(routesDir string) ([]RouteInfo, error) {
	type routeKey struct {
		urlPath  string
		isLayout bool
	}
	bestRoutes := make(map[routeKey]RouteInfo)

	err := filepath.Walk(routesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Skip hidden directories (e.g., .kilo, .git, .github)
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .templ files
		if !strings.HasSuffix(path, ".templ") {
			return nil
		}

		relPath, err := filepath.Rel(routesDir, path)
		if err != nil {
			return err
		}

		// Skip generated files
		if strings.HasPrefix(filepath.Base(relPath), "_") {
			return nil
		}

		route := parseRoute(relPath, routesDir)
		route.FilePath = relPath

		// Check for corresponding loader file
		dir := filepath.Dir(path)
		cleanBase := strings.TrimPrefix(filepath.Base(path), "+")
		loaderFile := ""
		if strings.HasPrefix(cleanBase, "page") {
			loaderFile = filepath.Join(dir, "+page.server.go")
		} else if strings.HasPrefix(cleanBase, "layout") {
			loaderFile = filepath.Join(dir, "+layout.server.go")
		}
		if loaderFile != "" {
			if _, err := os.Stat(loaderFile); err == nil {
				// Verify it has a Load function
				if hasLoadFunction(loaderFile) {
					route.HasLoader = true
				}
				// Discover actions in +page.server.go
				if strings.HasSuffix(loaderFile, "+page.server.go") {
					actions := getActions(loaderFile)
					if len(actions) > 0 {
						route.HasActions = true
						route.Actions = actions
					}
				}
			}
		}

		// Extract RuntimeTier from .templ file comments
		// #nosec G122 - this is a generator tool running on local filesystem
		if content, err := os.ReadFile(filepath.Clean(path)); err == nil {
			if idx := bytes.Index(content, []byte("@gospa:tier ")); idx != -1 {
				line := string(content[idx+12:])
				if end := strings.IndexAny(line, "\n\r"); end != -1 {
					route.RuntimeTier = strings.TrimSpace(line[:end])
				}
			}
		}

		key := routeKey{urlPath: route.URLPath, isLayout: route.IsLayout}
		existing, ok := bestRoutes[key]
		if !ok {
			bestRoutes[key] = route
			return nil
		}

		// Prioritize + prefix
		if strings.HasPrefix(filepath.Base(route.FilePath), "+") && !strings.HasPrefix(filepath.Base(existing.FilePath), "+") {
			bestRoutes[key] = route
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	var routes []RouteInfo
	for _, route := range bestRoutes {
		routes = append(routes, route)
	}

	// Sort routes for deterministic output
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].URLPath != routes[j].URLPath {
			return routes[i].URLPath < routes[j].URLPath
		}
		return routes[i].IsLayout && !routes[j].IsLayout
	})

	return routes, nil
}

// parseRoute converts a file path to route information.
func parseRoute(relPath, routesDir string) RouteInfo {
	route := RouteInfo{}

	// Get directory and filename
	dir := filepath.Dir(relPath)
	filename := filepath.Base(relPath)
	cleanFilename := strings.TrimPrefix(filename, "+")

	// Check if it's a layout
	route.IsLayout = cleanFilename == "layout.templ" || cleanFilename == "root_layout.templ" ||
		cleanFilename == "generated_layout.templ" || cleanFilename == "generated_root_layout.templ"

	// Determine package name and import path based on directory
	// For subdirectory routes like blog/page.templ, the package should be "blog"
	// For root routes like page.templ, the package is "routes"
	if dir == "." {
		// Root routes directory
		route.PackageName = "routes"
		route.ImportPath = ""
	} else {
		// Subdirectory - use the first directory component as package name
		parts := strings.Split(dir, string(filepath.Separator))

		// Build the package name from the path, converting _id to id
		// and stripping route groups (name)
		pkgParts := []string{}
		for _, part := range parts {
			// Skip route groups (name) - they don't affect package names
			if strings.HasPrefix(part, "(") && strings.HasSuffix(part, ")") {
				continue
			}
			if strings.HasPrefix(part, "_") {
				// Convert _id to id for package name
				pkgParts = append(pkgParts, strings.TrimPrefix(part, "_"))
			} else {
				pkgParts = append(pkgParts, part)
			}
		}

		// Use the full path as package name (e.g., "blog" or "blogid")
		// This ensures unique package names for nested routes
		rawPkgName := strings.Join(pkgParts, "")
		route.PackageName = rePkgName.ReplaceAllString(rawPkgName, "")

		// The import path is the relative directory path
		route.ImportPath = dir
	}

	// Try to parse actual function name from _templ.go file
	templGoPath := filepath.Join(routesDir, strings.TrimSuffix(relPath, ".templ")+"_templ.go")
	if fn, params := parseTemplGoFile(templGoPath); fn != "" {
		route.ComponentFn = fn
		route.Params = params
	} else {
		// Fallback to guessing from filename
		baseName := strings.TrimSuffix(cleanFilename, ".templ")
		route.ComponentFn = toPascalCase(baseName)
	}

	// Convert file path to URL path
	urlPath := filePathToURLPath(dir, filename)
	route.URLPath = urlPath

	// Extract all dynamic route params from URL path
	matches := reDynamicParam.FindAllStringSubmatch(urlPath, -1)
	for _, match := range matches {
		if len(match) > 1 {
			route.RouteParams = append(route.RouteParams, match[1])
		}
	}
	route.IsDynamic = len(route.RouteParams) > 0
	if len(route.RouteParams) == 1 {
		route.DynamicParam = route.RouteParams[0]
	}

	return route
}

// parseTemplGoFile parses a _templ.go file to extract the component function name and parameters using go/parser.
func parseTemplGoFile(path string) (string, []FuncParam) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filepath.Clean(path), nil, parser.ParseComments)
	if err != nil {
		return "", nil
	}

	// Iterate through declarations to find the first exported function returning templ.Component
	for _, decl := range node.Decls {
		fnDecl, ok := decl.(*ast.FuncDecl)
		if !ok || fnDecl.Name == nil {
			continue
		}

		// Must be exported (starts with capital letter)
		if !fnDecl.Name.IsExported() {
			continue
		}

		// Check return types for templ.Component
		if fnDecl.Type.Results == nil {
			continue
		}

		hasTemplComponent := false
		for _, retField := range fnDecl.Type.Results.List {
			switch t := retField.Type.(type) {
			case *ast.SelectorExpr:
				if ident, ok := t.X.(*ast.Ident); ok && ident.Name == "templ" && t.Sel.Name == "Component" {
					hasTemplComponent = true
					break
				}
			}
			if hasTemplComponent {
				break
			}
		}

		if !hasTemplComponent {
			continue
		}

		fnName := fnDecl.Name.Name
		var params []FuncParam

		// Extract parameters
		if fnDecl.Type.Params != nil {
			for _, field := range fnDecl.Type.Params.List {
				var typeBuf bytes.Buffer
				if err := printer.Fprint(&typeBuf, fset, field.Type); err != nil {
					continue
				}
				typeStr := typeBuf.String()

				if len(field.Names) == 0 {
					// Anonymous parameter
					params = append(params, FuncParam{Name: "", Type: typeStr})
				} else {
					for _, name := range field.Names {
						params = append(params, FuncParam{Name: name.Name, Type: typeStr})
					}
				}
			}
		}

		return fnName, params
	}

	return "", nil
}

// filePathToURLPath converts a file path to a URL path.
// Route groups (name) are stripped from the URL path entirely.
func filePathToURLPath(dir, filename string) string {
	cleanFilename := strings.TrimPrefix(filename, "+")

	// Handle root page
	if dir == "." {
		base := strings.TrimPrefix(cleanFilename, "generated_")
		if base == "page.templ" {
			return "/"
		}
	}

	// Build path from directory
	parts := strings.Split(dir, string(filepath.Separator))
	var urlParts []string

	for _, part := range parts {
		if part == "." || part == "" {
			continue
		}

		// Skip route groups (name) - they organize routes without affecting URL
		if strings.HasPrefix(part, "(") && strings.HasSuffix(part, ")") {
			continue
		}

		// Convert _param to :param (dynamic segment)
		if strings.HasPrefix(part, "_") {
			paramName := strings.TrimPrefix(part, "_")
			urlParts = append(urlParts, ":"+paramName)
		} else {
			urlParts = append(urlParts, part)
		}
	}

	// Add the page name if it's not an index page
	base := strings.TrimSuffix(cleanFilename, ".templ")
	base = strings.TrimPrefix(base, "generated_")
	if base != "page" && base != "layout" && base != "root_layout" {
		urlParts = append(urlParts, base)
	}

	if len(urlParts) == 0 {
		return "/"
	}

	return "/" + strings.Join(urlParts, "/")
}

// hasLoadFunction checks if a .go file contains a Load function.
func hasLoadFunction(path string) bool {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return false
	}

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil {
			continue
		}

		if fn.Name.Name == "Load" && fn.Name.IsExported() {
			return true
		}
	}

	return false
}

// getActions extracts action names from an exported Actions map in a .go file.
func getActions(path string) []string {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	var actions []string
	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		for _, spec := range genDecl.Specs {
			vSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for i, name := range vSpec.Names {
				if name.Name == "Actions" && name.IsExported() {
					// Found Actions map, extract keys
					if i < len(vSpec.Values) {
						if compLit, ok := vSpec.Values[i].(*ast.CompositeLit); ok {
							for _, elt := range compLit.Elts {
								if kv, ok := elt.(*ast.KeyValueExpr); ok {
									if lit, ok := kv.Key.(*ast.BasicLit); ok && lit.Kind == token.STRING {
										actions = append(actions, strings.Trim(lit.Value, "\"`"))
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return actions
}

// hasHandleFunction checks if a .go file contains a Handle function compatible with HookFunc.
func hasHandleFunction(path string) bool {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return false
	}

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil {
			continue
		}

		if fn.Name.Name == "Handle" && fn.Name.IsExported() {
			return true
		}
	}

	return false
}

// toPascalCase converts a string to PascalCase.
func toPascalCase(s string) string {
	if s == "page" {
		return "Page"
	}
	if s == "layout" {
		return "Layout"
	}

	// Split by non-alphanumeric
	parts := rePkgName.Split(s, -1)
	var result strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		result.WriteString(strings.ToUpper(part[:1]) + strings.ToLower(part[1:]))
	}
	return result.String()
}

func generateCode(routes []RouteInfo, routesDir string, hasHooks bool) (string, error) {
	var sb strings.Builder

	// Package declaration
	sb.WriteString("// Code generated by gospa route generator. DO NOT EDIT.\n")
	sb.WriteString("// Run: go generate ./...\n\n")

	// Get package name from directory
	pkgName := filepath.Base(routesDir)
	if pkgName == "" || pkgName == "." {
		pkgName = "routes"
	}
	fmt.Fprintf(&sb, "package %s\n\n", pkgName)

	// Group routes by directory
	var pages, layouts []RouteInfo
	for _, route := range routes {
		if route.IsLayout {
			layouts = append(layouts, route)
		} else {
			pages = append(pages, route)
		}
	}

	// Check if we need context and io imports for layout wrapping
	needsContextIO := false
	for _, route := range layouts {
		hasChildren := false
		for _, p := range route.Params {
			if p.Type == "templ.Component" {
				hasChildren = true
				break
			}
		}
		if !hasChildren {
			needsContextIO = true
			break
		}
	}

	// Collect unique import paths for subdirectory packages
	imports := make(map[string]string) // importPath -> packageName
	imports["github.com/a-h/templ"] = "templ"
	imports["github.com/aydenstechdungeon/gospa/routing"] = "routing"
	if needsContextIO {
		imports["context"] = ""
		imports["io"] = ""
	}

	// Get module path from go.mod
	modulePath, moduleRoot := getModuleInfo(routesDir)

	// Calculate the relative path from the module root to the routes directory
	// This ensures imports are correctly qualified even when nested in subdirectories (like website/routes)
	absRoutesDir, _ := filepath.Abs(routesDir)
	relRoutesPath, _ := filepath.Rel(moduleRoot, absRoutesDir)
	relRoutesPath = filepath.ToSlash(relRoutesPath)

	for _, route := range routes {
		if route.ImportPath != "" {
			// Construct full import path
			fullImportPath := modulePath + "/" + filepath.ToSlash(filepath.Join(relRoutesPath, route.ImportPath))
			imports[fullImportPath] = route.PackageName
		}
	}

	// Imports - sort and group (standard library first, then third-party)
	var stdlibImports, thirdPartyImports []string
	for importPath := range imports {
		// Check if it's a standard library import (no dot in first path segment, not github.com)
		if strings.Contains(importPath, ".") || strings.HasPrefix(importPath, "github.com") {
			thirdPartyImports = append(thirdPartyImports, importPath)
		} else {
			stdlibImports = append(stdlibImports, importPath)
		}
	}

	// Sort both groups
	sort.Strings(stdlibImports)
	sort.Strings(thirdPartyImports)

	sb.WriteString("import (\n")

	// Write standard library imports first
	for _, importPath := range stdlibImports {
		alias := imports[importPath]
		if alias != "" && alias != filepath.Base(importPath) {
			fmt.Fprintf(&sb, "\t%s %q\n", alias, importPath)
		} else {
			fmt.Fprintf(&sb, "\t%q\n", importPath)
		}
	}

	// Add blank line between groups if both exist
	if len(stdlibImports) > 0 && len(thirdPartyImports) > 0 {
		sb.WriteString("\n")
	}

	// Write third-party imports
	for _, importPath := range thirdPartyImports {
		alias := imports[importPath]
		if alias != "" && alias != filepath.Base(importPath) {
			fmt.Fprintf(&sb, "\t%s %q\n", alias, importPath)
		} else {
			fmt.Fprintf(&sb, "\t%q\n", importPath)
		}
	}

	sb.WriteString(")\n\n")

	// init function
	sb.WriteString("func init() {\n")

	if hasHooks {
		sb.WriteString("\t// Register global hooks\n")
		sb.WriteString("\trouting.RegisterHook(Handle)\n\n")
	}

	// pages and layouts already grouped above

	_ = pages   // Use the pages variable
	_ = layouts // Use the layouts variable

	// Register pages
	if len(pages) > 0 {
		sb.WriteString("\t// Register pages\n")
		for _, route := range pages {
			fmt.Fprintf(&sb, "\trouting.RegisterPageWithOptions(%q, func(props map[string]interface{}) templ.Component {\n", route.URLPath)
			fmt.Fprintf(&sb, "\t\treturn %s\n", generatePageCallWithPackage(route))
			fmt.Fprintf(&sb, "\t}, routing.RouteOptions{RuntimeTier: %q})\n", route.RuntimeTier)

			if route.HasLoader {
				pkgPrefix := ""
				if route.PackageName != "routes" && route.ImportPath != "" {
					pkgPrefix = route.PackageName + "."
				}
				fmt.Fprintf(&sb, "\trouting.RegisterLoad(%q, %sLoad)\n", route.URLPath, pkgPrefix)
			}

			if route.HasActions {
				pkgPrefix := ""
				if route.PackageName != "routes" && route.ImportPath != "" {
					pkgPrefix = route.PackageName + "."
				}
				for _, actionName := range route.Actions {
					fmt.Fprintf(&sb, "\trouting.RegisterAction(%q, %q, %sActions[%q])\n", route.URLPath, actionName, pkgPrefix, actionName)
				}
			}
		}
	}

	// Register layouts
	if len(layouts) > 0 {
		sb.WriteString("\n\t// Register layouts\n")
		for _, route := range layouts {
			isRoot := filepath.Base(route.FilePath) == "root_layout.templ" || filepath.Base(route.FilePath) == "+root_layout.templ"
			if isRoot {
				fmt.Fprintf(&sb, "\trouting.RegisterRootLayout(func(children templ.Component, props map[string]interface{}) templ.Component {\n")
			} else {
				fmt.Fprintf(&sb, "\trouting.RegisterLayoutWithOptions(%q, func(children templ.Component, props map[string]interface{}) templ.Component {\n", route.URLPath)
			}
			// Generate function call with proper parameters
			callArgs := generateLayoutCallArgsWithPackage(route)

			// Wrap layout call to pass children via context if it doesn't accept children directly
			hasChildren := false
			for _, p := range route.Params {
				if p.Type == "templ.Component" {
					hasChildren = true
					break
				}
			}

			if !hasChildren {
				sb.WriteString("\t\treturn templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {\n")
				sb.WriteString("\t\t\tctx = templ.WithChildren(ctx, children)\n")
				fmt.Fprintf(&sb, "\t\t\treturn %s.Render(ctx, w)\n", callArgs)
				sb.WriteString("\t\t})\n")
			} else {
				fmt.Fprintf(&sb, "\t\treturn %s\n", callArgs)
			}
			fmt.Fprintf(&sb, "\t}, %q)\n", route.RuntimeTier)

			if route.HasLoader {
				pkgPrefix := ""
				if route.PackageName != "routes" && route.ImportPath != "" {
					pkgPrefix = route.PackageName + "."
				}
				if isRoot {
					fmt.Fprintf(&sb, "\trouting.RegisterLayoutLoad(\"\", %sLoad)\n", pkgPrefix)
				} else {
					fmt.Fprintf(&sb, "\trouting.RegisterLayoutLoad(%q, %sLoad)\n", route.URLPath, pkgPrefix)
				}
			}
		}
	}

	sb.WriteString("}\n")

	return sb.String(), nil
}

// getModuleInfo reads the module path from go.mod file and returns the module name and root path.
func getModuleInfo(dir string) (moduleName string, moduleRoot string) {
	absDir, _ := filepath.Abs(dir)
	curr := absDir

	// Walk up directories to find go.mod
	for {
		goModPath := filepath.Join(curr, "go.mod")
		content, err := os.ReadFile(filepath.Clean(goModPath))
		if err == nil {
			// Parse module path from go.mod
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					return strings.TrimSpace(strings.TrimPrefix(line, "module ")), curr
				}
			}
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	return "example.com/project", absDir
}

// generatePageCallWithPackage generates the function call with package prefix if needed.
func generatePageCallWithPackage(route RouteInfo) string {
	fnName := route.ComponentFn
	if route.PackageName != "routes" && route.ImportPath != "" {
		fnName = route.PackageName + "." + fnName
	}

	// If no params, just call the function
	if len(route.Params) == 0 {
		return fnName + "()"
	}

	// Build argument list by matching route params to function params
	var args []string
	for _, param := range route.Params {
		// Generate extraction from props
		switch param.Type {
		case "string":
			args = append(args, fmt.Sprintf(`func() string {
		if v, ok := props["%s"].(string); ok {
			return v
		}
		return ""
	}()`, param.Name))
		case "int", "int64", "int32":
			args = append(args, fmt.Sprintf(`func() %s {
		if v, ok := props["%s"].(%s); ok {
			return v
		}
		return 0
	}()`, param.Type, param.Name, param.Type))
		case "bool":
			args = append(args, fmt.Sprintf(`func() bool {
		if v, ok := props["%s"].(bool); ok {
			return v
		}
		return false
	}()`, param.Name))
		case "float64", "float32":
			args = append(args, fmt.Sprintf(`func() %s {
		if v, ok := props["%s"].(%s); ok {
			return v
		}
		return 0.0
	}()`, param.Type, param.Name, param.Type))
		case "templ.Component":
			args = append(args, fmt.Sprintf(`func() templ.Component {
		if v, ok := props["%s"].(templ.Component); ok {
			return v
		}
		return nil
	}()`, param.Name))
		default:
			// For complex types, try interface{}
			args = append(args, fmt.Sprintf(`props["%s"]`, param.Name))
		}
	}

	return fnName + "(" + strings.Join(args, ", ") + ")"
}

// generateLayoutCallArgsWithPackage generates the function call for a layout component with package prefix.
func generateLayoutCallArgsWithPackage(route RouteInfo) string {
	fnName := route.ComponentFn
	if route.PackageName != "routes" && route.ImportPath != "" {
		fnName = route.PackageName + "." + fnName
	}

	if len(route.Params) == 0 {
		return fnName + "(children)"
	}

	// Build argument list
	var args []string
	for _, param := range route.Params {
		switch param.Type {
		case "templ.Component":
			args = append(args, "children")
		case "string":
			args = append(args, fmt.Sprintf(`func() string {
		if v, ok := props["%s"].(string); ok {
			return v
		}
		return ""
	}()`, param.Name))
		case "int", "int64", "int32":
			args = append(args, fmt.Sprintf(`func() %s {
		if v, ok := props["%s"].(%s); ok {
			return v
		}
		return 0
	}()`, param.Type, param.Name, param.Type))
		case "bool":
			args = append(args, fmt.Sprintf(`func() bool {
		if v, ok := props["%s"].(bool); ok {
			return v
		}
		return false
	}()`, param.Name))
		case "float64", "float32":
			args = append(args, fmt.Sprintf(`func() %s {
		if v, ok := props["%s"].(%s); ok {
			return v
		}
		return 0.0
	}()`, param.Type, param.Name, param.Type))
		case "map[string]interface{}":
			args = append(args, fmt.Sprintf(`func() map[string]interface{} {
		if v, ok := props["%s"].(map[string]interface{}); ok {
			return v
		}
		return nil
	}()`, param.Name))
		default:
			args = append(args, fmt.Sprintf(`props["%s"]`, param.Name))
		}
	}

	return fnName + "(" + strings.Join(args, ", ") + ")"
}
