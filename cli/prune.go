// Package cli provides the state pruning functionality for GoSPA.
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aydenstechdungeon/gospa/state"
)

// PruneConfig holds configuration for the prune command.
type PruneConfig struct {
	RootDir    string
	OutputDir  string
	ReportFile string
	KeepUnused bool
	Aggressive bool
	Exclude    []string
	Include    []string
	DryRun     bool
	Verbose    bool
	JSONOutput bool
}

// Prune executes the state pruning command.
func Prune(config *PruneConfig) {
	if config == nil {
		config = &PruneConfig{}
	}

	// Set default root directory
	if config.RootDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to get current directory: %v\n", err)
			os.Exit(1)
		}
		config.RootDir = cwd
	}

	// Build pruning config
	pruningConfig := state.DefaultPruningConfig()
	pruningConfig.RootDir = config.RootDir
	pruningConfig.OutputDir = config.OutputDir
	pruningConfig.ReportFile = config.ReportFile
	pruningConfig.KeepUnused = config.KeepUnused
	pruningConfig.Aggressive = config.Aggressive

	if len(config.Exclude) > 0 {
		pruningConfig.ExcludePatterns = config.Exclude
	}
	if len(config.Include) > 0 {
		pruningConfig.IncludePatterns = config.Include
	}

	// Create pruner
	pruner := state.NewStatePruner(pruningConfig)

	var report *state.PruningReport
	var err error

	if config.DryRun {
		// Only analyze, don't prune
		report, err = pruner.Analyze()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to analyze state: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Prune
		report, err = pruner.Prune()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to prune state: %v\n", err)
			os.Exit(1)
		}
	}

	// Output results
	if config.JSONOutput {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to marshal report: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
	} else if config.Verbose {
		printVerboseReport(report)
	} else {
		printSummaryReport(report, config.DryRun)
	}

	// Write report file if specified
	if config.ReportFile != "" {
		if err := pruner.WriteReport(config.ReportFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to write report: %v\n", err)
			os.Exit(1)
		}
		if config.Verbose {
			fmt.Printf("\nReport written to: %s\n", config.ReportFile)
		}
	}
}

// StateAnalyze executes the state analysis command.
func StateAnalyze(config *PruneConfig) {
	if config == nil {
		config = &PruneConfig{}
	}

	// Set default root directory
	if config.RootDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to get current directory: %v\n", err)
			os.Exit(1)
		}
		config.RootDir = cwd
	}

	// Build pruning config
	pruningConfig := state.DefaultPruningConfig()
	pruningConfig.RootDir = config.RootDir

	// Analyze
	pruner := state.NewStatePruner(pruningConfig)
	report, err := pruner.Analyze()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to analyze state: %v\n", err)
		os.Exit(1)
	}

	// Output
	if config.JSONOutput {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to marshal report: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
	} else if config.Verbose {
		printVerboseReport(report)
	} else {
		printSummaryReport(report, true)
	}
}

// StateTree executes the state tree visualization command.
func StateTree(stateFile string, usedPaths []string, jsonOut bool) {
	// Load state from file or stdin
	var stateData map[string]any
	if stateFile != "" {
		data, err := os.ReadFile(stateFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to read state file: %v\n", err)
			os.Exit(1)
		}
		if err := json.Unmarshal(data, &stateData); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to parse state file: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Try stdin
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			if err := json.NewDecoder(os.Stdin).Decode(&stateData); err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to parse state from stdin: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintln(os.Stderr, "Error: no state file provided and no stdin input")
			os.Exit(1)
		}
	}

	// Build tree
	tree := state.BuildStateTree(stateData)

	// Prune if used paths specified
	if len(usedPaths) > 0 {
		usedMap := make(map[string]bool)
		for _, p := range usedPaths {
			usedMap[p] = true
		}
		tree = state.PruneStateTree(tree, usedMap)
	}

	// Output
	if jsonOut {
		data, err := json.MarshalIndent(tree, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to marshal tree: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
	} else {
		printStateTree(tree, 0)
	}
}

// printSummaryReport prints a summary of the pruning report.
func printSummaryReport(report *state.PruningReport, dryRun bool) {
	action := "Pruned"
	if dryRun {
		action = "Would prune"
	}

	fmt.Printf("\n=== State Pruning Summary ===\n")
	fmt.Printf("Total state variables: %d\n", report.TotalStateVars)
	fmt.Printf("Used state variables:  %d\n", report.UsedStateVars)
	fmt.Printf("%s variables:      %d\n", action, report.PrunedStateVars)
	fmt.Printf("Estimated savings:     %d bytes\n", report.EstimatedSavings)

	if len(report.PrunedFiles) > 0 {
		fmt.Printf("\n%s files: %d\n", action, len(report.PrunedFiles))
	}

	if len(report.Errors) > 0 {
		fmt.Printf("\nErrors: %d\n", len(report.Errors))
	}
}

// printVerboseReport prints a detailed pruning report.
func printVerboseReport(report *state.PruningReport) {
	fmt.Printf("\n=== State Pruning Report ===\n\n")

	fmt.Printf("Statistics:\n")
	fmt.Printf("  Total state variables: %d\n", report.TotalStateVars)
	fmt.Printf("  Used state variables:  %d\n", report.UsedStateVars)
	fmt.Printf("  Pruned variables:      %d\n", report.PrunedStateVars)
	fmt.Printf("  Estimated savings:     %d bytes\n\n", report.EstimatedSavings)

	// Group by status
	var used, unused, exported []state.StateUsage
	for _, usage := range report.StateUsage {
		if usage.IsUsed {
			used = append(used, usage)
		} else if usage.IsExported {
			exported = append(exported, usage)
		} else {
			unused = append(unused, usage)
		}
	}

	if len(used) > 0 {
		fmt.Printf("Used state variables (%d):\n", len(used))
		for _, usage := range used {
			fmt.Printf("  ✓ %s (%s:%d)\n", usage.Name, filepath.Base(usage.File), usage.Line)
		}
		fmt.Println()
	}

	if len(exported) > 0 {
		fmt.Printf("Exported but unused (%d) - not pruned:\n", len(exported))
		for _, usage := range exported {
			fmt.Printf("  ○ %s (%s:%d)\n", usage.Name, filepath.Base(usage.File), usage.Line)
		}
		fmt.Println()
	}

	if len(unused) > 0 {
		fmt.Printf("Pruned state variables (%d):\n", len(unused))
		for _, usage := range unused {
			fmt.Printf("  ✗ %s (%s:%d)\n", usage.Name, filepath.Base(usage.File), usage.Line)
		}
		fmt.Println()
	}

	if len(report.PrunedFiles) > 0 {
		fmt.Printf("Modified files (%d):\n", len(report.PrunedFiles))
		for _, file := range report.PrunedFiles {
			fmt.Printf("  %s\n", file)
		}
		fmt.Println()
	}

	if len(report.Errors) > 0 {
		fmt.Printf("Errors (%d):\n", len(report.Errors))
		for _, err := range report.Errors {
			fmt.Printf("  ! %s\n", err)
		}
	}
}

// printStateTree prints a state tree with indentation.
func printStateTree(tree *state.StateTree, depth int) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}

	marker := " "
	if tree.Used {
		marker = "✓"
	}

	fmt.Printf("%s%s %s (%s)\n", indent, marker, tree.Name, tree.Type)

	if tree.Children != nil {
		for _, child := range tree.Children {
			printStateTree(child, depth+1)
		}
	}
}
