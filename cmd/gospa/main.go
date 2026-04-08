// Package main provides the GoSPA CLI entry point.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/aydenstechdungeon/gospa"
	"github.com/aydenstechdungeon/gospa/cli"

	// Register built-in plugins
	_ "github.com/aydenstechdungeon/gospa/plugin/image"
	_ "github.com/aydenstechdungeon/gospa/plugin/postcss"
	_ "github.com/aydenstechdungeon/gospa/plugin/qrcode"
	_ "github.com/aydenstechdungeon/gospa/plugin/tailwind"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Println(gospa.Version)
	case "create":
		fs := flag.NewFlagSet("create", flag.ExitOnError)
		nonInteractive := fs.Bool("y", false, "Non-interactive mode (use defaults for prompts)")
		nonInteractiveLong := fs.Bool("non-interactive", false, "Non-interactive mode")
		_ = fs.Parse(os.Args[2:])
		
		args := fs.Args()
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: gospa create <name> [-y]")
			os.Exit(1)
		}
		
		name := args[0]
		if err := cli.ValidateProjectName(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid project name: %v\n", err)
			os.Exit(1)
		}
		
		isNonInteractive := *nonInteractive || *nonInteractiveLong
		cli.CreateProjectWithOptions(name, "", isNonInteractive)
	case "dev":
		fs := flag.NewFlagSet("dev", flag.ExitOnError)
		port := fs.Int("port", 3000, "Port to advertise in dev output")
		host := fs.String("host", "localhost", "Host to advertise in dev output")
		routesDir := fs.String("routes-dir", "./routes", "Routes directory")
		_ = fs.Parse(os.Args[2:])
		cli.Dev(&cli.DevConfig{Port: *port, Host: *host, RoutesDir: *routesDir})
	case "build":
		fs := flag.NewFlagSet("build", flag.ExitOnError)
		out := fs.String("o", "dist", "Output directory")
		platform := fs.String("platform", "", "Target GOOS")
		arch := fs.String("arch", "", "Target GOARCH")
		minify := fs.Bool("minify", true, "Minify client assets")
		compress := fs.Bool("compress", true, "Precompress static assets")
		cgo := fs.Bool("cgo", false, "Enable CGO for the Go binary build")
		_ = fs.Parse(os.Args[2:])
		cfg := &cli.BuildConfig{OutputDir: *out, Minify: *minify, Compress: *compress, CGO: *cgo}
		if *platform != "" {
			cfg.Platform = *platform
		}
		if *arch != "" {
			cfg.Arch = *arch
		}
		cli.Build(cfg)
	case "generate":
		fs := flag.NewFlagSet("generate", flag.ExitOnError)
		out := fs.String("o", "./generated", "Output directory")
		inputDir := fs.String("input-dir", ".", "Input directory to scan for routes and state")
		componentType := fs.String("type", "island", "Default .gospa component type: island, page, layout, static, server")
		_ = fs.Parse(os.Args[2:])
		cli.Generate(&cli.GenerateConfig{OutputDir: *out, InputDir: *inputDir, ComponentType: *componentType})
	case "doctor":
		fs := flag.NewFlagSet("doctor", flag.ExitOnError)
		routesDir := fs.String("routes-dir", "./routes", "Routes directory to validate")
		_ = fs.Parse(os.Args[2:])
		cli.Doctor(&cli.DoctorConfig{RoutesDir: *routesDir})
	case "prune":
		fs := flag.NewFlagSet("prune", flag.ExitOnError)
		rootDir := fs.String("root-dir", ".", "Project root directory to analyze")
		outputDir := fs.String("output-dir", "", "Optional output directory for rewritten files")
		reportFile := fs.String("report-file", "", "Write pruning report to file")
		keepUnused := fs.Bool("keep-unused", false, "Keep unused state variables (analysis-only behavior)")
		aggressive := fs.Bool("aggressive", false, "Enable aggressive pruning heuristics")
		dryRun := fs.Bool("dry-run", false, "Analyze only; do not modify files")
		verbose := fs.Bool("verbose", false, "Print detailed report output")
		jsonOut := fs.Bool("json", false, "Emit report as JSON")
		exclude := fs.String("exclude", "", "Comma-separated exclude glob patterns")
		include := fs.String("include", "", "Comma-separated include glob patterns")
		_ = fs.Parse(os.Args[2:])
		cli.Prune(&cli.PruneConfig{
			RootDir:    *rootDir,
			OutputDir:  *outputDir,
			ReportFile: *reportFile,
			KeepUnused: *keepUnused,
			Aggressive: *aggressive,
			Exclude:    splitCSV(*exclude),
			Include:    splitCSV(*include),
			DryRun:     *dryRun,
			Verbose:    *verbose,
			JSONOutput: *jsonOut,
		})
	case "clean":
		cli.Clean()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`GoSPA CLI

Usage:
  gospa <command> [flags]

	Commands:
	  create <name>   Create a new project
	  dev             Start the development server
	  build           Build for production
	  generate        Generate routes and client artifacts
	  doctor          Validate local project/tooling setup
	  prune           Analyze and prune unused state
	  clean           Remove generated/build artifacts
	  version         Print the CLI/framework version`)
}

func splitCSV(input string) []string {
	if input == "" {
		return nil
	}
	parts := make([]string, 0)
	for _, part := range strings.Split(input, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	if len(parts) == 0 {
		return nil
	}
	return parts
}
