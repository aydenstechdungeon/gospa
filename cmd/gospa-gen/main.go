// Package main provides a code generator for GoSPA route registration.
// It scans .templ files and generates a registration file that wires up all components.
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// RouteInfo holds information about a discovered route.
type RouteInfo struct {
	Filepath     string   // Original file path
	URLPath      string   // Mapped URL path
	ComponentFn  string   // Component function name
	IsLayout     bool     // Whether this is a layout file
	IsRootLayout bool     // Whether this is the root layout
	Params       []string // Dynamic parameters (e.g., ["id"] for [id])
}

func main() {
	watchMode := flag.Bool("watch", false, "Watch the routes directory for changes")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: gospa-gen [-watch] <routes-dir>")
		os.Exit(1)
	}

	routesDir := args[0]

	// Initial generation
	if err := generate(routesDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating registration: %v\n", err)
		os.Exit(1)
	}

	if *watchMode {
		watch(routesDir)
	}
}

func generate(routesDir string) error {
	routes, err := scanRoutes(routesDir)
	if err != nil {
		return fmt.Errorf("error scanning routes: %w", err)
	}

	if err := generateRegistrationFile(routes); err != nil {
		return fmt.Errorf("error generating registration file: %w", err)
	}

	fmt.Printf("[%s] Generated routes_registration.go successfully\n", time.Now().Format("15:04:05"))
	return nil
}

func watch(routesDir string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Add all subdirectories to watcher
	err = filepath.Walk(routesDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Watching", routesDir, "for changes...")

	var timer *time.Timer
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// Only care about templ.go files being modified/created/deleted
			if strings.HasSuffix(event.Name, "_templ.go") || strings.HasSuffix(event.Name, ".templ") {
				// Debounce generation by 100ms to avoid multiple triggers
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(100*time.Millisecond, func() {
					_ = generate(routesDir)
				})
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("watch error:", err)
		}
	}
}

// scanRoutes scans the routes directory for .templ files and extracts route info.
func scanRoutes(routesDir string) ([]RouteInfo, error) {
	var routes []RouteInfo

	err := filepath.Walk(routesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-templ files
		if info.IsDir() || !strings.HasSuffix(path, "_templ.go") {
			return nil
		}

		// Get relative path from routes dir
		relPath, err := filepath.Rel(routesDir, path)
		if err != nil {
			return err
		}

		// Parse the Go file to find component functions
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}

		// Extract component functions
		componentFuncs := extractComponentFunctions(node)

		// Determine if this is a page or layout
		isRootLayout := filepath.Base(path) == "root_layout_templ.go"
		isLayout := strings.Contains(filepath.Base(path), "layout") && !isRootLayout

		// Convert file path to URL path
		urlPath, params := filePathToURLPath(relPath)

		for _, fn := range componentFuncs {
			routes = append(routes, RouteInfo{
				Filepath:     relPath,
				URLPath:      urlPath,
				ComponentFn:  fn,
				IsLayout:     isLayout,
				IsRootLayout: isRootLayout,
				Params:       params,
			})
		}

		return nil
	})

	return routes, err
}

// extractComponentFunctions extracts templ component function names from the AST.
func extractComponentFunctions(node *ast.File) []string {
	var funcs []string

	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			name := fn.Name.Name
			// Check if it's a templ component (returns templ.Component)
			if returnsTemplComponent(fn) {
				funcs = append(funcs, name)
			}
		}
	}

	return funcs
}

// returnsTemplComponent checks if a function returns a templ.Component.
func returnsTemplComponent(fn *ast.FuncDecl) bool {
	if fn.Type.Results == nil {
		return false
	}

	for _, field := range fn.Type.Results.List {
		if ident, ok := field.Type.(*ast.Ident); ok {
			if ident.Name == "Component" {
				return true
			}
		}
		// Check for templ.Component
		if selExpr, ok := field.Type.(*ast.SelectorExpr); ok {
			if ident, ok := selExpr.X.(*ast.Ident); ok {
				if ident.Name == "templ" && selExpr.Sel.Name == "Component" {
					return true
				}
			}
		}
	}

	return false
}

// filePathToURLPath converts a file path to a URL path.
// e.g., "blog/[id]/page_templ.go" -> "/blog/:id", ["id"]
func filePathToURLPath(relPath string) (string, []string) {
	// Remove filename
	dir := filepath.Dir(relPath)

	// Handle root case
	if dir == "." {
		return "/", nil
	}

	// Split path components
	parts := strings.Split(dir, string(filepath.Separator))
	var params []string
	var urlParts []string

	for _, part := range parts {
		// Check for dynamic segment [param]
		if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
			param := strings.Trim(part, "[]")
			params = append(params, param)
			urlParts = append(urlParts, ":"+param)
		} else if strings.HasPrefix(part, "[...") {
			// Catch-all route [...rest]
			param := strings.TrimPrefix(part, "[...")
			param = strings.TrimSuffix(param, "]")
			params = append(params, param)
			urlParts = append(urlParts, "*")
		} else {
			urlParts = append(urlParts, part)
		}
	}

	return "/" + strings.Join(urlParts, "/"), params
}

// generateRegistrationFile generates the routes_registration.go file.
func generateRegistrationFile(routes []RouteInfo) error {
	var sb strings.Builder

	sb.WriteString("// Code generated by gospa-gen. DO NOT EDIT.\n")
	sb.WriteString("// Package main provides auto-generated route registration.\n")
	sb.WriteString("package main\n\n")
	sb.WriteString("import (\n")
	sb.WriteString("\t\"github.com/a-h/templ\"\n")
	sb.WriteString("\t\"github.com/aydenstechdungeon/gospa/routing\"\n")
	sb.WriteString(")\n\n")
	sb.WriteString("// init auto-registers all route components when this package is initialized.\n")
	sb.WriteString("func init() {\n")

	for _, route := range routes {
		if route.IsRootLayout {
			sb.WriteString(generateRootLayoutRegistration(route))
		} else if route.IsLayout {
			sb.WriteString(generateLayoutRegistration(route))
		} else {
			sb.WriteString(generatePageRegistration(route))
		}
	}

	sb.WriteString("}\n")

	return os.WriteFile("routes_registration.go", []byte(sb.String()), 0644)
}

// generatePageRegistration generates registration code for a page component.
func generatePageRegistration(route RouteInfo) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "\trouting.RegisterPage(%q, func(props map[string]interface{}) templ.Component {\n", route.URLPath)

	// Handle dynamic parameters
	if len(route.Params) > 0 {
		for _, param := range route.Params {
			fmt.Fprintf(&sb, "\t\t%s, _ := props[%q].(string)\n", param, param)
		}
		fmt.Fprintf(&sb, "\t\treturn %s(%s)\n", route.ComponentFn, strings.Join(route.Params, ", "))
	} else {
		fmt.Fprintf(&sb, "\t\treturn %s()\n", route.ComponentFn)
	}

	sb.WriteString("\t})\n")

	return sb.String()
}

// generateLayoutRegistration generates registration code for a layout component.
func generateLayoutRegistration(route RouteInfo) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "\trouting.RegisterLayout(%q, func(children templ.Component, props map[string]interface{}) templ.Component {\n", route.URLPath)

	// For now, assume layouts take children as first param
	// This could be enhanced to detect additional parameters
	sb.WriteString("\t\treturn " + route.ComponentFn + "(children)\n")
	sb.WriteString("\t})\n")

	return sb.String()
}

// generateRootLayoutRegistration generates registration code for the root layout.
func generateRootLayoutRegistration(route RouteInfo) string {
	var sb strings.Builder

	sb.WriteString("\trouting.RegisterRootLayout(func(children templ.Component, props map[string]interface{}) templ.Component {\n")
	sb.WriteString("\t\treturn " + route.ComponentFn + "(children, props)\n")
	sb.WriteString("\t})\n")

	return sb.String()
}
