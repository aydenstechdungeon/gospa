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
	BoundaryTypeState     BoundaryType = "state"
	BoundaryTypeDerived   BoundaryType = "derived"
	BoundaryTypeEffect    BoundaryType = "effect"
	BoundaryTypeComponent BoundaryType = "component"
	BoundaryTypeEvent     BoundaryType = "event"
	BoundaryTypeComputed  BoundaryType = "computed"
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
func NewReactiveDetector() *ReactiveDetector {
	return &ReactiveDetector{
		statePatterns: []*regexp.Regexp{
			regexp.MustCompile(`\$state\s*\(\s*([^)]+)\s*\)`),
			regexp.MustCompile(`gospa\.State\s*\(\s*([^)]+)\s*\)`),
			regexp.MustCompile(`NewState\s*\(\s*([^)]+)\s*\)`),
		},
		derivedPatterns: []*regexp.Regexp{
			regexp.MustCompile(`\$derived\s*\(\s*([^)]+)\s*\)`),
			regexp.MustCompile(`gospa\.Derived\s*\(\s*([^)]+)\s*\)`),
			regexp.MustCompile(`NewDerived\s*\(\s*([^)]+)\s*\)`),
		},
		effectPatterns: []*regexp.Regexp{
			regexp.MustCompile(`\$effect\s*\(\s*func\s*\(\)`),
			regexp.MustCompile(`gospa\.Effect\s*\(`),
			regexp.MustCompile(`NewEffect\s*\(`),
		},
		eventPatterns: []*regexp.Regexp{
			regexp.MustCompile(`onclick\s*=\s*`),
			regexp.MustCompile(`on:click\s*=\s*`),
			regexp.MustCompile(`gospa\.On\s*\(\s*"([^"]+)"`),
			regexp.MustCompile(`@click\s*=`),
		},
		componentPatterns: []*regexp.Regexp{
			regexp.MustCompile(`templ\.Component`),
			regexp.MustCompile(`templ\.ComponentFunc`),
			regexp.MustCompile(`func\s*\(\w+\)\s*Render\s*\(`),
		},
		islandPatterns: []*regexp.Regexp{
			regexp.MustCompile(`gospa\.Island\s*\(\s*"([^"]+)"`),
			regexp.MustCompile(`@gospa:island\s+name="([^"]+)"`),
			regexp.MustCompile(`data-gospa-island`),
		},
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

	lines := strings.Split(source, "\n")

	for lineNum, line := range lines {
		// Detect state boundaries
		rd.detectStateBoundaries(line, lineNum+1, result)

		// Detect derived boundaries
		rd.detectDerivedBoundaries(line, lineNum+1, result)

		// Detect effect boundaries
		rd.detectEffectBoundaries(line, lineNum+1, result)

		// Detect event handlers
		rd.detectEventBoundaries(line, lineNum+1, result)

		// Detect islands
		rd.detectIslands(line, lineNum+1, result)
	}

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
func (rd *ReactiveDetector) detectStateBoundaries(line string, lineNum int, result *DetectionResult) {
	for _, pattern := range rd.statePatterns {
		matches := pattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			varName := rd.extractVariableName(line)
			boundary := ReactiveBoundary{
				Name:       varName,
				Type:       BoundaryTypeState,
				LineNumber: lineNum,
				StateVars:  []string{varName},
			}
			if len(match) > 1 {
				boundary.Dependencies = []string{match[1]}
			}
			result.Boundaries = append(result.Boundaries, boundary)
		}
	}
}

// detectDerivedBoundaries detects derived/computed values.
func (rd *ReactiveDetector) detectDerivedBoundaries(line string, lineNum int, result *DetectionResult) {
	for _, pattern := range rd.derivedPatterns {
		matches := pattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			varName := rd.extractVariableName(line)
			boundary := ReactiveBoundary{
				Name:       varName,
				Type:       BoundaryTypeDerived,
				LineNumber: lineNum,
				StateVars:  []string{varName},
			}
			if len(match) > 1 {
				boundary.Dependencies = rd.extractDependencies(match[1])
			}
			result.Boundaries = append(result.Boundaries, boundary)
		}
	}
}

// detectEffectBoundaries detects effect declarations.
func (rd *ReactiveDetector) detectEffectBoundaries(line string, lineNum int, result *DetectionResult) {
	for _, pattern := range rd.effectPatterns {
		if pattern.MatchString(line) {
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
func (rd *ReactiveDetector) detectEventBoundaries(line string, lineNum int, result *DetectionResult) {
	for _, pattern := range rd.eventPatterns {
		if pattern.MatchString(line) {
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
func (rd *ReactiveDetector) detectIslands(line string, lineNum int, result *DetectionResult) {
	for _, pattern := range rd.islandPatterns {
		matches := pattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			name := "island"
			if len(match) > 1 {
				name = match[1]
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

// extractVariableName extracts the variable name from a line.
func (rd *ReactiveDetector) extractVariableName(line string) string {
	// Look for variable assignment pattern: varName := or varName =
	assignPattern := regexp.MustCompile(`(\w+)\s*:?=`)
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
