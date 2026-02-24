// Package generator provides code generation for automatic route registration.
package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

	// Generate code
	code, err := generateCode(routes, routesDir)
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, []byte(code), 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	// Generate TypeScript definitions
	if err := GenerateTypeScriptDefinitions(routesDir); err != nil {
		fmt.Printf("Warning: failed to generate TypeScript definitions: %v\n", err)
	}

	fmt.Printf("Generated %s with %d routes\n", outputPath, len(routes))
	return nil
}

// scanRoutes scans the routes directory for .templ files.
func scanRoutes(routesDir string) ([]RouteInfo, error) {
	var routes []RouteInfo

	err := filepath.Walk(routesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
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
		if strings.HasPrefix(relPath, "_") {
			return nil
		}

		route := parseRoute(relPath, routesDir)
		route.FilePath = relPath
		routes = append(routes, route)

		return nil
	})

	return routes, err
}

// parseRoute converts a file path to route information.
func parseRoute(relPath, routesDir string) RouteInfo {
	route := RouteInfo{}

	// Get directory and filename
	dir := filepath.Dir(relPath)
	filename := filepath.Base(relPath)

	// Check if it's a layout
	route.IsLayout = filename == "layout.templ" || filename == "root_layout.templ"

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
		route.PackageName = regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(rawPkgName, "")

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
		baseName := strings.TrimSuffix(filename, ".templ")
		route.ComponentFn = toPascalCase(baseName)
	}

	// Convert file path to URL path
	urlPath := filePathToURLPath(dir, filename)
	route.URLPath = urlPath

	// Extract all dynamic route params from URL path
	re := regexp.MustCompile(`:([a-zA-Z]+)`)
	matches := re.FindAllStringSubmatch(urlPath, -1)
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

// parseTemplGoFile parses a _templ.go file to extract the component function name and parameters.
func parseTemplGoFile(path string) (string, []FuncParam) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", nil
	}

	contentStr := string(content)

	// Look for the main component function - it returns templ.Component
	// Pattern: func FunctionName(...) templ.Component {
	// or: func FunctionName(...) (templ.Component, error) {

	// First try to find function with templ.Component return type
	re := regexp.MustCompile(`func\s+([A-Z][a-zA-Z0-9]*)\s*\(([^)]*)\)\s*(?:\([^)]*templ\.Component[^)]*\)|templ\.Component)`)
	matches := re.FindStringSubmatch(contentStr)
	if len(matches) < 2 {
		// Try simpler pattern for functions returning templ.Component directly
		re = regexp.MustCompile(`func\s+([A-Z][a-zA-Z0-9]*)\s*\(([^)]*)\)\s*templ\.Component`)
		matches = re.FindStringSubmatch(contentStr)
	}

	if len(matches) >= 2 {
		fnName := matches[1]
		paramsStr := matches[2]
		params := parseFunctionParams(paramsStr)
		return fnName, params
	}

	return "", nil
}

// parseFunctionParams parses function parameter string into FuncParam slice.
func parseFunctionParams(paramsStr string) []FuncParam {
	if paramsStr == "" {
		return nil
	}

	var params []FuncParam

	// Split by comma, but handle nested types like "map[string]interface{}"
	depth := 0
	current := ""
	parts := []string{}

	for _, ch := range paramsStr {
		switch ch {
		case '(', '[', '{':
			depth++
			current += string(ch)
		case ')', ']', '}':
			depth--
			current += string(ch)
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(current))
				current = ""
			} else {
				current += string(ch)
			}
		default:
			current += string(ch)
		}
	}
	if strings.TrimSpace(current) != "" {
		parts = append(parts, strings.TrimSpace(current))
	}

	// Parse each param: "name type" or "name1, name2 type"
	for _, part := range parts {
		// Split into name and type
		fields := strings.Fields(part)
		if len(fields) >= 2 {
			// Last field(s) is the type, rest are names
			typeStart := len(fields) - 1
			for i := len(fields) - 2; i >= 0; i-- {
				// Check if this looks like part of a type (e.g., "templ.Component", "[]string", "map[string]interface{}")
				if strings.HasPrefix(fields[i], "*") ||
					strings.HasPrefix(fields[i], "[]") ||
					strings.HasPrefix(fields[i], "map[") ||
					strings.Contains(fields[i], ".") ||
					fields[i] == "chan" ||
					fields[i] == "func" {
					typeStart = i
				} else {
					break
				}
			}

			paramType := strings.Join(fields[typeStart:], " ")
			for i := 0; i < typeStart; i++ {
				params = append(params, FuncParam{
					Name: fields[i],
					Type: paramType,
				})
			}
		}
	}

	return params
}

// filePathToURLPath converts a file path to a URL path.
// Route groups (name) are stripped from the URL path entirely.
func filePathToURLPath(dir, filename string) string {
	// Handle root page
	if dir == "." && filename == "page.templ" {
		return "/"
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
	if filename != "page.templ" {
		// For non-page files, use the filename
		name := strings.TrimSuffix(filename, ".templ")
		if name != "layout" {
			urlParts = append(urlParts, name)
		}
	}

	if len(urlParts) == 0 {
		return "/"
	}

	return "/" + strings.Join(urlParts, "/")
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
	parts := regexp.MustCompile(`[^a-zA-Z0-9]+`).Split(s, -1)
	var result strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		result.WriteString(strings.ToUpper(part[:1]) + strings.ToLower(part[1:]))
	}
	return result.String()
}

func generateCode(routes []RouteInfo, routesDir string) (string, error) {
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

	// Collect unique import paths for subdirectory packages
	imports := make(map[string]string) // importPath -> packageName
	imports["github.com/a-h/templ"] = "templ"
	imports["github.com/aydenstechdungeon/gospa/routing"] = "routing"

	// Get module path from go.mod
	modulePath := getModulePath(routesDir)

	for _, route := range routes {
		if route.ImportPath != "" {
			// Construct full import path
			fullImportPath := modulePath + "/" + filepath.ToSlash(filepath.Join(filepath.Base(routesDir), route.ImportPath))
			imports[fullImportPath] = route.PackageName
		}
	}

	// Imports
	sb.WriteString("import (\n")
	for importPath, alias := range imports {
		if alias != "" && alias != filepath.Base(importPath) {
			fmt.Fprintf(&sb, "\t%s %q\n", alias, importPath)
		} else {
			fmt.Fprintf(&sb, "\t%q\n", importPath)
		}
	}
	sb.WriteString(")\n\n")

	// init function
	sb.WriteString("func init() {\n")

	// Group routes by directory for better organization
	pages := []RouteInfo{}
	layouts := []RouteInfo{}

	for _, route := range routes {
		if route.IsLayout {
			layouts = append(layouts, route)
		} else {
			pages = append(pages, route)
		}
	}

	// Register pages
	if len(pages) > 0 {
		sb.WriteString("\t// Register pages\n")
		for _, route := range pages {
			fmt.Fprintf(&sb, "\trouting.RegisterPage(%q, func(props map[string]interface{}) templ.Component {\n", route.URLPath)
			fmt.Fprintf(&sb, "\t\treturn %s\n", generatePageCallWithPackage(route))
			sb.WriteString("\t})\n")
		}
	}

	// Register layouts
	if len(layouts) > 0 {
		sb.WriteString("\n\t// Register layouts\n")
		for _, route := range layouts {
			if filepath.Base(route.FilePath) == "root_layout.templ" {
				fmt.Fprintf(&sb, "\trouting.RegisterRootLayout(func(children templ.Component, props map[string]interface{}) templ.Component {\n")
			} else {
				fmt.Fprintf(&sb, "\trouting.RegisterLayout(%q, func(children templ.Component, props map[string]interface{}) templ.Component {\n", route.URLPath)
			}
			// Generate function call with proper parameters
			callArgs := generateLayoutCallArgsWithPackage(route)
			fmt.Fprintf(&sb, "\t\treturn %s\n", callArgs)
			sb.WriteString("\t})\n")
		}
	}

	sb.WriteString("}\n")

	return sb.String(), nil
}

// getModulePath reads the module path from go.mod file.
func getModulePath(dir string) string {
	// Walk up directories to find go.mod
	for {
		goModPath := filepath.Join(dir, "go.mod")
		content, err := os.ReadFile(goModPath)
		if err == nil {
			// Parse module path from go.mod
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					return strings.TrimPrefix(line, "module ")
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "example.com/project"
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
