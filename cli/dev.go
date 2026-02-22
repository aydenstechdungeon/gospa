package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/aydenstechdungeon/gospa/plugin"
)

// Dev starts the development server with hot reload.
func Dev() {
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
	fmt.Println("Generating files...")
	_ = regenerateTempl()
	runGenerate()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start file watcher
	watcher := NewDevWatcher("./routes", "./components")
	if err := watcher.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting watcher: %v\n", err)
		os.Exit(1)
	}
	defer watcher.Stop()

	// Start the server
	serverCmd := startServer(ctx)

	// Handle file changes
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-watcher.Events:
				fmt.Printf("\nFile changed: %s\n", event.File)
				handleFileChange(event, serverCmd, ctx)
			case err := <-watcher.Errors:
				fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
			}
		}
	}()

	// Wait for interrupt
	<-sigChan
	fmt.Println("\nShutting down...")

	// Stop the server
	if serverCmd != nil && serverCmd.Process != nil {
		_ = serverCmd.Process.Signal(os.Interrupt)
		_ = serverCmd.Wait()
	}
}

// DevConfig holds configuration for the development server.
type DevConfig struct {
	Port          int
	Host          string
	RoutesDir     string
	ComponentsDir string
	WatchPaths    []string
	IgnorePaths   []string
	Debounce      time.Duration
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

	// Start file watcher
	watcher := NewDevWatcher(config.RoutesDir, config.ComponentsDir)
	if err := watcher.Start(); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}
	defer watcher.Stop()

	// Start the server
	serverCmd := startServerWithConfig(ctx, config)

	// Handle file changes
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-watcher.Events:
				handleFileChange(event, serverCmd, ctx)
			case err := <-watcher.Errors:
				fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
			}
		}
	}()

	// Wait for interrupt
	<-sigChan

	// Stop the server
	if serverCmd != nil && serverCmd.Process != nil {
		_ = serverCmd.Process.Signal(os.Interrupt)
		_ = serverCmd.Wait()
	}

	return nil
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
	FileOpCreate FileOp = iota
	FileOpModify
	FileOpDelete
	FileOpRename
)

// DevWatcher watches files for changes.
type DevWatcher struct {
	dirs      []string
	Events    chan FileEvent
	Errors    chan error
	stop      chan struct{}
	interval  time.Duration
	fileTimes map[string]time.Time
}

// NewDevWatcher creates a new file watcher.
func NewDevWatcher(dirs ...string) *DevWatcher {
	return &DevWatcher{
		dirs:      dirs,
		Events:    make(chan FileEvent, 100),
		Errors:    make(chan error, 10),
		stop:      make(chan struct{}),
		interval:  500 * time.Millisecond,
		fileTimes: make(map[string]time.Time),
	}
}

// Start starts the file watcher.
func (w *DevWatcher) Start() error {
	// Initial scan
	for _, dir := range w.dirs {
		if err := w.scanDir(dir); err != nil {
			return err
		}
	}

	// Start watching
	go w.watch()

	return nil
}

// Stop stops the file watcher.
func (w *DevWatcher) Stop() {
	close(w.stop)
}

func (w *DevWatcher) watch() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			for _, dir := range w.dirs {
				w.checkDir(dir)
			}
		}
	}
}

func (w *DevWatcher) scanDir(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			w.fileTimes[path] = info.ModTime()
		}

		return nil
	})
}

func (w *DevWatcher) checkDir(dir string) {
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Ignore errors
		}

		if info.IsDir() {
			return nil
		}

		oldTime, exists := w.fileTimes[path]
		modTime := info.ModTime()

		if !exists {
			// New file
			w.fileTimes[path] = modTime
			w.Events <- FileEvent{
				File:    path,
				Op:      FileOpCreate,
				ModTime: modTime,
			}
		} else if !modTime.Equal(oldTime) {
			// Modified file
			w.fileTimes[path] = modTime
			w.Events <- FileEvent{
				File:    path,
				Op:      FileOpModify,
				ModTime: modTime,
			}
		}

		return nil
	})

	// Check for deleted files
	for path := range w.fileTimes {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			delete(w.fileTimes, path)
			w.Events <- FileEvent{
				File:    path,
				Op:      FileOpDelete,
				ModTime: time.Now(),
			}
		}
	}
}

func startServer(ctx context.Context) *exec.Cmd {
	return startServerWithConfig(ctx, &DevConfig{
		Port: 3000,
		Host: "localhost",
	})
}

func startServerWithConfig(ctx context.Context, config *DevConfig) *exec.Cmd {
	// Build and run the server
	args := []string{"run", "."}
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "GOSPA_DEV=1")

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		return nil
	}

	fmt.Printf("Server running at http://%s:%d\n", config.Host, config.Port)

	return cmd
}

func handleFileChange(event FileEvent, serverCmd *exec.Cmd, ctx context.Context) {
	ext := filepath.Ext(event.File)

	switch ext {
	case ".templ":
		fmt.Println("Regenerating templates...")
		if err := regenerateTempl(); err != nil {
			fmt.Fprintf(os.Stderr, "Error regenerating templates: %v\n", err)
			return
		}
		fmt.Println("✓ Templates regenerated")

		// Restart server
		if serverCmd != nil && serverCmd.Process != nil {
			_ = serverCmd.Process.Signal(os.Interrupt)
			_ = serverCmd.Wait()
		}

	case ".go":
		fmt.Println("Go file changed, restarting server...")

		// Restart server
		if serverCmd != nil && serverCmd.Process != nil {
			_ = serverCmd.Process.Signal(os.Interrupt)
			_ = serverCmd.Wait()
		}

	case ".css", ".js":
		// Static files don't need server restart
		fmt.Println("Static file changed, browser will reload")
	}

	// Generate types
	if strings.HasSuffix(event.File, ".go") || strings.HasSuffix(event.File, ".templ") {
		runGenerate()
	}
}

func regenerateTempl() error {
	// Using go run to ensure it works even if templ is not in the PATH
	cmd := exec.Command("go", "run", "github.com/a-h/templ/cmd/templ@latest", "generate")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runGenerate() {
	// Run the generate command
	Generate()
}

func isGoSPAProject() bool {
	// Check for go.mod
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		return false
	}

	// Check for routes directory
	if _, err := os.Stat("routes"); os.IsNotExist(err) {
		return false
	}

	return true
}

// HotReload represents a hot reload event.
type HotReload struct {
	Type    string `json:"type"`
	File    string `json:"file"`
	Message string `json:"message"`
}

// DevServer represents the development server.
type DevServer struct {
	config   *DevConfig
	watcher  *DevWatcher
	server   *exec.Cmd
	clients  map[string]bool
	reloadCh chan HotReload
}

// NewDevServer creates a new development server.
func NewDevServer(config *DevConfig) *DevServer {
	return &DevServer{
		config:   config,
		clients:  make(map[string]bool),
		reloadCh: make(chan HotReload, 100),
	}
}

// Start starts the development server.
func (s *DevServer) Start() error {
	// Start watcher
	s.watcher = NewDevWatcher(s.config.RoutesDir, s.config.ComponentsDir)
	if err := s.watcher.Start(); err != nil {
		return err
	}

	// Start server
	s.server = startServerWithConfig(context.Background(), s.config)

	// Handle reloads
	go s.handleReloads()

	return nil
}

// Stop stops the development server.
func (s *DevServer) Stop() {
	if s.watcher != nil {
		s.watcher.Stop()
	}

	if s.server != nil && s.server.Process != nil {
		_ = s.server.Process.Signal(os.Interrupt)
		_ = s.server.Wait()
	}
}

func (s *DevServer) handleReloads() {
	for event := range s.watcher.Events {
		reload := HotReload{
			File: event.File,
		}

		switch event.Op {
		case FileOpCreate:
			reload.Type = "create"
			reload.Message = "File created"
		case FileOpModify:
			reload.Type = "modify"
			reload.Message = "File modified"
		case FileOpDelete:
			reload.Type = "delete"
			reload.Message = "File deleted"
		}

		s.reloadCh <- reload
	}
}

// Broadcast sends a reload event to all connected clients.
func (s *DevServer) Broadcast(reload HotReload) {
	s.reloadCh <- reload
}
