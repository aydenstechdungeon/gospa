// Package component provides reactive boundary detection for GoSPA.
package component

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
)

// ReactiveBoundary represents a detected reactive boundary in a component.
type ReactiveBoundary struct {
	Name          string              `json:"name"`
	Type          BoundaryType        `json:"type"`
	LineNumber    int                 `json:"lineNumber"`
	Dependencies  []string            `json:"dependencies"`
	StateVars     []string            `json:"stateVars"`
	Props         []string            `json:"props"`
	IsIsland      bool                `json:"isIsland"`
	HydrationMode IslandHydrationMode `json:"hydrationMode,omitempty"`
	Priority      IslandPriority      `json:"priority,omitempty"`
	Metadata      map[string]any      `json:"metadata,omitempty"`
}

// BoundaryType represents the type of reactive boundary.
type BoundaryType string

const (
	// BoundaryTypeState represents a state boundary.
	BoundaryTypeState BoundaryType = "state"
	// BoundaryTypeDerived indicates an automatically derived dependency
	BoundaryTypeDerived BoundaryType = "derived"
	// BoundaryTypeEffect indicates an effect dependency
	BoundaryTypeEffect BoundaryType = "effect"
	// BoundaryTypeComponent indicates a component boundary
	BoundaryTypeComponent BoundaryType = "component"
	// BoundaryTypeEvent indicates an event handler boundary
	BoundaryTypeEvent BoundaryType = "event"
	// BoundaryTypeComputed indicates a computed property boundary
	BoundaryTypeComputed BoundaryType = "computed"
)

// Pre-compiled patterns for reactive boundary detection.
// These are compiled once at package init to avoid per-call compilation overhead.
var (
	reStatePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?s)\$state\s*\(\s*([^)]+)\s*\)`),
		regexp.MustCompile(`(?s)gospa\.State\s*\(\s*([^)]+)\s*\)`),
	}
	reDerivedPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?s)\$derived\s*\(\s*([^)]+)\s*\)`),
		regexp.MustCompile(`(?s)gospa\.Derived\s*\(\s*([^)]+)\s*\)`),
	}
	reEffectPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?s)\$effect\s*\(\s*func\s*\(`),
		regexp.MustCompile(`(?s)gospa\.Effect\s*\(`),
	}
	reEventPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?s)on[a-z:]+\s*=\s*\{`),
		regexp.MustCompile(`(?s)@\w+\s*=\s*\{`),
	}
	reComponentPatterns = []*regexp.Regexp{
		regexp.MustCompile(`templ\.Component`),
		regexp.MustCompile(`templ\.ComponentFunc`),
		regexp.MustCompile(`func\s*\(\w+\)\s*Render\s*\(`),
	}
	reIslandPatterns = []*regexp.Regexp{
		regexp.MustCompile(`gospa\.Island\s*\(\s*"([^"]+)"`),
		regexp.MustCompile(`data-gospa-island`),
	}
)

// ReactiveDetector detects reactive boundaries in templ components.
type ReactiveDetector struct {
	statePatterns     []*regexp.Regexp
	derivedPatterns   []*regexp.Regexp
	effectPatterns    []*regexp.Regexp
	eventPatterns     []*regexp.Regexp
	componentPatterns []*regexp.Regexp
	islandPatterns    []*regexp.Regexp
}

// NewReactiveDetector creates a new reactive boundary detector.
// All patterns reference pre-compiled package-level regexps so there is
// no per-instance compilation cost.
func NewReactiveDetector() *ReactiveDetector {
	return &ReactiveDetector{
		statePatterns:     reStatePatterns,
		derivedPatterns:   reDerivedPatterns,
		effectPatterns:    reEffectPatterns,
		eventPatterns:     reEventPatterns,
		componentPatterns: reComponentPatterns,
		islandPatterns:    reIslandPatterns,
	}
}

// DetectionResult contains the results of reactive boundary detection.
type DetectionResult struct {
	Boundaries    []ReactiveBoundary `json:"boundaries"`
	StateCount    int                `json:"stateCount"`
	DerivedCount  int                `json:"derivedCount"`
	EffectCount   int                `json:"effectCount"`
	IslandCount   int                `json:"islandCount"`
	ComponentName string             `json:"componentName"`
	FilePath      string             `json:"filePath"`
}

// Detect analyzes source code and returns detected reactive boundaries.
func (rd *ReactiveDetector) Detect(source, filePath string) *DetectionResult {
	newlineIndices := cGetNewlineIndices(source)
	result := &DetectionResult{
		Boundaries: make([]ReactiveBoundary, 0),
		FilePath:   filePath,
	}

	// Extract script if this is an SFC
	script := source
	scriptStartOffset := 0
	if strings.Contains(source, "<script") {
		// Simple extraction for detection purposes
		startIdx := strings.Index(source, "<script")
		if openTagEnd := strings.Index(source[startIdx:], ">"); openTagEnd != -1 {
			startIdx += openTagEnd + 1
			scriptStartOffset = startIdx
			if endIdx := strings.Index(source[startIdx:], "</script>"); endIdx != -1 {
				script = source[startIdx : startIdx+endIdx]
			} else {
				script = source[startIdx:]
			}
		}
	}

	// Parse the Go script using AST
	fset := token.NewFileSet()
	// SFC scripts are often snippets, so we wrap them in a pseudo-package/function
	// to ensure the parser can handle top-level assignments and statements.
	prefix := "package main\nfunc _gospa_wrapper() {\n"
	wrappedSource := prefix + script + "\n}"
	
	f, err := parser.ParseFile(fset, "", wrappedSource, 0)
	if err != nil {
		// Fallback to simpler regex detection if AST parsing fails 
		// (e.g. invalid syntax during typing in dev mode)
		rd.detectStateBoundariesLegacy(source, newlineIndices, result)
		rd.detectDerivedBoundariesLegacy(source, newlineIndices, result)
		rd.detectEffectBoundariesLegacy(source, newlineIndices, result)
	} else {
		rd.detectBoundariesAST(f, scriptStartOffset - len(prefix), newlineIndices, result)
	}

	// Detect event handlers (HTML attributes, still regex/string based)
	rd.detectEventBoundaries(source, newlineIndices, result)

	// Detect islands (meta-tags)
	rd.detectIslands(source, newlineIndices, result)

	// Update counts
	for _, b := range result.Boundaries {
		switch b.Type {
		case BoundaryTypeState:
			result.StateCount++
		case BoundaryTypeDerived:
			result.DerivedCount++
		case BoundaryTypeEffect:
			result.EffectCount++
		case BoundaryTypeComponent:
			if b.IsIsland {
				result.IslandCount++
			}
		}
	}

	return result
}


func (rd *ReactiveDetector) detectBoundariesAST(f *ast.File, offsetAdjustment int, newlineIndices []int, result *DetectionResult) {
	// Track parents using a stack
	var stack []ast.Node
	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		stack = append(stack, n)

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		ident, ok := call.Fun.(*ast.Ident)
		if !ok {
			return true
		}

		var bType BoundaryType
		switch ident.Name {
		case "$state":
			bType = BoundaryTypeState
		case "$derived":
			bType = BoundaryTypeDerived
		case "$effect":
			bType = BoundaryTypeEffect
		default:
			return true
		}

		realPos := int(call.Pos()) + offsetAdjustment
		lineNum := cGetLineNumber(newlineIndices, realPos)

		boundary := ReactiveBoundary{
			Type:       bType,
			LineNumber: lineNum,
		}

		// Improved variable name detection using parent stack
		if len(stack) > 1 {
			parent := stack[len(stack)-2]
			switch p := parent.(type) {
			case *ast.AssignStmt:
				if len(p.Lhs) > 0 {
					if id, ok := p.Lhs[0].(*ast.Ident); ok {
						boundary.Name = id.Name
					}
				}
			case *ast.ValueSpec:
				if len(p.Names) > 0 {
					boundary.Name = p.Names[0].Name
				}
			case *ast.ExprStmt:
				// If part of an expression (e.g. nested), check one level higher
				if len(stack) > 2 {
					grandParent := stack[len(stack)-3]
					if spec, ok := grandParent.(*ast.ValueSpec); ok && len(spec.Names) > 0 {
						boundary.Name = spec.Names[0].Name
					}
				}
			}
		}

		if boundary.Name == "" {
			boundary.Name = fmt.Sprintf("%s_%d", strings.TrimPrefix(ident.Name, "$"), lineNum)
		}

		if bType == BoundaryTypeDerived || bType == BoundaryTypeEffect {
			boundary.Dependencies = rd.extractDependenciesAST(call)
		}

		if bType == BoundaryTypeState || bType == BoundaryTypeDerived {
			boundary.StateVars = []string{boundary.Name}
		}

		result.Boundaries = append(result.Boundaries, boundary)
		return true
	})
}

func (rd *ReactiveDetector) extractDependenciesAST(call *ast.CallExpr) []string {
	deps := make(map[string]bool)
	ast.Inspect(call, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			// Skip the rune itself
			if strings.HasPrefix(id.Name, "$") {
				return true
			}
			// Filter out keywords/built-ins
			if !isKeyword(id.Name) {
				deps[id.Name] = true
			}
		}
		return true
	})
	
	res := make([]string, 0, len(deps))
	for k := range deps {
		res = append(res, k)
	}
	return res
}

func isKeyword(s string) bool {
	keywords := map[string]bool{
		"func": true, "return": true, "if": true, "else": true,
		"for": true, "range": true, "true": true, "false": true,
		"nil": true, "string": true, "int": true, "bool": true,
		"any": true, "var": true, "type": true, "struct": true,
	}
	return keywords[s]
}

// detectStateBoundariesLegacy detects state declarations using regex as a fallback.
func (rd *ReactiveDetector) detectStateBoundariesLegacy(source string, newlineIndices []int, result *DetectionResult) {
	for _, pattern := range rd.statePatterns {
		matches := pattern.FindAllStringSubmatchIndex(source, -1)
		for _, match := range matches {
			lineNum := cGetLineNumber(newlineIndices, match[0])

			// Extract variable name from the context before the match
			lookback := match[0]
			if lookback > 512 {
				lookback = 512
			}
			context := source[match[0]-lookback : match[0]]
			varName := rd.extractVariableName(context)

			boundary := ReactiveBoundary{
				Name:       varName,
				Type:       BoundaryTypeState,
				LineNumber: lineNum,
				StateVars:  []string{varName},
			}
			if len(match) >= 4 {
				boundary.Dependencies = rd.extractDependencies(source[match[2]:match[3]])
			}
			result.Boundaries = append(result.Boundaries, boundary)
		}
	}
}

// detectDerivedBoundariesLegacy detects derived/computed values using regex as a fallback.
func (rd *ReactiveDetector) detectDerivedBoundariesLegacy(source string, newlineIndices []int, result *DetectionResult) {
	for _, pattern := range rd.derivedPatterns {
		matches := pattern.FindAllStringSubmatchIndex(source, -1)
		for _, match := range matches {
			lineNum := cGetLineNumber(newlineIndices, match[0])

			// Extract variable name from the context before the match
			lookback := match[0]
			if lookback > 512 {
				lookback = 512
			}
			context := source[match[0]-lookback : match[0]]
			varName := rd.extractVariableName(context)

			boundary := ReactiveBoundary{
				Name:       varName,
				Type:       BoundaryTypeDerived,
				LineNumber: lineNum,
				StateVars:  []string{varName},
			}
			if len(match) >= 4 {
				boundary.Dependencies = rd.extractDependencies(source[match[2]:match[3]])
			}
			result.Boundaries = append(result.Boundaries, boundary)
		}
	}
}

// detectEffectBoundariesLegacy detects effect declarations using regex as a fallback.
func (rd *ReactiveDetector) detectEffectBoundariesLegacy(source string, newlineIndices []int, result *DetectionResult) {
	for _, pattern := range rd.effectPatterns {
		matches := pattern.FindAllStringIndex(source, -1)
		for _, match := range matches {
			lineNum := cGetLineNumber(newlineIndices, match[0])
			boundary := ReactiveBoundary{
				Name:       fmt.Sprintf("effect_%d", lineNum),
				Type:       BoundaryTypeEffect,
				LineNumber: lineNum,
			}
			result.Boundaries = append(result.Boundaries, boundary)
		}
	}
}

// detectEventBoundaries detects event handlers.
func (rd *ReactiveDetector) detectEventBoundaries(source string, newlineIndices []int, result *DetectionResult) {
	for _, pattern := range rd.eventPatterns {
		matches := pattern.FindAllStringIndex(source, -1)
		for _, match := range matches {
			lineNum := cGetLineNumber(newlineIndices, match[0])
			boundary := ReactiveBoundary{
				Name:       fmt.Sprintf("event_%d", lineNum),
				Type:       BoundaryTypeEvent,
				LineNumber: lineNum,
			}
			result.Boundaries = append(result.Boundaries, boundary)
		}
	}
}

// detectIslands detects island declarations.
func (rd *ReactiveDetector) detectIslands(source string, newlineIndices []int, result *DetectionResult) {
	for _, pattern := range rd.islandPatterns {
		matches := pattern.FindAllStringSubmatchIndex(source, -1)
		for _, match := range matches {
			lineNum := cGetLineNumber(newlineIndices, match[0])
			name := "island"
			if len(match) >= 4 {
				name = source[match[2]:match[3]]
			}
			boundary := ReactiveBoundary{
				Name:          name,
				Type:          BoundaryTypeComponent,
				LineNumber:    lineNum,
				IsIsland:      true,
				HydrationMode: HydrationVisible,
				Priority:      PriorityNormal,
			}
			result.Boundaries = append(result.Boundaries, boundary)
		}
	}
}

// assignPattern is compiled once at package level to avoid per-call overhead.
var assignPattern = regexp.MustCompile(`(\w+)\s*:?=`)

// extractVariableName extracts the variable name from a context.
func (rd *ReactiveDetector) extractVariableName(context string) string {
	// Simple assignment patterns: varName := or varName =
	// Search from right to left to find the nearest/correct variable name.
	lines := strings.Split(context, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		matches := assignPattern.FindStringSubmatch(line)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return "unknown"
}

func cGetNewlineIndices(s string) []int {
	var indices []int
	for i, ch := range s {
		if ch == '\n' {
			indices = append(indices, i)
		}
	}
	return indices
}

func cGetLineNumber(newlineIndices []int, offset int) int {
	if len(newlineIndices) == 0 {
		return 1
	}
	// Binary search
	low, high := 0, len(newlineIndices)-1
	line := 0
	for low <= high {
		mid := (low + high) / 2
		if newlineIndices[mid] < offset {
			line = mid + 1
			low = mid + 1
		} else {
			high = mid - 1
		}
	}
	return line + 1
}

// extractDependencies extracts dependencies from an expression.
func (rd *ReactiveDetector) extractDependencies(expr string) []string {
	// Simple extraction: find variable references
	depPattern := regexp.MustCompile(`\b([a-zA-Z_]\w*)\b`)
	matches := depPattern.FindAllString(expr, -1)

	// Filter out keywords and built-ins
	keywords := map[string]bool{
		"func": true, "return": true, "if": true, "else": true,
		"for": true, "range": true, "true": true, "false": true,
		"nil": true, "string": true, "int": true, "bool": true,
	}

	deps := make([]string, 0)
	seen := make(map[string]bool)
	for _, m := range matches {
		if !keywords[m] && !seen[m] {
			deps = append(deps, m)
			seen[m] = true
		}
	}
	return deps
}

// AnalyzeComponent analyzes a component file for reactive boundaries.
func (rd *ReactiveDetector) AnalyzeComponent(source, filePath, componentName string) *ComponentAnalysis {
	result := rd.Detect(source, filePath)
	result.ComponentName = componentName

	analysis := &ComponentAnalysis{
		ComponentName: componentName,
		FilePath:      filePath,
		Boundaries:    result.Boundaries,
		IsReactive:    len(result.Boundaries) > 0,
		ShouldIsland:  rd.shouldBeIsland(result),
	}

	// Determine optimal hydration strategy
	analysis.HydrationStrategy = rd.determineHydrationStrategy(result)

	return analysis
}

// ComponentAnalysis contains detailed analysis of a component.
//
//nolint:revive // changing name would break API
type ComponentAnalysis struct {
	ComponentName     string             `json:"componentName"`
	FilePath          string             `json:"filePath"`
	Boundaries        []ReactiveBoundary `json:"boundaries"`
	IsReactive        bool               `json:"isReactive"`
	ShouldIsland      bool               `json:"shouldIsland"`
	HydrationStrategy HydrationStrategy  `json:"hydrationStrategy"`
}

// HydrationStrategy represents the recommended hydration strategy.
type HydrationStrategy struct {
	Mode         IslandHydrationMode `json:"mode"`
	Priority     IslandPriority      `json:"priority"`
	Reason       string              `json:"reason"`
	Dependencies []string            `json:"dependencies"`
	CriticalPath bool                `json:"criticalPath"`
}

// shouldBeIsland determines if a component should be an island.
func (rd *ReactiveDetector) shouldBeIsland(result *DetectionResult) bool {
	// Component should be an island if it has:
	// - State that changes
	// - Effects that run
	// - Event handlers
	// - Interactive elements
	return result.StateCount > 0 || result.EffectCount > 0 || result.IslandCount > 0
}

// determineHydrationStrategy determines the optimal hydration strategy.
func (rd *ReactiveDetector) determineHydrationStrategy(result *DetectionResult) HydrationStrategy {
	strategy := HydrationStrategy{
		Mode:     HydrationImmediate,
		Priority: PriorityNormal,
	}

	// Check for critical path indicators
	if result.EffectCount > 0 && result.StateCount > 0 {
		strategy.CriticalPath = true
		strategy.Priority = PriorityHigh
		strategy.Mode = HydrationImmediate
		strategy.Reason = "Component has effects and state - hydrate immediately"
		return strategy
	}

	// Check for interactive elements
	if hasEventHandlers(result.Boundaries) {
		strategy.Mode = HydrationInteraction
		strategy.Reason = "Component has event handlers - hydrate on interaction"
		return strategy
	}

	// Check for derived-only components
	if result.DerivedCount > 0 && result.StateCount == 0 {
		strategy.Mode = HydrationIdle
		strategy.Priority = PriorityLow
		strategy.Reason = "Component only has derived values - hydrate when idle"
		return strategy
	}

	// Default: visible hydration
	strategy.Mode = HydrationVisible
	strategy.Reason = "Standard hydration when visible"
	return strategy
}

// hasEventHandlers checks if boundaries contain event handlers.
func hasEventHandlers(boundaries []ReactiveBoundary) bool {
	for _, b := range boundaries {
		if b.Type == BoundaryTypeEvent {
			return true
		}
	}
	return false
}

// Global detector instance
var globalDetector = NewReactiveDetector()

// DetectReactiveBoundaries is a convenience function using the global detector.
func DetectReactiveBoundaries(source, filePath string) *DetectionResult {
	return globalDetector.Detect(source, filePath)
}

// AnalyzeComponentFile analyzes a component file using the global detector.
func AnalyzeComponentFile(source, filePath, componentName string) *ComponentAnalysis {
	return globalDetector.AnalyzeComponent(source, filePath, componentName)
}
