package cli

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/aydenstechdungeon/gospa/plugin"
)

// Generate generates TypeScript types and routes from Go templates.
func Generate() {
	// Trigger BeforeGenerate hook
	if err := plugin.TriggerHook(plugin.BeforeGenerate, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: BeforeGenerate hook failed: %v\n", err)
	}

	// Generate TypeScript types from Go state structs
	if err := generateTypes(); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating types: %v\n", err)
		os.Exit(1)
	}

	// Generate route definitions
	if err := generateRoutes(); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating routes: %v\n", err)
		os.Exit(1)
	}

	// Trigger AfterGenerate hook
	_ = plugin.TriggerHook(plugin.AfterGenerate, nil)

	fmt.Println("✓ Generated TypeScript types and routes")
}

// GenerateConfig holds configuration for code generation.
type GenerateConfig struct {
	InputDir   string
	OutputDir  string
	StateFiles []string
	RouteFiles []string
}

// GenerateWithConfig generates code with custom configuration.
func GenerateWithConfig(config *GenerateConfig) error {
	if err := generateTypesWithConfig(config); err != nil {
		return err
	}
	return generateRoutesWithConfig(config)
}

func generateTypes() error {
	config := &GenerateConfig{
		InputDir:  ".",
		OutputDir: "./generated",
	}
	return generateTypesWithConfig(config)
}

func generateTypesWithConfig(config *GenerateConfig) error {
	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Find all Go files with state structs
	stateFiles, err := findStateFiles(config.InputDir)
	if err != nil {
		return err
	}

	// Parse state structs and generate TypeScript types
	types := make(map[string]TypeScriptType)

	for _, file := range stateFiles {
		fileTypes, err := parseStateFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", file, err)
			continue
		}

		for name, t := range fileTypes {
			types[name] = t
		}
	}

	// Generate TypeScript file
	if err := writeTypeScriptFile(config.OutputDir, types); err != nil {
		return err
	}

	return nil
}

func generateRoutes() error {
	config := &GenerateConfig{
		InputDir:  "./routes",
		OutputDir: "./generated",
	}
	return generateRoutesWithConfig(config)
}

func generateRoutesWithConfig(config *GenerateConfig) error {
	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Find all .templ files
	templFiles, err := findTemplFiles(config.InputDir)
	if err != nil {
		return err
	}

	// Generate route definitions
	routes := make([]RouteDefinition, 0)

	for _, file := range templFiles {
		route, err := parseTemplRoute(file, config.InputDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", file, err)
			continue
		}

		if route != nil {
			routes = append(routes, *route)
		}
	}

	// Generate TypeScript file
	if err := writeRoutesFile(config.OutputDir, routes); err != nil {
		return err
	}

	return nil
}

// TypeScriptType represents a TypeScript type definition.
type TypeScriptType struct {
	Name   string
	Fields []TypeScriptField
}

// TypeScriptField represents a field in a TypeScript type.
type TypeScriptField struct {
	Name     string
	Type     string
	Optional bool
}

// RouteDefinition represents a route definition.
type RouteDefinition struct {
	Path       string
	File       string
	Params     []string
	IsDynamic  bool
	IsCatchAll bool
}

func findStateFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor and generated directories
		if info.IsDir() && (info.Name() == "vendor" || info.Name() == "generated" || info.Name() == "node_modules") {
			return filepath.SkipDir
		}

		// Look for Go files that might contain state structs
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func findTemplFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".templ") {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func parseStateFile(filename string) (map[string]TypeScriptType, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	types := make(map[string]TypeScriptType)

	ast.Inspect(node, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		// Check if struct has state-related comment
		if hasStateComment(typeSpec) || hasStateSuffix(typeSpec.Name.Name) {
			tsTypeDef := TypeScriptType{
				Name:   typeSpec.Name.Name,
				Fields: make([]TypeScriptField, 0),
			}

			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 {
					continue
				}

				name := field.Names[0].Name
				fieldType := convertGoTypeToTS(field.Type)
				optional := false

				// Check if field is a pointer (optional)
				if _, isPtr := field.Type.(*ast.StarExpr); isPtr {
					optional = true
				}

				tsTypeDef.Fields = append(tsTypeDef.Fields, TypeScriptField{
					Name:     name,
					Type:     fieldType,
					Optional: optional,
				})
			}

			types[typeSpec.Name.Name] = tsTypeDef
		}

		return true
	})

	return types, nil
}

func hasStateComment(typeSpec *ast.TypeSpec) bool {
	if typeSpec.Doc == nil {
		return false
	}

	for _, comment := range typeSpec.Doc.List {
		if strings.Contains(comment.Text, "gospa:state") {
			return true
		}
	}

	return false
}

func hasStateSuffix(name string) bool {
	return strings.HasSuffix(name, "State") || strings.HasSuffix(name, "Props")
}

func convertGoTypeToTS(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		switch t.Name {
		case "string":
			return "string"
		case "int", "int8", "int16", "int32", "int64":
			return "number"
		case "uint", "uint8", "uint16", "uint32", "uint64":
			return "number"
		case "float32", "float64":
			return "number"
		case "bool":
			return "boolean"
		case "any":
			return "any"
		default:
			return t.Name
		}
	case *ast.StarExpr:
		return convertGoTypeToTS(t.X)
	case *ast.ArrayType:
		return "Array<" + convertGoTypeToTS(t.Elt) + ">"
	case *ast.MapType:
		return "Record<" + convertGoTypeToTS(t.Key) + ", " + convertGoTypeToTS(t.Value) + ">"
	case *ast.SelectorExpr:
		return t.Sel.Name
	default:
		return "any"
	}
}

func parseTemplRoute(filename, baseDir string) (*RouteDefinition, error) {
	// Get relative path
	relPath, err := filepath.Rel(baseDir, filename)
	if err != nil {
		return nil, err
	}

	// Convert file path to route path
	dir := filepath.Dir(relPath)
	file := filepath.Base(relPath)

	// Skip layout files
	if file == "layout.templ" {
		return nil, nil
	}

	// Parse route path
	path := "/"
	if dir != "." {
		path = "/" + strings.ReplaceAll(dir, string(filepath.Separator), "/")
	}

	// Handle page.templ files
	if file == "page.templ" {
		// Path is already correct
	} else {
		// Add filename to path
		path = path + "/" + strings.TrimSuffix(file, ".templ")
	}

	// Clean up path
	path = strings.ReplaceAll(path, "//", "/")

	// Extract dynamic parameters
	params := extractRouteParams(path)
	isDynamic := len(params) > 0
	isCatchAll := strings.Contains(path, "[...")

	return &RouteDefinition{
		Path:       path,
		File:       filename,
		Params:     params,
		IsDynamic:  isDynamic,
		IsCatchAll: isCatchAll,
	}, nil
}

func extractRouteParams(path string) []string {
	params := make([]string, 0)

	// Find all [param] patterns
	for i := 0; i < len(path); i++ {
		if path[i] == '[' {
			end := strings.Index(path[i:], "]")
			if end == -1 {
				continue
			}

			param := path[i+1 : i+end]
			// Handle [...rest] syntax
			param = strings.TrimPrefix(param, "...")
			// Handle [[optional]] syntax
			param = strings.Trim(param, "[]")

			params = append(params, param)
			i += end
		}
	}

	return params
}

func writeTypeScriptFile(outputDir string, types map[string]TypeScriptType) error {
	var sb strings.Builder

	sb.WriteString("// Auto-generated by GoSPA. DO NOT EDIT.\n\n")

	// Write interface declarations
	for _, t := range types {
		fmt.Fprintf(&sb, "export interface %s {\n", t.Name)

		for _, field := range t.Fields {
			optional := ""
			if field.Optional {
				optional = "?"
			}
			fmt.Fprintf(&sb, "  %s%s: %s;\n", field.Name, optional, field.Type)
		}

		sb.WriteString("}\n\n")
	}

	// Write type exports
	sb.WriteString("export type AppState = {\n")
	for name := range types {
		fmt.Fprintf(&sb, "  %s: %s;\n", strings.ToLower(name[:1])+name[1:], name)
	}
	sb.WriteString("};\n")

	// Write to file
	outputPath := filepath.Join(outputDir, "types.ts")
	return os.WriteFile(outputPath, []byte(sb.String()), 0644)
}

func writeRoutesFile(outputDir string, routes []RouteDefinition) error {
	var sb strings.Builder

	sb.WriteString("// Auto-generated by GoSPA. DO NOT EDIT.\n\n")

	// Write route interface
	sb.WriteString("export interface Route {\n")
	sb.WriteString("  path: string;\n")
	sb.WriteString("  file: string;\n")
	sb.WriteString("  params: string[];\n")
	sb.WriteString("  isDynamic: boolean;\n")
	sb.WriteString("  isCatchAll: boolean;\n")
	sb.WriteString("}\n\n")

	// Write routes array
	sb.WriteString("export const routes: Route[] = [\n")
	for _, route := range routes {
		routeJSON, _ := json.Marshal(route)
		fmt.Fprintf(&sb, "  %s,\n", string(routeJSON))
	}
	sb.WriteString("];\n")

	// Write route map
	sb.WriteString("\nexport const routeMap: Record<string, Route> = {\n")
	for _, route := range routes {
		fmt.Fprintf(&sb, "  '%s': routes.find(r => r.path === '%s')!,\n", route.Path, route.Path)
	}
	sb.WriteString("};\n")

	// Write helper functions
	sb.WriteString("\nexport function getRoute(path: string): Route | undefined {\n")
	sb.WriteString("  return routeMap[path];\n")
	sb.WriteString("}\n")

	sb.WriteString("\nexport function buildPath(route: Route, params: Record<string, string>): string {\n")
	sb.WriteString("  let path = route.path;\n")
	sb.WriteString("  for (const [key, value] of Object.entries(params)) {\n")
	sb.WriteString("    path = path.replace(`[${key}]`, value);\n")
	sb.WriteString("    path = path.replace(`[...${key}]`, value);\n")
	sb.WriteString("  }\n")
	sb.WriteString("  return path;\n")
	sb.WriteString("}\n")

	// Write to file
	outputPath := filepath.Join(outputDir, "routes.ts")
	return os.WriteFile(outputPath, []byte(sb.String()), 0644)
}
