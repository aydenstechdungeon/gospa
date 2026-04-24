package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/aydenstechdungeon/gospa/plugin"
	"github.com/fsnotify/fsnotify"
)

const templVersion = "v0.3.1001"

// Dev starts the development server with hot reload.
func Dev(config *DevConfig) {
	fmt.Println("Starting development server...")

	// Set GOSPA_DEV in environment so BuildIslands detects dev mode
	_ = os.Setenv("GOSPA_DEV", "1")

	// Check if we're in a GoSPA project
	if !isGoSPAProject() {
		fmt.Fprintln(os.Stderr, "Error: Not a GoSPA project. Run 'gospa create' first.")
		os.Exit(1)
	}
	if os.Getenv("GOSPA_SKIP_PREFLIGHT") != "1" {
		Verify(&VerifyConfig{
			RoutesDir:  "./routes",
			Strict:     true,
			JSONOutput: false,
			Quiet:      false,
		})
	}

	// Trigger BeforeDev hook
	if err := plugin.TriggerHook(plugin.BeforeDev, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: BeforeDev hook failed: %v\n", err)
	}

	// Initial generation
	fmt.Println("Generating files (Development Mode)...")
	if err := regenerateTempl(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: templ regeneration failed: %v\n", err)
	}
	Generate(&GenerateConfig{
		InputDir:  ".",
		OutputDir: "./generated",
		DevMode:   true,
	})
	// In dev mode, output islands to static/ so the dev server can serve them directly
	_ = BuildIslands(&BuildConfig{
		OutputDir: "static",
		Env:       "development",
	}, nil)

	// Use defaults if config is nil
	if config == nil {
		config = &DevConfig{
			Port:      3000,
			Host:      "localhost",
			RoutesDir: "./routes",
			Timeout:   30 * time.Second,
			LogFormat: "text",
			Debounce:  100 * time.Millisecond,
		}
	}

	// Use startDevWithConfig which handles restart logic properly
	err := startDevWithConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// DevConfig holds configuration for the development server.
type DevConfig struct {
	Port       int           // Server port
	Host       string        // Bind address
	RoutesDir  string        // Routes directory
	WatchPaths []string      // extra directories to watch in addition to RoutesDir
	Open       bool          // open browser automatically
	Verbose    bool          // verbose logging
	NoRestart  bool          // disable automatic server restart on file changes
	Timeout    time.Duration // Server start timeout before kill
	LogFormat  string        // Log format: text, json
	HMRPort    int           // HMR WebSocket port (0 = auto)
	Proxy      string        // Proxy API requests to backend
	Debounce   time.Duration // File change debounce interval
}

// DevWithConfig starts the development server with custom configuration.
func DevWithConfig(config *DevConfig) error {
	return startDevWithConfig(config)
}

func startDevWithConfig(config *DevConfig) error {
	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Build the list of directories to watch; only include dirs that exist
	// so that components/, lib/, and static/ remain optional.
	candidateDirs := []string{config.RoutesDir, "./components", "./lib", "./state"}
	candidateDirs = append(candidateDirs, config.WatchPaths...)
	watchDirs := make([]string, 0, len(candidateDirs))
	for _, d := range candidateDirs {
		if d == "" {
			continue
		}
		if _, err := os.Stat(d); err == nil {
			watchDirs = append(watchDirs, d)
		}
	}

	// Start file watcher
	watcher := NewDevWatcher(watchDirs...)
	watcher.SetDebounce(config.Debounce)
	if err := watcher.Start(); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}
	defer watcher.Stop()

	// Start the server manager goroutine
	restartCh := make(chan struct{}, 1)

	// Initial start signal
	restartCh <- struct{}{}

	var cmdMu sync.Mutex
	var currentCmd *exec.Cmd

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-restartCh:
				// Stop existing server if any
				cmdMu.Lock()
				if currentCmd != nil && currentCmd.Process != nil {
					terminateProcess(currentCmd)
				}

				// Start new server with fresh context (not shared with other restarts)
				currentCmd = startServerProcess(context.Background(), config)
				cmdMu.Unlock()
			}
		}
	}()

	// Handle file changes
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-watcher.Events:
				handleFileChange(ctx, event, restartCh, config.NoRestart, watcher)
			case err := <-watcher.Errors:
				fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
			}
		}
	}()

	// Wait for interrupt
	<-sigChan

	// Stop the server
	cmdMu.Lock()
	if currentCmd != nil && currentCmd.Process != nil {
		terminateProcess(currentCmd)
	}
	cmdMu.Unlock()

	return nil
}

func startServerProcess(ctx context.Context, config *DevConfig) *exec.Cmd {
	// Build the server binary to a temp location for faster restarts
	tmpDir := filepath.Join(os.TempDir(), "gospa-dev")
	if err := os.MkdirAll(tmpDir, 0750); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp dir: %v\n", err)
	}
	serverBinary := filepath.Join(tmpDir, "server")

	// Build the server
	buildArgs := []string{"build", "-ldflags", "-s -w", "-o", serverBinary, "."}
	buildCmd := exec.Command("go", buildArgs...) //nolint:gosec // G204: buildArgs is constructed from trusted input
	buildCmd.Stdout = os.Stdout
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error building server: %v\n", err)
		return nil
	}

	// Run the pre-built binary
	cmd := exec.CommandContext(ctx, serverBinary) //nolint:gosec // G204: serverBinary is our own build output
	cmd.Stdout = os.Stdout
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(
		os.Environ(),
		"GOSPA_DEV=1",
		fmt.Sprintf("PORT=%d", config.Port),
		"HOST="+config.Host,
	)

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		return nil
	}

	fmt.Printf("Server running at http://%s:%d\n", config.Host, config.Port)

	// Open browser if requested
	if config.Open {
		openBrowser(fmt.Sprintf("http://%s:%d", config.Host, config.Port))
	}

	return cmd
}

// FileEvent represents a file change event.
type FileEvent struct {
	File    string
	Op      FileOp
	ModTime time.Time
}

// FileOp represents the type of file operation.
type FileOp int

const (
	// FileOpCreate is a file creation event
	FileOpCreate FileOp = iota
	// FileOpModify is a file modification event
	FileOpModify
	// FileOpDelete is a file deletion event
	FileOpDelete
	// FileOpRename is a file rename event
	FileOpRename
)

// DevWatcher watches files for changes.
type DevWatcher struct {
	dirs             []string
	Events           chan FileEvent
	Errors           chan error
	watcher          *fsnotify.Watcher
	debounce         time.Duration
	islandBuildCh    chan struct{}
	islandBuildDone  chan struct{}
	islandMu         sync.Mutex
	isBuilding       bool
	templRegenMu     sync.Mutex
	templRegenTimer  *time.Timer
	templRegenActive bool
	stopCh           chan struct{}
}

// NewDevWatcher creates a new file watcher with configurable debounce.
func NewDevWatcher(dirs ...string) *DevWatcher {
	return &DevWatcher{
		dirs:            dirs,
		Events:          make(chan FileEvent, 1000),
		Errors:          make(chan error, 10),
		debounce:        100 * time.Millisecond, // Default; overridden by config
		islandBuildCh:   make(chan struct{}, 1),
		islandBuildDone: make(chan struct{}, 1),
		stopCh:          make(chan struct{}),
	}
}

// SetDebounce sets the debounce duration for the watcher.
func (dw *DevWatcher) SetDebounce(d time.Duration) {
	dw.debounce = d
}

// Start begins watching the configured directories.
func (dw *DevWatcher) Start() error {
	var err error
	dw.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	// Add directories to watch
	for _, dir := range dw.dirs {
		if err := dw.addRecursive(dir); err != nil {
			log.Printf("Warning: failed to watch %s: %v", dir, err)
		}
	}

	// Start processing events
	go dw.run()

	return nil
}

func (dw *DevWatcher) addRecursive(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			if name == "node_modules" || name == ".git" || name == ".bin" || name == "dist" || name == "generated" {
				return filepath.SkipDir
			}
			return dw.watcher.Add(path)
		}
		return nil
	})
}

// Stop closes the watcher.
func (dw *DevWatcher) Stop() {
	// Signal stop to background goroutines
	if dw.stopCh != nil {
		close(dw.stopCh)
	}
	if dw.watcher != nil {
		_ = dw.watcher.Close()
	}
}

func (dw *DevWatcher) run() {
	lastEvents := make(map[string]time.Time)
	var mu sync.Mutex

	// Start debounced island build processor
	go dw.processIslandBuilds()

	for {
		select {
		case event, ok := <-dw.watcher.Events:
			if !ok {
				return
			}

			// Ignore generated files explicitly to avoid hot-reload loops.
			// templ outputs *_templ.go files; route/type generators use generated_* names.
			if strings.Contains(event.Name, "generated_") ||
				strings.HasSuffix(event.Name, ".templ.go") ||
				strings.HasSuffix(event.Name, "_templ.go") {
				continue
			}

			// Debounce events for the same file
			mu.Lock()
			lastTime, exists := lastEvents[event.Name]
			if exists && time.Since(lastTime) < dw.debounce {
				mu.Unlock()
				continue
			}
			lastEvents[event.Name] = time.Now()
			mu.Unlock()

			// Map fsnotify Op to FileOp
			var op FileOp
			switch {
			case event.Op&fsnotify.Create == fsnotify.Create:
				op = FileOpCreate
				// If a new directory is created, watch it too
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					_ = dw.addRecursive(event.Name)
				}
			case event.Op&fsnotify.Write == fsnotify.Write:
				op = FileOpModify
			case event.Op&fsnotify.Remove == fsnotify.Remove:
				op = FileOpDelete
			case event.Op&fsnotify.Rename == fsnotify.Rename:
				op = FileOpRename
			default:
				continue
			}

			dw.Events <- FileEvent{
				File:    event.Name,
				Op:      op,
				ModTime: time.Now(),
			}

		case err, ok := <-dw.watcher.Errors:
			if !ok {
				return
			}
			dw.Errors <- err
		}
	}
}

// processIslandBuilds handles debounced island rebuilding
func (dw *DevWatcher) processIslandBuilds() {
	debounce := 500 * time.Millisecond
	timer := time.NewTimer(debounce)
	defer timer.Stop()

	for {
		select {
		case <-dw.stopCh:
			return
		case <-dw.islandBuildCh:
			// Reset timer on new build request
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(debounce)
		case <-timer.C:
			// Perform the build
			dw.islandMu.Lock()
			dw.isBuilding = true
			dw.islandMu.Unlock()

			_ = BuildIslands(nil, nil)

			dw.islandMu.Lock()
			dw.isBuilding = false
			dw.islandMu.Unlock()
		}
	}
}

// triggerTemplRegen triggers a debounced templ regenerate
func (dw *DevWatcher) triggerTemplRegen() {
	dw.templRegenMu.Lock()
	defer dw.templRegenMu.Unlock()

	// If regeneration is already active, just signal a new request (debounce)
	if dw.templRegenActive {
		// Reset existing timer if any
		if dw.templRegenTimer != nil {
			if !dw.templRegenTimer.Stop() {
				select {
				case <-dw.templRegenTimer.C:
				default:
				}
			}
		}
		dw.templRegenTimer = time.NewTimer(200 * time.Millisecond)
		return
	}

	// Mark as active and start new timer
	dw.templRegenActive = true
	if dw.templRegenTimer != nil {
		if !dw.templRegenTimer.Stop() {
			select {
			case <-dw.templRegenTimer.C:
			default:
			}
		}
	}
	dw.templRegenTimer = time.NewTimer(200 * time.Millisecond)

	go func() {
		<-dw.templRegenTimer.C
		if err := regenerateTempl(); err != nil {
			fmt.Fprintf(os.Stderr, "Error regenerating templates: %v\n", err)
		} else {
			fmt.Printf("✓ templ regenerated\n")
		}
		dw.templRegenMu.Lock()
		dw.templRegenActive = false
		dw.templRegenMu.Unlock()
	}()
}

// triggerIslandBuild signals a debounced island rebuild
func (dw *DevWatcher) triggerIslandBuild() {
	select {
	case dw.islandBuildCh <- struct{}{}:
	default:
	}
}

func handleFileChange(ctx context.Context, event FileEvent, restartCh chan struct{}, noRestart bool, watcher *DevWatcher) {
	// Check if shutdown is in progress
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Ignore generated files defensively in case they slip past watcher-level filters.
	if strings.Contains(event.File, "generated_") ||
		strings.HasSuffix(event.File, ".templ.go") ||
		strings.HasSuffix(event.File, "_templ.go") {
		return
	}

	ext := filepath.Ext(event.File)

	switch ext {
	case ".templ", ".gospa":
		fmt.Printf("%s changed, triggering templ regeneration...\n", ext)
		watcher.triggerTemplRegen()

		// Restart server (unless disabled)
		if !noRestart {
			select {
			case restartCh <- struct{}{}:
			default:
			}
		}

	case ".go":
		fmt.Println("Go file changed, restarting server...")
		if !noRestart {
			select {
			case restartCh <- struct{}{}:
			default:
			}
		}

	case ".static", ".css", ".js":
		fmt.Println("Static file changed, triggering island rebuild...")
		if watcher != nil {
			watcher.triggerIslandBuild()
		}
	}

	// Generate types
	if strings.HasSuffix(event.File, ".go") || strings.HasSuffix(event.File, ".templ") || strings.HasSuffix(event.File, ".gospa") {
		// Run generate in development mode to enable HMR cache-busting
		Generate(&GenerateConfig{
			InputDir:  ".",
			OutputDir: "./generated",
			DevMode:   true,
		})
		// Trigger debounced island rebuild (for .go changes affecting islands)
		if watcher != nil {
			watcher.triggerIslandBuild()
		}
	}
}

func regenerateTempl() error {
	// Use a pinned templ version to avoid supply-chain drift from @latest.
	cmd := exec.Command("go", "run", "github.com/a-h/templ/cmd/templ@"+templVersion, "generate")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url) //nolint:gosec // G204: standard browser open
	case "darwin":
		cmd = exec.Command("open", url) //nolint:gosec // G204: standard browser open
	default:
		cmd = exec.Command("xdg-open", url) //nolint:gosec // G204: standard browser open
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not open browser: %v\n", err)
	}
}

func runGenerate() {
	// Run the generate command
	Generate(nil)
}

func isGoSPAProject() bool {
	// A GoSPA project only requires a go.mod. The routes, components, lib,
	// and static directories are all optional.
	_, err := os.Stat("go.mod")
	return err == nil
}

func terminateProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Signal(os.Interrupt)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		<-done
	}
}
