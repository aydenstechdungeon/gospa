// Package compiler provides a compiler for GoSPA Single File Components (.gospa).
package compiler

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"

	"github.com/aydenstechdungeon/gospa/compiler/sfc"
)

// GospaCompiler handles the compilation of .gospa files.
type GospaCompiler struct{}

// ComponentType represents the type of a compiled component.
type ComponentType string

// Component type constants.
const (
	ComponentTypeIsland ComponentType = "island"
	ComponentTypePage   ComponentType = "page"
	ComponentTypeLayout ComponentType = "layout"
	ComponentTypeStatic ComponentType = "static"
	ComponentTypeServer ComponentType = "server"
)

// RuntimeTier represents the complexity of the client runtime needed.
type RuntimeTier string

// RuntimeTier constants define the different levels of client-side runtime functionality.
const (
	// RuntimeTierMicro provides basic reactivity only.
	RuntimeTierMicro RuntimeTier = "micro"
	// RuntimeTierCore provides DOM bindings and event handling.
	RuntimeTierCore RuntimeTier = "core"
	// RuntimeTierFull provides all features including dynamic modules.
	RuntimeTierFull RuntimeTier = "full"
)

// CompileOptions configures the compilation of a .gospa component.
type CompileOptions struct {
	Type        ComponentType
	IslandID    string
	Name        string
	PkgName     string
	Hydrate     bool
	HydrateMode string // immediate, visible, idle, interaction
	ServerOnly  bool
	// RuntimeTier overrides the detected runtime complexity.
	RuntimeTier RuntimeTier
	// SafeMode enables stricter validation of script content.
	// When true, the compiler checks that Go scripts parse as valid Go and
	// rejects dangerous patterns such as exec.Command, os/exec, unsafe, or
	// syscall imports. Use this when compiling .gospa from sources you do
	// not fully trust (e.g., CMS-generated SFCs).
	SafeMode bool
}

// DangerousImportNames are packages whose presence indicates a trust
// boundary violation when compiling SFCs from untrusted sources.
var DangerousImportNames = []string{
	"os/exec", "exec",
	"os",
	"unsafe",
	"syscall",
	"plugin",
	"runtime/debug",
	"runtime/pprof",
	"reflect", // reflection-based code can bypass type safety
	"C",       // cgo
}

// DangerousCallPatterns are regex patterns for unsafe function calls that
// should be rejected in SafeMode.
var DangerousCallPatterns = []*regexp.Regexp{
	regexp.MustCompile(`exec\.Command`),     // direct command execution
	regexp.MustCompile(`os\.OpenFile`),      // filesystem writes
	regexp.MustCompile(`os\.Create`),        // filesystem writes
	regexp.MustCompile(`os\.Remove`),        // filesystem deletes
	regexp.MustCompile(`os\.Rename`),        // filesystem moves
	regexp.MustCompile(`os\.Mkdir`),         // filesystem operations
	regexp.MustCompile(`os\.MkdirAll`),      // filesystem operations
	regexp.MustCompile(`os\.WriteFile`),     // filesystem writes
	regexp.MustCompile(`os\.RemoveAll`),     // recursive filesystem deletes
	regexp.MustCompile(`os\.Symlink`),       // symlink creation
	regexp.MustCompile(`os\.Setenv`),        // environment mutation
	regexp.MustCompile(`os\.Getenv`),        // environment reads (information leak)
	regexp.MustCompile(`system\s*\(`),       // system() call pattern
	regexp.MustCompile(`syscall\.Exec`),     // exec syscall
	regexp.MustCompile(`syscall\.ForkExec`), // fork/exec syscall
	regexp.MustCompile(`unix\.Exec`),        // unix exec
	regexp.MustCompile(`unix\.ForkExec`),    // unix fork/exec
}

// ValidateSafeScript validates that script content does not contain dangerous patterns.
// Returns nil if the script is safe, or an error describing the dangerous pattern found.
func ValidateSafeScript(script string) error {
	if strings.TrimSpace(script) == "" {
		return nil
	}

	// 1. Parse as Go to verify syntactic validity and perform AST inspection
	fset := token.NewFileSet()

	// Determine if script is a full file or just a body
	// Replace runes ($state -> _state) so it's valid Go for the parser
	sanitized := strings.ReplaceAll(script, "$", "_")
	src := sanitized
	isBody := !strings.Contains(sanitized, "package ")
	if isBody {
		src = "package p\n" + sanitized
	}

	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		if isBody {
			// Try wrapping in a function if it's just statements
			src = "package p\nfunc __check() {\n" + sanitized + "\n}"
			f, err = parser.ParseFile(fset, "", src, parser.ParseComments)
		}
		if err != nil {
			return fmt.Errorf("script does not parse as valid Go: %w", err)
		}
	}

	var validationErr error
	blockedTypes := make(map[string]string) // alias -> package

	ast.Inspect(f, func(n ast.Node) bool {
		if validationErr != nil {
			return false
		}

		switch x := n.(type) {
		case *ast.ImportSpec:
			path := strings.Trim(x.Path.Value, "\"")
			for _, d := range DangerousImportNames {
				if path == d || strings.HasSuffix(path, "/"+d) {
					validationErr = fmt.Errorf("safe mode: script contains import of disallowed package %q", path)
					return false
				}
			}
			if x.Name != nil {
				if x.Name.Name == "." {
					validationErr = fmt.Errorf("safe mode: dot imports are disallowed for security")
					return false
				}
				blockedTypes[x.Name.Name] = path
			} else {
				parts := strings.Split(path, "/")
				blockedTypes[parts[len(parts)-1]] = path
			}

		case *ast.SelectorExpr:
			// check for pkg.Function calls
			if ident, ok := x.X.(*ast.Ident); ok {
				if pkgPath, ok := blockedTypes[ident.Name]; ok {
					fullCall := ident.Name + "." + x.Sel.Name
					for _, re := range DangerousCallPatterns {
						if re.MatchString(fullCall) || strings.Contains(fullCall, pkgPath) {
							validationErr = fmt.Errorf("safe mode: script contains dangerous call: %s", fullCall)
							return false
						}
					}
				}
			}
		case *ast.CallExpr:
			// Handle direct dangerous calls if they were dot-imported or are built-ins
			if ident, ok := x.Fun.(*ast.Ident); ok {
				for _, re := range DangerousCallPatterns {
					if re.MatchString(ident.Name) {
						validationErr = fmt.Errorf("safe mode: script contains dangerous call: %s", ident.Name)
						return false
					}
				}
			}
		}
		return true
	})

	return validationErr
}

// NewCompiler creates a new GospaCompiler.
func NewCompiler() *GospaCompiler {
	return &GospaCompiler{}
}

// Compile compiles a .gospa component into Templ and TypeScript.
// If opts.SafeMode is true, the script content is validated against a set of
// dangerous patterns before compilation proceeds.
func (c *GospaCompiler) Compile(opts CompileOptions, input string) (templ, ts string, err error) {
	componentType := opts.Type
	if componentType == "" {
		componentType = ComponentTypeIsland
	}

	name := c.sanitizeName(opts.Name)
	islandID := c.sanitizeName(opts.IslandID)
	parsed, err := sfc.Parse(input)
	if err != nil {
		return "", "", err
	}

	if parsed.FrontMatter != nil {
		if frontType := strings.TrimSpace(parsed.FrontMatter["type"]); frontType != "" {
			componentType = ComponentType(strings.ToLower(frontType))
		}
		if hydrateRaw := strings.TrimSpace(parsed.FrontMatter["hydrate"]); hydrateRaw != "" {
			switch {
			case strings.EqualFold(hydrateRaw, "true"):
				opts.Hydrate = true
			case strings.EqualFold(hydrateRaw, "false"):
				opts.Hydrate = false
			default:
				// Treat any other value as a hydration mode (e.g., visible, idle)
				opts.Hydrate = true
				opts.HydrateMode = hydrateRaw
			}
		}
		if serverOnlyRaw := strings.TrimSpace(parsed.FrontMatter["server_only"]); serverOnlyRaw != "" {
			opts.ServerOnly = strings.EqualFold(serverOnlyRaw, "true")
		}
		if modeRaw := strings.TrimSpace(parsed.FrontMatter["hydrate_mode"]); modeRaw != "" {
			opts.HydrateMode = modeRaw
		}
		if pkgRaw := strings.TrimSpace(parsed.FrontMatter["package"]); pkgRaw != "" {
			opts.PkgName = pkgRaw
		}
	}

	// Infer type from component name if not explicitly set
	// page.gospa -> page, layout.gospa -> layout, root_layout.gospa -> root_layout
	if componentType == "" || componentType == ComponentTypeIsland {
		inferredType := inferTypeFromName(opts.Name)
		if inferredType != "" {
			componentType = inferredType
		}
	}

	if name == "" {
		name = "Component"
	}
	if islandID == "" {
		islandID = name
	}

	if opts.PkgName == "" {
		opts.PkgName = inferPackage(componentType)
	}
	if componentType == ComponentTypeIsland && !opts.ServerOnly && !opts.Hydrate {
		// default hydration for islands unless explicitly disabled
		opts.Hydrate = true
	}

	// 0. Detect Runtime Tier
	if opts.RuntimeTier == "" {
		opts.RuntimeTier = c.detectRuntimeTier(parsed)
	}

	// 1. Process Reactive DSL in Script and extract Props/State
	scriptContent := parsed.Script.Content

	// SafeMode: validate script content before processing
	if opts.SafeMode {
		// Validate Go script
		if err := ValidateSafeScript(scriptContent); err != nil {
			return "", "", fmt.Errorf("safe mode violation in Go script: %w", err)
		}
		// Validate TypeScript script if present
		if parsed.ScriptTS.Content != "" {
			if err := ValidateSafeScript(parsed.ScriptTS.Content); err != nil {
				return "", "", fmt.Errorf("safe mode violation in TS script: %w", err)
			}
		}
	}

	props, states := ExtractTypes(scriptContent)
	processedScript, _ := c.transformDSL(scriptContent)

	// 2. Generate Unique Hash for Scoping (only if there is CSS)
	hash := ""
	if parsed.Style.Content != "" {
		hash = c.generateHash(islandID)
	}

	// 3. Transform Template (AST-based)
	tp := sfc.NewTemplateParser(parsed.Template.Content, parsed.Template.ByteOffset, parsed.Template.Line, parsed.Template.Column)
	nodes, err := tp.Parse()
	if err != nil {
		return "", "", err
	}

	if opts.SafeMode {
		if err := c.validateTemplateNodes(nodes); err != nil {
			return "", "", fmt.Errorf("safe mode violation in template: %w", err)
		}
	}
	transformedTemplate := c.codegenTemplate(nodes, hash)

	// 4. Generate Templ with Scoped CSS
	templTypes := GenerateGoStruct(name, props)
	var templTypesSnippet string
	if strings.TrimSpace(templTypes) != "" {
		templTypesSnippet = templTypes
	}

	// Detect if component has client-side code (TypeScript or reactive Go)
	hasScriptTS := parsed.ScriptTS.Content != ""
	hasReactiveGo := parsed.Script.Content != "" && (strings.Contains(parsed.Script.Content, "$state") || strings.Contains(parsed.Script.Content, "$derived") || strings.Contains(parsed.Script.Content, "$effect"))
	hasClientCode := hasScriptTS || hasReactiveGo

	// 4. Generate Output
	switch componentType {
	case ComponentTypePage:
		templ = c.generatePageTempl(name, islandID, transformedTemplate, processedScript, hash, opts.PkgName, opts.HydrateMode, props, templTypesSnippet, hasClientCode, opts.RuntimeTier)
	case ComponentTypeLayout:
		templ = c.generateLayoutTempl(name, transformedTemplate, processedScript, hash, opts.PkgName, opts.HydrateMode, props, templTypesSnippet, opts.RuntimeTier)
	case ComponentTypeStatic, ComponentTypeServer:
		templ = c.generateStaticTempl(name, transformedTemplate, processedScript, hash, opts.PkgName, opts.HydrateMode, props, templTypesSnippet, opts.RuntimeTier)
	default:
		templ = c.generateIslandTempl(name, islandID, transformedTemplate, processedScript, hash, opts.PkgName, opts.HydrateMode, props, templTypesSnippet, opts.RuntimeTier)
	}

	// 5. Generate TypeScript for client-side interactivity
	// Pages with reactive Go code or explicit ts blocks get client-side hydration
	// Islands always get client-side hydration if hydrate is enabled
	// Generate TS only if there's actual client-side code needed:
	// - Islands with hydration enabled, OR
	// - Pages with reactive Go code (state/derived/effect), OR
	// - Any component with explicit <script lang="ts">
	shouldGenerateTS := false
	if !opts.ServerOnly {
		isHydratedIsland := componentType == ComponentTypeIsland && opts.Hydrate
		shouldGenerateTS = isHydratedIsland || hasClientCode
	}

	if shouldGenerateTS {
		tsScript := parsed.Script.Content
		tsFromGo := true
		if parsed.ScriptTS.Content != "" {
			tsScript = parsed.ScriptTS.Content
			tsFromGo = false
		}
		tsTypes := GenerateTSInterface(name, props, states)
		ts = c.generateTS(islandID, tsScript, tsFromGo, parsed.Style.Content, hash)
		ts = tsTypes + "\n" + ts
		ts += c.generateScopedCSS(parsed.Style.Content, hash)
	}

	return templ, ts, nil
}

func (c *GospaCompiler) codegenTemplate(nodes []sfc.Node, hash string) string {
	var sb strings.Builder
	for _, node := range nodes {
		switch n := node.(type) {
		case *sfc.ElementNode:
			sb.WriteString("<")
			sb.WriteString(n.TagName)
			// Add scoping hash (only if hash is provided)
			if hash != "" {
				hasHash := false
				for i, attr := range n.Attributes {
					if attr.Name == "class" {
						n.Attributes[i].Value += " " + hash
						hasHash = true
						break
					}
				}
				if !hasHash {
					n.Attributes = append(n.Attributes, sfc.Attribute{Name: "class", Value: hash})
				}
			}

			for _, attr := range n.Attributes {
				sb.WriteString(" ")
				sb.WriteString(attr.Name)
				if attr.Value != "" || attr.IsExpression {
					sb.WriteString("=")
					if attr.IsExpression {
						sb.WriteString("{")
					} else {
						sb.WriteString("\"")
					}
					sb.WriteString(attr.Value)
					if attr.IsExpression {
						sb.WriteString("}")
					} else {
						sb.WriteString("\"")
					}
				}
			}
			if n.SelfClosing {
				sb.WriteString(" />")
			} else {
				sb.WriteString(">")
				sb.WriteString(c.codegenTemplate(n.Children, hash))
				sb.WriteString("</")
				sb.WriteString(n.TagName)
				sb.WriteString(">")
			}
		case *sfc.TextNode:
			sb.WriteString(n.Content)
		case *sfc.ExpressionNode:
			sb.WriteString("{ ")
			sb.WriteString(n.Content)
			sb.WriteString(" }")
		case *sfc.IfNode:
			sb.WriteString("if ")
			sb.WriteString(n.Condition)
			sb.WriteString(" {\n")
			sb.WriteString(c.codegenTemplate(n.Then, hash))
			for _, elseif := range n.ElseIfs {
				sb.WriteString("\n} else if ")
				sb.WriteString(elseif.Condition)
				sb.WriteString(" {\n")
				sb.WriteString(c.codegenTemplate(elseif.Then, hash))
			}
			if len(n.Else) > 0 {
				sb.WriteString("\n} else {\n")
				sb.WriteString(c.codegenTemplate(n.Else, hash))
			}
			sb.WriteString("\n}")
		case *sfc.EachNode:
			sb.WriteString("for _, ")
			sb.WriteString(n.As)
			sb.WriteString(" := range ")
			sb.WriteString(n.Iteratee)
			sb.WriteString(" {\n")
			sb.WriteString(c.codegenTemplate(n.Children, hash))
			sb.WriteString("\n}")
		case *sfc.SnippetNode:
			// Snippets are handled separately in generateTempl, but we can emit them as templ functions
			// Actually, let's just emit them as definitions if they were inside the template
		case *sfc.ComponentNode:
			sb.WriteString("@")
			sb.WriteString(n.Name)
			sb.WriteString("(")
			for i, attr := range n.Attributes {
				if i > 0 {
					sb.WriteString(", ")
				}
				// For positional arguments (from @Component(...) syntax), don't output the name
				if strings.HasPrefix(attr.Name, "_arg") {
					if attr.IsExpression {
						// Wrap in backticks for raw string literals in templ
						sb.WriteString("`")
						sb.WriteString(attr.Value)
						sb.WriteString("`")
					} else {
						sb.WriteString("\"")
						sb.WriteString(attr.Value)
						sb.WriteString("\"")
					}
				} else {
					sb.WriteString(attr.Name)
					sb.WriteString(": ")
					if attr.IsExpression {
						sb.WriteString(attr.Value)
					} else {
						sb.WriteString("\"")
						sb.WriteString(attr.Value)
						sb.WriteString("\"")
					}
				}
			}
			sb.WriteString(")")
			if len(n.Children) > 0 {
				sb.WriteString(" {\n")
				sb.WriteString(c.codegenTemplate(n.Children, hash))
				sb.WriteString("\n}")
			}
		case *sfc.CommentNode:
			sb.WriteString("<!-- ")
			sb.WriteString(n.Content)
			sb.WriteString(" -->")
		}
	}
	return sb.String()
}

// CompileLegacy preserves the old island-only API.
func (c *GospaCompiler) CompileLegacy(goName, islandID, input, pkgName string) (templ, ts string, err error) {
	return c.Compile(CompileOptions{
		Type:     ComponentTypeIsland,
		IslandID: islandID,
		Name:     goName,
		PkgName:  pkgName,
		Hydrate:  true,
	}, input)
}

func (c *GospaCompiler) generateHash(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		normalized = "component"
	}
	sum := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("gospa-%x", sum[:2])
}

var (
	snippetRegex       = regexp.MustCompile(`\{#snippet\s+([a-zA-Z0-9_]+)\((.*?)\)\}`)
	endSnippetRegex    = regexp.MustCompile(`\{/snippet\}`)
	reactiveLabelRegex = regexp.MustCompile(`\$:\s*([a-zA-Z0-9_]+)\s*=\s*([^;\n]+)`)
	nameSafeRegex      = regexp.MustCompile(`[^a-zA-Z0-9]`)
)

func (c *GospaCompiler) sanitizeName(name string) string {
	safe := nameSafeRegex.ReplaceAllString(name, "")
	if safe == "" {
		return "Component"
	}
	return safe
}

func (c *GospaCompiler) transformDSL(script string) (string, []Prop) {
	props, _ := ExtractTypes(script)

	// Implicit reactive statements $: val = expr -> var val = expr
	script = reactiveLabelRegex.ReplaceAllString(script, "var $1 = $2")

	// For Go (SSR), var name = $state(val) -> var name = val, $derived(expr) -> expr
	// We use the regexes from types.go (same package)
	script = StateRegex.ReplaceAllString(script, "var $1 = $2")
	script = DerivedRegex.ReplaceAllString(script, "$1")
	script = EffectRegex.ReplaceAllString(script, "")

	return script, props
}

func inferPackage(t ComponentType) string {
	switch t {
	case ComponentTypeIsland:
		return "islands"
	case ComponentTypePage:
		return "pages"
	case ComponentTypeLayout:
		return "layouts"
	default:
		return "components"
	}
}

// inferTypeFromName determines the component type based on the filename.
// Special filenames like page, layout, root_layout get SSR types.
// All others default to islands (with hydration).
func inferTypeFromName(name string) ComponentType {
	lower := strings.ToLower(name)

	// Exact matches (case-insensitive)
	switch lower {
	case "page":
		return ComponentTypePage
	case "layout":
		return ComponentTypeLayout
	case "root_layout":
		return ComponentTypeLayout
	case "error":
		return ComponentTypeStatic
	case "loading":
		return ComponentTypeStatic
	}

	// Prefix matches
	if strings.HasPrefix(lower, "page") {
		return ComponentTypePage
	}
	if strings.HasPrefix(lower, "layout") {
		return ComponentTypeLayout
	}
	if strings.HasPrefix(lower, "error") {
		return ComponentTypeStatic
	}
	if strings.HasPrefix(lower, "loading") || strings.HasPrefix(lower, "_loading") {
		return ComponentTypeStatic
	}

	// Default to island (will get hydration)
	return ""
}

// extractStructDefs extracts type struct definitions from script content.
// Struct definitions must be at the package level in templ, not inside {{ }} blocks.
func extractStructDefs(script string) (structs string, remaining string) {
	var sb strings.Builder
	buf := []byte(script)

	for {
		typeIdx := bytes.Index(buf, []byte("type "))
		if typeIdx == -1 {
			break
		}

		braceStart := bytes.Index(buf[typeIdx:], []byte("{"))
		if braceStart == -1 {
			break
		}
		braceStart += typeIdx

		depth := 0
		endIdx := -1
		for i := braceStart; i < len(buf); i++ {
			if buf[i] == '{' {
				depth++
			} else if buf[i] == '}' {
				depth--
				if depth == 0 {
					endIdx = i + 1
					break
				}
			}
		}
		if endIdx == -1 {
			break
		}

		sb.Write(buf[typeIdx:endIdx])
		sb.WriteString("\n")
		buf = append(buf[:typeIdx], buf[endIdx:]...)
	}

	return sb.String(), string(buf)
}

func (c *GospaCompiler) generateIslandTempl(name, islandID, template, script, hash, pkgName, hydrateMode string, props []Prop, templTypesSnippet string, tier RuntimeTier) string {
	var result string
	header := fmt.Sprintf("// Code generated by GoSPA; DO NOT EDIT.\n// @gospa:tier %s\n\n", tier)
	if pkgName == "" {
		pkgName = "islands"
	}

	cleanTemplate, extraImports, extraSnippets, scriptInjection := c.processScript(script, template, templTypesSnippet)

	if !strings.Contains(extraImports, "github.com/aydenstechdungeon/gospa/state") {
		switch {
		case extraImports == "":
			extraImports = "import \"github.com/aydenstechdungeon/gospa/state\""
		case strings.HasPrefix(extraImports, "import ("):
			extraImports = strings.Replace(extraImports, "import (", "import (\n\t\"github.com/aydenstechdungeon/gospa/state\"", 1)
		default:
			extraImports = "import \"github.com/aydenstechdungeon/gospa/state\"\n" + extraImports
		}
	}

	propNames := []string{}
	for _, p := range props {
		propNames = append(propNames, p.Name+" "+p.Type)
	}
	propArgs := typeArgs(strings.Join(propNames, ", "))
	signature := name + "(" + propArgs + ")"

	registration := fmt.Sprintf(`
	if r := state.FromContext(ctx); r != nil {
		pMap := map[string]interface{}{
			%s
		}
		r.Register("%s", pMap, nil)
	}`, generatePropMap(props), islandID)

	modeAttr := ""
	if hydrateMode != "" {
		modeAttr = fmt.Sprintf(" data-gospa-mode=\"%s\"", hydrateMode)
	}

	result = header + fmt.Sprintf(`package %s

%s

%s

templ %s {
%s
%s
	<div data-gospa-island="%s" class="%s"%s>
		%s
	</div>
}
`, pkgName, extraImports, extraSnippets, signature, scriptInjection, registration, islandID, hash, modeAttr, cleanTemplate)

	return result
}

func generatePropMap(props []Prop) string {
	var sb strings.Builder
	for _, p := range props {
		fmt.Fprintf(&sb, "\"%s\": %s,\n", p.Name, p.Name)
	}
	return sb.String()
}

func (c *GospaCompiler) generatePageTempl(name, islandID, template, script, hash, pkgName, hydrateMode string, props []Prop, templTypesSnippet string, hasClientCode bool, tier RuntimeTier) string {
	var result string
	header := fmt.Sprintf("// Code generated by GoSPA; DO NOT EDIT.\n// @gospa:tier %s\n\n", tier)
	if pkgName == "" {
		pkgName = "pages"
	}

	cleanTemplate, extraImports, extraSnippets, scriptInjection := c.processScript(script, template, templTypesSnippet)

	propNames := []string{}
	for _, p := range props {
		propNames = append(propNames, p.Name+" "+p.Type)
	}
	propArgs := typeArgs(strings.Join(propNames, ", "))
	signature := name + "(" + propArgs + ")"

	needsRegistration := hasClientCode
	if needsRegistration {
		if !strings.Contains(extraImports, "github.com/aydenstechdungeon/gospa/state") {
			switch {
			case extraImports == "":
				extraImports = "import \"github.com/aydenstechdungeon/gospa/state\""
			case strings.HasPrefix(extraImports, "import ("):
				extraImports = strings.Replace(extraImports, "import (", "import (\n\t\"github.com/aydenstechdungeon/gospa/state\"", 1)
			default:
				extraImports = "import \"github.com/aydenstechdungeon/gospa/state\"\n" + extraImports
			}
		}
		registration := fmt.Sprintf(`{{ if r := state.FromContext(ctx); r != nil { pMap := map[string]interface{}{}; r.Register("%s", pMap, nil) } }}`, islandID)
		modeAttr := ""
		if hydrateMode != "" {
			modeAttr = fmt.Sprintf(" data-gospa-mode=\"%s\"", hydrateMode)
		}

		result = header + fmt.Sprintf(`package %s

%s

%s

templ %s {
%s
%s
	<div data-gospa-island="%s" class="%s"%s>
		%s
	</div>
}
`, pkgName, extraImports, extraSnippets, signature, scriptInjection, registration, islandID, hash, modeAttr, cleanTemplate)
	} else {
		// No registration needed for pure SSR pages
		result = header + fmt.Sprintf(`package %s

%s

%s

templ %s {
%s
	<div data-gospa-page="%s" class="%s">
		%s
	</div>
}
`, pkgName, extraImports, extraSnippets, signature, scriptInjection, islandID, hash, cleanTemplate)
	}

	return result
}

func (c *GospaCompiler) generateLayoutTempl(name, template, script, hash, pkgName, hydrateMode string, props []Prop, templTypesSnippet string, tier RuntimeTier) string {
	templ := c.generateIslandTempl(name, "", template, script, hash, pkgName, hydrateMode, props, templTypesSnippet, tier)
	templ = strings.Replace(templ, "<div data-gospa-island=\"\" class=\""+hash+"\">", "<div class=\""+hash+"\">", 1)
	templ = strings.ReplaceAll(templ, "@children", "{ children }")
	signatureNeedle := "templ " + name + "("
	signatureReplace := "templ " + name + "(children templ.Component"
	if len(props) > 0 {
		signatureReplace += ", "
	}
	return strings.Replace(templ, signatureNeedle, signatureReplace, 1)
}

func (c *GospaCompiler) generateStaticTempl(name, template, script, hash, pkgName, hydrateMode string, props []Prop, templTypesSnippet string, tier RuntimeTier) string {
	templ := c.generateIslandTempl(name, "", template, script, hash, pkgName, hydrateMode, props, templTypesSnippet, tier)
	templ = strings.Replace(templ, "\n\t<div data-gospa-island=\"\" class=\""+hash+"\">\n\t\t", "\n\t\t", 1)
	return strings.Replace(templ, "\n\t</div>\n}\n", "\n}\n", 1)
}

func (c *GospaCompiler) processScript(script, template, templTypesSnippet string) (cleanTemplate string, extraImports string, extraSnippets string, scriptInjection string) {
	snippetDefs := []string{}
	cleanTemplate = template

	typeArgs := func(args string) string {
		if strings.TrimSpace(args) == "" {
			return ""
		}
		parts := strings.Split(args, ",")
		typedParts := []string{}
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed == "" {
				continue
			}
			if !strings.Contains(trimmed, " ") {
				trimmed += " any"
			}
			typedParts = append(typedParts, trimmed)
		}
		return strings.Join(typedParts, ", ")
	}

	for {
		startLoc := snippetRegex.FindStringSubmatchIndex(cleanTemplate)
		if startLoc == nil {
			break
		}
		snippetName := cleanTemplate[startLoc[2]:startLoc[3]]
		snippetArgs := cleanTemplate[startLoc[4]:startLoc[5]]
		remaining := cleanTemplate[startLoc[1]:]
		endLoc := endSnippetRegex.FindStringIndex(remaining)
		if endLoc == nil {
			break
		}
		content := remaining[:endLoc[0]]
		snippetDefs = append(snippetDefs, fmt.Sprintf("templ %s(%s) {\n\t%s\n}", snippetName, typeArgs(snippetArgs), strings.TrimSpace(content)))
		cleanTemplate = cleanTemplate[:startLoc[0]] + cleanTemplate[startLoc[1]+endLoc[1]:]
	}

	importRegex := regexp.MustCompile(`(?m)^import\s+(?:"[^"]+"|\(.*\))`)
	imports := importRegex.FindAllString(script, -1)
	extraImports = strings.Join(imports, "\n")
	cleanScript := importRegex.ReplaceAllString(script, "")
	structDefs, cleanScript := extractStructDefs(cleanScript)

	if strings.Contains(cleanScript, "fmt.") && !strings.Contains(extraImports, "\"fmt\"") {
		switch {
		case extraImports == "":
			extraImports = "import \"fmt\""
		case strings.HasPrefix(extraImports, "import ("):
			extraImports = strings.Replace(extraImports, "import (", "import (\n\t\"fmt\"", 1)
		default:
			extraImports = "import \"fmt\"\n" + extraImports
		}
	}
	if strings.Contains(extraImports, "\"fmt\"") {
		cleanScript = "var _ = fmt.Sprint\n" + cleanScript
	}

	// Unused variable hack: identify variables and add var _ = name to avoid Go compilation errors.
	vRegex := regexp.MustCompile(`(?m)^(?:var\s+)?([a-zA-Z0-9_]+)\s*(?::=|=)`)
	matches := vRegex.FindAllStringSubmatch(cleanScript, -1)
	seen := make(map[string]bool)
	for _, m := range matches {
		if m[1] != "_" && !seen[m[1]] {
			cleanScript += fmt.Sprintf("\nvar _ = %s", m[1])
			seen[m[1]] = true
		}
	}

	// Transform local functions to function variables since named functions
	// cannot be defined inside the body of another function (templ's Render).
	funcBlockRegex := regexp.MustCompile(`(?s)func\s+([a-zA-Z0-9_]+)\((.*?)\)\s*\{(.*?)\}`)
	cleanScript = funcBlockRegex.ReplaceAllStringFunc(cleanScript, func(match string) string {
		parts := funcBlockRegex.FindStringSubmatch(match)
		fnName := parts[1]
		fnArgs := parts[2]
		fnBody := parts[3]
		flatBody := strings.ReplaceAll(strings.TrimSpace(fnBody), "\n", "; ")
		return fmt.Sprintf("%s := func(%s) { %s }", fnName, fnArgs, flatBody)
	})

	extraSnippets = strings.TrimSpace(structDefs)
	if strings.TrimSpace(templTypesSnippet) != "" {
		if extraSnippets != "" {
			extraSnippets += "\n\n"
		}
		extraSnippets += strings.TrimSpace(templTypesSnippet)
	}
	if len(snippetDefs) > 0 {
		if extraSnippets != "" {
			extraSnippets += "\n\n"
		}
		extraSnippets += strings.Join(snippetDefs, "\n\n")
	}

	scriptInjection = ""
	if strings.TrimSpace(cleanScript) != "" {
		// Wrap the entire script in a single block for stability.
		// Templ handles multiline Go code inside {{ }} blocks.
		scriptInjection = "{{\n" + strings.TrimSpace(cleanScript) + "\n}}\n"
	}
	return
}

func typeArgs(args string) string {
	if strings.TrimSpace(args) == "" {
		return ""
	}
	parts := strings.Split(args, ",")
	typedParts := []string{}
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		if !strings.Contains(trimmed, " ") {
			trimmed += " any"
		}
		typedParts = append(typedParts, trimmed)
	}
	return strings.Join(typedParts, ", ")
}

func (c *GospaCompiler) validateTemplateNodes(nodes []sfc.Node) error {
	for _, node := range nodes {
		var err error
		switch n := node.(type) {
		case *sfc.ElementNode:
			for _, attr := range n.Attributes {
				if attr.IsExpression {
					if err = ValidateSafeScript(attr.Value); err != nil {
						return fmt.Errorf("attribute %q: %w", attr.Name, err)
					}
				}
			}
			err = c.validateTemplateNodes(n.Children)
		case *sfc.ExpressionNode:
			err = ValidateSafeScript(n.Content)
		case *sfc.IfNode:
			if err = ValidateSafeScript(n.Condition); err != nil {
				return err
			}
			if err = c.validateTemplateNodes(n.Then); err != nil {
				return err
			}
			for _, elseif := range n.ElseIfs {
				if err = ValidateSafeScript(elseif.Condition); err != nil {
					return err
				}
				if err = c.validateTemplateNodes(elseif.Then); err != nil {
					return err
				}
			}
			err = c.validateTemplateNodes(n.Else)
		case *sfc.EachNode:
			if err = ValidateSafeScript(n.Iteratee); err != nil {
				return err
			}
			err = c.validateTemplateNodes(n.Children)
		case *sfc.ComponentNode:
			for _, attr := range n.Attributes {
				if attr.IsExpression {
					if err = ValidateSafeScript(attr.Value); err != nil {
						return fmt.Errorf("component attribute %q: %w", attr.Name, err)
					}
				}
			}
			err = c.validateTemplateNodes(n.Children)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *GospaCompiler) generateTS(islandID, script string, fromGo bool, _ string, hash string) string {
	tsScript := script
	funcNames := []string{}

	if fromGo {
		// 1. Basic conversion from Go reactive runes to TS
		// We use state.$state etc to avoid collision and use our runtime
		tsScript = PropsRegex.ReplaceAllString(tsScript, "let { $1 } = props")
		tsScript = StateRegex.ReplaceAllString(tsScript, "let $1 = state.$$state($2)")
		tsScript = DerivedRegex.ReplaceAllString(tsScript, "state.$$derived(() => $1)")
		// Effects remain as state.$effect
		tsScript = EffectRegex.ReplaceAllString(tsScript, "state.$$effect(() => {$1})")

		// Convert Go func to JS function
		funcRegex := regexp.MustCompile(`(?m)^\s*func\s+([a-zA-Z0-9_]+)\((.*?)\)\s*\{`)
		tsScript = funcRegex.ReplaceAllString(tsScript, "function $1($2) {")

		// Reactive labels $: name = expr -> const name = state.$derived(() => expr)
		tsScript = ReactiveLabelRegex.ReplaceAllString(tsScript, "const $1 = state.$derived(() => $2)")

		// Clean up Go-isms
		goForRangeRegex := regexp.MustCompile(`(?m)for\s+_,\s*([a-zA-Z0-9_]+)\s*:=\s*range\s*(.*?)\s*\{`)
		tsScript = goForRangeRegex.ReplaceAllString(tsScript, "for (const $1 of $2) {")

		tsScript = strings.ReplaceAll(tsScript, " := ", " = ")
		tsScript = strings.ReplaceAll(tsScript, "fmt.Sprint", "String")
		tsScript = strings.ReplaceAll(tsScript, "fmt.Sprintf", "String") // fallback
		tsScript = strings.ReplaceAll(tsScript, "fmt.Printf", "console.log")

		// Extract function names for event binding (must do after conversion)
		namedFuncRegex := regexp.MustCompile(`function\s+([a-zA-Z0-9_]+)`)
		matches := namedFuncRegex.FindAllStringSubmatch(tsScript, -1)
		for _, m := range matches {
			funcNames = append(funcNames, m[1])
		}
	}

	header := "/**\n * Code generated by GoSPA; DO NOT EDIT.\n */\n\n"
	funcsObject := "{ " + strings.Join(funcNames, ", ") + " }"
	if len(funcNames) == 0 {
		funcsObject = "{}"
	}

	return header + fmt.Sprintf(`function __gospa_setup_%s(element: Element, { props, state }: { props: Record<string, any>; state: any }) {
%s

    // Event delegation registration
    const __GOSPA_HANDLERS__ = %s;
    (window as any)["__GOSPA_ISLAND_" + "%s" + "__"] = { handlers: __GOSPA_HANDLERS__ };
    
    // Scoped hydration selector
    const scope = (selector: string) => element.querySelector(selector + '.' + '%s');
}

export function mount(element: Element, props: Record<string, any>, state: any) {
  __gospa_setup_%s(element, { props, state });
}

export function hydrate(element: Element, props: Record<string, any>, state: any) {
  __gospa_setup_%s(element, { props, state });
}

export default { mount, hydrate };
`, islandID, tsScript, funcsObject, islandID, hash, islandID, islandID)
}

func (c *GospaCompiler) generateScopedCSS(style, hash string) string {
	if style == "" {
		return ""
	}
	// Scoping: use context-aware CSS scoping that avoids mutating selectors
	// inside strings, URLs, and comments.
	scopedStyle := scopeCSS(style, hash)

	encodedStyle, _ := json.Marshal(scopedStyle)
	return fmt.Sprintf("\n\n/* Scoped CSS */\nif (!document.querySelector(`style[data-gospa-style=\"%s\"]`)) {\n\tconst style = document.createElement('style');\n\tstyle.setAttribute('data-gospa-style', '%s');\n\tstyle.textContent = %s;\n\tdocument.head.appendChild(style);\n}\n", hash, hash, string(encodedStyle))
}

// cssContext tracks the state of the CSS parser as it walks through the input.
type cssContext int

const (
	ctxNormal     cssContext = iota
	ctxInString              // inside a string ("...") or ('...')
	ctxInURL                 // inside a url(...) value
	ctxInComment             // inside /* ... */
	ctxInFunction            // inside a CSS function (e.g., content: "...") — no scoping
)

// scopeCSS applies the scoping hash to CSS class and element selectors while
// avoiding false positives inside string literals, url() values, and comments.
// This is a best-effort implementation that handles common CSS patterns.
func scopeCSS(style, hash string) string {
	var sb strings.Builder
	i := 0
	n := len(style)
	ctx := ctxNormal
	quoteChar := byte(0) // track which quote char opened a string

	// Track the last non-whitespace character before a rule start to detect selectors.
	lastNonWS := byte(0)
	// Track brace depth to detect selectors vs values.
	braceDepth := 0

	for i < n {
		ch := style[i]

		// Handle comment context
		if ctx == ctxInComment {
			sb.WriteByte(ch)
			if ch == '*' && i+1 < n && style[i+1] == '/' {
				sb.WriteByte(style[i+1])
				i += 2
				ctx = ctxNormal
				continue
			}
			i++
			continue
		}

		// Handle string context
		if ctx == ctxInString {
			sb.WriteByte(ch)
			if ch == '\\' && i+1 < n {
				sb.WriteByte(style[i+1])
				i += 2
				continue
			}
			if ch == quoteChar {
				ctx = ctxNormal
			}
			i++
			continue
		}

		// Normal context
		switch {
		case ch == '/' && i+1 < n && style[i+1] == '*':
			sb.WriteByte(ch)
			sb.WriteByte(style[i+1])
			ctx = ctxInComment
			i += 2
			continue

		case ch == '"' || ch == '\'':
			sb.WriteByte(ch)
			quoteChar = ch
			ctx = ctxInString
			i++
			continue

		case ch == '{':
			braceDepth++
			sb.WriteByte(ch)
			i++
			continue

		case ch == '}':
			braceDepth--
			sb.WriteByte(ch)
			i++
			continue

		default:
			// Detect class selectors (.) and element selectors at rule-start position
			// A rule starts when braceDepth == 0 and we're about to see a selector
			// (preceded by newline/whitespace/','/';' or at the start).
			if braceDepth == 0 && (ch == '.' || isElementSelectorStart(style, i, n)) {
				// Only scope if we're at the start of a rule
				isRuleStart := (i == 0 || lastNonWS == 0 || lastNonWS == ',' || lastNonWS == '{' || lastNonWS == '}' || lastNonWS == ';')
				if isRuleStart {
					if ch == '.' {
						// Class selector: scope it
						sb.WriteByte('.')
						i++
						// Read the class name
						for i < n && (isIdentChar(style[i])) {
							sb.WriteByte(style[i])
							i++
						}
						// Insert scoping class
						sb.WriteByte('.')
						sb.WriteString(hash)
						continue
					} else if isLetter(style[i]) {
						// Element selector: read the tag name
						start := i
						for i < n && isIdentChar(style[i]) {
							i++
						}
						tagName := style[start:i]
						if isCSSElementTag(tagName) {
							sb.WriteString(tagName)
							sb.WriteByte('.')
							sb.WriteString(hash)
						} else {
							sb.WriteString(tagName)
						}
						continue
					}
				}
			}

			sb.WriteByte(ch)
			if !isWhitespace(ch) {
				lastNonWS = ch
			}
			i++
		}
	}

	return sb.String()
}

func isIdentChar(ch byte) bool {
	return ch == '-' || ch == '_' || ch == '.' || ch == ':' ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9')
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

// isElementSelectorStart checks if a character at position i could be the
// start of an element selector (e.g., h1, div, button, etc.)
func isElementSelectorStart(style string, i, _ int) bool {
	ch := style[i]
	if !isLetter(ch) {
		return false
	}
	// Check context: element selectors appear at rule start
	return true
}

// isCSSElementTag returns true for common CSS element selectors.
func isCSSElementTag(tag string) bool {
	elementTags := map[string]bool{
		"a": true, "abbr": true, "address": true, "area": true, "article": true,
		"aside": true, "audio": true, "b": true, "base": true, "bdi": true,
		"bdo": true, "blockquote": true, "body": true, "br": true, "button": true,
		"canvas": true, "caption": true, "cite": true, "code": true, "col": true,
		"colgroup": true, "data": true, "datalist": true, "dd": true, "del": true,
		"details": true, "dfn": true, "dialog": true, "div": true, "dl": true,
		"dt": true, "em": true, "embed": true, "fieldset": true, "figcaption": true,
		"figure": true, "footer": true, "form": true, "h1": true, "h2": true,
		"h3": true, "h4": true, "h5": true, "h6": true, "head": true, "header": true,
		"hgroup": true, "hr": true, "html": true, "i": true, "iframe": true,
		"img": true, "input": true, "ins": true, "kbd": true, "label": true,
		"legend": true, "li": true, "link": true, "main": true, "map": true,
		"mark": true, "menu": true, "meta": true, "meter": true, "nav": true,
		"noscript": true, "object": true, "ol": true, "optgroup": true,
		"option": true, "output": true, "p": true, "param": true, "picture": true,
		"pre": true, "progress": true, "q": true, "rp": true, "rt": true,
		"ruby": true, "s": true, "samp": true, "script": true, "section": true,
		"select": true, "small": true, "source": true, "span": true, "strong": true,
		"style": true, "sub": true, "summary": true, "sup": true, "table": true,
		"tbody": true, "td": true, "template": true, "textarea": true, "tfoot": true,
		"th": true, "thead": true, "time": true, "title": true, "tr": true,
		"track": true, "u": true, "ul": true, "var": true, "video": true, "wbr": true,
	}
	return elementTags[tag]
}
func (c *GospaCompiler) detectRuntimeTier(parsed *sfc.SFC) RuntimeTier {
	template := parsed.Template.Content
	script := parsed.Script.Content
	ts := parsed.ScriptTS.Content

	// Full Tier: Enhanced navigation, form actions, or explicit "full" requirement
	if strings.Contains(template, "gospa-enhance") ||
		strings.Contains(template, "gospa-link") ||
		strings.Contains(template, "@form") ||
		strings.Contains(ts, "navigation") {
		return RuntimeTierFull
	}

	// Core Tier: DOM bindings, events, or lifecycle hooks
	if strings.Contains(template, "@on:") ||
		strings.Contains(template, "@bind") ||
		strings.Contains(template, "@class") ||
		strings.Contains(template, "@style") ||
		strings.Contains(template, "@attr") ||
		strings.Contains(template, "@use:") ||
		strings.Contains(ts, "onMount") ||
		strings.Contains(ts, "onDestroy") {
		return RuntimeTierCore
	}

	// Micro Tier: Basic reactivity or just static text interpolation
	if strings.TrimSpace(script) != "" || strings.TrimSpace(ts) != "" {
		return RuntimeTierMicro
	}

	return RuntimeTierMicro
}
