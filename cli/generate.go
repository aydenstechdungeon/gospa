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
	if config.DevMode {
		fmt.Println("Running in development mode (HMR enabled)")
	}
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

	// Generate TypeScript wrappers for remote actions
	if err := generateRemoteActions(config); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to generate remote action wrappers: %v\n", err)
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
	DevMode       bool
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
	var islandNames []string
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

		// 3. Determine Go component name and type (Standardize Page/Layout)
		baseName := filepath.Base(file)
		goName := strings.TrimSuffix(baseName, ".gospa")
		selectedType := compiler.ComponentType(config.ComponentType)

		// Determine type from filename if not explicitly set
		cleanBaseName := strings.TrimPrefix(baseName, "+")
		switch cleanBaseName {
		case "page.gospa":
			goName = "Page"
			selectedType = compiler.ComponentTypePage
		case "layout.gospa":
			goName = "Layout"
			selectedType = compiler.ComponentTypeLayout
		case "root_layout.gospa":
			goName = "RootLayout"
			selectedType = compiler.ComponentTypeLayout
		case "error.gospa":
			goName = "Error"
			selectedType = compiler.ComponentTypeStatic
		case "_loading.gospa", "loading.gospa":
			goName = "Loading"
			selectedType = compiler.ComponentTypeStatic
		}

		// Fall back to config type if not determined from filename
		// (The compiler will also infer type from name, but we pass it here for clarity)
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
			islandNames = append(islandNames, uniqueName)
		}
	}

	// Generate islands entry file that registers all setup functions
	if err := generateIslandsEntry(config.OutputDir, islandNames, config.DevMode); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to generate islands entry: %v\n", err)
	}

	return nil
}

// generateIslandsEntry creates an entry TypeScript file that imports all
// island modules and registers their setup functions with the runtime.
func generateIslandsEntry(outputDir string, names []string, devMode bool) error {
	var sb strings.Builder
	sb.WriteString("/**\n * Auto-generated islands entry for GoSPA\n * Do not edit manually\n */\n\n")
	sb.WriteString("function registerLazySetup(name: string, loader: () => Promise<any>) {\n")
	sb.WriteString("  (window as any).__GOSPA_SETUPS__ = (window as any).__GOSPA_SETUPS__ || {};\n")
	sb.WriteString("  (window as any).__GOSPA_SETUPS__[name] = async (el: Element, props: Record<string, any>, state: any) => {\n")
	sb.WriteString("    const mod = await loader();\n")
	sb.WriteString("    const hydrateFn = mod.hydrate || mod.default?.hydrate || mod.mount || mod.default?.mount;\n")
	sb.WriteString("    if (hydrateFn) {\n")
	sb.WriteString("      return hydrateFn(el, props, state);\n")
	sb.WriteString("    }\n")
	sb.WriteString("  };\n")
	sb.WriteString("}\n\n")

	for _, name := range names {
		importPath := fmt.Sprintf("./%s.ts", name)
		if devMode {
			// In dev mode, append a timestamp evaluated in JS to force module reload on re-imports during HMR
			fmt.Fprintf(&sb, "registerLazySetup('%s', () => import('%s?v=' + Date.now()));\n", name, importPath)
		} else {
			fmt.Fprintf(&sb, "registerLazySetup('%s', () => import('%s'));\n", name, importPath)
		}
	}

	entryPath := filepath.Join(outputDir, "islands.ts")
	return os.WriteFile(entryPath, []byte(sb.String()), 0600)
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

		if info.IsDir() {
			// Skip hidden directories (e.g., .kilo, .git, .github)
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			// Skip vendor and generated directories
			if info.Name() == "vendor" || info.Name() == "generated" || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		// Look for Go files that might contain state structs
		if strings.HasSuffix(path, ".go") {
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

		if info.IsDir() {
			// Skip hidden directories (e.g., .kilo, .git, .github)
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(path, ".templ") || strings.HasSuffix(path, ".gospa") {
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

// RemoteActionInfo holds information about a registered remote action
type RemoteActionInfo struct {
	Name         string
	InputType    string
	OutputType   string
	FunctionType string
}

func generateRemoteActions(config *GenerateConfig) error {
	// Find all Go files that might contain RegisterRemoteAction calls
	routesDir := filepath.Join(config.InputDir, "routes")
	if _, err := os.Stat(routesDir); os.IsNotExist(err) {
		routesDir = config.InputDir
	}

	var actionFiles []string
	err := filepath.Walk(routesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") || info.Name() == "vendor" || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			actionFiles = append(actionFiles, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Parse each file for RegisterRemoteAction calls
	var actions []RemoteActionInfo
	for _, file := range actionFiles {
		fileActions, err := parseRemoteActions(file)
		if err != nil {
			continue // Skip files that can't be parsed
		}
		actions = append(actions, fileActions...)
	}

	if len(actions) == 0 {
		return nil // No remote actions found
	}

	// Generate TypeScript file with typed remote action wrappers
	var sb strings.Builder
	sb.WriteString("// Auto-generated by GoSPA. DO NOT EDIT.\n")
	sb.WriteString("// Type-safe remote action wrappers.\n\n")

	sb.WriteString("import { remoteAction } from '@gospa/runtime';\n\n")

	for _, action := range actions {
		// Generate typed wrapper
		inputType := action.InputType
		outputType := action.OutputType
		if inputType == "" {
			inputType = "Record<string, unknown>"
		}
		if outputType == "" {
			outputType = "unknown"
		}

		fmt.Fprintf(&sb, "export const %s = remoteAction<%s, %s>('%s');\n",
			action.Name, inputType, outputType, action.Name)
	}

	// Write to file
	outputPath := filepath.Join(config.OutputDir, "remote-actions.ts")
	return os.WriteFile(outputPath, []byte(sb.String()), 0600)
}

func parseRemoteActions(filename string) ([]RemoteActionInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var actions []RemoteActionInfo

	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Check if this is a RegisterRemoteAction call
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if sel.Sel.Name != "RegisterRemoteAction" {
			return true
		}

		// Get action name from first argument
		if len(call.Args) < 2 {
			return true
		}

		nameLit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || nameLit.Kind != token.STRING {
			return true
		}
		actionName := strings.Trim(nameLit.Value, "\"")

		// Try to extract type info from the second argument (function type)
		var inputType, outputType string
		if fnType, ok := call.Args[1].(*ast.FuncType); ok {
			inputType, outputType = extractRemoteActionTypes(fnType)
		}

		actions = append(actions, RemoteActionInfo{
			Name:       actionName,
			InputType:  inputType,
			OutputType: outputType,
		})

		return true
	})

	return actions, nil
}

func extractRemoteActionTypes(fnType *ast.FuncType) (inputType, outputType string) {
	// Extract input type from function parameters
	if len(fnType.Params.List) > 1 {
		// Skip context and RemoteContext, get the actual input
		if len(fnType.Params.List) >= 2 {
			inputType = extractTypeName(fnType.Params.List[1].Type)
		}
	}

	// Extract output type from function results
	if fnType.Results != nil && len(fnType.Results.List) > 0 {
		outputType = extractTypeName(fnType.Results.List[0].Type)
		// If it's a tuple (result, error), get the first element
		if len(fnType.Results.List) > 1 {
			outputType = extractTypeName(fnType.Results.List[0].Type)
		}
	}

	return inputType, outputType
}

func extractTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return extractTypeName(t.X) + " | null"
	case *ast.ArrayType:
		return "Array<" + extractTypeName(t.Elt) + ">"
	case *ast.MapType:
		return "Record<" + extractTypeName(t.Key) + ", " + extractTypeName(t.Value) + ">"
	case *ast.SelectorExpr:
		return t.Sel.Name
	default:
		return "unknown"
	}
}
