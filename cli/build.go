package cli

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/aydenstechdungeon/gospa/plugin"
)

// BuildConfig holds configuration for the production build.
type BuildConfig struct {
	OutputDir    string // Output directory
	Platform     string // Target GOOS
	Arch         string // Target GOARCH
	StaticAssets bool   // Copy static assets
	Minify       bool   // Enable minification
	Compress     bool   // Enable gzip/br compression
	Env          string // Build environment
	SourceMap    bool   // Generate source maps
	NoSourceMap  bool   // Explicitly disable source maps
	CGO          bool   // Enable CGO
	LDFlags      string // Custom linker flags
	Tags         string // Build tags (comma-separated)
	AssetsDir    string // Static assets source directory
	NoManifest   bool   // Skip build manifest generation
	Watch        bool   // Watch mode after build
	NoStatic     bool   // Skip static asset copying
	NoCompress   bool   // Skip compression
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

// BuildAllConfig holds configuration for multi-platform builds.
type BuildAllConfig struct {
	Targets   []string // Target platforms (e.g., linux/amd64, darwin/arm64)
	OutputDir string   // Output directory for builds
	Compress  bool     // Compress binaries with tar.gz
	Manifest  bool     // Generate release manifest
	Parallel  int      // Number of parallel builds (0 = number of CPUs)
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
			LDFlags:      "-s -w",
			AssetsDir:    "static",
			NoManifest:   false,
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

// buildMu protects the build state to prevent concurrent builds
var buildMu sync.Mutex

// isBuilding indicates if a build is currently in progress
var isBuilding bool

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

	// Step 3: Unified Bun Build (Runtime + Islands)
	fmt.Println("Building client assets (Runtime & Islands)...")
	if err := unifiedClientBuild(config, summary); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Unified client build failed: %v\n", err)
	}

	// Step 3.5: Ensure dependencies are tidied after generation
	// Skip if GOSPA_SKIP_MOD_TIDY is set (useful for offline builds or when go.mod has replace directives)
	if os.Getenv("GOSPA_SKIP_MOD_TIDY") == "" {
		fmt.Println("Tidying module dependencies...")
		tidyCmd := exec.Command("go", "mod", "tidy")
		tidyCmd.Stdout = os.Stdout
		tidyCmd.Stderr = os.Stderr
		if err := tidyCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: go mod tidy failed: %v (set GOSPA_SKIP_MOD_TIDY=1 to skip)\n", err)
		}
	}

	// Step 4: Build Go binary
	fmt.Println("Building Go binary...")
	binaryPath, err := buildGoBinary(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build Go binary: %w", err)
	}
	summary.GoBinaryPath = binaryPath

	// Step 5: Copy static assets
	if config.StaticAssets && !config.NoStatic {
		fmt.Println("Copying static assets...")
		count, err := copyStaticAssets(config)
		if err != nil {
			return nil, fmt.Errorf("failed to copy static assets: %w", err)
		}
		summary.StaticFilesCopied = count
	}

	// Step 6: Pre-compress static assets if requested
	if config.Compress && !config.NoCompress {
		fmt.Println("Pre-compressing static assets...")
		count, err := compressStaticAssets(config)
		if err != nil {
			return nil, fmt.Errorf("failed to compress static assets: %w", err)
		}
		summary.CompressedFiles = count
	}

	// Step 7: Generate build manifest
	if !config.NoManifest {
		fmt.Println("Generating build manifest...")
		if err := generateBuildManifest(config); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to generate manifest: %v\n", err)
		}
	}

	return summary, nil
}

func unifiedClientBuild(config *BuildConfig, summary *BuildSummary) error {
	clientDir := "client"
	islandsEntry := "generated/islands.ts"
	outputDir := filepath.Join(config.OutputDir, "static", "js")

	// Collect entry points
	entries := []string{}

	// 1. Runtime Entry
	runtimeEntry := ""
	for _, candidate := range []string{"src/runtime.ts", "src/index.ts", "src/main.ts"} {
		if _, err := os.Stat(filepath.Join(clientDir, candidate)); err == nil {
			runtimeEntry = filepath.Join(clientDir, candidate)
			entries = append(entries, runtimeEntry)
			break
		}
	}

	// 2. Secure Runtime Entry
	if _, err := os.Stat(filepath.Join(clientDir, "src/runtime-secure.ts")); err == nil {
		entries = append(entries, filepath.Join(clientDir, "src/runtime-secure.ts"))
	}

	// 3. Islands Entry
	isIslandsExist := false
	if _, err := os.Stat(islandsEntry); err == nil {
		entries = append(entries, islandsEntry)
		isIslandsExist = true
	}

	if len(entries) == 0 {
		return nil
	}

	// Detect package manager
	pm := GetPackageManager()
	if pm == NonePM {
		fmt.Println("Warning: No package manager (bun, pnpm, npm) found, skipping client build")
		return nil
	}

	// Prepare output directory
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return err
	}

	var cmd *exec.Cmd
	if pm == BunPM {
		bunPath, _ := exec.LookPath("bun")
		summary.BunPath = bunPath

		args := []string{"build"}
		args = append(args, entries...)
		args = append(args, "--outdir", outputDir, "--target", "browser", "--format", "esm", "--splitting")
		if config.Minify {
			args = append(args, "--minify")
		}
		if config.Env == "production" {
			args = append(args, "--drop:console", "--drop:debugger")
		}
		if config.SourceMap && !config.NoSourceMap {
			args = append(args, "--source-map")
		}

		// #nosec //nolint:gosec
		cmd = exec.Command(bunPath, args...)
	} else {
		// Fallback to esbuild via npx/pnpm dlx
		// Parse the execute command properly - it returns things like "pnpm dlx" or "npx"
		execCmd, execArgs := parseExecuteCommand(GetExecuteCommand(pm))

		args := append(execArgs, "esbuild") //nolint:gocritic // Intentionally creating new slice to preserve execArgs
		args = append(args, entries...)
		args = append(args, "--outdir="+outputDir, "--target=browser", "--format=esm", "--splitting", "--bundle")

		if config.Minify {
			args = append(args, "--minify")
		}
		if config.Env == "production" {
			args = append(args, "--drop:console", "--drop:debugger")
		}
		if config.SourceMap && !config.NoSourceMap {
			args = append(args, "--sourcemap")
		}

		// #nosec //nolint:gosec
		cmd = exec.Command(execCmd, args...)
		fmt.Printf("Warning: Bun not found. Using %s esbuild for bundling (slower & not preferred)\n", pm)
	}

	summary.BunPath = pm.String() // Track which PM was used
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "NODE_ENV="+config.Env)

	if err := cmd.Run(); err != nil {
		return err
	}

	summary.ClientRuntimeBuilt = true
	summary.ClientRuntimePath = filepath.Join(outputDir, "runtime.js")
	if isIslandsExist {
		fmt.Printf("✓ Client assets and Islands bundle built in %s\n", outputDir)
	} else {
		fmt.Printf("✓ Client runtime built in %s\n", outputDir)
	}

	return nil
}

// BuildIslands builds the islands TypeScript bundle into a single JavaScript file.
func BuildIslands(config *BuildConfig, summary *BuildSummary) error {
	if config == nil {
		config = &BuildConfig{
			OutputDir: "dist",
			Env:       "development",
		}
	}
	if summary == nil {
		summary = &BuildSummary{}
	}
	return unifiedClientBuild(config, summary)
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
		"-trimpath",
		"-ldflags", config.LDFlags,
		"-o", outputPath,
		".",
	}

	// Add build tags if specified
	if config.Tags != "" {
		args = append(args, "-tags", config.Tags)
	}

	// #nosec //nolint:gosec // args are safe static inputs
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set environment for cross-compilation
	env := os.Environ()
	cgo := "0"
	if config.CGO {
		cgo = "1"
	}
	env = append(env, "CGO_ENABLED="+cgo)
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
	staticDir := config.AssetsDir
	if staticDir == "" {
		staticDir = "static"
	}
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		return 0, nil
	}

	destDir := filepath.Join(config.OutputDir, "static")
	if err := os.MkdirAll(destDir, 0750); err != nil {
		return 0, err
	}

	var files []string
	err := filepath.WalkDir(staticDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	var copied struct {
		sync.Mutex
		count int
	}
	var firstErr error
	var errOnce sync.Once
	var wg sync.WaitGroup
	sem := make(chan struct{}, 20) // Concurrency limit for copying

	for _, file := range files {
		wg.Add(1)
		sem <- struct{}{}
		go func(srcPath string) {
			defer wg.Done()
			defer func() { <-sem }()

			relPath, err := filepath.Rel(staticDir, srcPath)
			if err != nil {
				errOnce.Do(func() { firstErr = err })
				return
			}

			destPath := filepath.Join(destDir, relPath)
			cleanDestPath := filepath.Clean(destPath)

			if err := os.MkdirAll(filepath.Dir(cleanDestPath), 0750); err != nil {
				errOnce.Do(func() { firstErr = err })
				return
			}

			if err := copyFile(srcPath, cleanDestPath); err != nil {
				errOnce.Do(func() { firstErr = err })
				return
			}

			copied.Lock()
			copied.count++
			copied.Unlock()
		}(file)
	}
	wg.Wait()

	return copied.count, firstErr
}

func copyFile(src, dst string) error {
	// #nosec //nolint:gosec // G304: copying project files during build
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()

	// #nosec //nolint:gosec // G304: creating project files during build
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()

	_, err = io.Copy(d, s)
	return err
}

func compressStaticAssets(config *BuildConfig) (int, error) {
	destDir := filepath.Join(config.OutputDir, "static")
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return 0, nil
	}

	var files []string
	err := filepath.WalkDir(destDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
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

		files = append(files, path)
		return nil
	})
	if err != nil {
		return 0, err
	}

	// Track errors from both compression passes
	var (
		gzipErr   error
		brotliErr error
		wg        sync.WaitGroup
		sem       = make(chan struct{}, 10) // Concurrency limit per algorithm
	)

	for _, file := range files {
		// Parallel Gzip
		wg.Add(1)
		sem <- struct{}{}
		go func(path string) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := compressFileGzip(path); err != nil {
				gzipErr = err
			}
		}(file)

		// Parallel Brotli
		wg.Add(1)
		sem <- struct{}{}
		go func(path string) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := compressFileBrotli(path); err != nil {
				brotliErr = err
			}
		}(file)
	}
	wg.Wait()

	// Return first error encountered, or nil
	if gzipErr != nil {
		return 0, gzipErr
	}
	if brotliErr != nil {
		return 0, brotliErr
	}

	// Count is the number of unique files compressed (not compression algorithms)
	return len(files), nil
}

func compressFileGzip(path string) error {
	// #nosec //nolint:gosec // G304: compressing project files during build
	input, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()

	// #nosec //nolint:gosec // G304: creating compressed project files during build
	output, err := os.Create(path + ".gz")
	if err != nil {
		return err
	}
	defer func() { _ = output.Close() }()

	writer, err := gzip.NewWriterLevel(output, gzip.DefaultCompression)
	if err != nil {
		return err
	}
	defer func() { _ = writer.Close() }()

	_, err = io.Copy(writer, input)
	return err
}

func compressFileBrotli(path string) error {
	// #nosec //nolint:gosec // G304: compressing project files during build
	input, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()

	// #nosec //nolint:gosec // G304: creating compressed project files during build
	output, err := os.Create(path + ".br")
	if err != nil {
		return err
	}
	defer func() { _ = output.Close() }()

	writer := brotli.NewWriterLevel(output, 4) // Level 4 is a good default
	defer func() { _ = writer.Close() }()

	_, err = io.Copy(writer, input)
	return err
}

func generateBuildManifest(config *BuildConfig) error {
	manifest := make(map[string]string)

	// Walk through output directory and create file hashes
	destDir := config.OutputDir
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return nil
	}

	err := filepath.WalkDir(destDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// Skip the binary itself
		if path == filepath.Join(destDir, "server") || path == filepath.Join(destDir, "server.exe") {
			return nil
		}

		relPath, err := filepath.Rel(destDir, path)
		if err != nil {
			return err
		}

		// Stream file content into hasher to avoid loading entire file into memory
		hashWriter := sha256.New()
		file, err := os.Open(path) //nolint:gosec // G304: walking our own build output
		if err != nil {
			return err
		}
		if _, err := io.Copy(hashWriter, file); err != nil {
			file.Close() //nolint:errcheck,gosec
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}

		hash := fmt.Sprintf("%x", hashWriter.Sum(nil))
		manifest[relPath] = hash
		return nil
	})
	if err != nil {
		return err
	}

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(destDir, "manifest.json"), manifestJSON, 0600)
}

// BuildAll builds for all platforms.
func BuildAll(config *BuildAllConfig) {
	if config == nil {
		config = &BuildAllConfig{
			Targets:   []string{"linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64", "windows/amd64", "windows/arm64"},
			OutputDir: "./dist",
			Compress:  true,
			Manifest:  true,
			Parallel:  runtime.NumCPU(),
		}
	}

	targets := parseBuildTargets(config.Targets)
	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no valid targets specified")
		os.Exit(1)
	}

	fmt.Printf("Building for %d platforms...\n", len(targets))

	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0750); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Build results
	type buildResult struct {
		platform string
		arch     string
		success  bool
		output   string
		err      error
	}

	results := make(chan buildResult, len(targets))
	var wg sync.WaitGroup

	// Limit concurrent builds
	sem := make(chan struct{}, config.Parallel)

	for _, t := range targets {
		wg.Add(1)
		sem <- struct{}{}

		go func(platform, arch string) {
			defer wg.Done()
			defer func() { <-sem }()

			outputDir := filepath.Join(config.OutputDir, fmt.Sprintf("%s-%s", platform, arch))
			cfg := &BuildConfig{
				OutputDir:    outputDir,
				Platform:     platform,
				Arch:         arch,
				StaticAssets: true,
				Minify:       true,
				Compress:     config.Compress,
				Env:          "production",
				LDFlags:      "-s -w",
			}

			fmt.Printf("\nBuilding for %s/%s...\n", platform, arch)
			_, err := BuildWithConfig(cfg)

			result := buildResult{
				platform: platform,
				arch:     arch,
				output:   outputDir,
			}

			if err != nil {
				result.success = false
				result.err = err
			} else {
				result.success = true
			}

			results <- result
		}(t.platform, t.arch)
	}

	// Close results channel when all builds complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var failed []string
	var succeeded int
	for result := range results {
		if result.success {
			fmt.Printf("✓ Built for %s/%s\n", result.platform, result.arch)
			succeeded++
		} else {
			fmt.Printf("✗ Failed for %s/%s: %v\n", result.platform, result.arch, result.err)
			failed = append(failed, fmt.Sprintf("%s/%s", result.platform, result.arch))
		}
	}

	// Summary
	fmt.Printf("\nBuild summary: %d/%d succeeded", succeeded, len(targets))
	if len(failed) > 0 {
		fmt.Printf(", %d failed: %v\n", len(failed), failed)
	} else {
		fmt.Println()
	}

	if len(failed) > 0 {
		os.Exit(1)
	}
}

type targetSpec struct {
	platform string
	arch     string
}

func parseBuildTargets(targets []string) []targetSpec {
	var result []targetSpec
	platforms := map[string]bool{
		"linux":   true,
		"darwin":  true,
		"windows": true,
		"freebsd": true,
		"openbsd": true,
	}
	arches := map[string]bool{
		"amd64": true,
		"arm64": true,
		"386":   true,
		"arm":   true,
	}

	for _, t := range targets {
		parts := strings.Split(t, "/")
		if len(parts) != 2 {
			continue
		}
		platform := strings.ToLower(strings.TrimSpace(parts[0]))
		arch := strings.ToLower(strings.TrimSpace(parts[1]))

		if platforms[platform] && arches[arch] {
			result = append(result, targetSpec{platform: platform, arch: arch})
		}
	}

	return result
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
	for i := 0; i < len(units)-1 && size >= 1024; i++ {
		size /= 1024
		unit = units[i+1]
	}

	if unit == "B" {
		return fmt.Sprintf("%d %s", info.Size(), unit)
	}
	return fmt.Sprintf("%.1f %s", size, unit)
}

// parseExecuteCommand splits an execute command like "pnpm dlx" or "npx" into
// the binary and arguments. This properly handles the command without breaking
// on whitespace in paths (unlike strings.Fields).
func parseExecuteCommand(executeCmd string) (string, []string) {
	// Simple parser that splits on first whitespace only
	// Commands like "pnpm dlx", "npx", "bun x" are supported
	parts := strings.SplitN(executeCmd, " ", 2)
	if len(parts) == 1 {
		return parts[0], nil
	}
	// For "pnpm dlx", returns ("pnpm", ["dlx"])
	// For "bun x", returns ("bun", ["x"])
	subArgs := strings.Fields(parts[1])
	return parts[0], subArgs
}

// CleanConfig holds configuration for the clean command.
type CleanConfig struct {
	DryRun      bool // Show what would be deleted
	NodeModules bool // Include node_modules
	Generated   bool // Include generated files
	Dist        bool // Include dist directory
	All         bool // Clean everything including cache
	Cache       bool // Clean gospa cache (~/.gospa)
}

// Clean removes build artifacts.
func Clean(config *CleanConfig) {
	if config == nil {
		config = &CleanConfig{
			NodeModules: true,
			Generated:   true,
			Dist:        true,
		}
	}

	action := "Cleaning"
	if config.DryRun {
		action = "Would remove"
	}
	fmt.Printf("%s build artifacts...\n", action)

	// Collect directories to clean
	var dirs []string
	if config.Dist {
		dirs = append(dirs, "dist")
	}
	if config.NodeModules {
		dirs = append(dirs, "node_modules")
	}
	if config.All {
		dirs = append(dirs, ".gospa")
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); err == nil {
			if config.DryRun {
				fmt.Printf("  %s %s (dry-run)\n", action, dir)
			} else {
				if err := os.RemoveAll(dir); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to remove %s: %v\n", dir, err)
				} else {
					fmt.Printf("✓ Removed %s\n", dir)
				}
			}
		}
	}

	// Remove generated templ files
	if config.Generated {
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
				if config.DryRun {
					fmt.Printf("  %s %s (dry-run)\n", action, cleanPath)
				} else {
					// Remove the cleaned path
					// #nosec //nolint:gosec // clean command is intended to remove files identified during walk; path validated with filepath.Abs
					if err := os.Remove(absPath); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to remove %s: %v\n", cleanPath, err)
					} else {
						fmt.Printf("✓ Removed %s\n", cleanPath)
					}
				}
			}

			return nil
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error walking directory: %v\n", err)
		}
	}

	// Clean gospa cache if requested
	if config.Cache || config.All {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			cacheDir := filepath.Join(homeDir, ".gospa")
			if _, err := os.Stat(cacheDir); err == nil {
				if config.DryRun {
					fmt.Printf("  %s %s (dry-run)\n", action, cacheDir)
				} else {
					if err := os.RemoveAll(cacheDir); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to remove %s: %v\n", cacheDir, err)
					} else {
						fmt.Printf("✓ Removed %s\n", cacheDir)
					}
				}
			}
		}
	}

	fmt.Println("✓ Clean complete!")
}

// Watch builds and watches for changes.
func Watch() {
	fmt.Println("Building and watching for changes...")

	// Initial build (synchronous)
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

	// Debounce timer for build events
	var debounceTimer *time.Timer
	const debounceInterval = 500 * time.Millisecond

	for {
		select {
		case <-sigChan:
			fmt.Println("\nStopping watcher...")
			return
		case event := <-watcher.Events:
			fmt.Printf("\nFile changed: %s\n", event.File)
			// Reset debounce timer
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.NewTimer(debounceInterval)
			// Wait for debounce interval, then build asynchronously
			go func(_ FileEvent) {
				<-time.After(debounceInterval)
				buildMu.Lock()
				if isBuilding {
					buildMu.Unlock()
					return
				}
				isBuilding = true
				buildMu.Unlock()

				Build(nil)

				buildMu.Lock()
				isBuilding = false
				buildMu.Unlock()
				fmt.Println("✓ Rebuilt")
			}(event)
		case err := <-watcher.Errors:
			fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
		}
	}
}
