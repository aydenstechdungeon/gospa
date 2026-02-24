// Package state provides build-time state pruning for GoSPA applications.
// This reduces bundle size by eliminating unused state at build time.
package state

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PruningConfig configures state pruning behavior.
type PruningConfig struct {
	// RootDir is the project root directory.
	RootDir string `json:"rootDir"`
	// OutputDir is where pruned state files are written.
	OutputDir string `json:"outputDir"`
	// ExcludePatterns are regex patterns for files to exclude.
	ExcludePatterns []string `json:"excludePatterns"`
	// IncludePatterns are regex patterns for files to include.
	IncludePatterns []string `json:"includePatterns"`
	// KeepUnused keeps state even if not directly referenced.
	KeepUnused bool `json:"keepUnused"`
	// Aggressive enables more aggressive pruning.
	Aggressive bool `json:"aggressive"`
	// ReportFile is where to write the pruning report.
	ReportFile string `json:"reportFile"`
}

// DefaultPruningConfig returns sensible defaults.
func DefaultPruningConfig() PruningConfig {
	return PruningConfig{
		ExcludePatterns: []string{
			`_test\.go$`,
			`_templ\.go$`,
			`generated_.*\.go$`,
		},
		IncludePatterns: []string{
			`\.go$`,
		},
		KeepUnused: false,
		Aggressive: false,
	}
}

// StateUsage represents how a state variable is used.
type StateUsage struct {
	Name       string   `json:"name"`
	File       string   `json:"file"`
	Line       int      `json:"line"`
	Type       string   `json:"type"`
	References []string `json:"references,omitempty"`
	IsExported bool     `json:"isExported"`
	IsUsed     bool     `json:"isUsed"`
	CanPrune   bool     `json:"canPrune"`
}

// PruningReport contains the results of state analysis.
type PruningReport struct {
	TotalStateVars   int                   `json:"totalStateVars"`
	UsedStateVars    int                   `json:"usedStateVars"`
	PrunedStateVars  int                   `json:"prunedStateVars"`
	EstimatedSavings int                   `json:"estimatedSavings"`
	StateUsage       map[string]StateUsage `json:"stateUsage"`
	PrunedFiles      []string              `json:"prunedFiles"`
	Errors           []string              `json:"errors,omitempty"`
}

// StatePruner analyzes and prunes unused state.
type StatePruner struct {
	config    PruningConfig
	fset      *token.FileSet
	stateVars map[string]StateUsage
	usedVars  map[string]bool
	report    *PruningReport
}

// NewStatePruner creates a new state pruner.
func NewStatePruner(config PruningConfig) *StatePruner {
	return &StatePruner{
		config:    config,
		fset:      token.NewFileSet(),
		stateVars: make(map[string]StateUsage),
		usedVars:  make(map[string]bool),
		report: &PruningReport{
			StateUsage: make(map[string]StateUsage),
		},
	}
}

// Analyze scans the codebase for state usage.
func (sp *StatePruner) Analyze() (*PruningReport, error) {
	// Walk the directory tree
	err := filepath.Walk(sp.config.RootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if file should be processed
		if !sp.shouldProcessFile(path) {
			return nil
		}

		// Analyze the file
		return sp.analyzeFile(path)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to analyze codebase: %w", err)
	}

	// Mark used variables
	sp.markUsedVariables()

	// Calculate statistics
	sp.calculateStatistics()

	return sp.report, nil
}

// shouldProcessFile checks if a file should be processed.
func (sp *StatePruner) shouldProcessFile(path string) bool {
	// Check exclude patterns
	for _, pattern := range sp.config.ExcludePatterns {
		matched, err := regexp.MatchString(pattern, filepath.Base(path))
		if err == nil && matched {
			return false
		}
	}

	// Check include patterns
	for _, pattern := range sp.config.IncludePatterns {
		matched, err := regexp.MatchString(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}
	}

	return false
}

// analyzeFile analyzes a single Go file for state usage.
func (sp *StatePruner) analyzeFile(path string) error {
	// Parse the file
	node, err := parser.ParseFile(sp.fset, path, nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		sp.report.Errors = append(sp.report.Errors, fmt.Sprintf("failed to parse %s: %v", path, err))
		return nil
	}

	// Find state declarations
	ast.Inspect(node, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.GenDecl:
			sp.processGenDecl(decl, path)
		case *ast.Ident:
			sp.processIdent(decl, path)
		case *ast.SelectorExpr:
			sp.processSelectorExpr(decl, path)
		}
		return true
	})

	return nil
}

// processGenDecl processes a general declaration.
func (sp *StatePruner) processGenDecl(decl *ast.GenDecl, path string) {
	// Look for state variable declarations
	for _, spec := range decl.Specs {
		vspec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		for i, name := range vspec.Names {
			// Check if this is a state variable (has $ prefix in comments or specific patterns)
			isState := sp.isStateVariable(name.Name, decl.Doc)

			if isState {
				varType := ""
				if i < len(vspec.Values) {
					varType = sp.exprToString(vspec.Values[i])
				}

				usage := StateUsage{
					Name:       name.Name,
					File:       path,
					Line:       sp.fset.Position(name.Pos()).Line,
					Type:       varType,
					IsExported: ast.IsExported(name.Name),
					IsUsed:     false,
					CanPrune:   !ast.IsExported(name.Name),
				}

				sp.stateVars[name.Name] = usage
			}
		}
	}
}

// processIdent processes an identifier reference.
func (sp *StatePruner) processIdent(ident *ast.Ident, path string) {
	// Check if this references a known state variable
	if _, exists := sp.stateVars[ident.Name]; exists {
		sp.usedVars[ident.Name] = true
	}
}

// processSelectorExpr processes a selector expression.
func (sp *StatePruner) processSelectorExpr(sel *ast.SelectorExpr, path string) {
	// Check for state access patterns like state.Var
	if x, ok := sel.X.(*ast.Ident); ok {
		key := x.Name + "." + sel.Sel.Name
		if _, exists := sp.stateVars[key]; exists {
			sp.usedVars[key] = true
		}
	}
}

// isStateVariable checks if a variable is a state variable.
func (sp *StatePruner) isStateVariable(name string, doc *ast.CommentGroup) bool {
	// Check for state-related naming patterns
	statePatterns := []string{
		"State",
		"state",
		"Store",
		"store",
		"Rune",
		"rune",
	}

	for _, pattern := range statePatterns {
		if strings.Contains(name, pattern) {
			return true
		}
	}

	// Check for gospa state annotations in comments
	if doc != nil {
		for _, comment := range doc.List {
			if strings.Contains(comment.Text, "@gospa:state") ||
				strings.Contains(comment.Text, "@state") {
				return true
			}
		}
	}

	return false
}

// exprToString converts an AST expression to a string.
func (sp *StatePruner) exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.BasicLit:
		return e.Value
	case *ast.CallExpr:
		return sp.exprToString(e.Fun) + "(...)"
	case *ast.CompositeLit:
		return sp.exprToString(e.Type) + "{...}"
	default:
		return ""
	}
}

// markUsedVariables marks variables as used based on analysis.
func (sp *StatePruner) markUsedVariables() {
	for name := range sp.usedVars {
		if usage, exists := sp.stateVars[name]; exists {
			usage.IsUsed = true
			sp.stateVars[name] = usage
		}
	}

	// Copy to report
	for name, usage := range sp.stateVars {
		sp.report.StateUsage[name] = usage
	}
}

// calculateStatistics calculates pruning statistics.
func (sp *StatePruner) calculateStatistics() {
	sp.report.TotalStateVars = len(sp.stateVars)

	for _, usage := range sp.stateVars {
		if usage.IsUsed {
			sp.report.UsedStateVars++
		} else if usage.CanPrune {
			sp.report.PrunedStateVars++
			sp.report.EstimatedSavings += 100 // Rough estimate in bytes
		}
	}
}

// Prune removes unused state from the codebase.
func (sp *StatePruner) Prune() (*PruningReport, error) {
	// First analyze
	report, err := sp.Analyze()
	if err != nil {
		return nil, err
	}

	if sp.config.KeepUnused {
		// Don't actually prune, just report
		return report, nil
	}

	// Group state by file
	fileState := make(map[string][]StateUsage)
	for _, usage := range sp.stateVars {
		if !usage.IsUsed && usage.CanPrune {
			fileState[usage.File] = append(fileState[usage.File], usage)
		}
	}

	// Process each file
	for file, usages := range fileState {
		if err := sp.pruneFile(file, usages); err != nil {
			sp.report.Errors = append(sp.report.Errors, fmt.Sprintf("failed to prune %s: %v", file, err))
			continue
		}
		sp.report.PrunedFiles = append(sp.report.PrunedFiles, file)
	}

	return sp.report, nil
}

// pruneFile removes unused state from a single file.
func (sp *StatePruner) pruneFile(path string, usages []StateUsage) error {
	// Read the file
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	// Remove lines with unused state (in reverse order to preserve line numbers)
	for i := len(usages) - 1; i >= 0; i-- {
		usage := usages[i]
		if usage.Line > 0 && usage.Line <= len(lines) {
			// Comment out or remove the line
			lines[usage.Line-1] = "// PRUNED: " + lines[usage.Line-1]
		}
	}

	// Write the modified file
	outputPath := path
	if sp.config.OutputDir != "" {
		relPath, err := filepath.Rel(sp.config.RootDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}
		outputPath = filepath.Join(sp.config.OutputDir, relPath)
	}

	// Ensure output directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	output := strings.Join(lines, "\n")
	if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// WriteReport writes the pruning report to a file.
func (sp *StatePruner) WriteReport(path string) error {
	data, err := json.MarshalIndent(sp.report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// GetReport returns the current pruning report.
func (sp *StatePruner) GetReport() *PruningReport {
	return sp.report
}

// PruneState is a convenience function for state pruning.
func PruneState(config PruningConfig) (*PruningReport, error) {
	pruner := NewStatePruner(config)
	return pruner.Prune()
}

// AnalyzeState is a convenience function for state analysis.
func AnalyzeState(config PruningConfig) (*PruningReport, error) {
	pruner := NewStatePruner(config)
	return pruner.Analyze()
}

// StateTree represents a hierarchical state structure for analysis.
type StateTree struct {
	Name     string                `json:"name"`
	Type     string                `json:"type"`
	Size     int                   `json:"size"`
	Children map[string]*StateTree `json:"children,omitempty"`
	Used     bool                  `json:"used"`
	Path     string                `json:"path"`
}

// BuildStateTree builds a hierarchical representation of state.
func BuildStateTree(state map[string]any) *StateTree {
	root := &StateTree{
		Name:     "root",
		Type:     "object",
		Children: make(map[string]*StateTree),
	}

	for key, value := range state {
		root.Children[key] = buildStateTreeRecursive(key, value, "")
	}

	return root
}

func buildStateTreeRecursive(name string, value any, path string) *StateTree {
	node := &StateTree{
		Name: name,
		Path: path + "." + name,
	}

	switch v := value.(type) {
	case map[string]any:
		node.Type = "object"
		node.Children = make(map[string]*StateTree)
		for key, val := range v {
			node.Children[key] = buildStateTreeRecursive(key, val, node.Path)
		}
	case []any:
		node.Type = "array"
		node.Size = len(v)
	case string:
		node.Type = "string"
		node.Size = len(v)
	case int, int64, float64:
		node.Type = "number"
	case bool:
		node.Type = "boolean"
	default:
		node.Type = "unknown"
	}

	return node
}

// PruneStateTree removes unused branches from a state tree.
func PruneStateTree(tree *StateTree, usedPaths map[string]bool) *StateTree {
	if tree.Children == nil {
		return tree
	}

	pruned := &StateTree{
		Name:     tree.Name,
		Type:     tree.Type,
		Size:     tree.Size,
		Path:     tree.Path,
		Used:     usedPaths[tree.Path],
		Children: make(map[string]*StateTree),
	}

	for key, child := range tree.Children {
		prunedChild := PruneStateTree(child, usedPaths)
		if prunedChild.Used || hasUsedDescendants(prunedChild, usedPaths) {
			pruned.Children[key] = prunedChild
		}
	}

	return pruned
}

func hasUsedDescendants(tree *StateTree, usedPaths map[string]bool) bool {
	if tree.Children == nil {
		return false
	}

	for _, child := range tree.Children {
		if usedPaths[child.Path] || hasUsedDescendants(child, usedPaths) {
			return true
		}
	}

	return false
}

// CalculateTreeSize calculates the estimated size of a state tree.
func CalculateTreeSize(tree *StateTree) int {
	size := tree.Size

	for _, child := range tree.Children {
		size += CalculateTreeSize(child)
	}

	return size
}

// SerializePrunedState serializes only the used portions of state.
func SerializePrunedState(state map[string]any, usedPaths map[string]bool) map[string]any {
	result := make(map[string]any)

	for key, value := range state {
		path := "." + key
		if usedPaths[path] {
			result[key] = value
		} else if nested, ok := value.(map[string]any); ok {
			pruned := serializePrunedStateRecursive(nested, path, usedPaths)
			if len(pruned) > 0 {
				result[key] = pruned
			}
		}
	}

	return result
}

func serializePrunedStateRecursive(state map[string]any, basePath string, usedPaths map[string]bool) map[string]any {
	result := make(map[string]any)

	for key, value := range state {
		path := basePath + "." + key
		if usedPaths[path] {
			result[key] = value
		} else if nested, ok := value.(map[string]any); ok {
			pruned := serializePrunedStateRecursive(nested, path, usedPaths)
			if len(pruned) > 0 {
				result[key] = pruned
			}
		}
	}

	return result
}
