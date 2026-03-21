// Package main provides the GoSPA CLI entry point.
package main

import (
	"flag"
	"fmt"
	"os"

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
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: gospa create <name>")
			os.Exit(1)
		}
		cli.CreateProject(os.Args[2])
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
		_ = fs.Parse(os.Args[2:])
		cfg := &cli.BuildConfig{OutputDir: *out, Minify: *minify, Compress: *compress}
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
		_ = fs.Parse(os.Args[2:])
		cli.Generate(&cli.GenerateConfig{OutputDir: *out, InputDir: *inputDir})
	case "doctor":
		fs := flag.NewFlagSet("doctor", flag.ExitOnError)
		routesDir := fs.String("routes-dir", "./routes", "Routes directory to validate")
		_ = fs.Parse(os.Args[2:])
		cli.Doctor(&cli.DoctorConfig{RoutesDir: *routesDir})
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
  clean           Remove generated/build artifacts
  version         Print the CLI/framework version`)
}
