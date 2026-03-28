package cli

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aydenstechdungeon/gospa/compiler"
	"github.com/aydenstechdungeon/gospa/plugin"
	routing_generator "github.com/aydenstechdungeon/gospa/routing/generator"
)

var (
	rePkgName = regexp.MustCompile(`[^a-zA-Z0-9]+`)
)

// Generate generates TypeScript types and routes from Go templates.
func Generate(config *GenerateConfig) {
	// Trigger BeforeGenerate hook
	if err := plugin.TriggerHook(plugin.BeforeGenerate, map[string]interface{}{"config": config}); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: BeforeGenerate hook failed: %v\n", err)
	}

	// Use defaults if config is nil
	if config == nil {
		config = &GenerateConfig{
			InputDir:      ".",
			OutputDir:     "./generated",
			ComponentType: string(compiler.ComponentTypeIsland),
		}
	}

	// Compile .gospa files first
	if err := compileSFCs(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error compiling SFCs: %v\n", err)
	}

	// Run templ generate to ensure .go files are created/updated before route generation
	fmt.Println("Running templ generate...")
	if err := regenerateTempl(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: templ generate failed: %v\n", err)
	}

	// Generate Go route registry (e.g., generated_routes.go)
	routesDir := filepath.Join(config.InputDir, "routes")
	if _, err := os.Stat(routesDir); os.IsNotExist(err) {
		// Try current directory if routesDir doesn't exist
		routesDir = config.InputDir
	}

	if err := routing_generator.Generate(routesDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating Go routes: %v\n", err)
		// Non-fatal when called from a hot-reload goroutine; just return.
		return
	}

	// Generate TypeScript types from Go state structs
	if err := generateTypesWithConfig(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating types: %v\n", err)
		return
	}

	// Generate routes and type-safe helpers via the routing generator
	// This is already called inside routing_generator.Generate

	// Trigger AfterGenerate hook
	_ = plugin.TriggerHook(plugin.AfterGenerate, map[string]interface{}{"config": config})

	fmt.Println("✓ Generated Go routes, TypeScript types, and TS routes")
}

// GenerateConfig holds configuration for code generation.
type GenerateConfig struct {
	InputDir      string
	OutputDir     string
	StateFiles    []string
	RouteFiles    []string
	ComponentType string
}

// GenerateWithConfig generates code with custom configuration.
func GenerateWithConfig(config *GenerateConfig) error {
	return generateTypesWithConfig(config)
}

func generateTypesWithConfig(config *GenerateConfig) error {
	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0750); err != nil {
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

func compileSFCs(config *GenerateConfig) error {
	sourceFiles, err := findSourceFiles(config.InputDir)
	if err != nil {
		return err
	}

	c := compiler.NewCompiler()
	for _, file := range sourceFiles {
		if !strings.HasSuffix(file, ".gospa") {
			continue
		}

		content, err := os.ReadFile(filepath.Clean(file))
		if err != nil {
			return err
		}

		// Determine relative path for unique naming
		relPath, _ := filepath.Rel(config.InputDir, file)

		// 1. Generate unique island name (e.g., routes_blog_id_page)
		uniqueName := strings.TrimSuffix(relPath, ".gospa")
		uniqueName = strings.ReplaceAll(uniqueName, string(filepath.Separator), "_")
		uniqueName = strings.ReplaceAll(uniqueName, "[", "")
		uniqueName = strings.ReplaceAll(uniqueName, "]", "")
		uniqueName = strings.ReplaceAll(uniqueName, ".", "")
		uniqueName = rePkgName.ReplaceAllString(uniqueName, "")

		dir := filepath.Dir(file)

		// 3. Determine Go component name (Standardize Page/Layout)
		baseName := filepath.Base(file)
		goName := strings.TrimSuffix(baseName, ".gospa")
		switch baseName {
		case "page.gospa":
			goName = "Page"
		case "layout.gospa":
			goName = "Layout"
		}

		selectedType := compiler.ComponentType(config.ComponentType)
		if selectedType == "" {
			selectedType = compiler.ComponentTypeIsland
		}
		opts := compiler.CompileOptions{
			Type:     selectedType,
			Name:     goName,
			PkgName:  inferPackage(selectedType),
			Hydrate:  selectedType == compiler.ComponentTypeIsland,
			IslandID: uniqueName,
		}

		templ, ts, err := c.Compile(opts, string(content))
		if err != nil {
			return fmt.Errorf("failed to compile %s: %w", file, err)
		}

		// Write .templ file in the same directory with a "generated_" prefix
		templPath := filepath.Join(dir, "generated_"+strings.TrimSuffix(baseName, ".gospa")+".templ")
		// #nosec G703
		if err := os.WriteFile(filepath.Clean(templPath), []byte(templ), 0600); err != nil {
			return err
		}

		// Write .ts file in the output directory (using unique name to avoid collisions)
		if strings.TrimSpace(ts) != "" {
			tsPath := filepath.Join(config.OutputDir, uniqueName+".ts")
			// #nosec G703
			if err := os.WriteFile(filepath.Clean(tsPath), []byte(ts), 0600); err != nil {
				return err
			}
		}
	}
	return nil
}

func inferPackage(t compiler.ComponentType) string {
	switch t {
	case compiler.ComponentTypeIsland:
		return "islands"
	case compiler.ComponentTypePage:
		return "pages"
	case compiler.ComponentTypeLayout:
		return "layouts"
	default:
		return "components"
	}
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

func findSourceFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (strings.HasSuffix(path, ".templ") || strings.HasSuffix(path, ".gospa")) {
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
		if name == "" {
			continue
		}
		camelName := strings.ToLower(name[:1]) + name[1:]
		fmt.Fprintf(&sb, "  %s: %s;\n", camelName, name)
	}
	sb.WriteString("};\n")

	// Write to file
	outputPath := filepath.Join(outputDir, "types.ts")
	return os.WriteFile(outputPath, []byte(sb.String()), 0600)
}
