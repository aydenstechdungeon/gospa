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

	// Check if we're in a GoSPA project
	if !isGoSPAProject() {
		fmt.Fprintln(os.Stderr, "Error: Not a GoSPA project. Run 'gospa create' first.")
		os.Exit(1)
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
	_ = BuildIslands(nil, nil)

	// Use defaults if config is nil
	if config == nil {
		config = &DevConfig{
			Port:      3000,
			Host:      "localhost",
			RoutesDir: "./routes",
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
	Port       int
	Host       string
	RoutesDir  string
	WatchPaths []string // extra directories to watch in addition to RoutesDir
	Open       bool     // open browser automatically
	Verbose    bool     // verbose logging
	NoRestart  bool     // disable automatic server restart on file changes
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
	if err := watcher.Start(); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}
	defer watcher.Stop()

	// Start the server manager goroutine
	restartCh := make(chan struct{}, 1)

	// Create context for running the server process
	serverCtx, cancelServer := context.WithCancel(ctx)
	defer cancelServer()

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

				// Start new server
				currentCmd = startServerProcess(serverCtx, config)
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
				handleFileChange(ctx, event, restartCh, config.NoRestart)
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
	// Build and run the server
	args := []string{"run", "."}
	if config.Verbose {
		args = append(args, "-v")
	}
	cmd := exec.CommandContext(ctx, "go", args...)
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
	dirs     []string
	Events   chan FileEvent
	Errors   chan error
	watcher  *fsnotify.Watcher
	debounce time.Duration
}

// NewDevWatcher creates a new file watcher.
func NewDevWatcher(dirs ...string) *DevWatcher {
	return &DevWatcher{
		dirs:     dirs,
		Events:   make(chan FileEvent, 1000),
		Errors:   make(chan error, 10),
		debounce: 100 * time.Millisecond,
	}
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
	if dw.watcher != nil {
		_ = dw.watcher.Close()
	}
}

func (dw *DevWatcher) run() {
	lastEvents := make(map[string]time.Time)
	var mu sync.Mutex

	for {
		select {
		case event, ok := <-dw.watcher.Events:
			if !ok {
				return
			}

			// Ignore generated files explicitly
			if strings.Contains(event.Name, "generated_") || strings.HasSuffix(event.Name, ".templ.go") {
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


func handleFileChange(_ context.Context, event FileEvent, restartCh chan struct{}, noRestart bool) {
	ext := filepath.Ext(event.File)

	switch ext {
	case ".templ", ".gospa":
		fmt.Printf("%s changed, regenerating...\n", ext)
		if err := regenerateTempl(); err != nil {
			fmt.Fprintf(os.Stderr, "Error regenerating templates: %v\n", err)
			return
		}
		fmt.Printf("✓ %s regenerated\n", ext)

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
		fmt.Println("Static file changed, browser will reload")
		_ = BuildIslands(nil, nil)
	}

	// Generate types
	if strings.HasSuffix(event.File, ".go") || strings.HasSuffix(event.File, ".templ") || strings.HasSuffix(event.File, ".gospa") {
		// Run generate in development mode to enable HMR cache-busting
		Generate(&GenerateConfig{
			InputDir:  ".",
			OutputDir: "./generated",
			DevMode:   true,
		})
		_ = BuildIslands(nil, nil)
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
