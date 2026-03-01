// Package main provides the gospa CLI tool.
package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/aydenstechdungeon/gospa/cli"
	"github.com/aydenstechdungeon/gospa/plugin"
	"github.com/aydenstechdungeon/gospa/plugin/tailwind"
)

// Version is the current version of GoSPA
const Version = "0.1.5"

func main() {
	// Register built-in plugins
	plugin.Register(tailwind.New())

	printer := cli.NewColorPrinter()

	if len(os.Args) < 2 {
		cli.PrintBanner()
		printUsage(printer)
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "dev":
		handleDevCommand(printer)
	case "build":
		handleBuildCommand(printer)
	case "generate", "gen":
		handleGenerateCommand(printer)
	case "create":
		handleCreateCommand(printer)
	case "add":
		if len(os.Args) < 3 {
			printer.Error("Feature name required")
			printer.Info("Usage: gospa add <feature>")
			printer.Info("Available features: tailwind")
			os.Exit(1)
		}
		feature := os.Args[2]
		// Try to run plugin command "add:<feature>"
		found, err := plugin.RunCommand("add:"+feature, os.Args[3:])
		if err != nil {
			printer.Error("Failed to add %s: %v", feature, err)
			os.Exit(1)
		}
		if !found {
			printer.Error("Unknown feature: %s", feature)
			printer.Info("Available features: tailwind")
			os.Exit(1)
		}
		printer.Success("Added %s feature", feature)
	case "prune":
		handlePruneCommand(printer)
	case "state:analyze":
		handleStateAnalyzeCommand(printer)
	case "state:tree":
		handleStateTreeCommand(printer)
	case "version", "-v", "--version":
		fmt.Printf("GoSPA v%s\n", Version)
	case "help", "-h", "--help":
		cli.PrintBanner()
		printUsage(printer)
	default:
		printer.Error("Unknown command: %s", cmd)
		printer.Info("Run 'gospa help' for usage information")
		os.Exit(1)
	}
}

func printUsage(printer *cli.ColorPrinter) {
	fmt.Printf("%s\n\n", printer.Bold("USAGE"))
	fmt.Printf("    gospa <command> [arguments]\n\n")

	fmt.Printf("%s\n\n", printer.Bold("COMMANDS"))
	commands := []struct {
		cmd  string
		desc string
	}{
		{"create <name>", "Create a new GoSPA project"},
		{"dev", "Start development server with hot reload"},
		{"build", "Build for production"},
		{"generate, gen", "Generate route registration code"},
		{"add <feature>", "Add a feature (e.g., tailwind)"},
		{"version", "Show GoSPA version"},
		{"help", "Show this help message"},
	}

	for _, c := range commands {
		fmt.Printf("    %-20s %s\n", printer.Cyan(c.cmd), printer.Dim(c.desc))
	}

	fmt.Printf("\n%s\n", printer.Dim("Use 'gospa <command> --help' for more information on a command."))

	fmt.Printf("\n%s\n\n", printer.Bold("EXAMPLES"))
	fmt.Printf("    gospa create myapp\n")
	fmt.Printf("    cd myapp && gospa dev\n")
	fmt.Printf("    gospa build -o ./dist\n")
	fmt.Println()
}

// handlePruneCommand handles the prune command.
func handlePruneCommand(printer *cli.ColorPrinter) {
	config := &cli.PruneConfig{}

	// Parse flags
	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--root", "-r":
			if i+1 < len(args) {
				config.RootDir = args[i+1]
				i++
			}
		case "--output", "-o":
			if i+1 < len(args) {
				config.OutputDir = args[i+1]
				i++
			}
		case "--report", "-R":
			if i+1 < len(args) {
				config.ReportFile = args[i+1]
				i++
			}
		case "--keep-unused", "-k":
			config.KeepUnused = true
		case "--aggressive", "-a":
			config.Aggressive = true
		case "--dry-run", "-d":
			config.DryRun = true
		case "--verbose", "-v":
			config.Verbose = true
		case "--json", "-j":
			config.JSONOutput = true
		case "--exclude", "-e":
			if i+1 < len(args) {
				config.Exclude = append(config.Exclude, args[i+1])
				i++
			}
		case "--include", "-i":
			if i+1 < len(args) {
				config.Include = append(config.Include, args[i+1])
				i++
			}
		case "--help", "-h":
			printPruneUsage(printer)
			return
		}
	}

	cli.Prune(config)
}

// handleStateAnalyzeCommand handles the state:analyze command.
func handleStateAnalyzeCommand(printer *cli.ColorPrinter) {
	config := &cli.PruneConfig{}

	// Parse flags
	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--root", "-r":
			if i+1 < len(args) {
				config.RootDir = args[i+1]
				i++
			}
		case "--json", "-j":
			config.JSONOutput = true
		case "--verbose", "-v":
			config.Verbose = true
		case "--help", "-h":
			printer.Info("Usage: gospa state:analyze [options]")
			fmt.Println("\nOptions:")
			fmt.Println("  --root, -r <dir>    Root directory to analyze")
			fmt.Println("  --json, -j          Output as JSON")
			fmt.Println("  --verbose, -v       Verbose output")
			return
		}
	}

	cli.StateAnalyze(config)
}

// handleStateTreeCommand handles the state:tree command.
func handleStateTreeCommand(printer *cli.ColorPrinter) {
	var stateFile string
	var usedPaths []string
	jsonOut := false

	// Parse flags
	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--file", "-f":
			if i+1 < len(args) {
				stateFile = args[i+1]
				i++
			}
		case "--used", "-u":
			if i+1 < len(args) {
				usedPaths = append(usedPaths, args[i+1])
				i++
			}
		case "--json", "-j":
			jsonOut = true
		case "--help", "-h":
			printer.Info("Usage: gospa state:tree [options]")
			fmt.Println("\nOptions:")
			fmt.Println("  --file, -f <file>   State file to analyze")
			fmt.Println("  --used, -u <path>   Used state paths (can be repeated)")
			fmt.Println("  --json, -j          Output as JSON")
			return
		}
	}

	cli.StateTree(stateFile, usedPaths, jsonOut)
}

// handleBuildCommand handles the build command.
func handleBuildCommand(printer *cli.ColorPrinter) {
	config := &cli.BuildConfig{
		OutputDir:    "dist",
		Platform:     runtime.GOOS,
		Arch:         runtime.GOARCH,
		StaticAssets: true,
		Minify:       true,
		Compress:     true,
		Env:          "production",
	}

	// Parse flags
	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--output", "-o":
			if i+1 < len(args) {
				config.OutputDir = args[i+1]
				i++
			}
		case "--platform", "-p":
			if i+1 < len(args) {
				config.Platform = args[i+1]
				i++
			}
		case "--arch", "-a":
			if i+1 < len(args) {
				config.Arch = args[i+1]
				i++
			}
		case "--env", "-e":
			if i+1 < len(args) {
				config.Env = args[i+1]
				i++
			}
		case "--no-minify":
			config.Minify = false
		case "--no-compress":
			config.Compress = false
		case "--no-static":
			config.StaticAssets = false
		case "--all":
			cli.BuildAll()
			return
		case "--help", "-h":
			printBuildUsage(printer)
			return
		}
	}

	cli.Build(config)
}

// handleDevCommand handles the dev command.
func handleDevCommand(printer *cli.ColorPrinter) {
	config := &cli.DevConfig{
		Port:          3000,
		Host:          "localhost",
		RoutesDir:     "./routes",
		ComponentsDir: "./components",
	}

	// Parse flags
	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port", "-p":
			if i+1 < len(args) {
				var port int
				if _, err := fmt.Sscanf(args[i+1], "%d", &port); err != nil {
					printer.Error("Invalid port: %s", args[i+1])
					os.Exit(1)
				}
				config.Port = port
				i++
			}
		case "--host", "-H":
			if i+1 < len(args) {
				config.Host = args[i+1]
				i++
			}
		case "--routes", "-r":
			if i+1 < len(args) {
				config.RoutesDir = args[i+1]
				i++
			}
		case "--components", "-c":
			if i+1 < len(args) {
				config.ComponentsDir = args[i+1]
				i++
			}
		case "--help", "-h":
			printDevUsage(printer)
			return
		}
	}

	cli.Dev(config)
}

// handleGenerateCommand handles the generate command.
func handleGenerateCommand(printer *cli.ColorPrinter) {
	// For now, generate doesn't take many flags, but we can add InputDir/OutputDir
	config := &cli.GenerateConfig{
		InputDir:  ".",
		OutputDir: "./generated",
	}

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--input", "-i":
			if i+1 < len(args) {
				config.InputDir = args[i+1]
				i++
			}
		case "--output", "-o":
			if i+1 < len(args) {
				config.OutputDir = args[i+1]
				i++
			}
		case "--help", "-h":
			printGenerateUsage(printer)
			return
		}
	}

	cli.Generate(config)
}

// handleCreateCommand handles the create command.
func handleCreateCommand(printer *cli.ColorPrinter) {
	if len(os.Args) < 3 {
		printer.Error("Project name required")
		printer.Info("Usage: gospa create <project-name> [options]")
		os.Exit(1)
	}

	name := os.Args[2]
	config := &cli.ProjectConfig{
		Name:      name,
		Module:    name,
		OutputDir: name,
		WithGit:   true,
	}

	args := os.Args[3:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--module", "-m":
			if i+1 < len(args) {
				config.Module = args[i+1]
				i++
			}
		case "--output", "-o":
			if i+1 < len(args) {
				config.OutputDir = args[i+1]
				i++
			}
		case "--no-git":
			config.WithGit = false
		case "--docker":
			config.WithDocker = true
		case "--help", "-h":
			printCreateUsage(printer)
			return
		}
	}

	printer.Title("Creating GoSPA project: %s", config.Name)

	if err := cli.CreateProjectWithConfig(config); err != nil {
		printer.Error("Failed to create project: %v", err)
		os.Exit(1)
	}

	// Success message
	fmt.Println()
	printer.Success("Project '%s' created successfully!", config.Name)
	fmt.Println()
	printer.Bold("Next steps:")
	fmt.Printf("    cd %s\n", config.OutputDir)
	fmt.Println("    go mod tidy")
	fmt.Println("    templ generate")
	fmt.Println("    gospa generate")
	fmt.Printf("    go run .\n")
	fmt.Println()
}

// printBuildUsage prints usage for the build command.
func printBuildUsage(printer *cli.ColorPrinter) {
	printer.Info("Usage: gospa build [options]")
	fmt.Println("\nOptions:")
	fmt.Println("  --output, -o <dir>     Output directory (default: dist)")
	fmt.Println("  --platform, -p <os>    Target platform (default: current)")
	fmt.Println("  --arch, -a <arch>      Target architecture (default: current)")
	fmt.Println("  --env, -e <env>        Build environment (default: production)")
	fmt.Println("  --no-minify            Disable minification")
	fmt.Println("  --no-compress          Disable pre-compression")
	fmt.Println("  --no-static            Do not copy static assets")
	fmt.Println("  --all                  Build for all platforms")
	fmt.Println("  --help, -h             Show this help message")
}

// printDevUsage prints usage for the dev command.
func printDevUsage(printer *cli.ColorPrinter) {
	printer.Info("Usage: gospa dev [options]")
	fmt.Println("\nOptions:")
	fmt.Println("  --port, -p <port>      Server port (default: 3000)")
	fmt.Println("  --host, -H <host>      Server host (default: localhost)")
	fmt.Println("  --routes, -r <dir>     Routes directory (default: ./routes)")
	fmt.Println("  --components, -c <dir> Components directory (default: ./components)")
	fmt.Println("  --help, -h             Show this help message")
}

// printGenerateUsage prints usage for the generate command.
func printGenerateUsage(printer *cli.ColorPrinter) {
	printer.Info("Usage: gospa generate [options]")
	fmt.Println("\nOptions:")
	fmt.Println("  --input, -i <dir>      Input directory (default: .)")
	fmt.Println("  --output, -o <dir>     Output directory (default: ./generated)")
	fmt.Println("  --help, -h             Show this help message")
}

// printCreateUsage prints usage for the create command.
func printCreateUsage(printer *cli.ColorPrinter) {
	printer.Info("Usage: gospa create <name> [options]")
	fmt.Println("\nOptions:")
	fmt.Println("  --module, -m <name>    Go module name (default: <name>)")
	fmt.Println("  --output, -o <dir>     Output directory (default: <name>)")
	fmt.Println("  --no-git               Skip .gitignore creation")
	fmt.Println("  --docker               Add Dockerfile")
	fmt.Println("  --help, -h             Show this help message")
}

// printPruneUsage prints usage for the prune command.
func printPruneUsage(printer *cli.ColorPrinter) {
	printer.Info("Usage: gospa prune [options]")
	fmt.Println("\nOptions:")
	fmt.Println("  --root, -r <dir>       Root directory to analyze")
	fmt.Println("  --output, -o <dir>     Output directory for pruned files")
	fmt.Println("  --report, -R <file>    Write pruning report to file")
	fmt.Println("  --keep-unused, -k      Keep unused state (only analyze)")
	fmt.Println("  --aggressive, -a       Enable aggressive pruning")
	fmt.Println("  --exclude, -e <pattern> Exclude patterns (can be repeated)")
	fmt.Println("  --include, -i <pattern> Include patterns (can be repeated)")
	fmt.Println("  --dry-run, -d          Analyze without making changes")
	fmt.Println("  --verbose, -v          Verbose output")
	fmt.Println("  --json, -j             Output as JSON")
	fmt.Println("\nExamples:")
	fmt.Println("  gospa prune --dry-run")
	fmt.Println("  gospa prune --report pruned.json")
	fmt.Println("  gospa prune --aggressive --output ./pruned")
}
