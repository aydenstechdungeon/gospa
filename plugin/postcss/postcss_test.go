package postcss

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassToEscapedSelector(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"bg-white", `.bg-white`},
		{"bg-white/5", `.bg-white\/5`},
		{"bg-gradient-to-br", `.bg-gradient-to-br`},
		{"bg-clip-text", `.bg-clip-text`},
		{"text-transparent", `.text-transparent`},
		{"from-[var(--accent-primary)]", `.from-\[var\(--accent-primary\)\]`},
		{"md:text-8xl", `.md\:text-8xl`},
		{"rounded-2xl", `.rounded-2xl`},
		{"shadow-cyan-500/30", `.shadow-cyan-500\/30`},
		{"hover:scale-105", `.hover\:scale-105`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := classToEscapedSelector(tt.input)
			if got != tt.expected {
				t.Errorf("classToEscapedSelector(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtractClassesFromTempl(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "postcss-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a mock templ file
	templContent := `package components

templ Hero() {
	<section class="relative pt-24 pb-20 md:pt-40">
		<h1 class="text-6xl md:text-8xl font-display font-black bg-gradient-to-br bg-clip-text text-transparent">
			Hello
		</h1>
		<p class="text-xl text-[var(--text-secondary)] max-w-3xl">
			World
		</p>
	</section>
}
`
	templDir := filepath.Join(tmpDir, "components")
	if err := os.MkdirAll(templDir, 0750); err != nil {
		t.Fatalf("failed to create components dir: %v", err)
	}

	templPath := filepath.Join(templDir, "hero.templ")
	if err := os.WriteFile(templPath, []byte(templContent), 0600); err != nil {
		t.Fatalf("failed to write templ file: %v", err)
	}

	classes := extractClassesFromTempl(tmpDir, []string{"components/*.templ"})

	// Check expected classes are found
	expected := []string{
		"relative", "pt-24", "pb-20", "md:pt-40",
		"text-6xl", "md:text-8xl", "font-display", "font-black",
		"bg-gradient-to-br", "bg-clip-text", "text-transparent",
		"text-xl", "max-w-3xl",
	}
	for _, cls := range expected {
		if !classes[cls] {
			t.Errorf("expected class %q to be extracted", cls)
		}
	}

	// Check dynamic class attributes are NOT extracted
	if classes["dynamicClass"] {
		t.Errorf("should not extract classes from dynamic attribute syntax")
	}
}

func TestExtractClassesFromTemplGlobPatterns(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "postcss-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create nested templ files
	routesDir := filepath.Join(tmpDir, "routes")
	componentsDir := filepath.Join(tmpDir, "components")
	if err := os.MkdirAll(routesDir, 0750); err != nil {
		t.Fatalf("failed to create routes dir: %v", err)
	}
	if err := os.MkdirAll(componentsDir, 0750); err != nil {
		t.Fatalf("failed to create components dir: %v", err)
	}

	routeContent := `package routes
templ Home() {
	<div class="home-container bg-white">Home</div>
}
`
	compContent := `package components
templ Nav() {
	<nav class="nav-bar flex items-center">Nav</nav>
}
`
	if err := os.WriteFile(filepath.Join(routesDir, "home.templ"), []byte(routeContent), 0600); err != nil {
		t.Fatalf("failed to write route templ: %v", err)
	}
	if err := os.WriteFile(filepath.Join(componentsDir, "nav.templ"), []byte(compContent), 0600); err != nil {
		t.Fatalf("failed to write component templ: %v", err)
	}

	classes := extractClassesFromTempl(tmpDir, []string{
		"routes/*.templ",
		"components/*.templ",
	})

	for _, cls := range []string{"home-container", "bg-white", "nav-bar", "flex", "items-center"} {
		if !classes[cls] {
			t.Errorf("expected class %q to be extracted from nested dirs", cls)
		}
	}
}

func TestExtractClassesFromTemplEmptyGlobs(t *testing.T) {
	classes := extractClassesFromTempl(".", []string{})
	if len(classes) != 0 {
		t.Errorf("expected empty classes for empty globs, got %d", len(classes))
	}

	classes = extractClassesFromTempl(".", nil)
	if len(classes) != 0 {
		t.Errorf("expected empty classes for nil globs, got %d", len(classes))
	}
}

func TestSplitCSSNoCriticalClasses(t *testing.T) {
	// When criticalClasses is nil/empty, behaves like the old byte-size split.
	// Use a CSS structure with multiple depth-0 blocks so the split can happen.
	css := []byte(`@layer base{html{color-scheme:dark}}@layer components{.btn{padding:1rem}}@layer utilities{.a{color:red}.b{color:blue}.c{color:green}.d{color:yellow}.e{color:orange}.f{color:purple}.g{color:pink}}`)

	// With maxSize large enough to include everything
	critical, nonCritical := splitCSS(css, 99999, nil)
	if string(critical) != string(css) {
		t.Errorf("expected full CSS when maxSize > len(css)")
	}
	if len(nonCritical) != 0 {
		t.Errorf("expected empty non-critical when no split needed")
	}

	// With maxSize that falls after @layer base ends but before utilities starts
	critical, nonCritical = splitCSS(css, 50, nil)
	if len(critical) == 0 {
		t.Errorf("expected non-empty critical CSS")
	}
	if len(nonCritical) == 0 {
		t.Errorf("expected non-empty non-critical CSS")
	}
	// Verify the split is at a rule boundary
	if !strings.HasSuffix(string(critical), "}") {
		t.Errorf("expected critical CSS to end at a rule boundary (})")
	}
}

func TestSplitCSSWithCriticalClasses(t *testing.T) {
	// Minified CSS similar to Tailwind v4 output. Utilities must start early enough
	// that maxSize falls within it for reordering to matter.
	css := []byte(`@layer base{html{color-scheme:dark}}@layer utilities{.bg-red-500{background-color:#ef4444}.bg-blue-500{background-color:#3b82f6}.bg-gradient-to-br{background-image:linear-gradient(to bottom right,var(--tw-gradient-stops))}.bg-clip-text{-webkit-background-clip:text;background-clip:text}.text-transparent{color:transparent}.text-white{color:#fff}.p-4{padding:1rem}}`)

	// Request bg-gradient-to-br, bg-clip-text, text-transparent as critical
	criticalClasses := map[string]bool{
		"bg-gradient-to-br": true,
		"bg-clip-text":      true,
		"text-transparent":  true,
	}

	critical, nonCritical := splitCSS(css, 250, criticalClasses)

	// The matched rules should be in critical CSS
	criticalStr := string(critical)
	for _, cls := range []string{"bg-gradient-to-br", "bg-clip-text", "text-transparent"} {
		escaped := classToEscapedSelector(cls)
		if !strings.Contains(criticalStr, escaped) {
			t.Errorf("expected critical CSS to contain %s, got: %s", cls, criticalStr)
		}
	}

	// Non-critical should contain some rules
	if len(nonCritical) == 0 {
		t.Errorf("expected non-empty non-critical CSS")
	}
}

func TestSplitCSSReordering(t *testing.T) {
	// CSS with base layer early and utilities layer large enough that maxSize falls inside it.
	// The matched class "foxtrot" is alphabetically last in the utilities,
	// so without reordering it would be in non-critical.
	css := []byte(`@layer base{html{color-scheme:dark}}@layer utilities{.alpha{color:a}.bravo{color:b}.charlie{color:c}.delta{color:d}.echo{color:e}.foxtrot{color:f}.golf{color:g}.hotel{color:h}.india{color:i}.juliet{color:j}}`)

	// "foxtrot" would normally fall after the split point, but reordering should pull it forward
	criticalClasses := map[string]bool{
		"foxtrot": true,
	}

	critical, nonCritical := splitCSS(css, 60, criticalClasses)

	// foxtrot should be in the critical portion because reordering moves it first
	criticalStr := string(critical)
	if !strings.Contains(criticalStr, ".foxtrot") {
		t.Errorf("expected reordering to place .foxtrot in critical CSS, got: %s", criticalStr)
	}
	// Non-critical should still contain some rules
	if len(nonCritical) == 0 {
		t.Errorf("expected non-empty non-critical CSS")
	}
}

func TestFindUtilitiesLayer(t *testing.T) {
	css := []byte(`@layer base{html{color-scheme:dark}}@layer utilities{.bg-red{background-color:red}.text-white{color:#fff}}`)

	start, end := findUtilitiesLayer(css)
	if start == -1 {
		t.Fatal("expected to find utilities layer")
	}

	utilBlock := string(css[start:end])
	if !strings.HasPrefix(utilBlock, "@layer utilities{") {
		t.Errorf("utilities block should start with @layer utilities{")
	}
	if !strings.HasSuffix(utilBlock, "}") {
		t.Errorf("utilities block should end with }")
	}
	if !strings.Contains(utilBlock, ".bg-red") {
		t.Errorf("utilities block should contain .bg-red rule")
	}
}

func TestFindUtilitiesLayerNotFound(t *testing.T) {
	css := []byte(`@layer base{html{color-scheme:dark}}`)
	start, end := findUtilitiesLayer(css)
	if start != -1 || end != -1 {
		t.Errorf("expected -1, -1 when no utilities layer, got %d, %d", start, end)
	}
}

func TestParseRules(t *testing.T) {
	// parseRules operates on inner utilities content (without @layer wrapper)
	content := []byte(`.a{color:red}.b{color:blue}@media(min-width:640px){.c{color:green}}.d{color:yellow}`)
	rules := parseRules(content)

	if len(rules) < 3 {
		t.Errorf("expected at least 3 rules, got %d: %v", len(rules), rules)
	}

	// First rule should be .a{color:red}
	if !strings.HasPrefix(string(rules[0]), ".a{") {
		t.Errorf("first rule should start with .a{, got %s", string(rules[0]))
	}
}

func TestMatchesAnySelector(t *testing.T) {
	selectors := map[string]bool{
		`.bg-gradient-to-br`: true,
		`.bg-clip-text`:      true,
	}

	tests := []struct {
		rule     string
		expected bool
	}{
		{`.bg-gradient-to-br{background-image:linear-gradient(to bottom right)}`, true},
		{`.bg-clip-text{-webkit-background-clip:text}`, true},
		{`.text-white{color:#fff}`, false},
		{`.bg-gradient-to-br:hover{opacity:0.8}`, true},
	}

	for _, tt := range tests {
		got := matchesAnySelector([]byte(tt.rule), selectors)
		if got != tt.expected {
			t.Errorf("matchesAnySelector(%q) = %v, want %v", tt.rule, got, tt.expected)
		}
	}
}

func TestSplitCSSPreservesNonUtilities(t *testing.T) {
	// Verify that @layer base and other non-utilities content is preserved
	css := []byte(`@layer base{html{color-scheme:dark;line-height:1.5}}@layer utilities{.a{color:red}.b{color:blue}}`)

	criticalClasses := map[string]bool{"a": true}
	critical, nonCritical := splitCSS(css, 20, criticalClasses)

	// Critical should contain base layer
	if !strings.Contains(string(critical), "@layer base") {
		t.Errorf("critical CSS should preserve @layer base block")
	}

	// Both should contain some content
	if len(critical) == 0 || len(nonCritical) == 0 {
		t.Errorf("both critical and non-critical should have content")
	}
}

func TestSplitCSSEmptyCSS(t *testing.T) {
	critical, nonCritical := splitCSS([]byte(""), 1000, nil)
	if len(critical) != 0 || len(nonCritical) != 0 {
		t.Errorf("expected empty output for empty input")
	}
}

func TestSplitCSSDefaultMaxSize(t *testing.T) {
	// maxSize <= 0 should default to 14KB
	css := []byte(`@layer utilities{.a{color:red}}`)
	critical, nonCritical := splitCSS(css, 0, nil)
	if string(critical) != string(css) {
		t.Errorf("small CSS should be returned as-is with default maxSize")
	}
	if len(nonCritical) != 0 {
		t.Errorf("no split needed for small CSS")
	}
}
