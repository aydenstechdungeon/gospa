// Package fiber provides HMR (Hot Module Replacement) support for GoSPA.
package fiber

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	json "github.com/goccy/go-json"

	websocket "github.com/gofiber/contrib/v3/websocket"
	fiberpkg "github.com/gofiber/fiber/v3"
)

// HMRConfig configures the HMR system.
type HMRConfig struct {
	Enabled         bool          `json:"enabled"`
	WatchPaths      []string      `json:"watchPaths"`
	IgnorePaths     []string      `json:"ignorePaths"`
	DebounceTime    time.Duration `json:"debounceTime"`
	AllowInsecureWS bool          `json:"allowInsecureWS"`
}

// HMRManager manages hot module replacement.
type HMRManager struct {
	config        HMRConfig
	clients       map[*websocket.Conn]bool
	clientsMu     sync.RWMutex
	fileWatcher   *HMRFileWatcher
	debounceMap   map[string]time.Time
	debounceMu    sync.Mutex
	moduleStates  map[string]any
	stateMu       sync.RWMutex
	changeChan    chan HMRFileChangeEvent
	broadcastChan chan HMRMessage
	stopOnce      sync.Once
}

// HMRFileChangeEvent represents a file change event.
type HMRFileChangeEvent struct {
	Path      string    `json:"path"`
	EventType string    `json:"eventType"` // "create", "modify", "delete"
	Timestamp time.Time `json:"timestamp"`
}

// HMRMessage represents a message sent to clients.
type HMRMessage struct {
	Type         string `json:"type"` // "update", "reload", "error", "state-preserve", "connected"
	Path         string `json:"path,omitempty"`
	ModuleID     string `json:"moduleId,omitempty"`
	Event        string `json:"event,omitempty"`
	ReloadReason string `json:"reloadReason,omitempty"` // "template-safe" | "style-safe" | "runtime-break" | "config-break"
	State        any    `json:"state,omitempty"`
	Error        string `json:"error,omitempty"`
	Timestamp    int64  `json:"timestamp"`
}

// HMRUpdatePayload contains update information.
type HMRUpdatePayload struct {
	ModuleID      string `json:"moduleId"`
	Path          string `json:"path"`
	UpdateType    string `json:"updateType"` // "template", "script", "style", "full"
	Content       string `json:"content,omitempty"`
	StatePreserve bool   `json:"statePreserve"`
}

// HMRFileWatcher watches files for changes for HMR using fsnotify.
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
		// 500ms default: reduces polling CPU cost vs the old 100ms with negligible UX impact.
		// For sub-100ms responsiveness, replace HMRFileWatcher with github.com/fsnotify/fsnotify.
		config.DebounceTime = 500 * time.Millisecond
	}

	mgr := &HMRManager{
		config:        config,
		clients:       make(map[*websocket.Conn]bool),
		debounceMap:   make(map[string]time.Time),
		moduleStates:  make(map[string]any),
		changeChan:    make(chan HMRFileChangeEvent, 100),
		broadcastChan: make(chan HMRMessage, 50), // Buffered for async broadcast
	}

	if config.Enabled {
		mgr.fileWatcher = NewHMRFileWatcher(config.WatchPaths, config.IgnorePaths, mgr.changeChan)
		go mgr.processChanges()
		go mgr.broadcastLoop()      // Async broadcast processing
		go mgr.cleanupDebounceMap() // Periodic cleanup of stale debounce entries
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
	// Recreate channel so a subsequent Start() works without panicking
	fw.stopChan = make(chan struct{})
	fw.running = false
}

// watch implements the file watching loop using fsnotify for event-based watching.
func (fw *HMRFileWatcher) watch() {
	// Create new fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("[HMR] Failed to create watcher: %v\n", err)
		return
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			fmt.Printf("[HMR] Failed to close watcher: %v\n", err)
		}
	}()

	// Add all watch paths recursively
	for _, path := range fw.paths {
		// Walk directory to add all subdirectories
		_ = filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				// Check if directory should be ignored
				shouldIgnore := false
				for _, ignore := range fw.ignore {
					if matched, err := filepath.Match(ignore, info.Name()); err == nil && matched {
						shouldIgnore = true
						break
					}
				}
				if !shouldIgnore {
					if err := watcher.Add(walkPath); err != nil {
						fmt.Printf("[HMR] Failed to watch %s: %v\n", walkPath, err)
					}
				}
			}
			return nil
		})
	}

	fmt.Printf("[HMR] Watching %d paths for changes\n", len(fw.paths))

	// Event loop
	for {
		select {
		case <-fw.stopChan:
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Only handle write, create, and rename events
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
				// If a new directory is created, start watching it
				if event.Has(fsnotify.Create) {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						// Check if directory should be ignored
						shouldIgnore := false
						for _, ignore := range fw.ignore {
							if matched, err := filepath.Match(ignore, filepath.Base(event.Name)); err == nil && matched {
								shouldIgnore = true
								break
							}
						}
						if !shouldIgnore {
							if err := watcher.Add(event.Name); err != nil {
								fmt.Printf("[HMR] Failed to watch new directory %s: %v\n", event.Name, err)
							} else {
								fmt.Printf("[HMR] Now watching new directory: %s\n", event.Name)
							}
						}
					}
				}

				// Check if it's a watched file type
				if fw.isWatchedFile(event.Name) {
					// Check ignore patterns
					shouldIgnore := false
					for _, ignore := range fw.ignore {
						if matched, err := filepath.Match(ignore, filepath.Base(event.Name)); err == nil && matched {
							shouldIgnore = true
							break
						}
					}
					if !shouldIgnore {
						eventType := "modify"
						if event.Has(fsnotify.Create) {
							eventType = "create"
						}
						fw.changeChan <- HMRFileChangeEvent{
							Path:      event.Name,
							EventType: eventType,
							Timestamp: time.Now(),
						}
					}
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("[HMR] Watch error: %v\n", err)
		}
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

		// Determine update type and reason
		updateType, reloadReason := mgr.determineUpdateType(event.Path)
		moduleID := mgr.pathToModuleID(event.Path)

		// Build update message
		msg := HMRMessage{
			Type:         "update",
			Path:         event.Path,
			ModuleID:     moduleID,
			Event:        event.EventType,
			ReloadReason: reloadReason,
			Timestamp:    time.Now().UnixMilli(),
		}
		if reloadReason == "runtime-break" || reloadReason == "config-break" {
			msg.Type = "reload"
		}

		// Include state preservation info for components
		if updateType != "full" {
			mgr.stateMu.RLock()
			if state, exists := mgr.moduleStates[moduleID]; exists {
				msg.State = state
			}
			mgr.stateMu.RUnlock()
		}

		// Send to broadcast channel (non-blocking)
		select {
		case mgr.broadcastChan <- msg:
		default:
			// Broadcast channel full, skip this update
			fmt.Printf("[HMR] Warning: broadcast channel full, dropping update for %s\n", event.Path)
		}
	}
}

// broadcastLoop processes broadcasts asynchronously to avoid blocking file event processing.
func (mgr *HMRManager) broadcastLoop() {
	for msg := range mgr.broadcastChan {
		mgr.Broadcast(msg)
	}
}

// cleanupDebounceMap periodically removes stale entries from debounceMap.
// Entries older than 2x the debounce time are considered stale.
func (mgr *HMRManager) cleanupDebounceMap() {
	ticker := time.NewTicker(mgr.config.DebounceTime * 4)
	defer ticker.Stop()

	for range ticker.C {
		mgr.debounceMu.Lock()
		threshold := time.Now().Add(-mgr.config.DebounceTime * 2)
		for path, lastTime := range mgr.debounceMap {
			if lastTime.Before(threshold) {
				delete(mgr.debounceMap, path)
			}
		}
		mgr.debounceMu.Unlock()
	}
}

// determineUpdateType determines the type of update needed.
func (mgr *HMRManager) determineUpdateType(path string) (string, string) {
	switch {
	case strings.HasSuffix(path, ".templ"):
		return "template", "template-safe"
	case strings.HasSuffix(path, ".gospa"):
		// Heuristic: treat SFC source edits as template-safe to preserve stateful
		// dev loops for most markup/style changes. Runtime-breaking script edits
		// still get caught by client-side update failures and fallback reloads.
		return "template", "template-safe"
	case strings.HasSuffix(path, ".go"):
		return "full", "config-break"
	case strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".js"):
		return "script", "runtime-break"
	case strings.HasSuffix(path, ".css"):
		return "style", "style-safe"
	default:
		return "full", "config-break"
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
// Failed connections are removed after iteration to avoid mutating the map under RLock.
func (mgr *HMRManager) Broadcast(msg HMRMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	// Collect failed connections under RLock — do NOT delete during iteration
	mgr.clientsMu.RLock()
	var failed []*websocket.Conn
	for conn := range mgr.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			failed = append(failed, conn)
		}
	}
	mgr.clientsMu.RUnlock()

	// Remove failed connections under write lock
	if len(failed) > 0 {
		mgr.clientsMu.Lock()
		for _, conn := range failed {
			_ = conn.Close()
			delete(mgr.clients, conn)
		}
		mgr.clientsMu.Unlock()
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
func (mgr *HMRManager) HMREndpoint() fiberpkg.Handler {
	return websocket.New(func(c *websocket.Conn) {
		mgr.HandleWebSocket(c)
	})
}

// HMRMiddleware returns middleware that adds HMR script to HTML responses.
func (mgr *HMRManager) HMRMiddleware() fiberpkg.Handler {
	return func(c fiberpkg.Ctx) error {
		if err := c.Next(); err != nil {
			return err
		}

		// Only process HTML responses
		if !strings.Contains(c.GetRespHeader("Content-Type"), "text/html") {
			return nil
		}

		// Add HMR script before </body> using bytes for efficiency
		body := c.Response().Body()
		nonce, _ := c.Locals("gospa.csp_nonce").(string)
		hmrScript := mgr.generateHMRScript(nonce)

		// Find </body> position and create new body with injected script
		bodyTag := []byte("</body>")
		idx := bytes.LastIndex(body, bodyTag)
		if idx == -1 {
			return nil
		}

		// Build new body: content before </body> + script + </body>
		var buf bytes.Buffer
		buf.Write(body[:idx])
		buf.WriteString(hmrScript)
		buf.Write(body[idx:])

		c.Response().SetBody(buf.Bytes())
		return nil
	}
}

// generateHMRScript generates the client-side HMR script.
func (mgr *HMRManager) generateHMRScript(nonce string) string {
	nonceAttr := ""
	if nonce != "" {
		nonceAttr = fmt.Sprintf(` nonce="%s"`, nonce)
	}
	return fmt.Sprintf(`
<script%s>
(function() {
	// Use wss if the page is https AND we aren't allowing insecure connections.
	const wsProto = (window.location.protocol === 'https:' && !%v) ? 'wss://' : 'ws://';
	const ws = new WebSocket(wsProto + window.location.host + '/__hmr');
	
	ws.onopen = function() {
		console.log('[HMR] Connected');
	};
	
	ws.onmessage = function(event) {
		try {
			const msg = JSON.parse(event.data);
			handleHMRMessage(msg);
		} catch (e) {
			console.error('[HMR] Failed to parse message:', e);
		}
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
				console.log('[HMR] Full reload required. reason=', msg.reloadReason || 'unknown');
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
`, nonceAttr, mgr.config.AllowInsecureWS)
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
	mgr.stopOnce.Do(func() {
		close(mgr.changeChan)
		close(mgr.broadcastChan)
	})
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
