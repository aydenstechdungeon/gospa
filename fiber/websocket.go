package fiber

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/aydenstechdungeon/gospa/state"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// WSClient represents a connected WebSocket client.
type WSClient struct {
	ID     string
	Conn   *websocket.Conn
	Send   chan []byte
	State  *state.StateMap
	mu     sync.Mutex
	closed bool
}

// WSMessage represents a WebSocket message.
type WSMessage struct {
	Type        string                 `json:"type"`
	ComponentID string                 `json:"componentId,omitempty"`
	Action      string                 `json:"action,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Payload     json.RawMessage        `json:"payload,omitempty"`
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
			h.Clients[client.ID] = client
			h.mu.Unlock()
			log.Printf("Client connected: %s", client.ID)

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client.ID]; ok {
				delete(h.Clients, client.ID)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("Client disconnected: %s", client.ID)

		case message := <-h.Broadcast:
			h.mu.RLock()
			for _, client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.Clients, client.ID)
				}
			}
			h.mu.RUnlock()
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

	// Setup differential sync
	client.State.OnChange = func(key string, value any) {
		_ = client.SendJSON(map[string]interface{}{
			"type":  "sync",
			"key":   key,
			"value": value,
		})
	}

	return client
}

// ReadPump pumps messages from the WebSocket connection to the hub.
func (c *WSClient) ReadPump(hub *WSHub, onMessage func(*WSClient, WSMessage)) {
	defer func() {
		hub.Unregister <- c
		_ = c.Conn.Close()
	}()

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
	defer func() {
		_ = c.Conn.Close()
	}()

	for message := range c.Send {
		c.mu.Lock()
		if c.closed {
			c.mu.Unlock()
			return
		}
		err := c.Conn.WriteMessage(websocket.TextMessage, message)
		c.mu.Unlock()

		if err != nil {
			break
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

	// DEBUG: Log initial state being sent to client
	log.Printf("DEBUG: SendState to client %s: %s", c.ID, string(stateJSON))

	_ = c.SendJSON(map[string]interface{}{
		"type":  "init",
		"state": json.RawMessage(stateJSON),
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
		// Generate client ID
		clientID := config.GenerateID()

		// Create client
		client := NewWSClient(clientID, c)

		// Register client
		config.Hub.Register <- client

		// Call global connect handlers (for initial state sync)
		callConnectHandlers(client)

		// Call onConnect hook
		if config.OnConnect != nil {
			config.OnConnect(client)
		}

		// Send initial state
		client.SendState()

		// Handle messages
		onMessage := config.OnMessage
		if onMessage == nil {
			onMessage = DefaultMessageHandler
		}

		// Start read/write pumps
		go client.WritePump()
		client.ReadPump(config.Hub, onMessage)

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

		// Update state
		r := state.NewRune(update.Value)
		client.State.Add(update.Key, r)

		// Send success to requesting client
		sendResponse(map[string]interface{}{
			"type":    "sync",
			"key":     update.Key,
			"value":   update.Value,
			"success": true,
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
		// Get client ID from query or header
		clientID := c.Query("client_id")
		if clientID == "" {
			clientID = c.Get("X-Client-ID")
		}

		if clientID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Client ID required",
			})
		}

		// Get client
		client, ok := hub.GetClient(clientID)
		if !ok {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Client not found",
			})
		}

		// Parse state update
		var update WSStateUpdate
		if err := json.Unmarshal(c.Body(), &update); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid update payload",
			})
		}

		// Update state
		r := state.NewRune(update.Value)
		client.State.Add(update.Key, r)

		// Broadcast to other clients
		hub.BroadcastExcept(clientID, []byte(`{"type":"sync","key":"`+update.Key+`"}`))

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
