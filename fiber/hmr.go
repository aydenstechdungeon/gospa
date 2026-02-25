// Package fiber provides HMR (Hot Module Replacement) support for GoSPA.
package fiber

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// HMRConfig configures the HMR system.
type HMRConfig struct {
	Enabled      bool          `json:"enabled"`
	WatchPaths   []string      `json:"watchPaths"`
	IgnorePaths  []string      `json:"ignorePaths"`
	DebounceTime time.Duration `json:"debounceTime"`
	BroadcastAll bool          `json:"broadcastAll"`
}

// HMRManager manages hot module replacement.
type HMRManager struct {
	config       HMRConfig
	clients      map[*websocket.Conn]bool
	clientsMu    sync.RWMutex
	fileWatcher  *HMRFileWatcher
	debounceMap  map[string]time.Time
	debounceMu   sync.Mutex
	moduleStates map[string]any
	stateMu      sync.RWMutex
	changeChan   chan HMRFileChangeEvent
}

// HMRFileChangeEvent represents a file change event.
type HMRFileChangeEvent struct {
	Path        string    `json:"path"`
	EventType   string    `json:"eventType"` // "create", "modify", "delete"
	Timestamp   time.Time `json:"timestamp"`
	ContentHash string    `json:"contentHash,omitempty"`
}

// HMRMessage represents a message sent to clients.
type HMRMessage struct {
	Type      string `json:"type"` // "update", "reload", "error", "state-preserve", "connected"
	Path      string `json:"path,omitempty"`
	ModuleID  string `json:"moduleId,omitempty"`
	Event     string `json:"event,omitempty"`
	State     any    `json:"state,omitempty"`
	Error     string `json:"error,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// HMRUpdatePayload contains update information.
type HMRUpdatePayload struct {
	ModuleID      string `json:"moduleId"`
	Path          string `json:"path"`
	UpdateType    string `json:"updateType"` // "template", "script", "style", "full"
	Content       string `json:"content,omitempty"`
	StatePreserve bool   `json:"statePreserve"`
}

// HMRFileWatcher watches files for changes for HMR.
type HMRFileWatcher struct {
	paths      []string
	ignore     []string
	changeChan chan HMRFileChangeEvent
	stopChan   chan struct{}
	running    bool
	mu         sync.Mutex
}

// NewHMRManager creates a new HMR manager.
func NewHMRManager(config HMRConfig) *HMRManager {
	if config.DebounceTime == 0 {
		config.DebounceTime = 100 * time.Millisecond
	}

	mgr := &HMRManager{
		config:       config,
		clients:      make(map[*websocket.Conn]bool),
		debounceMap:  make(map[string]time.Time),
		moduleStates: make(map[string]any),
		changeChan:   make(chan HMRFileChangeEvent, 100),
	}

	if config.Enabled {
		mgr.fileWatcher = NewHMRFileWatcher(config.WatchPaths, config.IgnorePaths, mgr.changeChan)
		go mgr.processChanges()
	}

	return mgr
}

// NewHMRFileWatcher creates a new HMR file watcher.
func NewHMRFileWatcher(paths, ignore []string, changeChan chan HMRFileChangeEvent) *HMRFileWatcher {
	return &HMRFileWatcher{
		paths:      paths,
		ignore:     ignore,
		changeChan: changeChan,
		stopChan:   make(chan struct{}),
	}
}

// Start begins watching files.
func (fw *HMRFileWatcher) Start() {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.running {
		return
	}

	fw.running = true
	go fw.watch()
}

// Stop stops watching files.
func (fw *HMRFileWatcher) Stop() {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if !fw.running {
		return
	}

	close(fw.stopChan)
	fw.running = false
}

// watch implements the file watching loop.
func (fw *HMRFileWatcher) watch() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	fileModTimes := make(map[string]time.Time)

	for {
		select {
		case <-fw.stopChan:
			return
		case <-ticker.C:
			fw.checkFiles(fileModTimes)
		}
	}
}

// checkFiles checks for file modifications.
func (fw *HMRFileWatcher) checkFiles(modTimes map[string]time.Time) {
	for _, watchPath := range fw.paths {
		_ = filepath.Walk(watchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			// Check ignore patterns
			for _, ignore := range fw.ignore {
				if strings.Contains(path, ignore) {
					return nil
				}
			}

			// Check for relevant file types
			if !fw.isWatchedFile(path) {
				return nil
			}

			currentMod := info.ModTime()
			if lastMod, exists := modTimes[path]; exists {
				if currentMod.After(lastMod) {
					fw.changeChan <- HMRFileChangeEvent{
						Path:      path,
						EventType: "modify",
						Timestamp: time.Now(),
					}
				}
			}
			modTimes[path] = currentMod

			return nil
		})
	}
}

// isWatchedFile checks if a file should be watched.
func (fw *HMRFileWatcher) isWatchedFile(path string) bool {
	watchedExts := []string{
		".templ", ".go", ".ts", ".js", ".css", ".html",
		".svelte", ".vue", ".jsx", ".tsx",
	}

	for _, ext := range watchedExts {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

// processChanges processes file change events.
func (mgr *HMRManager) processChanges() {
	for event := range mgr.changeChan {
		// Debounce
		mgr.debounceMu.Lock()
		lastTime, exists := mgr.debounceMap[event.Path]
		if exists && time.Since(lastTime) < mgr.config.DebounceTime {
			mgr.debounceMu.Unlock()
			continue
		}
		mgr.debounceMap[event.Path] = time.Now()
		mgr.debounceMu.Unlock()

		// Determine update type
		updateType := mgr.determineUpdateType(event.Path)
		moduleID := mgr.pathToModuleID(event.Path)

		// Broadcast update to clients
		msg := HMRMessage{
			Type:      "update",
			Path:      event.Path,
			ModuleID:  moduleID,
			Event:     event.EventType,
			Timestamp: time.Now().UnixMilli(),
		}

		// Include state preservation info for components
		if updateType != "full" {
			mgr.stateMu.RLock()
			if state, exists := mgr.moduleStates[moduleID]; exists {
				msg.State = state
			}
			mgr.stateMu.RUnlock()
		}

		mgr.Broadcast(msg)
	}
}

// determineUpdateType determines the type of update needed.
func (mgr *HMRManager) determineUpdateType(path string) string {
	switch {
	case strings.HasSuffix(path, ".templ"):
		return "template"
	case strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".js"):
		return "script"
	case strings.HasSuffix(path, ".css"):
		return "style"
	default:
		return "full"
	}
}

// pathToModuleID converts a file path to a module ID.
func (mgr *HMRManager) pathToModuleID(path string) string {
	// Normalize path separators
	path = filepath.ToSlash(path)

	// Remove common prefixes
	for _, watchPath := range mgr.config.WatchPaths {
		watchPath = filepath.ToSlash(watchPath)
		if strings.HasPrefix(path, watchPath) {
			path = strings.TrimPrefix(path, watchPath)
			break
		}
	}

	// Remove leading slash and extension
	path = strings.TrimPrefix(path, "/")
	ext := filepath.Ext(path)
	path = strings.TrimSuffix(path, ext)

	return path
}

// RegisterClient registers a new WebSocket client.
func (mgr *HMRManager) RegisterClient(conn *websocket.Conn) {
	mgr.clientsMu.Lock()
	defer mgr.clientsMu.Unlock()
	mgr.clients[conn] = true
}

// UnregisterClient removes a WebSocket client.
func (mgr *HMRManager) UnregisterClient(conn *websocket.Conn) {
	mgr.clientsMu.Lock()
	defer mgr.clientsMu.Unlock()
	delete(mgr.clients, conn)
}

// Broadcast sends a message to all connected clients.
func (mgr *HMRManager) Broadcast(msg HMRMessage) {
	mgr.clientsMu.RLock()
	defer mgr.clientsMu.RUnlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	for conn := range mgr.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			conn.Close()
			delete(mgr.clients, conn)
		}
	}
}

// PreserveState saves state for a module.
func (mgr *HMRManager) PreserveState(moduleID string, state any) {
	mgr.stateMu.Lock()
	defer mgr.stateMu.Unlock()
	mgr.moduleStates[moduleID] = state
}

// GetState retrieves preserved state for a module.
func (mgr *HMRManager) GetState(moduleID string) (any, bool) {
	mgr.stateMu.RLock()
	defer mgr.stateMu.RUnlock()
	state, exists := mgr.moduleStates[moduleID]
	return state, exists
}

// ClearState removes preserved state for a module.
func (mgr *HMRManager) ClearState(moduleID string) {
	mgr.stateMu.Lock()
	defer mgr.stateMu.Unlock()
	delete(mgr.moduleStates, moduleID)
}

// HandleWebSocket handles WebSocket connections for HMR.
func (mgr *HMRManager) HandleWebSocket(c *websocket.Conn) {
	mgr.RegisterClient(c)
	defer mgr.UnregisterClient(c)

	// Send initial connection message
	mgr.sendWelcome(c)

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			break
		}

		var data map[string]any
		if err := json.Unmarshal(msg, &data); err != nil {
			continue
		}

		mgr.handleClientMessage(c, data)
	}
}

// sendWelcome sends initial connection message.
func (mgr *HMRManager) sendWelcome(c *websocket.Conn) {
	msg := HMRMessage{
		Type:      "connected",
		Timestamp: time.Now().UnixMilli(),
	}
	data, _ := json.Marshal(msg)
	_ = c.WriteMessage(websocket.TextMessage, data)
}

// handleClientMessage handles messages from clients.
func (mgr *HMRManager) handleClientMessage(c *websocket.Conn, data map[string]any) {
	msgType, ok := data["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "state-preserve":
		if moduleID, ok := data["moduleId"].(string); ok {
			if state, ok := data["state"]; ok {
				mgr.PreserveState(moduleID, state)
			}
		}

	case "state-request":
		if moduleID, ok := data["moduleId"].(string); ok {
			if state, exists := mgr.GetState(moduleID); exists {
				msg := HMRMessage{
					Type:      "state-preserve",
					ModuleID:  moduleID,
					State:     state,
					Timestamp: time.Now().UnixMilli(),
				}
				data, _ := json.Marshal(msg)
				_ = c.WriteMessage(websocket.TextMessage, data)
			}
		}

	case "error":
		if errMsg, ok := data["error"].(string); ok {
			// Log client errors
			fmt.Printf("[HMR] Client error: %s\n", errMsg)
		}
	}
}

// HMREndpoint returns a Fiber handler for HMR WebSocket.
func (mgr *HMRManager) HMREndpoint() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		mgr.HandleWebSocket(c)
	})
}

// HMRMiddleware returns middleware that adds HMR script to HTML responses.
func (mgr *HMRManager) HMRMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Only process HTML responses
		if !strings.Contains(string(c.Response().Header.Peek("Content-Type")), "text/html") {
			return c.Next()
		}

		// Add HMR script before </body>
		body := string(c.Body())
		hmrScript := mgr.generateHMRScript()
		body = strings.Replace(body, "</body>", hmrScript+"</body>", 1)

		return c.SendString(body)
	}
}

// generateHMRScript generates the client-side HMR script.
func (mgr *HMRManager) generateHMRScript() string {
	return `
<script>
(function() {
	const ws = new WebSocket('ws://' + window.location.host + '/__hmr');
	
	ws.onopen = function() {
		console.log('[HMR] Connected');
	};
	
	ws.onmessage = function(event) {
		const msg = JSON.parse(event.data);
		handleHMRMessage(msg);
	};
	
	ws.onclose = function() {
		console.log('[HMR] Disconnected, reconnecting...');
		setTimeout(function() {
			window.location.reload();
		}, 1000);
	};
	
	function handleHMRMessage(msg) {
		switch(msg.type) {
			case 'update':
				console.log('[HMR] Update:', msg.moduleId);
				if (window.__gospaHMR) {
					window.__gospaHMR.handleUpdate(msg);
				} else {
					window.location.reload();
				}
				break;
			case 'reload':
				console.log('[HMR] Full reload required');
				window.location.reload();
				break;
			case 'error':
				console.error('[HMR] Error:', msg.error);
				if (window.__gospaHMRError) {
					window.__gospaHMRError(msg.error);
				}
				break;
		}
	}
	
	// Preserve state before unload
	window.addEventListener('beforeunload', function() {
		if (window.__gospaPreserveState) {
			const states = window.__gospaPreserveState();
			for (const [moduleId, state] of Object.entries(states)) {
				ws.send(JSON.stringify({
					type: 'state-preserve',
					moduleId: moduleId,
					state: state
				}));
			}
		}
	});
})();
</script>
`
}

// Start begins HMR operation.
func (mgr *HMRManager) Start() {
	if mgr.config.Enabled && mgr.fileWatcher != nil {
		mgr.fileWatcher.Start()
	}
}

// Stop stops HMR operation.
func (mgr *HMRManager) Stop() {
	if mgr.fileWatcher != nil {
		mgr.fileWatcher.Stop()
	}
	close(mgr.changeChan)
}

// Global HMR manager
var globalHMRManager *HMRManager

// InitHMR initializes the global HMR manager.
func InitHMR(config HMRConfig) *HMRManager {
	globalHMRManager = NewHMRManager(config)
	return globalHMRManager
}

// GetHMR returns the global HMR manager.
func GetHMR() *HMRManager {
	return globalHMRManager
}
