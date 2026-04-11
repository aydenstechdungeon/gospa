// Package main provides the GoSPA CLI entry point.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

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
		open := fs.Bool("open", false, "Open browser automatically")
		noRestart := fs.Bool("no-restart", false, "Disable auto-restart on .go changes")
		verbose := fs.Bool("verbose", false, "Verbose logging output")
		timeout := fs.Duration("timeout", 30*time.Second, "Server start timeout")
		debounce := fs.Duration("debounce", 100*time.Millisecond, "File change debounce interval")
		proxy := fs.String("proxy", "", "Proxy API requests to backend")
		_ = fs.Parse(os.Args[2:])
		cli.Dev(&cli.DevConfig{
			Port:      *port,
			Host:      *host,
			RoutesDir: *routesDir,
			Open:      *open,
			NoRestart: *noRestart,
			Verbose:   *verbose,
			Timeout:   *timeout,
			Debounce:  *debounce,
			Proxy:     *proxy,
		})
	case "build":
		fs := flag.NewFlagSet("build", flag.ExitOnError)
		out := fs.String("o", "dist", "Output directory")
		platform := fs.String("platform", "", "Target GOOS")
		arch := fs.String("arch", "", "Target GOARCH")
		minify := fs.Bool("minify", true, "Minify client assets")
		compress := fs.Bool("compress", true, "Precompress static assets")
		cgo := fs.Bool("cgo", false, "Enable CGO for the Go binary build")
		ldflags := fs.String("ldflags", "-s -w", "Custom linker flags")
		tags := fs.String("tags", "", "Build tags (comma-separated)")
		assetsDir := fs.String("assets-dir", "static", "Static assets source directory")
		noManifest := fs.Bool("no-manifest", false, "Skip build manifest generation")
		noStatic := fs.Bool("no-static", false, "Skip static asset copying")
		noCompress := fs.Bool("no-compress", false, "Skip compression")
		sourcemap := fs.Bool("sourcemap", false, "Generate source maps")
		_ = fs.Parse(os.Args[2:])
		cfg := &cli.BuildConfig{
			OutputDir:    *out,
			Minify:       *minify,
			Compress:     *compress,
			CGO:          *cgo,
			StaticAssets: true,
			Env:          "production",
			LDFlags:      *ldflags,
			Tags:         *tags,
			AssetsDir:    *assetsDir,
			NoManifest:   *noManifest,
			NoStatic:     *noStatic,
			NoCompress:   *noCompress,
			SourceMap:    *sourcemap,
		}
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
		noTypes := fs.Bool("no-types", false, "Skip TS type generation")
		noActions := fs.Bool("no-actions", false, "Skip remote action generation")
		routesOnly := fs.Bool("routes-only", false, "Only generate routes")
		strict := fs.Bool("strict", false, "Strict type checking")
		noTempl := fs.Bool("no-templ", false, "Skip templ generate")
		watch := fs.Bool("watch", false, "Watch mode")
		_ = fs.Parse(os.Args[2:])
		cli.Generate(&cli.GenerateConfig{
			OutputDir:     *out,
			InputDir:      *inputDir,
			ComponentType: *componentType,
			NoTypes:       *noTypes,
			NoActions:     *noActions,
			RoutesOnly:    *routesOnly,
			Strict:        *strict,
			NoTempl:       *noTempl,
			Watch:         *watch,
		})
	case "doctor":
		fs := flag.NewFlagSet("doctor", flag.ExitOnError)
		routesDir := fs.String("routes-dir", "./routes", "Routes directory to validate")
		fix := fs.Bool("fix", false, "Auto-fix detected issues")
		jsonOutput := fs.Bool("json", false, "JSON output")
		quiet := fs.Bool("quiet", false, "Only show errors")
		checkUpdates := fs.Bool("check-updates", false, "Check for package updates")
		_ = fs.Parse(os.Args[2:])
		cli.Doctor(&cli.DoctorConfig{
			RoutesDir:    *routesDir,
			Fix:          *fix,
			JSONOutput:   *jsonOutput,
			Quiet:        *quiet,
			CheckUpdates: *checkUpdates,
		})
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
		fs := flag.NewFlagSet("clean", flag.ExitOnError)
		dryRun := fs.Bool("dry-run", false, "Show what would be deleted")
		nodeModules := fs.Bool("node-modules", true, "Include node_modules")
		generated := fs.Bool("generated", true, "Include generated files")
		dist := fs.Bool("dist", true, "Include dist directory")
		all := fs.Bool("all", false, "Clean everything including cache")
		cache := fs.Bool("cache", false, "Clean gospa cache (~/.gospa)")
		_ = fs.Parse(os.Args[2:])
		cli.Clean(&cli.CleanConfig{
			DryRun:      *dryRun,
			NodeModules: *nodeModules,
			Generated:   *generated,
			Dist:        *dist,
			All:         *all,
			Cache:       *cache,
		})
	case "serve":
		fs := flag.NewFlagSet("serve", flag.ExitOnError)
		port := fs.Int("port", 8080, "Server port")
		host := fs.String("host", "localhost", "Bind address")
		dir := fs.String("dir", "dist", "Directory to serve")
		https := fs.Bool("https", false, "Enable HTTPS")
		cert := fs.String("cert", "", "TLS certificate file")
		key := fs.String("key", "", "TLS key file")
		gzip := fs.Bool("gzip", true, "Enable gzip compression")
		brotli := fs.Bool("brotli", true, "Enable brotli compression")
		cache := fs.Bool("cache", true, "Enable cache headers")
		_ = fs.Parse(os.Args[2:])
		cli.Serve(&cli.ServeConfig{
			Port:   *port,
			Host:   *host,
			Dir:    *dir,
			HTTPS:  *https,
			Cert:   *cert,
			Key:    *key,
			Gzip:   *gzip,
			Brotli: *brotli,
			Cache:  *cache,
		})
	case "build-all":
		fs := flag.NewFlagSet("build-all", flag.ExitOnError)
		targets := fs.String("targets", "linux/amd64,linux/arm64,darwin/amd64,darwin/arm64,windows/amd64,windows/arm64", "Comma-separated target platforms")
		outputDir := fs.String("output", "./releases", "Output directory")
		compress := fs.Bool("compress", true, "Compress binaries with tar.gz")
		manifest := fs.Bool("manifest", true, "Generate release manifest")
		parallel := fs.Int("parallel", 0, "Number of parallel builds (0 = number of CPUs)")
		_ = fs.Parse(os.Args[2:])
		cli.BuildAll(&cli.BuildAllConfig{
			Targets:   splitCSV(*targets),
			OutputDir: *outputDir,
			Compress:  *compress,
			Manifest:  *manifest,
			Parallel:  *parallel,
		})
	case "config":
		fs := flag.NewFlagSet("config", flag.ExitOnError)
		showCmd := fs.Bool("show", false, "Show effective config")
		initCmd := fs.Bool("init", false, "Create default config file")
		jsonOutput := fs.Bool("json", false, "JSON output")
		_ = fs.Parse(os.Args[2:])
		switch {
		case *showCmd:
			cfg, err := cli.LoadConfig("")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
				os.Exit(1)
			}
			cfg.MergeWithEnv()
			if *jsonOutput {
				fmt.Printf("%#v\n", cfg)
			} else {
				fmt.Printf("GoSPA Config:\n  Dev: port=%d, host=%s\n", cfg.Dev.Port, cfg.Dev.Host)
				fmt.Printf("  Build: output=%s, minify=%v\n", cfg.Build.Output, cfg.Build.Minify)
			}
		case *initCmd:
			cfg := cli.DefaultConfig()
			err := cli.SaveConfig(cfg, "gospa.config.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Created gospa.config.yaml")
		default:
			fs.Usage()
			os.Exit(1)
		}
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
  build-all       Build for all platforms
  generate        Generate routes and client artifacts
  serve           Serve production build
  doctor          Validate local project/tooling setup
  prune           Analyze and prune unused state
  clean           Remove generated/build artifacts
  config          Config file management
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
