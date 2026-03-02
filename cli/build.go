package cli

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

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
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
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

	// Build the client runtime
	outputPath := filepath.Join(config.OutputDir, "static", "js", "runtime.js")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	// Run bun build
	cmd := exec.Command(bunPath, "build", "src/runtime.ts", "--outfile", outputPath, "--minify")
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
	if err := os.MkdirAll(destDir, 0755); err != nil {
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
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		// Copy file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(destPath, data, info.Mode())
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
	input, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()

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
	if err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(path, "_templ.go") || strings.HasSuffix(path, "_templ.txt") {
			if err := os.Remove(path); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to remove %s: %v\n", path, err)
			} else {
				fmt.Printf("✓ Removed %s\n", path)
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

	// Start watcher
	watcher := NewDevWatcher("./routes", "./components", "./lib", "./static")
	if err := watcher.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting watcher: %v\n", err)
		return
	}
	defer watcher.Stop()

	fmt.Println("Watching for changes... Press Ctrl+C to stop")

	// Handle file changes
	for {
		select {
		case event := <-watcher.Events:
			fmt.Printf("\nFile changed: %s\n", event.File)
			Build(nil)
			fmt.Println("✓ Rebuilt")
		case err := <-watcher.Errors:
			fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
		}
	}
}
