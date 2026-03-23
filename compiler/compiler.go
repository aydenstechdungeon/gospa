// Package compiler provides a compiler for GoSPA Single File Components (.gospa).
package compiler

import (
	"encoding/json"
	"fmt"
	"regexp"
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
func (c *GospaCompiler) Compile(name, input string) (templ, ts string, err error) {
	parsed, err := sfc.Parse(input)
	if err != nil {
		return "", "", err
	}

	// 1. Process Reactive DSL in Script
	processedScript := c.transformDSL(parsed.Script.Content)

	// 2. Generate Unique Hash for Scoping
	hash := c.generateHash(name)

	// 3. Generate Templ with Scoped CSS classes
	scopedTemplate := c.scopeTemplate(parsed.Template.Content, hash)
	templ = c.generateTempl(name, scopedTemplate, processedScript, hash)

	// 4. Generate TypeScript Island
	ts = c.generateTS(name, parsed.Script.Content, parsed.Template.Content, hash)

	// 5. Generate Scoped CSS
	ts += c.generateScopedCSS(parsed.Style.Content, hash)

	return templ, ts, nil
}

func (c *GospaCompiler) generateHash(name string) string {
	return fmt.Sprintf("gospa-%x", strings.ToLower(name))[:10]
}

func (c *GospaCompiler) scopeTemplate(template, hash string) string {
	// Simple prototype: add class to all tags
	re := regexp.MustCompile(`<([a-z0-9]+)([^>]*)>`)
	return re.ReplaceAllString(template, `<$1 class="`+hash+` "$2>`)
}

var (
	stateRegex   = regexp.MustCompile(`\$state\((.*?)\)`)
	derivedRegex = regexp.MustCompile(`\$derived\((.*?)\)`)
)

func (c *GospaCompiler) transformDSL(script string) string {
	// For Go (SSR), $state(val) -> val
	s := stateRegex.ReplaceAllString(script, "$1")
	// For Go (SSR), $derived(expr) -> expr
	s = derivedRegex.ReplaceAllString(s, "$1")
	// For Go (SSR), $effect(...) -> empty (effects only run on client)
	effectRegex := regexp.MustCompile(`(?s)\$effect\(func\(\)\s*\{(.*?)\}\)`)
	s = effectRegex.ReplaceAllString(s, "")
	return s
}

func (c *GospaCompiler) generateTempl(name, template, _, hash string) string {
	return fmt.Sprintf(`package islands

import "github.com/aydenstechdungeon/gospa/component"

type %sProps struct {}

templ %s(props %sProps) {
	<div data-gospa-island="%s" class="%s">
		%s
	</div>
}
`, name, name, name, name, hash, template)
}

func (c *GospaCompiler) generateTS(name, script, _, hash string) string {
	tsScript := script
	tsScript = stateRegex.ReplaceAllString(tsScript, "state.$$state($1)")
	tsScript = derivedRegex.ReplaceAllString(tsScript, "state.$$derived(() => $1)")
	
	effectRegex := regexp.MustCompile(`(?s)\$effect\(func\(\)\s*\{(.*?)\}\)`)
	tsScript = effectRegex.ReplaceAllString(tsScript, "state.$$effect(() => {$1})")

	tsScript = strings.ReplaceAll(tsScript, "func() {", "() => {")
	tsScript = strings.ReplaceAll(tsScript, "fmt.Printf", "console.log")
	tsScript = strings.ReplaceAll(tsScript, "var ", "const ")

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
	// Safe scoping: use json.Marshal to prevent breakout from JS string
	encodedStyle, _ := json.Marshal(strings.ReplaceAll(style, ".", "."+hash))
	return fmt.Sprintf("\n\n/* Scoped CSS */\nconst style = document.createElement('style');\nstyle.textContent = %s;\ndocument.head.appendChild(style);\n", string(encodedStyle))
}
