package fiber

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aydenstechdungeon/gospa/state"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// DevConfig holds development configuration.
type DevConfig struct {
	// Enabled enables development mode
	Enabled bool
	// RoutesDir is the directory containing route files
	RoutesDir string
	// ComponentsDir is the directory containing component files
	ComponentsDir string
	// WatchPaths are additional paths to watch for changes
	WatchPaths []string
	// IgnorePaths are paths to ignore
	IgnorePaths []string
	// Debounce is the debounce time for file changes
	Debounce time.Duration
	// OnReload is called when files change
	OnReload func()
	// StateKey is the context key for state
	StateKey string
}

// DefaultDevConfig returns default development configuration.
func DefaultDevConfig() DevConfig {
	return DevConfig{
		Enabled:       false,
		RoutesDir:     "routes",
		ComponentsDir: "components",
		WatchPaths:    []string{},
		IgnorePaths:   []string{"node_modules", ".git", "dist", "build"},
		Debounce:      100 * time.Millisecond,
		StateKey:      "gospa.state",
	}
}

// FileWatcher watches for file changes.
type FileWatcher struct {
	config  DevConfig
	changes chan string
	stop    chan struct{}
	mu      sync.Mutex
	running bool
}

// NewFileWatcher creates a new file watcher.
func NewFileWatcher(config DevConfig) *FileWatcher {
	return &FileWatcher{
		config:  config,
		changes: make(chan string, 100),
		stop:    make(chan struct{}),
	}
}

// Start starts watching for file changes.
func (w *FileWatcher) Start() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return
	}

	w.running = true

	// Watch routes directory
	go w.watchDir(w.config.RoutesDir)

	// Watch components directory
	go w.watchDir(w.config.ComponentsDir)

	// Watch additional paths
	for _, path := range w.config.WatchPaths {
		go w.watchDir(path)
	}

	// Process changes
	go w.processChanges()

	log.Println("File watcher started")
}

// Stop stops watching for file changes.
func (w *FileWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return
	}

	w.running = false
	close(w.stop)
}

// watchDir watches a directory for changes.
func (w *FileWatcher) watchDir(dir string) {
	// Get initial file states
	fileStates := make(map[string]time.Time)
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			fileStates[path] = info.ModTime()
		}
		return nil
	})

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}

				// Skip ignored paths
				for _, ignore := range w.config.IgnorePaths {
					if strings.Contains(path, ignore) {
						return nil
					}
				}

				if !info.IsDir() {
					oldTime, exists := fileStates[path]
					if !exists || info.ModTime().After(oldTime) {
						fileStates[path] = info.ModTime()
						select {
						case w.changes <- path:
						default:
							// Channel full, skip
						}
					}
				}
				return nil
			})
		}
	}
}

// processChanges processes file changes with debouncing.
func (w *FileWatcher) processChanges() {
	var pendingFiles []string
	timer := time.NewTimer(w.config.Debounce)
	defer timer.Stop()

	for {
		select {
		case <-w.stop:
			return
		case file := <-w.changes:
			pendingFiles = append(pendingFiles, file)
			timer.Reset(w.config.Debounce)
		case <-timer.C:
			if len(pendingFiles) > 0 {
				log.Printf("Files changed: %v", pendingFiles)
				if w.config.OnReload != nil {
					w.config.OnReload()
				}
				pendingFiles = nil
			}
		}
	}
}

// DevTools provides development tools.
type DevTools struct {
	config    DevConfig
	watcher   *FileWatcher
	clients   map[string]*websocket.Conn
	stateLog  []StateLogEntry
	mu        sync.RWMutex
	stateKeys map[string]bool
}

// StateLogEntry represents a state change log entry.
type StateLogEntry struct {
	Timestamp time.Time   `json:"timestamp"`
	Key       string      `json:"key"`
	OldValue  interface{} `json:"oldValue,omitempty"`
	NewValue  interface{} `json:"newValue"`
	Source    string      `json:"source"` // "client" or "server"
}

// NewDevTools creates new development tools.
func NewDevTools(config DevConfig) *DevTools {
	return &DevTools{
		config:    config,
		watcher:   NewFileWatcher(config),
		clients:   make(map[string]*websocket.Conn),
		stateLog:  make([]StateLogEntry, 0),
		stateKeys: make(map[string]bool),
	}
}

// Start starts the development tools.
func (d *DevTools) Start() {
	if !d.config.Enabled {
		return
	}

	d.watcher.Start()
	log.Println("Development tools started")
}

// Stop stops the development tools.
func (d *DevTools) Stop() {
	if !d.config.Enabled {
		return
	}

	d.watcher.Stop()
	log.Println("Development tools stopped")
}

// LogStateChange logs a state change.
func (d *DevTools) LogStateChange(key string, oldValue, newValue interface{}, source string) {
	if !d.config.Enabled {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	entry := StateLogEntry{
		Timestamp: time.Now(),
		Key:       key,
		OldValue:  oldValue,
		NewValue:  newValue,
		Source:    source,
	}

	d.stateLog = append(d.stateLog, entry)
	d.stateKeys[key] = true

	// Keep only last 1000 entries
	if len(d.stateLog) > 1000 {
		d.stateLog = d.stateLog[len(d.stateLog)-1000:]
	}

	// Broadcast to dev tools clients
	d.broadcastStateChange(entry)
}

// GetStateLog returns the state change log.
func (d *DevTools) GetStateLog() []StateLogEntry {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.stateLog
}

// GetStateKeys returns all tracked state keys.
func (d *DevTools) GetStateKeys() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	keys := make([]string, 0, len(d.stateKeys))
	for k := range d.stateKeys {
		keys = append(keys, k)
	}
	return keys
}

// broadcastStateChange broadcasts a state change to dev tools clients.
func (d *DevTools) broadcastStateChange(entry StateLogEntry) {
	data, err := json.Marshal(map[string]interface{}{
		"type":  "state_change",
		"entry": entry,
	})
	if err != nil {
		return
	}

	for _, conn := range d.clients {
		_ = conn.WriteMessage(websocket.TextMessage, data)
	}
}

// DevToolsHandler creates a WebSocket handler for dev tools.
func (d *DevTools) DevToolsHandler() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		clientID := generateComponentID()

		d.mu.Lock()
		d.clients[clientID] = c
		d.mu.Unlock()

		defer func() {
			d.mu.Lock()
			delete(d.clients, clientID)
			d.mu.Unlock()
			_ = c.Close()
		}()

		// Send initial state
		d.sendInitialState(c)

		// Handle messages
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				break
			}

			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}

			msgType, _ := msg["type"].(string)
			switch msgType {
			case "get_state_log":
				d.sendStateLog(c)
			case "get_state_keys":
				d.sendStateKeys(c)
			case "clear_log":
				d.mu.Lock()
				d.stateLog = make([]StateLogEntry, 0)
				d.mu.Unlock()
			}
		}
	})
}

func (d *DevTools) sendInitialState(c *websocket.Conn) {
	data, _ := json.Marshal(map[string]interface{}{
		"type":    "init",
		"enabled": d.config.Enabled,
	})
	_ = c.WriteMessage(websocket.TextMessage, data)
}

func (d *DevTools) sendStateLog(c *websocket.Conn) {
	d.mu.RLock()
	log := d.stateLog
	d.mu.RUnlock()

	data, _ := json.Marshal(map[string]interface{}{
		"type": "state_log",
		"log":  log,
	})
	_ = c.WriteMessage(websocket.TextMessage, data)
}

func (d *DevTools) sendStateKeys(c *websocket.Conn) {
	keys := d.GetStateKeys()
	data, _ := json.Marshal(map[string]interface{}{
		"type": "state_keys",
		"keys": keys,
	})
	_ = c.WriteMessage(websocket.TextMessage, data)
}

// DevPanelHandler creates a handler for the dev panel UI.
func (d *DevTools) DevPanelHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		html := devPanelHTML()
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	}
}

// devPanelHTML returns the dev panel HTML.
func devPanelHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>GoSPA Dev Tools</title>
	<style>
		* { margin: 0; padding: 0; box-sizing: border-box; }
		body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #1a1a2e; color: #eee; min-height: 100vh; }
		.container { max-width: 1200px; margin: 0 auto; padding: 1rem; }
		header { display: flex; justify-content: space-between; align-items: center; padding: 1rem; background: #16213e; border-radius: 8px; margin-bottom: 1rem; }
		h1 { font-size: 1.5rem; }
		.status { display: flex; align-items: center; gap: 0.5rem; }
		.status-dot { width: 10px; height: 10px; border-radius: 50%; background: #4ade80; }
		.status-dot.inactive { background: #ef4444; }
		.panel { background: #16213e; border-radius: 8px; padding: 1rem; margin-bottom: 1rem; }
		.panel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
		.panel-title { font-size: 1.1rem; font-weight: 600; }
		.btn { padding: 0.5rem 1rem; border-radius: 6px; border: none; cursor: pointer; font-size: 0.9rem; transition: all 0.2s; }
		.btn-primary { background: #e94560; color: white; }
		.btn-primary:hover { background: #ff6b6b; }
		.btn-secondary { background: #333; color: #ccc; }
		.btn-secondary:hover { background: #444; }
		.state-keys { display: flex; flex-wrap: wrap; gap: 0.5rem; margin-bottom: 1rem; }
		.state-key { padding: 0.25rem 0.75rem; background: #0f0f23; border-radius: 4px; font-family: monospace; font-size: 0.85rem; }
		.log-container { max-height: 500px; overflow-y: auto; }
		.log-entry { display: grid; grid-template-columns: 100px 150px 1fr 1fr 80px; gap: 0.5rem; padding: 0.5rem; border-bottom: 1px solid #333; font-size: 0.85rem; }
		.log-entry:hover { background: #1a1a2e; }
		.log-time { color: #888; font-family: monospace; }
		.log-key { color: #4ade80; font-family: monospace; }
		.log-value { font-family: monospace; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
		.log-source { text-align: center; }
		.log-source.client { color: #60a5fa; }
		.log-source.server { color: #f59e0b; }
		.empty { text-align: center; padding: 2rem; color: #666; }
	</style>
</head>
<body>
	<div class="container">
		<header>
			<h1>GoSPA Dev Tools</h1>
			<div class="status">
				<div class="status-dot" id="statusDot"></div>
				<span id="statusText">Connected</span>
			</div>
		</header>

		<div class="panel">
			<div class="panel-header">
				<span class="panel-title">State Keys</span>
				<button class="btn btn-secondary" onclick="refreshKeys()">Refresh</button>
			</div>
			<div class="state-keys" id="stateKeys">
				<span class="empty">No state keys tracked</span>
			</div>
		</div>

		<div class="panel">
			<div class="panel-header">
				<span class="panel-title">State Change Log</span>
				<div>
					<button class="btn btn-secondary" onclick="clearLog()">Clear</button>
					<button class="btn btn-primary" onclick="refreshLog()">Refresh</button>
				</div>
			</div>
			<div class="log-container" id="logContainer">
				<div class="empty">No state changes logged</div>
			</div>
		</div>
	</div>

	<script>
		let ws = null;
		let connected = false;

		function connect() {
			const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
			ws = new WebSocket(protocol + '//' + window.location.host + '/_gospa/dev/ws');

			ws.onopen = function() {
				connected = true;
				updateStatus(true);
			};

			ws.onclose = function() {
				connected = false;
				updateStatus(false);
				setTimeout(connect, 1000);
			};

			ws.onmessage = function(event) {
				const data = JSON.parse(event.data);
				handleMessage(data);
			};
		}

		function updateStatus(connected) {
			const dot = document.getElementById('statusDot');
			const text = document.getElementById('statusText');
			if (connected) {
				dot.classList.remove('inactive');
				text.textContent = 'Connected';
			} else {
				dot.classList.add('inactive');
				text.textContent = 'Disconnected';
			}
		}

		function handleMessage(data) {
			switch (data.type) {
				case 'state_change':
					addLogEntry(data.entry);
					break;
				case 'state_log':
					renderLog(data.log);
					break;
				case 'state_keys':
					renderKeys(data.keys);
					break;
			}
		}

		function addLogEntry(entry) {
			const container = document.getElementById('logContainer');
			const empty = container.querySelector('.empty');
			if (empty) empty.remove();

			const div = document.createElement('div');
			div.className = 'log-entry';
			div.innerHTML = '<span class="log-time">' + new Date(entry.timestamp).toLocaleTimeString() + '</span>' +
				'<span class="log-key">' + entry.key + '</span>' +
				'<span class="log-value" title="' + entry.oldValue + '">' + JSON.stringify(entry.oldValue) + '</span>' +
				'<span class="log-value" title="' + entry.newValue + '">' + JSON.stringify(entry.newValue) + '</span>' +
				'<span class="log-source ' + entry.source + '">' + entry.source + '</span>';
			container.insertBefore(div, container.firstChild);
		}

		function renderLog(log) {
			const container = document.getElementById('logContainer');
			container.innerHTML = '';
			if (log.length === 0) {
				container.innerHTML = '<div class="empty">No state changes logged</div>';
				return;
			}
			for (var i = log.length - 1; i >= 0; i--) {
				addLogEntry(log[i]);
			}
		}

		function renderKeys(keys) {
			const container = document.getElementById('stateKeys');
			if (keys.length === 0) {
				container.innerHTML = '<span class="empty">No state keys tracked</span>';
				return;
			}
			var html = '';
			for (var i = 0; i < keys.length; i++) {
				html += '<span class="state-key">' + keys[i] + '</span>';
			}
			container.innerHTML = html;
		}

		function refreshLog() {
			if (ws && connected) {
				ws.send(JSON.stringify({ type: 'get_state_log' }));
			}
		}

		function refreshKeys() {
			if (ws && connected) {
				ws.send(JSON.stringify({ type: 'get_state_keys' }));
			}
		}

		function clearLog() {
			if (ws && connected) {
				ws.send(JSON.stringify({ type: 'clear_log' }));
			}
			document.getElementById('logContainer').innerHTML = '<div class="empty">No state changes logged</div>';
		}

		connect();
		refreshKeys();
		refreshLog();
	</script>
</body>
</html>`
}

// DebugMiddleware logs requests and state changes.
func DebugMiddleware(devTools *DevTools) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		// Log request
		log.Printf("[%s] %s %d %v", c.Method(), c.Path(), c.Response().StatusCode(), time.Since(start))

		return err
	}
}

// StateInspectorMiddleware inspects state changes.
func StateInspectorMiddleware(devTools *DevTools, config Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get state before
		var beforeState map[string]interface{}
		if stateMap, ok := c.Locals(config.StateKey).(*state.StateMap); ok && stateMap != nil {
			if jsonData, err := stateMap.ToJSON(); err == nil {
				_ = json.Unmarshal([]byte(jsonData), &beforeState)
			}
		}

		err := c.Next()

		// Get state after
		if stateMap, ok := c.Locals(config.StateKey).(*state.StateMap); ok && stateMap != nil {
			if jsonData, err := stateMap.ToJSON(); err == nil {
				var afterState map[string]interface{}
				_ = json.Unmarshal([]byte(jsonData), &afterState)

				// Compare and log changes
				for key, newVal := range afterState {
					oldVal := beforeState[key]
					if fmt.Sprintf("%v", oldVal) != fmt.Sprintf("%v", newVal) {
						devTools.LogStateChange(key, oldVal, newVal, "server")
					}
				}
			}
		}

		return err
	}
}
