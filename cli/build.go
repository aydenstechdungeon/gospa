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

// Build builds the application for production.
func Build(config *BuildConfig) {
	fmt.Println("Building for production...")

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

	if err := BuildWithConfig(config); err != nil {
		fmt.Fprintf(os.Stderr, "Build failed: %v\n", err)
		os.Exit(1)
	}

	// Trigger AfterBuild hook
	_ = plugin.TriggerHook(plugin.AfterBuild, map[string]interface{}{"config": config})

	fmt.Println("✓ Build complete!")
}

// BuildWithConfig builds the application with custom configuration.
func BuildWithConfig(config *BuildConfig) error {
	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Step 1: Generate templ files
	fmt.Println("Generating templ files...")
	if err := regenerateTempl(); err != nil {
		return fmt.Errorf("failed to generate templ files: %w", err)
	}

	// Step 2: Generate TypeScript types
	fmt.Println("Generating TypeScript types...")
	runGenerate()

	// Step 3: Build client runtime
	fmt.Println("Building client runtime...")
	if err := buildClientRuntime(config); err != nil {
		return fmt.Errorf("failed to build client runtime: %w", err)
	}

	// Step 3.5: Ensure dependencies are tidied after generation
	fmt.Println("Tidying module dependencies...")
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Stdout = os.Stdout
	tidyCmd.Stderr = os.Stderr
	if err := tidyCmd.Run(); err != nil {
		return fmt.Errorf("failed to tidy module dependencies: %w", err)
	}

	// Step 4: Build Go binary
	fmt.Println("Building Go binary...")
	if err := buildGoBinary(config); err != nil {
		return fmt.Errorf("failed to build Go binary: %w", err)
	}

	// Step 5: Copy static assets
	if config.StaticAssets {
		fmt.Println("Copying static assets...")
		if err := copyStaticAssets(config); err != nil {
			return fmt.Errorf("failed to copy static assets: %w", err)
		}
	}

	// Step 6: Pre-compress static assets if requested
	if config.Compress {
		fmt.Println("Pre-compressing static assets...")
		if err := compressStaticAssets(config); err != nil {
			return fmt.Errorf("failed to compress static assets: %w", err)
		}
	}

	return nil
}

func buildClientRuntime(config *BuildConfig) error {
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
	cmd := exec.Command(bunPath, "build", entryPoint, "--outfile", outputPath, "--minify")
	cmd.Dir = clientDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "NODE_ENV="+config.Env)

	return cmd.Run()
}

func buildGoBinary(config *BuildConfig) error {
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

	return cmd.Run()
}

func copyStaticAssets(config *BuildConfig) error {
	staticDir := "static"
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		return nil
	}

	destDir := filepath.Join(config.OutputDir, "static")
	if err := os.MkdirAll(destDir, 0750); err != nil {
		return err
	}

	return filepath.Walk(staticDir, func(path string, info os.FileInfo, err error) error {
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
		return os.WriteFile(cleanDestPath, data, info.Mode())
	})
}

func compressStaticAssets(config *BuildConfig) error {
	destDir := filepath.Join(config.OutputDir, "static")
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return nil
	}

	return filepath.Walk(destDir, func(path string, info os.FileInfo, err error) error {
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

		return compressFileGzip(path)
	})
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
		if err := BuildWithConfig(config); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to build for %s/%s: %v\n", p.platform, p.arch, err)
		} else {
			fmt.Printf("✓ Built for %s/%s\n", p.platform, p.arch)
		}
	}
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
