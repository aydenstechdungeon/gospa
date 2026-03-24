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

// NewCompiler creates a new GospaCompiler.
func NewCompiler() *GospaCompiler {
	return &GospaCompiler{}
}

// Compile compiles a .gospa component into Templ and TypeScript.
func (c *GospaCompiler) Compile(goName, islandID, input, pkgName string) (templ, ts string, err error) {
	name := c.sanitizeName(goName)
	islandID = c.sanitizeName(islandID)
	parsed, err := sfc.Parse(input)
	if err != nil {
		return "", "", err
	}

	// 1. Process Reactive DSL in Script
	processedScript := c.transformDSL(parsed.Script.Content)

	// 2. Generate Unique Hash for Scoping (based on islandID for uniqueness)
	hash := c.generateHash(islandID)

	// 3. Generate Templ with Scoped CSS classes
	scopedTemplate := c.scopeTemplate(parsed.Template.Content, hash)
	templ = c.generateTempl(name, islandID, scopedTemplate, processedScript, hash, pkgName)

	// 4. Generate TypeScript Island
	tsScript := parsed.Script.Content
	tsFromGo := true
	if parsed.ScriptTS.Content != "" {
		tsScript = parsed.ScriptTS.Content
		tsFromGo = false
	}
	ts = c.generateTS(islandID, tsScript, tsFromGo, parsed.Style.Content, hash)

	// 5. Generate Scoped CSS
	ts += c.generateScopedCSS(parsed.Style.Content, hash)

	return templ, ts, nil
}

func (c *GospaCompiler) scopeTemplate(template, hash string) string {
	// 1. Protect strings/backticks from scoping
	// This prevents tags inside code blocks/literals from being corrupted
	stringMap := make(map[string]string)
	placeholderCounter := 0

	// Regex for Go-style backtick strings and double-quoted strings
	strRegex := regexp.MustCompile("(?s)`[^`]*`|\"[^\"]*\"")
	
	protected := strRegex.ReplaceAllStringFunc(template, func(s string) string {
		// Only protect if it contains tags to keep output readable and prevent attribute collision
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
		if classAttrRegex.MatchString(tag) {
			return classAttrRegex.ReplaceAllString(tag, `class="$1 `+hash+`"`)
		}
		// If no class, insert it
		return tagRegex.ReplaceAllString(tag, `<$1 class="`+hash+` "$2>`)
	})

	// 3. Restore strings in a way that avoids partial placeholder replacements
	type mapping struct{ k, v string }
	sorted := make([]mapping, 0, len(stringMap))
	for k, v := range stringMap {
		sorted = append(sorted, mapping{k, v})
	}
	// Longest placeholders first to avoid replacing part of a longer one
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
	stateRegex     = regexp.MustCompile(`\$state\((.*?)\)`)
	derivedRegex   = regexp.MustCompile(`\$derived\((.*?)\)`)
	effectRegex    = regexp.MustCompile(`(?s)\$effect\(func\(\)\s*\{(.*?)\}\)`)
	tagRegex       = regexp.MustCompile(`(?i)<([a-z0-9]+)([^>]*)>`)
	classAttrRegex = regexp.MustCompile(`(?i)class="([^"]*)"`)
	cssDotRegex    = regexp.MustCompile(`\.([a-zA-Z][a-zA-Z0-9-_]*)`)
	nameSafeRegex  = regexp.MustCompile(`[^a-zA-Z0-9]`)
)

func (c *GospaCompiler) sanitizeName(name string) string {
	safe := nameSafeRegex.ReplaceAllString(name, "")
	if safe == "" {
		return "Component"
	}
	return safe
}

func (c *GospaCompiler) transformDSL(script string) string {
	// For Go (SSR), $state(val) -> val
	s := stateRegex.ReplaceAllString(script, "$1")
	// For Go (SSR), $derived(expr) -> expr
	s = derivedRegex.ReplaceAllString(s, "$1")
	// For Go (SSR), $effect(...) -> empty (effects only run on client)
	s = effectRegex.ReplaceAllString(s, "")
	return s
}

func (c *GospaCompiler) generateTempl(name, islandID, template, script, hash, pkgName string) string {
	if pkgName == "" {
		pkgName = "islands"
	}

	// Extract imports from script
	importRegex := regexp.MustCompile(`(?m)^import\s+(?:"[^"]+"|\(.*\))`)
	imports := importRegex.FindAllString(script, -1)
	cleanScript := importRegex.ReplaceAllString(script, "")

	extraImports := strings.Join(imports, "\n")

	// Inject script if non-empty
	scriptInjection := ""
	if strings.TrimSpace(cleanScript) != "" {
		scriptInjection = "\n\t" + strings.ReplaceAll(strings.TrimSpace(cleanScript), "\n", "\n\t") + "\n"
	}

	// Pages should have a simple Page() signature for the route generator
	// Layouts might need children, handled separately if we want to support nested .gospa layouts
	signature := "Page()"
	if strings.ToLower(name) == "layout" {
		signature = "Layout(children templ.Component)"
	}

	// Extract simple variables to make them available in templ scope for SSR
	return fmt.Sprintf(`package %s

%s

templ %s {
	%s
	<div data-gospa-island="%s" class="%s">
		%s
	</div>
}
`, pkgName, extraImports, signature, scriptInjection, islandID, hash, template)
}

func (c *GospaCompiler) generateTS(name, script string, fromGo bool, _ string, hash string) string {
	tsScript := script
	if fromGo {
		tsScript = stateRegex.ReplaceAllString(tsScript, "state.$$state($1)")
		tsScript = derivedRegex.ReplaceAllString(tsScript, "state.$$derived(() => $1)")
		tsScript = effectRegex.ReplaceAllString(tsScript, "state.$$effect(() => {$1})")

		// Slightly safer replacement for JS (avoids strings in simple cases)
		tsScript = strings.ReplaceAll(tsScript, "func() {", "() => {")
		tsScript = strings.ReplaceAll(tsScript, "fmt.Printf", "console.log")
		tsScript = strings.ReplaceAll(tsScript, "var ", "const ")
	}

	return fmt.Sprintf(`import { createIsland } from '@gospa/runtime';

export default createIsland({
  name: '%s',
  setup(element, { props, state }) {
%s
    
    // Scoped hydration selector
    const scope = (selector) => element.querySelector(selector + '.' + '%s');
  }
});
`, name, tsScript, hash)
}

func (c *GospaCompiler) generateScopedCSS(style, hash string) string {
	if style == "" {
		return ""
	}
	// Scoping: Append hash as a class selector (e.g., .card -> .card.gospa-hash)
	scopedStyle := cssDotRegex.ReplaceAllString(style, ".$1."+hash)

	encodedStyle, _ := json.Marshal(scopedStyle)
	return fmt.Sprintf("\n\n/* Scoped CSS */\nconst style = document.createElement('style');\nstyle.textContent = %s;\ndocument.head.appendChild(style);\n", string(encodedStyle))
}
