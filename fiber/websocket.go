package fiber

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/aydenstechdungeon/gospa/state"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

const (
	// Time allowed to keep an idle connection alive.
	pongWait = 60 * time.Second
	// Send pings to peer with this period.
	pingPeriod = (pongWait * 9) / 10
)

// SessionStore maps session tokens to client IDs for secure HTTP state sync.
type SessionStore struct {
	sessions map[string]string // token -> clientID
	mu       sync.RWMutex
}

// NewSessionStore creates a new session store.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]string),
	}
}

// CreateSession creates a new session token for a client ID.
// Returns the session token, or empty string if random generation fails.
func (s *SessionStore) CreateSession(clientID string) string {
	token := generateSecureToken()
	if token == "" {
		// Random generation failed - this is a critical error
		// Return empty string to indicate failure
		return ""
	}
	s.mu.Lock()
	s.sessions[token] = clientID
	s.mu.Unlock()
	return token
}

// ValidateSession returns the client ID for a valid session token.
func (s *SessionStore) ValidateSession(token string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	clientID, ok := s.sessions[token]
	return clientID, ok
}

// RemoveSession removes a session token.
func (s *SessionStore) RemoveSession(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

// RemoveClientSessions removes all sessions for a client ID.
func (s *SessionStore) RemoveClientSessions(clientID string) {
	s.mu.Lock()
	for token, id := range s.sessions {
		if id == clientID {
			delete(s.sessions, token)
		}
	}
	s.mu.Unlock()
}

// generateSecureToken generates a cryptographically secure random token.
// Returns an empty string if random generation fails (which should never happen
// on modern systems with a properly functioning OS).
func generateSecureToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// This should never happen on modern systems. If it does, log and return
		// empty string - callers should handle this case appropriately.
		log.Printf("CRITICAL: crypto/rand.Read failed: %v", err)
		return ""
	}
	return hex.EncodeToString(b)
}

// Global session store for HTTP state sync.
var globalSessionStore = NewSessionStore()

// ClientStateStore persists client state by client ID for session restoration.
type ClientStateStore struct {
	states map[string]*state.StateMap
	mu     sync.RWMutex
}

// NewClientStateStore creates a new client state store.
func NewClientStateStore() *ClientStateStore {
	return &ClientStateStore{
		states: make(map[string]*state.StateMap),
	}
}

// Save saves a client's state.
func (s *ClientStateStore) Save(clientID string, sm *state.StateMap) {
	s.mu.Lock()
	s.states[clientID] = sm
	s.mu.Unlock()
}

// Get retrieves a client's state.
func (s *ClientStateStore) Get(clientID string) (*state.StateMap, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sm, ok := s.states[clientID]
	return sm, ok
}

// Remove removes a client's state.
func (s *ClientStateStore) Remove(clientID string) {
	s.mu.Lock()
	delete(s.states, clientID)
	s.mu.Unlock()
}

// Global client state store for session persistence.
var globalClientStateStore = NewClientStateStore()

// WSClient represents a connected WebSocket client.
type WSClient struct {
	ID        string
	SessionID string
	Conn      *websocket.Conn
	Send      chan []byte
	State     *state.StateMap
	mu        sync.Mutex
	closed    bool
}

// WSMessage represents a WebSocket message.
type WSMessage struct {
	Type         string                 `json:"type"`
	ComponentID  string                 `json:"componentId,omitempty"`
	Action       string                 `json:"action,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Payload      json.RawMessage        `json:"payload,omitempty"`
	SessionToken string                 `json:"sessionToken,omitempty"` // SECURITY: Token sent in message, not URL
	ClientID     string                 `json:"clientId,omitempty"`     // Client ID for session association
}

// WSStateUpdate represents a state update message.
type WSStateUpdate struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// WSHub maintains the set of active clients and broadcasts messages.
type WSHub struct {
	Clients    map[string]*WSClient
	Register   chan *WSClient
	Unregister chan *WSClient
	Broadcast  chan []byte
	mu         sync.RWMutex
}

// NewWSHub creates a new WebSocket hub.
func NewWSHub() *WSHub {
	return &WSHub{
		Clients:    make(map[string]*WSClient),
		Register:   make(chan *WSClient),
		Unregister: make(chan *WSClient),
		Broadcast:  make(chan []byte, 256),
	}
}

// Run starts the hub's main loop.
func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			if oldClient, ok := h.Clients[client.ID]; ok {
				_ = oldClient.Conn.Close()
			}
			h.Clients[client.ID] = client
			h.mu.Unlock()
			log.Printf("Client connected: %s", client.ID)

		case client := <-h.Unregister:
			h.mu.Lock()
			if existing, ok := h.Clients[client.ID]; ok && existing == client {
				delete(h.Clients, client.ID)
			}
			close(client.Send)
			h.mu.Unlock()
			log.Printf("Client disconnected: %s", client.ID)

		case message := <-h.Broadcast:
			h.mu.RLock()
			var toRemove []string
			for _, client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					toRemove = append(toRemove, client.ID)
				}
			}
			h.mu.RUnlock()
			// Safely delete clients with write lock after iteration
			if len(toRemove) > 0 {
				h.mu.Lock()
				for _, id := range toRemove {
					delete(h.Clients, id)
				}
				h.mu.Unlock()
			}
		}
	}
}

// BroadcastTo broadcasts a message to specific clients.
func (h *WSHub) BroadcastTo(clientIDs []string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, id := range clientIDs {
		if client, ok := h.Clients[id]; ok {
			select {
			case client.Send <- message:
			default:
				// Client buffer full, skip
			}
		}
	}
}

// BroadcastExcept broadcasts to all clients except the specified one.
func (h *WSHub) BroadcastExcept(exceptID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for id, client := range h.Clients {
		if id != exceptID {
			select {
			case client.Send <- message:
			default:
				// Client buffer full, skip
			}
		}
	}
}

// GetClient retrieves a client by ID.
func (h *WSHub) GetClient(id string) (*WSClient, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	client, ok := h.Clients[id]
	return client, ok
}

// ClientCount returns the number of connected clients.
func (h *WSHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.Clients)
}

// NewWSClient creates a new WebSocket client.
func NewWSClient(id string, conn *websocket.Conn) *WSClient {
	client := &WSClient{
		ID:     id,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		State:  state.NewStateMap(),
		closed: false,
	}
	return client
}

// ReadPump pumps messages from the WebSocket connection to the hub.
func (c *WSClient) ReadPump(hub *WSHub, onMessage func(*WSClient, WSMessage)) {
	defer func() {
		hub.Unregister <- c
		_ = c.Conn.Close()
	}()

	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			c.SendError("Invalid message format")
			continue
		}

		onMessage(c, msg)
	}
}

// WritePump pumps messages from the hub to the WebSocket connection.
func (c *WSClient) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.mu.Lock()
			if c.closed || !ok {
				c.mu.Unlock()
				return
			}
			err := c.Conn.WriteMessage(websocket.TextMessage, message)
			c.mu.Unlock()

			if err != nil {
				return
			}
		case <-ticker.C:
			c.mu.Lock()
			if c.closed {
				c.mu.Unlock()
				return
			}
			err := c.Conn.WriteMessage(websocket.PingMessage, nil)
			c.mu.Unlock()

			if err != nil {
				return
			}
		}
	}
}

// SendJSON sends a JSON message to the client.
func (c *WSClient) SendJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	select {
	case c.Send <- data:
	default:
		// Buffer full
	}

	return nil
}

// SendError sends an error message to the client.
func (c *WSClient) SendError(message string) {
	_ = c.SendJSON(map[string]interface{}{
		"type":  "error",
		"error": message,
	})
}

// SendState sends the current state to the client.
func (c *WSClient) SendState() {
	stateJSON, err := c.State.ToJSON()
	if err != nil {
		c.SendError("Failed to serialize state")
		return
	}

	_ = c.SendJSON(map[string]interface{}{
		"type":  "init",
		"state": json.RawMessage(stateJSON),
	})
}

// SendInitWithSession sends the initial state with a session token for HTTP state sync.
func (c *WSClient) SendInitWithSession(sessionToken string) {
	stateJSON, err := c.State.ToJSON()
	if err != nil {
		c.SendError("Failed to serialize state")
		return
	}

	_ = c.SendJSON(map[string]interface{}{
		"type":         "init",
		"state":        json.RawMessage(stateJSON),
		"sessionToken": sessionToken,
		"clientId":     c.SessionID,
	})
}

// Close closes the client connection.
func (c *WSClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed {
		c.closed = true
		close(c.Send)
	}
}

// WebSocketConfig holds WebSocket configuration.
type WebSocketConfig struct {
	// Hub is the WebSocket hub for managing connections.
	Hub *WSHub
	// OnConnect is called when a client connects.
	OnConnect func(*WSClient)
	// OnDisconnect is called when a client disconnects.
	OnDisconnect func(*WSClient)
	// OnMessage is called when a message is received.
	OnMessage func(*WSClient, WSMessage)
	// GenerateID generates a client ID.
	GenerateID func() string
}

// DefaultWebSocketConfig returns default WebSocket configuration.
func DefaultWebSocketConfig() WebSocketConfig {
	return WebSocketConfig{
		Hub:        NewWSHub(),
		GenerateID: generateComponentID,
	}
}

// WebSocketHandler creates a WebSocket handler.
func WebSocketHandler(config WebSocketConfig) fiber.Handler {
	// Apply defaults for nil config values
	if config.Hub == nil {
		config.Hub = NewWSHub()
	}
	if config.GenerateID == nil {
		config.GenerateID = generateComponentID
	}

	// Start the hub
	go config.Hub.Run()

	return websocket.New(func(c *websocket.Conn) {
		var sessionID string
		var sessionToken string
		var restoredState *state.StateMap

		// SECURITY: Session token can come from:
		// 1. First message after connection (preferred - no URL leakage)
		// 2. URL query param (fallback for legacy clients - less secure)
		// We'll wait for the first message to check for session token

		// Generate unique connection ID so tabs don't kick each other off
		connID := "conn_" + generateSecureToken()[:8]

		// Create client with placeholder session (will be updated after auth)
		client := NewWSClient(connID, c)
		client.SessionID = "" // Will be set after session validation

		// Register client
		config.Hub.Register <- client

		// Set up read deadline for initial auth message
		_ = c.SetReadDeadline(time.Now().Add(10 * time.Second))

		// Wait for first message (should be init with session token)
		_, firstMsg, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("Failed to read initial message: %v", err)
			_ = c.Conn.Close()
			return
		}

		var initMsg WSMessage
		if err := json.Unmarshal(firstMsg, &initMsg); err != nil {
			client.SendError("Invalid initial message format")
			_ = c.Conn.Close()
			return
		}

		// Handle session authentication
		if initMsg.Type == "init" && initMsg.SessionToken != "" {
			// SECURITY: Session token provided in message (preferred method)
			if prevSessionID, ok := globalSessionStore.ValidateSession(initMsg.SessionToken); ok {
				// Check if we have saved state for this session
				if savedState, hasState := globalClientStateStore.Get(prevSessionID); hasState {
					log.Printf("Restoring session state for %s", prevSessionID)
					sessionID = prevSessionID
					restoredState = savedState
					sessionToken = initMsg.SessionToken
				}
			}
		} else {
			// Fallback: Check for session token in query params (legacy support)
			sessionParam := c.Query("session")
			if sessionParam != "" {
				if prevSessionID, ok := globalSessionStore.ValidateSession(sessionParam); ok {
					if savedState, hasState := globalClientStateStore.Get(prevSessionID); hasState {
						log.Printf("Restoring session state for %s", prevSessionID)
						sessionID = prevSessionID
						restoredState = savedState
						sessionToken = sessionParam
					}
				}
			}
		}

		// If no valid session, generate new session ID
		if sessionID == "" {
			sessionID = config.GenerateID()
			sessionToken = globalSessionStore.CreateSession(sessionID)
			if sessionToken == "" {
				client.SendError("Failed to create session")
				_ = c.Conn.Close()
				return
			}
		}

		// Update client with session ID
		client.SessionID = sessionID

		// Restore previous state if available, passing pointer
		if restoredState != nil {
			client.State = restoredState
		} else {
			// Setup differential sync for the first time for this state
			client.State.OnChange = func(key string, value any) {
				// Save state to persistent store safely
				globalClientStateStore.Save(sessionID, client.State)

				// Parse componentId and local key for Svelte updates
				componentID := ""
				localKey := key
				if dotIdx := strings.Index(key, "."); dotIdx > 0 {
					componentID = key[:dotIdx]
					localKey = key[dotIdx+1:]
				}

				// Broadcast state change to all clients sharing this session ID
				config.Hub.mu.RLock()
				for _, hubClient := range config.Hub.Clients {
					if hubClient.SessionID == sessionID {
						_ = hubClient.SendJSON(map[string]interface{}{
							"type":        "sync",
							"componentId": componentID,
							"key":         localKey,
							"value":       value,
						})
					}
				}
				config.Hub.mu.RUnlock()
			}
			globalClientStateStore.Save(sessionID, client.State)
		}

		// Reset read deadline for normal operation
		_ = c.SetReadDeadline(time.Now().Add(pongWait))

		// Call global connect handlers (for initial state sync)
		callConnectHandlers(client)

		// Call onConnect hook
		if config.OnConnect != nil {
			config.OnConnect(client)
		}

		// Send initial state with session token
		client.SendInitWithSession(sessionToken)

		// Handle messages
		onMessage := config.OnMessage
		if onMessage == nil {
			onMessage = DefaultMessageHandler
		}

		// Start write pump
		go client.WritePump()

		// Continue with normal read pump
		client.ReadPump(config.Hub, onMessage)

		// Save final state before disconnect
		globalClientStateStore.Save(sessionID, client.State)

		// Note: We don't remove the session on disconnect so the client can reconnect
		// Sessions will be cleaned up by the session store when they expire (if TTL is implemented)
		// globalSessionStore.RemoveClientSessions(clientID)

		// Call onDisconnect hook
		if config.OnDisconnect != nil {
			config.OnDisconnect(client)
		}
	})
}

// DefaultMessageHandler handles incoming WebSocket messages.
func DefaultMessageHandler(client *WSClient, msg WSMessage) {
	var reqID interface{}
	if msg.Data != nil {
		reqID = msg.Data["_requestId"]
	}

	sendResponse := func(payload map[string]interface{}) {
		if reqID != nil {
			if payload["data"] == nil {
				payload["data"] = map[string]interface{}{}
			}
			if dataMap, ok := payload["data"].(map[string]interface{}); ok {
				dataMap["_responseId"] = reqID
			}
		}
		_ = client.SendJSON(payload)
	}

	switch msg.Type {
	case "init":
		stateJSON, _ := client.State.ToJSON()
		sendResponse(map[string]interface{}{
			"type":        "init",
			"componentId": msg.ComponentID,
			"state":       json.RawMessage(stateJSON),
		})

	case "update":
		var update WSStateUpdate
		if err := json.Unmarshal(msg.Payload, &update); err != nil {
			sendResponse(map[string]interface{}{
				"type":  "error",
				"error": "Invalid update payload",
			})
			return
		}

		// Create component-scoped key (e.g., "counter.count")
		stateKey := update.Key
		if msg.ComponentID != "" {
			stateKey = msg.ComponentID + "." + update.Key
		}

		// Update state
		if obs, ok := client.State.Get(stateKey); ok {
			if settable, isSettable := obs.(state.Settable); isSettable {
				_ = settable.SetAny(update.Value)
			}
		} else {
			r := state.NewRune(update.Value)
			client.State.Add(stateKey, r)
		}

		// Send success to requesting client
		sendResponse(map[string]interface{}{
			"type":        "sync",
			"componentId": msg.ComponentID,
			"key":         update.Key,
			"value":       update.Value,
			"success":     true,
		})

	case "sync":
		client.SendState()

	case "ping":
		sendResponse(map[string]interface{}{
			"type": "pong",
		})

	case "action":
		action := msg.Action
		if action == "" {
			sendResponse(map[string]interface{}{
				"type":  "error",
				"error": "Action name required",
			})
			return
		}

		// Look for action handlers in the hub or app
		// For now, we'll just log and broadcast if no dedicated handler
		log.Printf("Action received: %s (Client: %s)", action, client.ID)

		// This is where we will hook into the app's action handlers
		// For the examples, we can use a global registry for now
		if handler, ok := GetActionHandler(action); ok {
			handler(client, msg.Payload)
		} else {
			sendResponse(map[string]interface{}{
				"type":  "error",
				"error": "Unknown action: " + action,
			})
		}

	default:
		sendResponse(map[string]interface{}{
			"type":  "error",
			"error": "Unknown message type: " + msg.Type,
		})
	}
}

// ActionHandler is a function that handles a WebSocket action.
type ActionHandler func(client *WSClient, payload json.RawMessage)

// ConnectHandler is a function that handles a new WebSocket connection.
type ConnectHandler func(client *WSClient)

var (
	actionHandlers  = make(map[string]ActionHandler)
	actionMu        sync.RWMutex
	connectHandlers []ConnectHandler
	connectMu       sync.RWMutex
)

// RegisterActionHandler registers a global action handler.
func RegisterActionHandler(name string, handler ActionHandler) {
	actionMu.Lock()
	defer actionMu.Unlock()
	actionHandlers[name] = handler
}

// GetActionHandler retrieves a global action handler.
func GetActionHandler(name string) (ActionHandler, bool) {
	actionMu.RLock()
	defer actionMu.RUnlock()
	handler, ok := actionHandlers[name]
	return handler, ok
}

// RegisterOnConnectHandler registers a global connect handler.
func RegisterOnConnectHandler(handler ConnectHandler) {
	connectMu.Lock()
	defer connectMu.Unlock()
	connectHandlers = append(connectHandlers, handler)
}

// callConnectHandlers calls all registered connect handlers.
func callConnectHandlers(client *WSClient) {
	connectMu.RLock()
	defer connectMu.RUnlock()
	for _, handler := range connectHandlers {
		handler(client)
	}
}

// WebSocketUpgradeMiddleware upgrades HTTP connections to WebSocket.
func WebSocketUpgradeMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if WebSocket upgrade request
		if string(c.Request().Header.Peek("Upgrade")) != "websocket" {
			return c.Next()
		}

		// WebSocket upgrade will be handled by the websocket handler
		return c.Next()
	}
}

// StateSyncHandler creates a handler for state synchronization.
func StateSyncHandler(hub *WSHub) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get session token from header or query
		sessionToken := c.Query("session")
		if sessionToken == "" {
			sessionToken = c.Get("X-Session-Token")
		}

		if sessionToken == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Session token required",
			})
		}

		sessionID, ok := globalSessionStore.ValidateSession(sessionToken)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid session",
			})
		}

		// Get shared state map
		stateMap, ok := globalClientStateStore.Get(sessionID)
		if !ok {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Session state not found",
			})
		}

		// Parse state update
		var update WSStateUpdate
		if err := json.Unmarshal(c.Body(), &update); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid update payload",
			})
		}

		// Update state - this triggers the shared OnChange safely, broadcasting to all tabs
		if obs, ok := stateMap.Get(update.Key); ok {
			if settable, isSettable := obs.(state.Settable); isSettable {
				_ = settable.SetAny(update.Value)
			}
		} else {
			r := state.NewRune(update.Value)
			stateMap.Add(update.Key, r)
		}

		return c.JSON(fiber.Map{
			"success": true,
		})
	}
}

// BroadcastState broadcasts state to all connected clients.
func BroadcastState(hub *WSHub, key string, value interface{}) error {
	if hub == nil {
		return nil
	}
	data, err := json.Marshal(map[string]interface{}{
		"type":  "sync",
		"key":   key,
		"value": value,
	})
	if err != nil {
		return err
	}

	hub.Broadcast <- data
	return nil
}

// SendToClient sends a message to a specific client.
func SendToClient(hub *WSHub, clientID string, message interface{}) error {
	client, ok := hub.GetClient(clientID)
	if !ok {
		return nil
	}
	return client.SendJSON(message)
}
