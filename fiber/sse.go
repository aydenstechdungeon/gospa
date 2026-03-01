// Package fiber provides Server-Sent Events (SSE) support for GoSPA applications.
// SSE enables real-time server-to-client push notifications over HTTP.
package fiber

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aydenstechdungeon/gospa/store"
	"github.com/gofiber/fiber/v2"
	fiberpkg "github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// SSEEvent represents a Server-Sent Event.
type SSEEvent struct {
	// ID is the event identifier
	ID string `json:"id,omitempty"`
	// Event is the event type/name
	Event string `json:"event,omitempty"`
	// Data is the event payload
	Data any `json:"data"`
	// Retry specifies reconnection time in milliseconds
	Retry int `json:"retry,omitempty"`
}

// SSEClient represents a connected SSE client.
type SSEClient struct {
	// ID is the unique client identifier
	ID string
	// Channel is the client's event channel
	Channel chan SSEEvent
	// ConnectedAt is the connection timestamp
	ConnectedAt time.Time
	// Metadata contains additional client data
	Metadata map[string]any
	// Topics the client is subscribed to
	Topics map[string]bool
}

// SSEBroker manages SSE connections and event distribution.
type SSEBroker struct {
	// clients holds all connected clients
	clients map[string]*SSEClient
	// mutex protects concurrent access
	mutex sync.RWMutex
	// eventBufferSize is the buffer size for client channels
	eventBufferSize int
	// heartbeatInterval is the keepalive interval
	heartbeatInterval time.Duration
	// onConnect is called when a client connects
	onConnect func(*SSEClient)
	// onDisconnect is called when a client disconnects
	onDisconnect func(*SSEClient)
	// pubsub backend for distributed environments
	pubsub store.PubSub
}

// SSEConfig holds SSE broker configuration.
type SSEConfig struct {
	// EventBufferSize is the buffer size for client channels
	EventBufferSize int
	// HeartbeatInterval is the keepalive interval (0 to disable)
	HeartbeatInterval time.Duration
	// OnConnect is called when a client connects
	OnConnect func(*SSEClient)
	// OnDisconnect is called when a client disconnects
	OnDisconnect func(*SSEClient)
	// PubSub backend for distributed environments
	PubSub store.PubSub
}

// NewSSEBroker creates a new SSE broker.
func NewSSEBroker(config *SSEConfig) *SSEBroker {
	if config == nil {
		config = &SSEConfig{}
	}
	if config.EventBufferSize == 0 {
		config.EventBufferSize = 100
	}
	if config.HeartbeatInterval == 0 {
		config.HeartbeatInterval = 30 * time.Second
	}
	if config.PubSub == nil {
		config.PubSub = store.NewMemoryPubSub()
	}

	b := &SSEBroker{
		clients:           make(map[string]*SSEClient),
		eventBufferSize:   config.EventBufferSize,
		heartbeatInterval: config.HeartbeatInterval,
		onConnect:         config.OnConnect,
		onDisconnect:      config.OnDisconnect,
		pubsub:            config.PubSub,
	}

	// Subscribe to distributed events
	_ = b.pubsub.Subscribe("gospa:sse", func(message []byte) {
		var sseMsg struct {
			Target string   `json:"target"` // "all", "topic:xyz", "client:xyz"
			Event  SSEEvent `json:"event"`
		}
		if err := json.Unmarshal(message, &sseMsg); err != nil {
			return
		}

		b.mutex.RLock()
		defer b.mutex.RUnlock()

		if sseMsg.Target == "all" {
			for _, client := range b.clients {
				select {
				case client.Channel <- sseMsg.Event:
				default:
				}
			}
		} else if len(sseMsg.Target) > 6 && sseMsg.Target[:6] == "topic:" {
			topic := sseMsg.Target[6:]
			for _, client := range b.clients {
				if client.Topics[topic] {
					select {
					case client.Channel <- sseMsg.Event:
					default:
					}
				}
			}
		} else if len(sseMsg.Target) > 7 && sseMsg.Target[:7] == "client:" {
			clientID := sseMsg.Target[7:]
			if client, exists := b.clients[clientID]; exists {
				select {
				case client.Channel <- sseMsg.Event:
				default:
				}
			}
		}
	})

	return b
}

// Connect registers a new SSE client.
func (b *SSEBroker) Connect(clientID string, metadata ...map[string]any) *SSEClient {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	client := &SSEClient{
		ID:          clientID,
		Channel:     make(chan SSEEvent, b.eventBufferSize),
		ConnectedAt: time.Now(),
		Topics:      make(map[string]bool),
	}

	if len(metadata) > 0 {
		client.Metadata = metadata[0]
	}

	b.clients[clientID] = client

	if b.onConnect != nil {
		go b.onConnect(client)
	}

	return client
}

// Disconnect removes an SSE client.
func (b *SSEBroker) Disconnect(clientID string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if client, exists := b.clients[clientID]; exists {
		close(client.Channel)
		delete(b.clients, clientID)

		if b.onDisconnect != nil {
			go b.onDisconnect(client)
		}
	}
}

// Subscribe adds a client to a topic.
func (b *SSEBroker) Subscribe(clientID string, topics ...string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if client, exists := b.clients[clientID]; exists {
		for _, topic := range topics {
			client.Topics[topic] = true
		}
	}
}

// Unsubscribe removes a client from a topic.
func (b *SSEBroker) Unsubscribe(clientID string, topics ...string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if client, exists := b.clients[clientID]; exists {
		for _, topic := range topics {
			delete(client.Topics, topic)
		}
	}
}

// Send sends an event to a specific client.
func (b *SSEBroker) Send(clientID string, event SSEEvent) bool {
	data, _ := json.Marshal(map[string]interface{}{
		"target": "client:" + clientID,
		"event":  event,
	})
	_ = b.pubsub.Publish("gospa:sse", data)
	return true
}

// Broadcast sends an event to all connected clients.
func (b *SSEBroker) Broadcast(event SSEEvent) int {
	data, _ := json.Marshal(map[string]interface{}{
		"target": "all",
		"event":  event,
	})
	_ = b.pubsub.Publish("gospa:sse", data)
	return 0 // Distributed, local count meaningless
}

// BroadcastToTopic sends an event to clients subscribed to a topic.
func (b *SSEBroker) BroadcastToTopic(topic string, event SSEEvent) int {
	data, _ := json.Marshal(map[string]interface{}{
		"target": "topic:" + topic,
		"event":  event,
	})
	_ = b.pubsub.Publish("gospa:sse", data)
	return 0
}

// ClientCount returns the number of connected clients.
func (b *SSEBroker) ClientCount() int {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return len(b.clients)
}

// GetClient returns a client by ID.
func (b *SSEBroker) GetClient(clientID string) (*SSEClient, bool) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	client, exists := b.clients[clientID]
	return client, exists
}

// GetClientsByTopic returns all clients subscribed to a topic.
func (b *SSEBroker) GetClientsByTopic(topic string) []*SSEClient {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	var clients []*SSEClient
	for _, client := range b.clients {
		if client.Topics[topic] {
			clients = append(clients, client)
		}
	}
	return clients
}

// SSEHandler returns a Fiber handler for SSE connections.
func (b *SSEBroker) SSEHandler(clientIDFunc func(*fiberpkg.Ctx) string) fiberpkg.Handler {
	return func(c *fiberpkg.Ctx) error {
		// Set SSE headers
		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")
		c.Set("Transfer-Encoding", "chunked")

		// Get client ID
		clientID := clientIDFunc(c)
		if clientID == "" {
			clientID = generateClientID()
		}

		// Connect client
		client := b.Connect(clientID, map[string]any{
			"remote_addr": c.IP(),
			"user_agent":  c.Get("User-Agent"),
		})
		defer b.Disconnect(clientID)

		// Send connected event
		b.Send(clientID, SSEEvent{
			Event: "connected",
			Data:  map[string]any{"clientId": clientID},
		})

		// Start heartbeat
		heartbeatTicker := time.NewTicker(b.heartbeatInterval)
		defer heartbeatTicker.Stop()

		// Create context for graceful shutdown
		ctx := c.Context()

		// Stream events
		for {
			select {
			case <-ctx.Done():
				return nil
			case event := <-client.Channel:
				if err := writeSSEEvent(c, event); err != nil {
					return err
				}
			case <-heartbeatTicker.C:
				if err := writeSSEEvent(c, SSEEvent{Event: "heartbeat", Data: "ping"}); err != nil {
					return err
				}
			}
		}
	}
}

// SSESubscribeHandler returns a handler for subscribing to topics.
//
// SECURITY WARNING: This handler verifies that the target clientId is connected,
// but it does NOT verify that the requester IS that client.
// Any authenticated user who knows another client's ID can subscribe that client
// to arbitrary topics.
//
// To prevent cross-client topic injection, wrap this handler with authentication
// middleware that validates req.ClientID against the session identity, e.g.:
//
//	sse.Post("/subscribe", authMw, broker.SSESubscribeHandler())
//
// Inside your authMw, reject if the session user ID doesn't match the clientId in the request body.
func (b *SSEBroker) SSESubscribeHandler() fiberpkg.Handler {
	return func(c *fiberpkg.Ctx) error {
		var req struct {
			ClientID string   `json:"clientId"`
			Topics   []string `json:"topics"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		// Validate that the client exists and is currently connected.
		// NOTE: This is an existence check, not an identity check.
		// See security warning in the function comment above.
		if !b.clientExists(req.ClientID) {
			return c.Status(404).JSON(fiberpkg.Map{
				"error": "client not found or not connected",
			})
		}

		b.Subscribe(req.ClientID, req.Topics...)

		return c.JSON(fiberpkg.Map{
			"success": true,
			"topics":  req.Topics,
		})
	}
}

// clientExists checks if a client is currently connected.
// This is used to validate subscription requests.
func (b *SSEBroker) clientExists(clientID string) bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	_, exists := b.clients[clientID]
	return exists
}

// SSEUnsubscribeHandler returns a handler for unsubscribing from topics.
func (b *SSEBroker) SSEUnsubscribeHandler() fiberpkg.Handler {
	return func(c *fiberpkg.Ctx) error {
		var req struct {
			ClientID string   `json:"clientId"`
			Topics   []string `json:"topics"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		// Validate that the client exists and is currently connected
		if !b.clientExists(req.ClientID) {
			return c.Status(404).JSON(fiberpkg.Map{
				"error": "client not found or not connected",
			})
		}

		b.Unsubscribe(req.ClientID, req.Topics...)

		return c.JSON(fiberpkg.Map{
			"success": true,
			"topics":  req.Topics,
		})
	}
}

// writeSSEEvent writes an SSE event to the response.
func writeSSEEvent(c *fiberpkg.Ctx, event SSEEvent) error {
	// Write event ID if present
	if event.ID != "" {
		_, _ = c.Write([]byte(fmt.Sprintf("id: %s\n", event.ID)))
	}

	// Write event name if present
	if event.Event != "" {
		_, _ = c.Write([]byte(fmt.Sprintf("event: %s\n", event.Event)))
	}

	// Write retry if present
	if event.Retry > 0 {
		_, _ = c.Write([]byte(fmt.Sprintf("retry: %d\n", event.Retry)))
	}

	// Write data
	var dataStr string
	switch v := event.Data.(type) {
	case string:
		dataStr = v
	case []byte:
		dataStr = string(v)
	default:
		data, err := json.Marshal(event.Data)
		if err != nil {
			return err
		}
		dataStr = string(data)
	}

	_, _ = c.Write([]byte(fmt.Sprintf("data: %s\n\n", dataStr)))

	return nil
}

// generateClientID generates a unique client ID using cryptographically secure random bytes.
// This prevents session hijacking via predictable client IDs.
func generateClientID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// This should never happen with crypto/rand, but fall back to timestamp if it does
		return fmt.Sprintf("sse_%d", time.Now().UnixNano())
	}
	return "sse_" + hex.EncodeToString(bytes)
}

// SetupSSE sets up SSE routes on a Fiber app.
func SetupSSE(app *fiberpkg.App, broker *SSEBroker, basePath string, corsConfig *cors.Config) {
	// Create router group
	sse := app.Group(basePath)

	// Apply CORS if configured
	if corsConfig != nil {
		sse.Use(cors.New(*corsConfig))
	}

	// SSE connection endpoint
	sse.Get("/connect", broker.SSEHandler(func(c *fiberpkg.Ctx) string {
		return c.Query("clientId", "")
	}))

	// Subscribe endpoint
	sse.Post("/subscribe", broker.SSESubscribeHandler())

	// Unsubscribe endpoint
	sse.Post("/unsubscribe", broker.SSEUnsubscribeHandler())

	// Stats endpoint
	sse.Get("/stats", func(c *fiberpkg.Ctx) error {
		return c.JSON(fiberpkg.Map{
			"clientCount": broker.ClientCount(),
		})
	})
}

// SSEHelper provides helper methods for common SSE patterns.
type SSEHelper struct {
	broker *SSEBroker
}

// NewSSEHelper creates a new SSE helper.
func NewSSEHelper(broker *SSEBroker) *SSEHelper {
	return &SSEHelper{broker: broker}
}

// Notify sends a notification event.
func (h *SSEHelper) Notify(clientID string, notification any) {
	h.broker.Send(clientID, SSEEvent{
		Event: "notification",
		Data:  notification,
	})
}

// NotifyAll broadcasts a notification to all clients.
func (h *SSEHelper) NotifyAll(notification any) {
	h.broker.Broadcast(SSEEvent{
		Event: "notification",
		Data:  notification,
	})
}

// NotifyTopic broadcasts a notification to a topic.
func (h *SSEHelper) NotifyTopic(topic string, notification any) {
	h.broker.BroadcastToTopic(topic, SSEEvent{
		Event: "notification",
		Data:  notification,
	})
}

// Update sends a state update event.
func (h *SSEHelper) Update(clientID string, key string, value any) {
	h.broker.Send(clientID, SSEEvent{
		Event: "update",
		Data:  map[string]any{"key": key, "value": value},
	})
}

// UpdateAll broadcasts a state update to all clients.
func (h *SSEHelper) UpdateAll(key string, value any) {
	h.broker.Broadcast(SSEEvent{
		Event: "update",
		Data:  map[string]any{"key": key, "value": value},
	})
}

// UpdateTopic broadcasts a state update to a topic.
func (h *SSEHelper) UpdateTopic(topic string, key string, value any) {
	h.broker.BroadcastToTopic(topic, SSEEvent{
		Event: "update",
		Data:  map[string]any{"key": key, "value": value},
	})
}

// Alert sends an alert event.
func (h *SSEHelper) Alert(clientID string, level string, message string) {
	h.broker.Send(clientID, SSEEvent{
		Event: "alert",
		Data:  map[string]any{"level": level, "message": message},
	})
}

// AlertAll broadcasts an alert to all clients.
func (h *SSEHelper) AlertAll(level string, message string) {
	h.broker.Broadcast(SSEEvent{
		Event: "alert",
		Data:  map[string]any{"level": level, "message": message},
	})
}

// Progress sends a progress event.
func (h *SSEHelper) Progress(clientID string, progress int, message string) {
	h.broker.Send(clientID, SSEEvent{
		Event: "progress",
		Data:  map[string]any{"progress": progress, "message": message},
	})
}

// ProgressTopic broadcasts progress to a topic.
func (h *SSEHelper) ProgressTopic(topic string, progress int, message string) {
	h.broker.BroadcastToTopic(topic, SSEEvent{
		Event: "progress",
		Data:  map[string]any{"progress": progress, "message": message},
	})
}
