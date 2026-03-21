package cli

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/aydenstechdungeon/gospa/plugin"
)

// BuildConfig holds configuration for the production build.
type BuildConfig struct {
	OutputDir    string
	Platform     string
	Arch         string
	StaticAssets bool
	Minify       bool
	Compress     bool
	Env          string
}

// BuildSummary captures the important outputs from a production build.
type BuildSummary struct {
	BunPath            string
	ClientRuntimeBuilt bool
	ClientRuntimePath  string
	GoBinaryPath       string
	StaticFilesCopied  int
	CompressedFiles    int
}

// Build builds the application for production.
func Build(config *BuildConfig) {
	printer := NewColorPrinter()
	printer.Title("GoSPA Build")
	printer.Subtitle("Creating a production build with Go + Bun tooling")

	// Check if we're in a GoSPA project
	if !isGoSPAProject() {
		fmt.Fprintln(os.Stderr, "Error: Not a GoSPA project. Run 'gospa create' first.")
		os.Exit(1)
	}

	// Use defaults if config is nil
	if config == nil {
		config = &BuildConfig{
			OutputDir:    "dist",
			Platform:     runtime.GOOS,
			Arch:         runtime.GOARCH,
			StaticAssets: true,
			Minify:       true,
			Compress:     true,
			Env:          "production",
		}
	}

	// Trigger BeforeBuild hook
	if err := plugin.TriggerHook(plugin.BeforeBuild, map[string]interface{}{"config": config}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: BeforeBuild hook failed: %v\n", err)
		os.Exit(1)
	}

	summary, err := BuildWithConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Build failed: %v\n", err)
		os.Exit(1)
	}

	// Trigger AfterBuild hook
	_ = plugin.TriggerHook(plugin.AfterBuild, map[string]interface{}{"config": config})

	printBuildSummary(printer, summary)
	printer.Success("Build complete!")
}

// BuildWithConfig builds the application with custom configuration.
func BuildWithConfig(config *BuildConfig) (*BuildSummary, error) {
	summary := &BuildSummary{}

	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Step 1: Generate templ files
	fmt.Println("Generating templ files...")
	if err := regenerateTempl(); err != nil {
		return nil, fmt.Errorf("failed to generate templ files: %w", err)
	}

	// Step 2: Generate TypeScript types
	fmt.Println("Generating TypeScript types...")
	runGenerate()

	// Step 3: Build client runtime
	fmt.Println("Building client runtime...")
	if err := buildClientRuntime(config, summary); err != nil {
		return nil, fmt.Errorf("failed to build client runtime: %w", err)
	}

	// Step 3.5: Ensure dependencies are tidied after generation
	fmt.Println("Tidying module dependencies...")
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Stdout = os.Stdout
	tidyCmd.Stderr = os.Stderr
	if err := tidyCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to tidy module dependencies: %w", err)
	}

	// Step 4: Build Go binary
	fmt.Println("Building Go binary...")
	binaryPath, err := buildGoBinary(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build Go binary: %w", err)
	}
	summary.GoBinaryPath = binaryPath

	// Step 5: Copy static assets
	if config.StaticAssets {
		fmt.Println("Copying static assets...")
		count, err := copyStaticAssets(config)
		if err != nil {
			return nil, fmt.Errorf("failed to copy static assets: %w", err)
		}
		summary.StaticFilesCopied = count
	}

	// Step 6: Pre-compress static assets if requested
	if config.Compress {
		fmt.Println("Pre-compressing static assets...")
		count, err := compressStaticAssets(config)
		if err != nil {
			return nil, fmt.Errorf("failed to compress static assets: %w", err)
		}
		summary.CompressedFiles = count
	}

	return summary, nil
}

func buildClientRuntime(config *BuildConfig, summary *BuildSummary) error {
	clientDir := "client"
	if _, err := os.Stat(clientDir); os.IsNotExist(err) {
		// No client directory, skip
		return nil
	}

	// Check if bun is available
	bunPath, err := exec.LookPath("bun")
	if err != nil {
		fmt.Println("Warning: bun not found, skipping client build")
		return nil
	}
	summary.BunPath = bunPath

	// Locate the entry point — prefer src/runtime.ts, fall back to src/index.ts
	entryPoint := ""
	for _, candidate := range []string{"src/runtime.ts", "src/index.ts", "src/main.ts"} {
		if _, err := os.Stat(filepath.Join(clientDir, candidate)); err == nil {
			entryPoint = candidate
			break
		}
	}
	if entryPoint == "" {
		fmt.Println("Warning: no client entry point found (src/runtime.ts, src/index.ts, src/main.ts), skipping client build")
		return nil
	}

	// Build the client runtime
	outputPath := filepath.Join(config.OutputDir, "static", "js", "runtime.js")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0750); err != nil {
		return err
	}

	// Run bun build
	//nolint:gosec // bunPath is safe executable from LookPath
	args := []string{"build", entryPoint, "--outfile", outputPath}
	if config.Minify {
		args = append(args, "--minify")
	}
	cmd := exec.Command(bunPath, args...)
	cmd.Dir = clientDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "NODE_ENV="+config.Env)

	if err := cmd.Run(); err != nil {
		return err
	}

	summary.ClientRuntimeBuilt = true
	summary.ClientRuntimePath = outputPath
	return nil
}

func buildGoBinary(config *BuildConfig) (string, error) {
	// Determine output filename
	outputName := "server"
	if config.Platform == "windows" {
		outputName = "server.exe"
	}

	outputPath := filepath.Join(config.OutputDir, outputName)

	// Build command
	args := []string{
		"build",
		"-ldflags", "-s -w", // Strip debug info
		"-o", outputPath,
		".",
	}

	//nolint:gosec // args are safe static inputs
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set environment for cross-compilation
	env := os.Environ()
	env = append(env, "CGO_ENABLED=0")
	env = append(env, "GOOS="+config.Platform)
	env = append(env, "GOARCH="+config.Arch)
	env = append(env, "GOSPA_ENV="+config.Env)
	cmd.Env = env

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return outputPath, nil
}

func copyStaticAssets(config *BuildConfig) (int, error) {
	staticDir := "static"
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		return 0, nil
	}

	destDir := filepath.Join(config.OutputDir, "static")
	if err := os.MkdirAll(destDir, 0750); err != nil {
		return 0, err
	}

	copied := 0
	err := filepath.Walk(staticDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(staticDir, path)
		if err != nil {
			return err
		}

		// Create destination path
		destPath := filepath.Join(destDir, relPath)
		// Validate path is within expected directory to prevent traversal
		cleanDestPath := filepath.Clean(destPath)
		if !strings.HasPrefix(cleanDestPath, filepath.Clean(destDir)) {
			return fmt.Errorf("invalid destination path: %s", destPath)
		}
		if err := os.MkdirAll(filepath.Dir(cleanDestPath), 0750); err != nil {
			return err
		}

		// Copy file
		//nolint:gosec
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		//nolint:gosec // path validated above with strings.HasPrefix check
		if err := os.WriteFile(cleanDestPath, data, info.Mode()); err != nil {
			return err
		}
		copied++
		return nil
	})
	return copied, err
}

func compressStaticAssets(config *BuildConfig) (int, error) {
	destDir := filepath.Join(config.OutputDir, "static")
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return 0, nil
	}

	compressed := 0
	err := filepath.Walk(destDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Only compress compressible files
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".js" && ext != ".css" && ext != ".html" && ext != ".svg" && ext != ".json" {
			return nil
		}

		// Skip already compressed files
		if ext == ".gz" || ext == ".br" {
			return nil
		}

		if err := compressFileGzip(path); err != nil {
			return err
		}
		compressed++
		return nil
	})
	return compressed, err
}

func compressFileGzip(path string) error {
	//nolint:gosec
	input, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()

	//nolint:gosec
	output, err := os.Create(path + ".gz")
	if err != nil {
		return err
	}
	defer func() { _ = output.Close() }()

	writer, err := gzip.NewWriterLevel(output, gzip.BestCompression)
	if err != nil {
		return err
	}
	defer func() { _ = writer.Close() }()

	_, err = io.Copy(writer, input)
	return err
}

// BuildAll builds for all platforms.
func BuildAll() {
	platforms := []struct {
		platform string
		arch     string
	}{
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"windows", "amd64"},
		{"windows", "arm64"},
	}

	for _, p := range platforms {
		outputDir := fmt.Sprintf("dist/%s-%s", p.platform, p.arch)
		config := &BuildConfig{
			OutputDir:    outputDir,
			Platform:     p.platform,
			Arch:         p.arch,
			StaticAssets: true,
			Minify:       true,
			Compress:     true,
			Env:          "production",
		}

		fmt.Printf("\nBuilding for %s/%s...\n", p.platform, p.arch)
		if _, err := BuildWithConfig(config); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to build for %s/%s: %v\n", p.platform, p.arch, err)
		} else {
			fmt.Printf("✓ Built for %s/%s\n", p.platform, p.arch)
		}
	}
}

func printBuildSummary(printer *ColorPrinter, summary *BuildSummary) {
	if summary == nil {
		return
	}

	printer.Info("Bun executable: %s", displayOrFallback(summary.BunPath, "not used"))
	if summary.ClientRuntimeBuilt {
		printer.Info("Client runtime: %s (%s)", summary.ClientRuntimePath, formatFileSize(summary.ClientRuntimePath))
	} else {
		printer.Warning("Client runtime: skipped")
	}

	if summary.GoBinaryPath != "" {
		printer.Info("Go binary: %s (%s)", summary.GoBinaryPath, formatFileSize(summary.GoBinaryPath))
	}
	printer.Info("Static files copied: %d", summary.StaticFilesCopied)
	printer.Info("Static files compressed: %d", summary.CompressedFiles)
}

func displayOrFallback(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func formatFileSize(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "size unavailable"
	}

	size := float64(info.Size())
	units := []string{"B", "KB", "MB", "GB"}
	unit := units[0]
	for i := 0; i < len(units) && size >= 1024; i++ {
		unit = units[i]
		if size < 1024 || i == len(units)-1 {
			break
		}
		size /= 1024
		unit = units[i+1]
	}

	if unit == "B" {
		return fmt.Sprintf("%d %s", info.Size(), unit)
	}
	return fmt.Sprintf("%.1f %s", size, unit)
}

// Clean removes build artifacts.
func Clean() {
	fmt.Println("Cleaning build artifacts...")

	dirs := []string{"dist", "node_modules"}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); err == nil {
			if err := os.RemoveAll(dir); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to remove %s: %v\n", dir, err)
			} else {
				fmt.Printf("✓ Removed %s\n", dir)
			}
		}
	}

	// Remove generated templ files
	// Use WalkDir for safer filesystem traversal (avoids TOCTOU race)
	if err := filepath.WalkDir(".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			return nil
		}

		name := d.Name()
		if strings.HasSuffix(name, "_templ.go") || strings.HasSuffix(name, "_templ.txt") {
			// Clean and validate path to prevent any potential path traversal
			cleanPath := filepath.Clean(path)
			// Resolve to absolute path to ensure we're within the project
			absPath, err := filepath.Abs(cleanPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to resolve path %s: %v\n", cleanPath, err)
				return nil
			}
			// Remove the cleaned path
			//nolint:gosec // clean command is intended to remove files identified during walk; path validated with filepath.Abs
			if err := os.Remove(absPath); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to remove %s: %v\n", cleanPath, err)
			} else {
				fmt.Printf("✓ Removed %s\n", cleanPath)
			}
		}

		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error walking directory: %v\n", err)
	}

	fmt.Println("✓ Clean complete!")
}

// Watch builds and watches for changes.
func Watch() {
	fmt.Println("Building and watching for changes...")

	// Initial build
	Build(nil)

	// Only watch directories that actually exist; components/, lib/, and
	// static/ are all optional in a GoSPA project.
	candidateDirs := []string{"./routes", "./components", "./lib", "./static"}
	watchDirs := make([]string, 0, len(candidateDirs))
	for _, d := range candidateDirs {
		if _, err := os.Stat(d); err == nil {
			watchDirs = append(watchDirs, d)
		}
	}

	watcher := NewDevWatcher(watchDirs...)
	if err := watcher.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting watcher: %v\n", err)
		return
	}
	defer watcher.Stop()

	fmt.Println("Watching for changes... Press Ctrl+C to stop")

	// Handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-sigChan:
			fmt.Println("\nStopping watcher...")
			return
		case event := <-watcher.Events:
			fmt.Printf("\nFile changed: %s\n", event.File)
			Build(nil)
			fmt.Println("✓ Rebuilt")
		case err := <-watcher.Errors:
			fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
		}
	}
}
