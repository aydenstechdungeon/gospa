package compiler

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aydenstechdungeon/gospa/compiler/sfc"
)

func TestParseSFC(t *testing.T) {
	input := `
<script lang="go">
  var count = 0
</script>

<template>
  <div>{count}</div>
</template>

<style>
  div { color: red; }
</style>
`
	parsed, err := sfc.Parse(input)
	if err != nil {
		t.Fatalf("Failed to parse SFC: %v", err)
	}

	if parsed.Script.Content != "var count = 0" {
		t.Errorf("Unexpected script content: %q", parsed.Script.Content)
	}

	if parsed.Template.Content != "<div>{count}</div>" {
		t.Errorf("Unexpected template content: %q", parsed.Template.Content)
	}

	if parsed.Style.Content != "div { color: red; }" {
		t.Errorf("Unexpected style content: %q", parsed.Style.Content)
	}

	fmt.Println("SFC Parse test passed")
}

func TestCompileCounter(t *testing.T) {
	c := NewCompiler()
	input := `
<script lang="go">
  var count = $state(0)
  var doubled = $derived(count * 2)
  
  $effect(func() {
    fmt.Printf("Count: %d\n", count)
  })

  func increment() {
    count++
  }
</script>

<template>
  <button on:click={increment}>{count}</button>
</template>
`
	templ, ts, err := c.Compile(CompileOptions{
		Type:     ComponentTypeIsland,
		Name:     "Counter",
		IslandID: "Counter",
		PkgName:  "islands",
		Hydrate:  true,
	}, input)
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	if !strings.Contains(templ, "data-gospa-island=\"Counter\"") {
		t.Errorf("Templ missing island attribute")
	}

	if !strings.Contains(ts, "__gospa_state(0)") {
		t.Errorf("TS missing reactive state")
	}

	if !strings.Contains(ts, "__gospa_derived(() => count * 2)") {
		t.Errorf("TS missing derived state")
	}

	if !strings.Contains(ts, "console.log(\"Count: %d\\n\", count)") {
		t.Errorf("TS missing effect/console.log: %q", ts)
	}

	fmt.Println("Counter compilation test passed")
}

func TestSanitizeName(t *testing.T) {
	c := NewCompiler()
	rawName := "Counter'); alert(1); //"
	_, ts, err := c.Compile(CompileOptions{
		Type:     ComponentTypeIsland,
		Name:     rawName,
		IslandID: rawName,
		PkgName:  "islands",
		Hydrate:  true,
	}, "<template><div>Test</div></template>")
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	if strings.Contains(ts, "alert(1)") {
		t.Errorf("Sanitization failed: TS still contains alert(1)")
	}

	if !strings.Contains(ts, "Counteralert1") {
		t.Errorf("Sanitized name 'Counteralert1' not found in TS: %v", ts)
	}
}

func TestCompileWithEmptySanitizedName(t *testing.T) {
	c := NewCompiler()
	templ, ts, err := c.Compile(CompileOptions{
		Type:     ComponentTypeIsland,
		Name:     "!!!",
		IslandID: "!!!",
		PkgName:  "islands",
		Hydrate:  true,
	}, "<template><div>Test</div></template>")
	if err != nil {
		t.Fatalf("Failed to compile with empty sanitized name: %v", err)
	}

	if !strings.Contains(ts, "__gospa_setup_Component") {
		t.Fatalf("Expected fallback component name 'Component' in TS setup function, got: %s", ts)
	}
	if !strings.Contains(templ, "data-gospa-island=\"Component\"") {
		t.Fatalf("Expected fallback component name in templ output, got: %s", templ)
	}
}

func TestParseRejectsMultipleTemplates(t *testing.T) {
	input := `
<template><div>One</div></template>
<template><div>Two</div></template>
`
	_, err := sfc.Parse(input)
	if err == nil {
		t.Fatal("Expected Parse to reject multiple template blocks")
	}
	if !strings.Contains(err.Error(), "multiple <template> blocks") {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestParseAllowsSeparateGoAndTSScripts(t *testing.T) {
	input := `
<script lang="go">
  var count = $state(1)
</script>
<script lang="ts">
  const count = state.$state(1)
  const doubled = state.$derived(() => count.value * 2)
</script>
<template><div>{count}</div></template>
`
	parsed, err := sfc.Parse(input)
	if err != nil {
		t.Fatalf("Expected parser to accept one go script and one ts script: %v", err)
	}
	if parsed.Script.Lang != "go" || parsed.ScriptTS.Lang != "ts" {
		t.Fatalf("Unexpected parsed script languages: go=%q ts=%q", parsed.Script.Lang, parsed.ScriptTS.Lang)
	}
}

func TestCompileUsesTSScriptWhenProvided(t *testing.T) {
	c := NewCompiler()
	input := `
<script lang="go">
  var count = $state(1)
</script>
<script lang="ts">
  const count = state.$state(1)
  const greet = "func() { should stay unchanged }"
</script>
<template><div>{count}</div></template>
`
	_, ts, err := c.Compile(CompileOptions{
		Type:     ComponentTypeIsland,
		Name:     "DualScript",
		IslandID: "DualScript",
		PkgName:  "islands",
		Hydrate:  true,
	}, input)
	if err != nil {
		t.Fatalf("Failed to compile dual-script component: %v", err)
	}
	if !strings.Contains(ts, `const greet = "func() { should stay unchanged }"`) {
		t.Fatalf("Expected TS script to be used as-is when lang=ts is present, got: %s", ts)
	}
	if strings.Contains(ts, "state.$$state(") {
		t.Fatalf("Did not expect Go DSL transform on explicit TS script, got: %s", ts)
	}
}

func TestParseRejectsDuplicateGoScripts(t *testing.T) {
	input := `
<script lang="go">var a = 1</script>
<script lang="go">var b = 2</script>
<template><div>ok</div></template>
`
	_, err := sfc.Parse(input)
	if err == nil {
		t.Fatal("Expected parse failure for duplicate go scripts")
	}
	if !strings.Contains(err.Error(), "multiple <script lang=\"go\">") {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestParseAcceptsModuleScript(t *testing.T) {
	input := `
<script context="module" lang="go">
  func Load(c routing.LoadContext) (map[string]interface{}, error) { return map[string]interface{}{}, nil }
</script>
<script lang="go">
  var count = $state(1)
</script>
<template><div>{count}</div></template>
`
	parsed, err := sfc.Parse(input)
	if err != nil {
		t.Fatalf("expected parser to accept module script: %v", err)
	}
	if parsed.ScriptModule.Context != "module" {
		t.Fatalf("expected module context, got %q", parsed.ScriptModule.Context)
	}
	if parsed.ScriptModule.Lang != "go" {
		t.Fatalf("expected module script lang go, got %q", parsed.ScriptModule.Lang)
	}
}

func TestParseRejectsDuplicateModuleScripts(t *testing.T) {
	input := `
<script context="module" lang="go">func Load(c routing.LoadContext) (map[string]interface{}, error) { return nil, nil }</script>
<script context="module" lang="go">func ActionDefault(c routing.LoadContext) (interface{}, error) { return nil, nil }</script>
<template><div>ok</div></template>
`
	_, err := sfc.Parse(input)
	if err == nil {
		t.Fatal("expected parse failure for duplicate module scripts")
	}
	if !strings.Contains(err.Error(), "multiple <script context=\"module\" lang=\"go\">") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompilePageNoIslandAndNoTS(t *testing.T) {
	c := NewCompiler()
	input := `<template><h1>Hello</h1></template>`

	templ, ts, err := c.Compile(CompileOptions{
		Type:    ComponentTypePage,
		Name:    "Home",
		PkgName: "pages",
	}, input)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if strings.Contains(templ, "data-gospa-island") {
		t.Fatalf("page templ should not include island wrapper: %s", templ)
	}
	if strings.TrimSpace(ts) != "" {
		t.Fatalf("page should not generate TS output: %s", ts)
	}
}

func TestCompileLayoutIncludesChildrenAndNoTS(t *testing.T) {
	c := NewCompiler()
	input := `<template><main>@children</main></template>`

	templ, ts, err := c.Compile(CompileOptions{
		Type:    ComponentTypeLayout,
		Name:    "MainLayout",
		PkgName: "layouts",
	}, input)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if !strings.Contains(templ, "children templ.Component") {
		t.Fatalf("layout should accept children in signature: %s", templ)
	}
	if !strings.Contains(templ, "{ children }") {
		t.Fatalf("layout should render children placeholder: %s", templ)
	}
	if strings.TrimSpace(ts) != "" {
		t.Fatalf("layout should not generate TS output: %s", ts)
	}
}

func TestCompileLowersOnDirectiveToDelegationAttribute(t *testing.T) {
	c := NewCompiler()
	input := `
<script lang="go">
  func increment() {}
</script>
<template><button on:click={increment}>+</button></template>
`
	templ, _, err := c.Compile(CompileOptions{
		Type:     ComponentTypeIsland,
		Name:     "Counter",
		IslandID: "Counter",
		PkgName:  "islands",
		Hydrate:  true,
	}, input)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if !strings.Contains(templ, `data-gospa-on="click:increment"`) {
		t.Fatalf("expected lowered runtime event attribute, got: %s", templ)
	}
	if strings.Contains(templ, " on:click=") {
		t.Fatalf("expected on:click to be lowered away, got: %s", templ)
	}
}

func TestCompileStaticNoWrapperAndNoTS(t *testing.T) {
	c := NewCompiler()
	input := `<template><p>Footer</p></template>`

	templ, ts, err := c.Compile(CompileOptions{
		Type:    ComponentTypeStatic,
		Name:    "Footer",
		PkgName: "components",
	}, input)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if strings.Contains(templ, "data-gospa-island") {
		t.Fatalf("static templ should not include island marker: %s", templ)
	}
	if strings.Contains(templ, "<div class=") {
		t.Fatalf("static templ should not include outer wrapper: %s", templ)
	}
	if strings.TrimSpace(ts) != "" {
		t.Fatalf("static should not generate TS output: %s", ts)
	}
}

func TestFrontMatterParsingAndTypeDefaulting(t *testing.T) {
	parsed, err := sfc.Parse(`---
type: page
hydrate: false
---
<template><div>ok</div></template>`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if parsed.FrontMatter["type"] != "page" {
		t.Fatalf("expected frontmatter type page, got %q", parsed.FrontMatter["type"])
	}

	c := NewCompiler()
	templ, ts, err := c.Compile(CompileOptions{
		Name: "FrontMatterPage",
	}, `---
type: page
---
<template><div>ok</div></template>`)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if strings.Contains(templ, "data-gospa-island") {
		t.Fatalf("frontmatter page should not include island wrapper: %s", templ)
	}
	if strings.TrimSpace(ts) != "" {
		t.Fatalf("frontmatter page should not generate TS output: %s", ts)
	}
}

func TestCompileDefaultsToIslandWithoutFrontMatter(t *testing.T) {
	c := NewCompiler()
	templ, ts, err := c.Compile(CompileOptions{
		Name:     "Counter",
		IslandID: "counter",
	}, `<template><div>Counter</div></template>`)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if !strings.Contains(templ, "data-gospa-island=\"counter\"") {
		t.Fatalf("default compile should remain island behavior: %s", templ)
	}
	if strings.TrimSpace(ts) == "" {
		t.Fatalf("default island compile should generate TS output")
	}
}

func TestValidateSafeScript(t *testing.T) {
	tests := []struct {
		name    string
		script  string
		wantErr bool
	}{
		{"empty script", "", false},
		{"safe script", "var x = 1\nfmt.Println(x)", false},
		{"unsafe import", "import \"os/exec\"\nfunc main() {}", true},
		{"unsafe import rename", "import e \"os/exec\"\nfunc main() {}", true},
		{"unsafe call direct", "import \"os/exec\"\nexec.Command(\"ls\")", true},
		{"unsafe os writeFile", "import \"os\"\nos.WriteFile(\"test\", nil, 0644)", true},
		{"invalid go script", "var x = ", true}, // Should fail go parsing
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSafeScript(tt.script)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSafeScript() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSafeScript_Hardening(t *testing.T) {
	tests := []struct {
		name    string
		script  string
		wantErr bool
	}{
		{"dot import", "import . \"net/http\"\nfunc main() { Get(\"http://evil.com\") }", true},
		{"cgo import", "import \"C\"\nfunc main() { C.system(nil) }", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSafeScript(tt.script)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSafeScript() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSafeTSScript_AcceptsValidTypeScript(t *testing.T) {
	script := `
const count: number = 1
type User = { id: string; age?: number }
const names: string[] = ["a", "b"]
`
	if err := ValidateSafeTSScript(script); err != nil {
		t.Fatalf("expected valid TypeScript to pass safe-mode validation, got %v", err)
	}
}

func TestCompile_SafeMode_AcceptsTypeScriptScriptBlock(t *testing.T) {
	c := NewCompiler()
	input := `
<script lang="ts">
const count: number = 1
const labels: string[] = ["ok"]
</script>
<template><button>{count}</button></template>
`
	_, _, err := c.Compile(CompileOptions{
		Type:     ComponentTypeIsland,
		Name:     "SafeTS",
		PkgName:  "islands",
		IslandID: "safe-ts",
		Hydrate:  true,
		SafeMode: true,
	}, input)
	if err != nil {
		t.Fatalf("expected SafeMode to accept valid TypeScript script block, got %v", err)
	}
}

func TestCompileMultilineScript(t *testing.T) {
	c := NewCompiler()
	input := `
<script lang="go">
  var x = 1
  if x > 0 {
    fmt.Println("positive")
  }
</script>
<template><div>{x}</div></template>
`
	templ, _, err := c.Compile(CompileOptions{
		Type:     ComponentTypeIsland,
		Name:     "Multiline",
		PkgName:  "islands",
		IslandID: "multiline",
	}, input)
	if err != nil {
		t.Fatalf("Failed to compile multiline script: %v", err)
	}
	if !strings.Contains(templ, "if x > 0 {") {
		t.Fatalf("Templ missing multiline if statement: %s", templ)
	}
}

func TestCompileLegacy(t *testing.T) {
	c := NewCompiler()
	templ, ts, err := c.CompileLegacy("MyLegacyIsland", "legacy-id", "<template><div>Legacy</div></template>", "legacy_pkg")

	if err != nil {
		t.Fatalf("CompileLegacy failed: %v", err)
	}

	if !strings.Contains(templ, "package legacy_pkg") {
		t.Errorf("Expected package legacy_pkg, got %s", templ)
	}

	if !strings.Contains(templ, "MyLegacyIsland()") {
		t.Errorf("Expected signature MyLegacyIsland(), got %s", templ)
	}

	if !strings.Contains(templ, "data-gospa-island=\"legacyid\"") {
		t.Errorf("Expected sanitized island-id legacyid, got %s", templ)
	}

	if ts == "" {
		t.Error("CompileLegacy should produce TS for an island by default")
	}
}

func TestCompile_ComponentCallExpressionArgStaysExpression(t *testing.T) {
	c := NewCompiler()
	templ, _, err := c.Compile(CompileOptions{
		Type:    ComponentTypePage,
		Name:    "ExprArgs",
		PkgName: "pages",
	}, `<template>@Child({count})</template>`)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if !strings.Contains(templ, "@Child(count)") {
		t.Fatalf("expected expression arg to remain expression, got: %s", templ)
	}
	if strings.Contains(templ, "@Child(`count`)") {
		t.Fatalf("expression arg should not be stringified, got: %s", templ)
	}
}

func TestCompile_ComponentCallBacktickArgStaysStringLiteral(t *testing.T) {
	c := NewCompiler()
	templ, _, err := c.Compile(CompileOptions{
		Type:    ComponentTypePage,
		Name:    "CodeBlockArg",
		PkgName: "pages",
	}, `<template>@components.CodeBlock(`+"`"+`import "x"`+"`"+`, "go", "sample.go")</template>`)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if !strings.Contains(templ, "@components.CodeBlock(`import \"x\"`, \"go\", \"sample.go\")") {
		t.Fatalf("expected backtick arg to stay a valid string literal, got: %s", templ)
	}
}

func TestCompileRejectsInvalidModuleLoadSignature(t *testing.T) {
	c := NewCompiler()
	input := `
<script context="module" lang="go">
  func Load(c routing.LoadContext) (interface{}, error) {
    return nil, nil
  }
</script>
<template><div>bad</div></template>
`
	_, _, err := c.Compile(CompileOptions{
		Type:    ComponentTypePage,
		Name:    "BadModule",
		PkgName: "routes",
	}, input)
	if err == nil {
		t.Fatal("expected compile failure for invalid module Load signature")
	}
	if !strings.Contains(err.Error(), "invalid module export Load") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompileRejectsModuleMethodExport(t *testing.T) {
	c := NewCompiler()
	input := `
<script context="module" lang="go">
  type S struct{}
  func (s S) Load(c routing.LoadContext) (map[string]interface{}, error) {
    return map[string]interface{}{}, nil
  }
</script>
<template><div>bad</div></template>
`
	_, _, err := c.Compile(CompileOptions{
		Type:    ComponentTypePage,
		Name:    "BadModuleMethod",
		PkgName: "routes",
	}, input)
	if err == nil {
		t.Fatal("expected compile failure for method export")
	}
	if !strings.Contains(err.Error(), "top-level function") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompileRejectsLowercaseModuleActionSuffix(t *testing.T) {
	c := NewCompiler()
	input := `
<script context="module" lang="go">
  func Actionsave(c routing.LoadContext) (interface{}, error) {
    return nil, nil
  }
</script>
<template><div>bad</div></template>
`
	_, _, err := c.Compile(CompileOptions{
		Type:    ComponentTypePage,
		Name:    "BadModuleActionName",
		PkgName: "routes",
	}, input)
	if err == nil {
		t.Fatal("expected compile failure for invalid action export name")
	}
	if !strings.Contains(err.Error(), "suffix must start with an uppercase letter") {
		t.Fatalf("unexpected error: %v", err)
	}
}
