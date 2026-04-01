// Package component provides reactive boundary detection for GoSPA.
package component

import (
	"fmt"
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
	result := &DetectionResult{
		Boundaries: make([]ReactiveBoundary, 0),
		FilePath:   filePath,
	}

	// Detect state boundaries
	rd.detectStateBoundaries(source, result)

	// Detect derived boundaries
	rd.detectDerivedBoundaries(source, result)

	// Detect effect boundaries
	rd.detectEffectBoundaries(source, result)

	// Detect event handlers
	rd.detectEventBoundaries(source, result)

	// Detect islands
	rd.detectIslands(source, result)

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

// detectStateBoundaries detects state declarations.
func (rd *ReactiveDetector) detectStateBoundaries(source string, result *DetectionResult) {
	for _, pattern := range rd.statePatterns {
		matches := pattern.FindAllStringSubmatchIndex(source, -1)
		for _, match := range matches {
			lineBefore := source[:match[0]]
			lineNum := strings.Count(lineBefore, "\n") + 1

			// Extract variable name from the context before the match
			lastLineIdx := strings.LastIndex(lineBefore, "\n")
			var context string
			if lastLineIdx == -1 {
				context = lineBefore
			} else {
				context = lineBefore[lastLineIdx:]
			}

			varName := rd.extractVariableName(context)
			boundary := ReactiveBoundary{
				Name:       varName,
				Type:       BoundaryTypeState,
				LineNumber: lineNum,
				StateVars:  []string{varName},
			}
			if len(match) >= 4 {
				boundary.Dependencies = []string{source[match[2]:match[3]]}
			}
			result.Boundaries = append(result.Boundaries, boundary)
		}
	}
}

// detectDerivedBoundaries detects derived/computed values.
func (rd *ReactiveDetector) detectDerivedBoundaries(source string, result *DetectionResult) {
	for _, pattern := range rd.derivedPatterns {
		matches := pattern.FindAllStringSubmatchIndex(source, -1)
		for _, match := range matches {
			lineBefore := source[:match[0]]
			lineNum := strings.Count(lineBefore, "\n") + 1

			// Extract variable name from the context before the match
			lastLineIdx := strings.LastIndex(lineBefore, "\n")
			var context string
			if lastLineIdx == -1 {
				context = lineBefore
			} else {
				context = lineBefore[lastLineIdx:]
			}

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

// detectEffectBoundaries detects effect declarations.
func (rd *ReactiveDetector) detectEffectBoundaries(source string, result *DetectionResult) {
	for _, pattern := range rd.effectPatterns {
		matches := pattern.FindAllStringIndex(source, -1)
		for _, match := range matches {
			lineNum := strings.Count(source[:match[0]], "\n") + 1
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
func (rd *ReactiveDetector) detectEventBoundaries(source string, result *DetectionResult) {
	for _, pattern := range rd.eventPatterns {
		matches := pattern.FindAllStringIndex(source, -1)
		for _, match := range matches {
			lineNum := strings.Count(source[:match[0]], "\n") + 1
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
func (rd *ReactiveDetector) detectIslands(source string, result *DetectionResult) {
	for _, pattern := range rd.islandPatterns {
		matches := pattern.FindAllStringSubmatchIndex(source, -1)
		for _, match := range matches {
			lineNum := strings.Count(source[:match[0]], "\n") + 1
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

// extractVariableName extracts the variable name from a line.
func (rd *ReactiveDetector) extractVariableName(line string) string {
	// Look for variable assignment pattern: varName := or varName =
	matches := assignPattern.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}
	return "unknown"
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
