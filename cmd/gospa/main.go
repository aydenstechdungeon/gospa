// Package main provides the gospa CLI tool.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aydenstechdungeon/gospa/cli"
	"github.com/aydenstechdungeon/gospa/plugin"
	"github.com/aydenstechdungeon/gospa/plugin/tailwind"
)

// Version is the current version of GoSPA
const Version = "0.1.0"

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
	case "generate", "gen":
		cli.Generate()
	case "create":
		if len(os.Args) < 3 {
			printer.Error("Project name required")
			printer.Info("Usage: gospa create <project-name> [options]")
			os.Exit(1)
		}
		createProject(os.Args[2], printer)
	case "dev":
		cli.Dev()
	case "build":
		cli.Build()
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

	fmt.Printf("\n%s\n\n", printer.Bold("EXAMPLES"))
	fmt.Printf("    gospa create myapp\n")
	fmt.Printf("    cd myapp && gospa dev\n")
	fmt.Printf("    gospa build\n")
	fmt.Println()
}

func createProject(name string, printer *cli.ColorPrinter) {
	printer.Title("Creating GoSPA project: %s", name)

	// Create project directory in current directory
	projectPath := name
	printer.Step(1, 5, "Creating project directory")
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		printer.Error("Failed to create project directory: %v", err)
		os.Exit(1)
	}

	// Create subdirectories
	printer.Step(2, 5, "Creating directory structure")
	dirs := []string{
		filepath.Join(projectPath, "routes"),
		filepath.Join(projectPath, "components"),
		filepath.Join(projectPath, "lib"),
		filepath.Join(projectPath, "static"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			printer.Error("Failed to create directory %s: %v", dir, err)
			os.Exit(1)
		}
	}

	// Create go.mod
	printer.Step(3, 5, "Generating go.mod")
	goMod := fmt.Sprintf(`module %s

go 1.21

require (
	github.com/aydenstechdungeon/gospa v0.1.0
	github.com/a-h/templ v0.2.543
	github.com/gofiber/fiber/v2 v2.51.0
)
`, name)
	if err := os.WriteFile(filepath.Join(projectPath, "go.mod"), []byte(goMod), 0644); err != nil {
		printer.Error("Failed to create go.mod: %v", err)
		os.Exit(1)
	}

	// Create main.go
	printer.Step(4, 5, "Generating main.go")
	mainGo := `package main

import (
	"log"

	"github.com/aydenstechdungeon/gospa"
	_ "` + name + `/routes" // Import routes to trigger init()
)

func main() {
	app := gospa.New(gospa.Config{
		RoutesDir:   "./routes",
		DevMode:     true,
		AppName:     "` + name + `",
	})

	if err := app.Run(":3000"); err != nil {
		log.Fatal(err)
	}
}
`
	if err := os.WriteFile(filepath.Join(projectPath, "main.go"), []byte(mainGo), 0644); err != nil {
		printer.Error("Failed to create main.go: %v", err)
		os.Exit(1)
	}

	// Create routes/layout.templ
	printer.Step(5, 5, "Generating route templates")
	layoutTempl := `package routes

import "github.com/a-h/templ"

templ Layout(title string, children templ.Component) {
	<nav>
		<a href="/">Home</a>
	</nav>
	<div>
		@templ.Component(children)
	</div>
}
`
	if err := os.WriteFile(filepath.Join(projectPath, "routes", "layout.templ"), []byte(layoutTempl), 0644); err != nil {
		printer.Error("Failed to create layout.templ: %v", err)
		os.Exit(1)
	}

	// Create routes/page.templ
	pageTempl := `package routes

templ Page() {
	<div>
		<h1>Welcome to ` + name + `</h1>
		<p>Your GoSPA application is ready!</p>
	</div>
}
`
	if err := os.WriteFile(filepath.Join(projectPath, "routes", "page.templ"), []byte(pageTempl), 0644); err != nil {
		printer.Error("Failed to create page.templ: %v", err)
		os.Exit(1)
	}

	// Success message
	fmt.Println()
	printer.Success("Project '%s' created successfully!", name)
	fmt.Println()
	printer.Bold("Next steps:")
	fmt.Printf("    cd %s\n", projectPath)
	fmt.Println("    go mod tidy")
	fmt.Println("    templ generate")
	fmt.Println("    gospa generate")
	fmt.Println("    go run ../../cmd/gospa")
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
