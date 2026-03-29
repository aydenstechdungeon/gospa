// Package compiler provides a compiler for GoSPA Single File Components (.gospa).
package compiler

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
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

// CompileOptions configures the compilation of a .gospa component.
type CompileOptions struct {
	Type       ComponentType
	IslandID   string
	Name       string
	PkgName    string
	Hydrate    bool
	ServerOnly bool
}

// NewCompiler creates a new GospaCompiler.
func NewCompiler() *GospaCompiler {
	return &GospaCompiler{}
}

// Compile compiles a .gospa component into Templ and TypeScript.
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
			opts.Hydrate = strings.EqualFold(hydrateRaw, "true")
		}
		if serverOnlyRaw := strings.TrimSpace(parsed.FrontMatter["server_only"]); serverOnlyRaw != "" {
			opts.ServerOnly = strings.EqualFold(serverOnlyRaw, "true")
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

	// 1. Process Reactive DSL in Script and extract Props/State
	scriptContent := parsed.Script.Content
	props, states := ExtractTypes(scriptContent)
	processedScript, _ := c.transformDSL(scriptContent)

	// 2. Generate Unique Hash for Scoping
	hash := c.generateHash(islandID)

	// 3. Transform Template (AST-based)
	tp := sfc.NewTemplateParser(parsed.Template.Content, parsed.Template.ByteOffset, parsed.Template.Line, parsed.Template.Column)
	nodes, err := tp.Parse()
	if err != nil {
		return "", "", err
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

	switch componentType {
	case ComponentTypePage:
		templ = c.generatePageTempl(name, islandID, transformedTemplate, processedScript, hash, opts.PkgName, props, templTypesSnippet, hasClientCode)
	case ComponentTypeLayout:
		templ = c.generateLayoutTempl(name, transformedTemplate, processedScript, hash, opts.PkgName, props, templTypesSnippet)
	case ComponentTypeStatic, ComponentTypeServer:
		templ = c.generateStaticTempl(name, transformedTemplate, processedScript, hash, opts.PkgName, props, templTypesSnippet)
	default:
		templ = c.generateIslandTempl(name, islandID, transformedTemplate, processedScript, hash, opts.PkgName, props, templTypesSnippet)
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
			// Add scoping hash
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
	remaining = script

	for {
		typeIdx := strings.Index(remaining, "type ")
		if typeIdx == -1 {
			break
		}

		braceStart := strings.Index(remaining[typeIdx:], "{")
		if braceStart == -1 {
			break
		}
		braceStart += typeIdx

		depth := 0
		endIdx := -1
		for i := braceStart; i < len(remaining); i++ {
			if remaining[i] == '{' {
				depth++
			} else if remaining[i] == '}' {
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

		sb.WriteString(remaining[typeIdx:endIdx])
		sb.WriteString("\n")
		remaining = remaining[:typeIdx] + remaining[endIdx:]
	}

	return sb.String(), remaining
}

func (c *GospaCompiler) generateIslandTempl(name, islandID, template, script, hash, pkgName string, props []Prop, templTypesSnippet string) string {
	header := "// Code generated by GoSPA; DO NOT EDIT.\n\n"
	if pkgName == "" {
		pkgName = "islands"
	}

	// 1. Extract Snippets from template
	snippetDefs := []string{}
	cleanTemplate := template

	// Helper to ensure args are typed (default to any)
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

	// 2. Extract imports and transform script
	importRegex := regexp.MustCompile(`(?m)^import\s+(?:"[^"]+"|\(.*\))`)

	imports := importRegex.FindAllString(script, -1)
	extraImports := strings.Join(imports, "\n")

	cleanScript := importRegex.ReplaceAllString(script, "")

	// Extract type struct definitions to package level (they can't be inside {{ }} blocks)
	structDefs, cleanScript := extractStructDefs(cleanScript)

	// Ensure fmt import if used
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

	// Add dummy usage for all variables in cleanScript to avoid "declared and not used"
	vRegex := regexp.MustCompile(`(?m)^(?:var\s+)?([a-zA-Z0-9_]+)\s*(?::=|=)`)
	matches := vRegex.FindAllStringSubmatch(cleanScript, -1)
	for _, m := range matches {
		if m[1] != "_" {
			cleanScript += fmt.Sprintf("\nvar _ = %s", m[1])
		}
	}

	// Transform named functions to single-line anonymous functions to keep them in local scope
	funcBlockRegex := regexp.MustCompile(`(?s)func\s+([a-zA-Z0-9_]+)\((.*?)\)\s*\{(.*?)\}`)
	cleanScript = funcBlockRegex.ReplaceAllStringFunc(cleanScript, func(match string) string {
		parts := funcBlockRegex.FindStringSubmatch(match)
		fnName := parts[1]
		fnArgs := parts[2]
		fnBody := parts[3]
		// Flatten body and replace newlines with ;
		flatBody := strings.ReplaceAll(strings.TrimSpace(fnBody), "\n", "; ")
		return fmt.Sprintf("%s := func(%s) { %s }", fnName, fnArgs, flatBody)
	})

	extraSnippets := strings.TrimSpace(structDefs)
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

	// Inject script initializations if non-empty
	// All Go code must be inside {{ }} delimiters in templ
	scriptInjection := ""
	if strings.TrimSpace(cleanScript) != "" {
		lines := strings.Split(strings.TrimSpace(cleanScript), "\n")
		i := 0
		for i < len(lines) {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.Contains(trimmed, "func(") || strings.Contains(trimmed, "func ") {
				i++
				continue
			}
			if strings.Contains(trimmed, "var _ = ") {
				i++
				continue
			}
			// Detect multi-line blocks (if/for/switch with opening brace)
			openBraces := strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
			if openBraces > 0 {
				// Collect the entire block into a single {{ }} wrapper
				block := trimmed
				for openBraces > 0 && i+1 < len(lines) {
					i++
					nextLine := strings.TrimSpace(lines[i])
					block += " " + nextLine
					openBraces += strings.Count(nextLine, "{") - strings.Count(nextLine, "}")
				}
				scriptInjection += "{{ " + block + " }}\n"
			} else {
				scriptInjection += "{{ " + trimmed + " }}\n"
			}
			i++
		}
	}

	// Add state registry import if needed
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

	// Signature generation
	propNames := []string{}
	for _, p := range props {
		propNames = append(propNames, p.Name+" "+p.Type)
	}
	propArgs := typeArgs(strings.Join(propNames, ", "))
	signature := name + "(" + propArgs + ")"

	// Registration logic
	registration := fmt.Sprintf(`
	if r := state.FromContext(ctx); r != nil {
		pMap := map[string]interface{}{
			%s
		}
		r.Register("%s", pMap, nil)
	}`, generatePropMap(props), islandID)

	return header + fmt.Sprintf(`package %s

%s

%s

templ %s {
%s
%s
	<div data-gospa-island="%s" class="%s">
		%s
	</div>
}
`, pkgName, extraImports, extraSnippets, signature, scriptInjection, registration, islandID, hash, cleanTemplate)
}

func generatePropMap(props []Prop) string {
	var sb strings.Builder
	for _, p := range props {
		fmt.Fprintf(&sb, "\"%s\": %s,\n", p.Name, p.Name)
	}
	return sb.String()
}

func (c *GospaCompiler) generatePageTempl(name, islandID, template, script, hash, pkgName string, props []Prop, templTypesSnippet string, hasClientCode bool) string {
	header := "// Code generated by GoSPA; DO NOT EDIT.\n\n"
	if pkgName == "" {
		pkgName = "pages"
	}

	// 1. Extract Snippets from template
	snippetDefs := []string{}
	cleanTemplate := template

	// Helper to ensure args are typed (default to any)
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

	// 2. Extract imports and transform script
	importRegex := regexp.MustCompile(`(?m)^import\s+(?:"[^"]+"|\(.*\))`)

	imports := importRegex.FindAllString(script, -1)
	extraImports := strings.Join(imports, "\n")

	cleanScript := importRegex.ReplaceAllString(script, "")

	// Extract type struct definitions to package level (they can't be inside {{ }} blocks)
	structDefs, cleanScript := extractStructDefs(cleanScript)

	// Ensure fmt import if used
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

	// Add dummy usage for all variables in cleanScript to avoid "declared and not used"
	vRegex := regexp.MustCompile(`(?m)^(?:var\s+)?([a-zA-Z0-9_]+)\s*(?::=|=)`)
	matches := vRegex.FindAllStringSubmatch(cleanScript, -1)
	for _, m := range matches {
		if m[1] != "_" {
			cleanScript += fmt.Sprintf("\nvar _ = %s", m[1])
		}
	}

	// Transform named functions to single-line anonymous functions to keep them in local scope
	funcBlockRegex := regexp.MustCompile(`(?s)func\s+([a-zA-Z0-9_]+)\((.*?)\)\s*\{(.*?)\}`)
	cleanScript = funcBlockRegex.ReplaceAllStringFunc(cleanScript, func(match string) string {
		parts := funcBlockRegex.FindStringSubmatch(match)
		fnName := parts[1]
		fnArgs := parts[2]
		fnBody := parts[3]
		// Flatten body and replace newlines with ;
		flatBody := strings.ReplaceAll(strings.TrimSpace(fnBody), "\n", "; ")
		return fmt.Sprintf("%s := func(%s) { %s }", fnName, fnArgs, flatBody)
	})

	extraSnippets := strings.TrimSpace(structDefs)
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

	// Inject script initializations if non-empty
	// All Go code must be inside {{ }} delimiters in templ
	scriptInjection := ""
	if strings.TrimSpace(cleanScript) != "" {
		lines := strings.Split(strings.TrimSpace(cleanScript), "\n")
		i := 0
		for i < len(lines) {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.Contains(trimmed, "func(") || strings.Contains(trimmed, "func ") {
				i++
				continue
			}
			if strings.Contains(trimmed, "var _ = ") {
				i++
				continue
			}
			// Detect multi-line blocks (if/for/switch with opening brace)
			openBraces := strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
			if openBraces > 0 {
				// Collect the entire block into a single {{ }} wrapper
				block := trimmed
				for openBraces > 0 && i+1 < len(lines) {
					i++
					nextLine := strings.TrimSpace(lines[i])
					block += " " + nextLine
					openBraces += strings.Count(nextLine, "{") - strings.Count(nextLine, "}")
				}
				scriptInjection += "{{ " + block + " }}\n"
			} else {
				scriptInjection += "{{ " + trimmed + " }}\n"
			}
			i++
		}
	}

	// Check if this page needs client-side hydration
	// It needs it if there's $state, $derived, or $effect in the script
	needsRegistration := hasClientCode

	// Signature generation
	propNames := []string{}
	for _, p := range props {
		propNames = append(propNames, p.Name+" "+p.Type)
	}
	propArgs := typeArgs(strings.Join(propNames, ", "))
	signature := name + "(" + propArgs + ")"

	// Generate output with optional registration
	var result string
	if needsRegistration {
		// Add state import and registration for pages that need hydration
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

		result = header + fmt.Sprintf(`package %s

%s

%s

templ %s {
%s
%s
	<div data-gospa-island="%s" class="%s">
		%s
	</div>
}
`, pkgName, extraImports, extraSnippets, signature, scriptInjection, registration, islandID, hash, cleanTemplate)
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

func (c *GospaCompiler) generateLayoutTempl(name, template, script, hash, pkgName string, props []Prop, templTypesSnippet string) string {
	templ := c.generateIslandTempl(name, "", template, script, hash, pkgName, props, templTypesSnippet)
	templ = strings.Replace(templ, "<div data-gospa-island=\"\" class=\""+hash+"\">", "<div class=\""+hash+"\">", 1)
	templ = strings.ReplaceAll(templ, "@children", "{ children }")
	signatureNeedle := "templ " + name + "("
	signatureReplace := "templ " + name + "(children templ.Component"
	if len(props) > 0 {
		signatureReplace += ", "
	}
	return strings.Replace(templ, signatureNeedle, signatureReplace, 1)
}

func (c *GospaCompiler) generateStaticTempl(name, template, script, hash, pkgName string, props []Prop, templTypesSnippet string) string {
	templ := c.generateIslandTempl(name, "", template, script, hash, pkgName, props, templTypesSnippet)
	templ = strings.Replace(templ, "\n\t<div data-gospa-island=\"\" class=\""+hash+"\">\n\t\t", "\n\t\t", 1)
	return strings.Replace(templ, "\n\t</div>\n}\n", "\n}\n", 1)
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
	// Scoping:
	// Scope all class selectors with the hash
	scopedStyle := CSSDotRegex.ReplaceAllString(style, ".$1."+hash)
	// 2. h1 -> h1.gospa-hash (simplified, works for elements too)
	scopedStyle = CSSElementRegex.ReplaceAllString(scopedStyle, "$1."+hash+" {")

	encodedStyle, _ := json.Marshal(scopedStyle)
	return fmt.Sprintf("\n\n/* Scoped CSS */\nif (!document.querySelector(`style[data-gospa-style=\"%s\"]`)) {\n\tconst style = document.createElement('style');\n\tstyle.setAttribute('data-gospa-style', '%s');\n\tstyle.textContent = %s;\n\tdocument.head.appendChild(style);\n}\n", hash, hash, string(encodedStyle))
}
