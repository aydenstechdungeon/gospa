// Package compiler provides a compiler for GoSPA Single File Components (.gospa).
package compiler

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
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

	// 1. Process Reactive DSL in Script and extract Props
	processedScript, props := c.transformDSL(parsed.Script.Content)

	// 2. Generate Unique Hash for Scoping
	hash := c.generateHash(islandID)

	// 3. Transform Template (Svelte-like syntax)
	transformedTemplate := c.transformTemplate(parsed.Template.Content)

	// 4. Generate Templ with Scoped CSS
	scopedTemplate := c.scopeTemplate(transformedTemplate, hash)
	switch componentType {
	case ComponentTypePage:
		templ = c.generatePageTempl(name, scopedTemplate, processedScript, hash, opts.PkgName, props)
	case ComponentTypeLayout:
		templ = c.generateLayoutTempl(name, scopedTemplate, processedScript, hash, opts.PkgName, props)
	case ComponentTypeStatic, ComponentTypeServer:
		templ = c.generateStaticTempl(name, scopedTemplate, processedScript, hash, opts.PkgName, props)
	default:
		templ = c.generateIslandTempl(name, islandID, scopedTemplate, processedScript, hash, opts.PkgName, props)
	}

	// 5. Generate TypeScript Island
	if componentType == ComponentTypeIsland && opts.Hydrate && !opts.ServerOnly {
		tsScript := parsed.Script.Content
		tsFromGo := true
		if parsed.ScriptTS.Content != "" {
			tsScript = parsed.ScriptTS.Content
			tsFromGo = false
		}
		ts = c.generateTS(islandID, tsScript, tsFromGo, parsed.Style.Content, hash)
		ts += c.generateScopedCSS(parsed.Style.Content, hash)
	}

	return templ, ts, nil
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

func (c *GospaCompiler) scopeTemplate(template, hash string) string {
	// 1. Protect strings/backticks
	stringMap := make(map[string]string)
	placeholderCounter := 0
	strRegex := regexp.MustCompile("(?s)`[^`]*`|\"[^\"]*\"")

	protected := strRegex.ReplaceAllStringFunc(template, func(s string) string {
		if !strings.ContainsAny(s, "<>") {
			return s
		}
		placeholder := fmt.Sprintf("__GOSPA_STR_ID_%d__", placeholderCounter)
		stringMap[placeholder] = s
		placeholderCounter++
		return placeholder
	})

	// 2. Scope remaining tags
	scoped := tagRegex.ReplaceAllStringFunc(protected, func(tag string) string {
		if strings.HasPrefix(strings.ToLower(tag), "<script") || strings.HasPrefix(strings.ToLower(tag), "<style") {
			return tag
		}

		// If it's a component call (e.g. @Component), don't scope it here as Templ handles it
		if strings.HasPrefix(tag, "@") {
			return tag
		}

		if classAttrRegex.MatchString(tag) {
			return classAttrRegex.ReplaceAllString(tag, `class="$1 `+hash+`"`)
		}
		// If no class, insert it (but be careful with self-closing tags and components)
		if strings.HasSuffix(tag, "/>") {
			return strings.Replace(tag, "/>", ` class="`+hash+`" />`, 1)
		}
		return tagRegex.ReplaceAllString(tag, `<$1 class="`+hash+` "$2>`)
	})

	// 3. Restore strings
	type mapping struct{ k, v string }
	sorted := make([]mapping, 0, len(stringMap))
	for k, v := range stringMap {
		sorted = append(sorted, mapping{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].k) > len(sorted[j].k)
	})

	for _, m := range sorted {
		scoped = strings.ReplaceAll(scoped, m.k, m.v)
	}

	return scoped
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
	stateRegex         = regexp.MustCompile(`\$state\((.*?)\)`)
	derivedRegex       = regexp.MustCompile(`\$derived\((.*?)\)`)
	effectRegex        = regexp.MustCompile(`(?s)\$effect\(func\(\)\s*\{(.*?)\}\)`)
	tagRegex           = regexp.MustCompile(`(?i)<([a-z0-9-]+)([^>]*)>`) // Added dash for custom components
	classAttrRegex     = regexp.MustCompile(`(?i)class="([^"]*)"`)
	cssDotRegex        = regexp.MustCompile(`\.([a-zA-Z][a-zA-Z0-9-_]*)`)
	cssElementRegex    = regexp.MustCompile(`(?m)^([a-z0-9]+)\s*\{`)
	nameSafeRegex      = regexp.MustCompile(`[^a-zA-Z0-9]`)
	propsRegex         = regexp.MustCompile(`(?m)var\s+\{\s*(.*?)\s*\}\s*=\s*\$props\(\)`)
	ifRegex            = regexp.MustCompile(`\{#if\s+(.*?)\}`)
	elseIfRegex        = regexp.MustCompile(`\{:else\s+if\s+(.*?)\}`)
	elseRegex          = regexp.MustCompile(`\{:else\}`)
	endIfRegex         = regexp.MustCompile(`\{/if\}`)
	eachRegex          = regexp.MustCompile(`\{#each\s+(.*?)\s+as\s+(.*?)\}`)
	endEachRegex       = regexp.MustCompile(`\{/each\}`)
	shorthandRegex     = regexp.MustCompile(`\s\{([a-zA-Z0-9_]+)\}([\s/>])`) // Only match in attribute position
	bindRegex          = regexp.MustCompile(`bind:([a-zA-Z0-9-]+)=\{([a-zA-Z0-9_.]+)\}`)
	transitionRegex    = regexp.MustCompile(`transition:([a-zA-Z0-9-]+)(?:=\{([^}]*)\})?`)
	componentTagRegex  = regexp.MustCompile(`<([A-Z][a-zA-Z0-9-_]*)([^>]*)>`) // Removed (?i)
	endComponentRegex  = regexp.MustCompile(`</([A-Z][a-zA-Z0-9-_]*)>`)       // Removed (?i)
	snippetRegex       = regexp.MustCompile(`\{#snippet\s+([a-zA-Z0-9_]+)\((.*?)\)\}`)
	endSnippetRegex    = regexp.MustCompile(`\{/snippet\}`)
	onRegex            = regexp.MustCompile(`\son:([a-zA-Z0-9:]+)=\{([a-zA-Z0-9_.]+)\}`)
	reactiveLabelRegex = regexp.MustCompile(`\$:\s*([a-zA-Z0-9_]+)\s*=\s*([^;\n]+)`)
)

func (c *GospaCompiler) sanitizeName(name string) string {
	safe := nameSafeRegex.ReplaceAllString(name, "")
	if safe == "" {
		return "Component"
	}
	return safe
}

func (c *GospaCompiler) transformDSL(script string) (string, []string) {
	var props []string
	if matches := propsRegex.FindStringSubmatch(script); len(matches) > 1 {
		pList := strings.Split(matches[1], ",")
		for _, p := range pList {
			props = append(props, strings.TrimSpace(p))
		}
		script = propsRegex.ReplaceAllString(script, "")
	}

	// Implicit reactive statements $: val = expr -> var val = expr
	script = reactiveLabelRegex.ReplaceAllString(script, "var $1 = $2")

	// For Go (SSR), $state(val) -> val, $derived(expr) -> expr
	script = stateRegex.ReplaceAllString(script, "$1")
	script = derivedRegex.ReplaceAllString(script, "$1")
	script = effectRegex.ReplaceAllString(script, "")

	return script, props
}

func (c *GospaCompiler) transformTemplate(template string) string {
	// 0. Protect backtick strings
	stringMap := make(map[string]string)
	placeholderCounter := 0

	backtickRegex := regexp.MustCompile("(?s)`[^`]*`")

	s := backtickRegex.ReplaceAllStringFunc(template, func(match string) string {
		placeholder := fmt.Sprintf("__GOSPA_PROTECTED_%d__", placeholderCounter)
		stringMap[placeholder] = match
		placeholderCounter++
		return placeholder
	})

	// 1. Logic Blocks
	s = ifRegex.ReplaceAllString(s, "if $1 {")
	s = elseIfRegex.ReplaceAllString(s, "} else if $1 {")
	s = elseRegex.ReplaceAllString(s, "} else {")
	s = endIfRegex.ReplaceAllString(s, "}")
	s = eachRegex.ReplaceAllString(s, "for _, $2 := range $1 {")
	s = endEachRegex.ReplaceAllString(s, "}")

	// 2. Snippets (Removed from here, will be extracted in generateTempl)

	// 3. Components (PascalCase)
	s = componentTagRegex.ReplaceAllStringFunc(s, func(match string) string {
		parts := componentTagRegex.FindStringSubmatch(match)
		name := parts[1]
		attrs := parts[2]
		if strings.HasSuffix(attrs, "/") {
			return fmt.Sprintf("@%s(%s)", name, strings.TrimSuffix(attrs, "/"))
		}
		return fmt.Sprintf("@%s(%s) {", name, attrs)
	})
	s = endComponentRegex.ReplaceAllString(s, "}")

	// 4. Bindings
	s = bindRegex.ReplaceAllString(s, `data-gospa-bind="$1:$2"`)

	// 4b. Events on:click={fn} -> data-gospa-on="click:fn"
	s = onRegex.ReplaceAllString(s, ` data-gospa-on="$1:$2"`)

	// 2. Snippet Calls {snippet(args)} -> @snippet(args)
	// We look for {name(...)} where name is a snippet
	// This is a bit broad, but snippets usually start with lowercase
	snippetCallRegex := regexp.MustCompile(`\{([a-z][a-zA-Z0-9_]*)\((.*?)\)\}`)
	s = snippetCallRegex.ReplaceAllString(s, "@$1($2)")

	// 5. Transitions
	s = transitionRegex.ReplaceAllStringFunc(s, func(match string) string {
		parts := transitionRegex.FindStringSubmatch(match)
		name := parts[1]
		params := ""
		if len(parts) > 2 {
			params = parts[2]
		}
		res := fmt.Sprintf(`data-transition="%s"`, name)
		if params != "" {
			res += fmt.Sprintf(` data-transition-params='%s'`, params)
		}
		return res
	})

	s = shorthandRegex.ReplaceAllString(s, " $1={$1}$2")

	// 7. Restore protected strings
	for k, v := range stringMap {
		s = strings.ReplaceAll(s, k, v)
	}

	return s
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

func (c *GospaCompiler) generateIslandTempl(name, islandID, template, script, hash, pkgName string, props []string) string {
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

	extraSnippets := strings.Join(snippetDefs, "\n\n")

	// Inject script initializations if non-empty, every line prefixed with @
	// Skip function definitions in Templ (they are for client-side TS)
	scriptInjection := ""
	if strings.TrimSpace(cleanScript) != "" {
		lines := strings.Split(strings.TrimSpace(cleanScript), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.Contains(trimmed, "func(") || strings.Contains(trimmed, "func ") {
				continue
			}
			// If it starts with var, keep it as is (with var) or convert to :=
			// Templ handles both, but var is safer for blank identifiers
			scriptInjection += "{{ " + trimmed + " }}\n"
		}
	}

	// Signature generation
	propArgs := typeArgs(strings.Join(props, ", "))
	signature := name + "(" + propArgs + ")"

	return header + fmt.Sprintf(`package %s

%s

%s

templ %s {
%s
	<div data-gospa-island="%s" class="%s">
		%s
	</div>
}
`, pkgName, extraImports, extraSnippets, signature, scriptInjection, islandID, hash, cleanTemplate)
}

func (c *GospaCompiler) generatePageTempl(name, template, script, hash, pkgName string, props []string) string {
	templ := c.generateIslandTempl(name, "", template, script, hash, pkgName, props)
	return strings.Replace(templ, "<div data-gospa-island=\"\" class=\""+hash+"\">", "<div class=\""+hash+"\">", 1)
}

func (c *GospaCompiler) generateLayoutTempl(name, template, script, hash, pkgName string, props []string) string {
	templ := c.generateIslandTempl(name, "", template, script, hash, pkgName, props)
	templ = strings.Replace(templ, "<div data-gospa-island=\"\" class=\""+hash+"\">", "<div class=\""+hash+"\">", 1)
	templ = strings.ReplaceAll(templ, "@children", "{ children }")
	signatureNeedle := "templ " + name + "("
	signatureReplace := "templ " + name + "(children templ.Component"
	if len(props) > 0 {
		signatureReplace += ", "
	}
	return strings.Replace(templ, signatureNeedle, signatureReplace, 1)
}

func (c *GospaCompiler) generateStaticTempl(name, template, script, hash, pkgName string, props []string) string {
	templ := c.generateIslandTempl(name, "", template, script, hash, pkgName, props)
	templ = strings.Replace(templ, "\n\t<div data-gospa-island=\"\" class=\""+hash+"\">\n\t\t", "\n\t\t", 1)
	return strings.Replace(templ, "\n\t</div>\n}\n", "\n}\n", 1)
}

func (c *GospaCompiler) generateTS(name, script string, fromGo bool, _ string, hash string) string {
	tsScript := script
	funcNames := []string{}

	if fromGo {
		// 1. $props() -> destructuring (Strip types)
		if matches := propsRegex.FindStringSubmatch(tsScript); len(matches) > 1 {
			pList := strings.Split(matches[1], ",")
			cleanProps := []string{}
			for _, p := range pList {
				trimmed := strings.TrimSpace(p)
				parts := strings.Fields(trimmed)
				if len(parts) > 0 {
					cleanProps = append(cleanProps, parts[0])
				}
			}
			tsScript = propsRegex.ReplaceAllString(tsScript, fmt.Sprintf("const { %s } = props", strings.Join(cleanProps, ", ")))
		}

		// 2. Convert Go func to JS function
		funcRegex := regexp.MustCompile(`(?m)^\s*func\s+([a-zA-Z0-9_]+)\((.*?)\)\s*\{`)
		tsScript = funcRegex.ReplaceAllString(tsScript, "function $1($2) {")

		// 3. Handle runes
		tsScript = stateRegex.ReplaceAllString(tsScript, "state.$$state($1)")
		tsScript = derivedRegex.ReplaceAllString(tsScript, "state.$$derived(() => $1)")
		tsScript = effectRegex.ReplaceAllString(tsScript, "state.$$effect($1)")

		// 4. Reactive labels $: name = expr -> const name = state.$$derived(() => expr)
		tsScript = reactiveLabelRegex.ReplaceAllString(tsScript, "const $1 = state.$$derived(() => $2)")

		// 5. Clean up Go-isms
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

	eventBindingLogic := fmt.Sprintf(`
    // Event binding logic
    const __GOSPA_FUNCS__ = %s;
    element.querySelectorAll('[data-gospa-on]').forEach(el => {
      const attr = el.getAttribute('data-gospa-on');
      if (!attr) return;
      const [eventStr, fnName] = attr.split(':');
      const handler = (__GOSPA_FUNCS__ as any)[fnName];
      if (handler) {
        import('@gospa/runtime').then(m => m.on(el, eventStr, handler));
      }
    });`, funcsObject)

	return header + fmt.Sprintf(`import { createIsland } from '@gospa/runtime';

export default createIsland({
  name: '%s',
  setup(element, { props, state }) {
%s

%s
    
    // Scoped hydration selector
    const scope = (selector: string) => element.querySelector(selector + '.' + '%s');
  }
});
`, name, tsScript, eventBindingLogic, hash)
}

func (c *GospaCompiler) generateScopedCSS(style, hash string) string {
	if style == "" {
		return ""
	}
	// Scoping:
	// 1. .card -> .card.gospa-hash
	scopedStyle := cssDotRegex.ReplaceAllString(style, ".$1."+hash)
	// 2. h1 -> h1.gospa-hash (simplified, works for elements too)
	scopedStyle = cssElementRegex.ReplaceAllString(scopedStyle, "$1."+hash+" {")

	encodedStyle, _ := json.Marshal(scopedStyle)
	return fmt.Sprintf("\n\n/* Scoped CSS */\nif (!document.querySelector(`style[data-gospa-style=\"%s\"]`)) {\n\tconst style = document.createElement('style');\n\tstyle.setAttribute('data-gospa-style', '%s');\n\tstyle.textContent = %s;\n\tdocument.head.appendChild(style);\n}\n", hash, hash, string(encodedStyle))
}
