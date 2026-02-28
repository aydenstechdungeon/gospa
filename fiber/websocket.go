package fiber

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/aydenstechdungeon/gospa/state"
	"github.com/aydenstechdungeon/gospa/store"
	"github.com/gofiber/fiber/v2"
	websocket "github.com/gofiber/websocket/v2"
)

const (
	// Time allowed to keep an idle connection alive.
	pongWait = 60 * time.Second
	// Send pings to peer with this period.
	pingPeriod = (pongWait * 9) / 10
)

// ConnectionRateLimiter implements per-IP rate limiting for WebSocket connections
// to prevent DoS attacks. Uses token bucket algorithm with burst capacity.
type ConnectionRateLimiter struct {
	mu      sync.RWMutex
	buckets map[string]*rateBucket // IP -> bucket
	// Configurable limits
	maxTokens       float64       // Maximum burst (default: 5)
	refillRate      float64       // Tokens per second (default: 0.2 = 1 per 5 sec)
	cleanupInterval time.Duration // How often to clean stale entries (default: 1 min)
}

type rateBucket struct {
	tokens     float64
	lastRefill time.Time
}

// Global connection rate limiter (singleton)
var globalConnRateLimiter = NewConnectionRateLimiter()

// NewConnectionRateLimiter creates a rate limiter with sensible defaults.
func NewConnectionRateLimiter() *ConnectionRateLimiter {
	rl := &ConnectionRateLimiter{
		buckets:         make(map[string]*rateBucket),
		maxTokens:       5.0,         // Allow burst of 5 connections
		refillRate:      0.2,         // 1 connection per 5 seconds sustained
		cleanupInterval: time.Minute, // Cleanup stale entries every minute
	}
	// Start cleanup goroutine
	go rl.cleanupLoop()
	return rl
}

// Allow checks if a connection from the given IP is allowed.
// Returns true if the connection should be accepted.
func (rl *ConnectionRateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.buckets[ip]

	if !exists {
		// First connection from this IP
		rl.buckets[ip] = &rateBucket{
			tokens:     rl.maxTokens - 1, // Consume 1 token
			lastRefill: now,
		}
		return true
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens += elapsed * rl.refillRate
	if bucket.tokens > rl.maxTokens {
		bucket.tokens = rl.maxTokens
	}
	bucket.lastRefill = now

	// Check if we can consume a token
	if bucket.tokens >= 1.0 {
		bucket.tokens -= 1.0
		return true
	}

	return false
}

// GetIPFromContext extracts the client IP from the Fiber context.
// Uses Fiber's built-in IP extraction which handles X-Forwarded-For
// based on the app's Proxy settings (TrustedProxies, etc).
func GetIPFromContext(c *fiber.Ctx) string {
	return c.IP()
}

// cleanupLoop periodically removes stale rate limit entries to prevent memory leaks.
func (rl *ConnectionRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, bucket := range rl.buckets {
			// Remove entries that haven't been used in 10 minutes
			if now.Sub(bucket.lastRefill) > 10*time.Minute {
				delete(rl.buckets, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// sessionEntry holds a session token and its expiry.
type sessionEntry struct {
	ClientID  string    `json:"clientId"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// SessionTTL is how long a session token remains valid (default 24 hours).
const SessionTTL = 24 * time.Hour

// SessionStore maps session tokens to client IDs for secure HTTP state sync.
type SessionStore struct {
	storage store.Storage
}

// NewSessionStore creates a new session store.
func NewSessionStore(storage store.Storage) *SessionStore {
	return &SessionStore{
		storage: storage,
	}
}

// CreateSession creates a new session token for a client ID.
// Returns the session token, or empty string if random generation fails.
func (s *SessionStore) CreateSession(clientID string) string {
	token := generateSecureToken()
	if token == "" {
		return ""
	}
	entry := sessionEntry{
		ClientID:  clientID,
		ExpiresAt: time.Now().Add(SessionTTL),
	}
	bytes, err := json.Marshal(entry)
	if err == nil {
		_ = s.storage.Set(token, bytes, SessionTTL)
	}
	return token
}

// ValidateSession returns the client ID for a valid, non-expired session token.
func (s *SessionStore) ValidateSession(token string) (string, bool) {
	bytes, err := s.storage.Get(token)
	if err != nil {
		return "", false
	}
	var entry sessionEntry
	if err := json.Unmarshal(bytes, &entry); err != nil {
		return "", false
	}
	if time.Now().After(entry.ExpiresAt) {
		s.RemoveSession(token)
		return "", false
	}
	return entry.ClientID, true
}

// RemoveSession removes a session token.
func (s *SessionStore) RemoveSession(token string) {
	_ = s.storage.Delete(token)
}

// RemoveClientSessions removes all sessions for a client ID.
// NOTE: Depending on the storage backend, this might not be easily achievable without secondary indices.
// For KV-only stores, this operation shouldn't be relied upon.
func (s *SessionStore) RemoveClientSessions(clientID string) {
	// Not practically supported in KV stores without key scanning.
	// Since sessions expire naturally through TTL, this is a no-op for now.
}

// Global session store for HTTP state sync. Defaulting to in-memory.
var globalSessionStore = NewSessionStore(store.NewMemoryStorage())

// generateSecureToken generates a cryptographically secure random token.
// Returns an empty string if random generation fails (which should never happen
// on modern systems with a properly functioning OS).
func generateSecureToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Printf("CRITICAL: crypto/rand.Read failed: %v", err)
		return ""
	}
	return hex.EncodeToString(b)
}

// ClientStateStore persists client state by client ID for session restoration.
type ClientStateStore struct {
	storage store.Storage
}

// NewClientStateStore creates a new client state store.
func NewClientStateStore(storage store.Storage) *ClientStateStore {
	return &ClientStateStore{
		storage: storage,
	}
}

// Save saves a client's state.
func (s *ClientStateStore) Save(clientID string, sm *state.StateMap) {
	bytes, err := sm.MarshalJSON()
	if err == nil {
		_ = s.storage.Set("state:"+clientID, bytes, SessionTTL)
	}
}

// Get retrieves a client's state.
func (s *ClientStateStore) Get(clientID string) (*state.StateMap, bool) {
	bytes, err := s.storage.Get("state:" + clientID)
	if err != nil {
		return nil, false
	}
	sm := state.NewStateMap()
	var raw map[string]interface{}
	if err := json.Unmarshal(bytes, &raw); err != nil {
		return nil, false
	}
	for k, v := range raw {
		r := state.NewRune(v)
		sm.Add(k, r)
	}
	return sm, true
}

// Remove removes a client's state.
func (s *ClientStateStore) Remove(clientID string) {
	_ = s.storage.Delete("state:" + clientID)
}

// Global client state store for session persistence. Defaulting to in-memory.
var globalClientStateStore = NewClientStateStore(store.NewMemoryStorage())

// InitStores updates the global stores to use the provided storage backend.
func InitStores(storage store.Storage) {
	globalSessionStore = NewSessionStore(storage)
	globalClientStateStore = NewClientStateStore(storage)
}

// WSClient represents a connected WebSocket client.
type WSClient struct {
	ID        string
	SessionID string
	Conn      *websocket.Conn
	Send      chan []byte
	State     *state.StateMap
	mu        sync.Mutex
	closed    bool
	// optional features wired from WebSocketConfig at creation time
	compress     bool
	stateDiffing bool
	serializer   func(interface{}) ([]byte, error)
	deserializer func([]byte, interface{}) error
	// lastSentState holds the snapshot used for StateDiffing
	lastSentStateMu sync.Mutex
	lastSentState   map[string]interface{}
	// Rate limiting
	actionMu         sync.Mutex
	actionTokens     float64
	actionLastRefill time.Time
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
	pubsub     store.PubSub
}

// NewWSHub creates a new WebSocket hub.
func NewWSHub(pubsub store.PubSub) *WSHub {
	if pubsub == nil {
		pubsub = store.NewMemoryPubSub()
	}
	h := &WSHub{
		Clients:    make(map[string]*WSClient),
		Register:   make(chan *WSClient),
		Unregister: make(chan *WSClient),
		Broadcast:  make(chan []byte, 256),
		pubsub:     pubsub,
	}

	// Subscribe to a global broadcast channel for state syncing across processes
	_ = h.pubsub.Subscribe("gospa:broadcast", func(message []byte) {
		h.mu.RLock()
		defer h.mu.RUnlock()

		var msgData map[string]interface{}
		var sessionID string
		// Best effort parse to restrict session scope
		if err := json.Unmarshal(message, &msgData); err == nil {
			if sid, ok := msgData["_sessionID"].(string); ok {
				sessionID = sid
			}
		}

		for _, client := range h.Clients {
			if sessionID == "" || client.SessionID == sessionID {
				select {
				case client.Send <- message:
				default:
					// Client buffer full
				}
			}
		}
	})

	return h
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
				// Use guarded Close() to prevent double-close panics
				client.Close()
			}
			h.mu.Unlock()
			log.Printf("Client disconnected: %s", client.ID)

		case message := <-h.Broadcast:
			// Instead of directly sending to local clients, publish to the PubSub system.
			// The PubSub subscription handler will broadcast it locally.
			_ = h.pubsub.Publish("gospa:broadcast", message)
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
		ID:               id,
		Conn:             conn,
		Send:             make(chan []byte, 256),
		State:            state.NewStateMap(),
		closed:           false,
		actionTokens:     10.0,
		actionLastRefill: time.Now(),
		lastSentState:    make(map[string]interface{}), // Initialize to prevent nil pointer
	}
	return client
}

// maxWSMessageSize is the maximum WebSocket message size we accept (64KB).
const maxWSMessageSize = 64 * 1024

// maxActionNameLen is the maximum length of an action name field.
const maxActionNameLen = 256

// ReadPump pumps messages from the WebSocket connection to the hub.
func (c *WSClient) ReadPump(hub *WSHub, onMessage func(*WSClient, WSMessage)) {
	defer func() {
		hub.Unregister <- c
		_ = c.Conn.Close()
	}()

	// Limit inbound message size to prevent DoS attacks
	c.Conn.SetReadLimit(maxWSMessageSize)
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

		// Sanitize field lengths to prevent injection via long strings
		if len(msg.Action) > maxActionNameLen {
			msg.Action = msg.Action[:maxActionNameLen]
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
// When StateDiffing is enabled it only sends keys that changed since the last
// successful send — using a "patch" message type that the client merges into
// its local state rather than replacing it wholesale.
// When CompressState is enabled the payload JSON is gzip-compressed and
// base64-encoded, with a "compressed":true flag so the client can decompress.
func (c *WSClient) SendState() {
	stateMap := c.State.ToMap()
	if c.stateDiffing {
		c.lastSentStateMu.Lock()
		prev := c.lastSentState
		c.lastSentStateMu.Unlock()
		if prev != nil {
			diff := computeStateDiff(prev, stateMap)
			if len(diff) == 0 {
				return // nothing changed
			}
			c.sendEncodedPayload(map[string]interface{}{
				"type":  "patch",
				"patch": diff,
			})
			c.lastSentStateMu.Lock()
			c.lastSentState = stateMap
			c.lastSentStateMu.Unlock()
			return
		}
		// First send — fall through to full snapshot
		c.lastSentStateMu.Lock()
		c.lastSentState = stateMap
		c.lastSentStateMu.Unlock()
	}

	var stateJSON []byte
	var err error
	if c.serializer != nil {
		stateJSON, err = c.serializer(stateMap)
	} else {
		var s string
		s, err = c.State.ToJSON()
		stateJSON = []byte(s)
	}
	if err != nil {
		c.SendError("Failed to serialize state")
		return
	}
	c.sendEncodedPayload(map[string]interface{}{
		"type":  "init",
		"state": json.RawMessage(stateJSON),
	})
}

// SendInitWithSession sends the initial state with a session token for HTTP state sync.
func (c *WSClient) SendInitWithSession(sessionToken string) {
	stateMap := c.State.ToMap()
	if c.stateDiffing {
		c.lastSentStateMu.Lock()
		c.lastSentState = stateMap
		c.lastSentStateMu.Unlock()
	}

	var stateJSON []byte
	var err error
	if c.serializer != nil {
		stateJSON, err = c.serializer(stateMap)
	} else {
		var s string
		s, err = c.State.ToJSON()
		stateJSON = []byte(s)
	}
	if err != nil {
		c.SendError("Failed to serialize state")
		return
	}
	c.sendEncodedPayload(map[string]interface{}{
		"type":         "init",
		"state":        json.RawMessage(stateJSON),
		"sessionToken": sessionToken,
		"clientId":     c.SessionID,
	})
}

// sendEncodedPayload marshals msg and optionally gzip-compresses it before
// queueing on the Send channel.
func (c *WSClient) sendEncodedPayload(msg map[string]interface{}) {
	if c.compress {
		data, err := json.Marshal(msg)
		if err != nil {
			c.SendError(fmt.Sprintf("state encode error: %v", err))
			return
		}
		compressed, err := compressBytes(data)
		if err != nil {
			c.SendError(fmt.Sprintf("state compress error: %v", err))
			return
		}
		_ = c.SendJSON(map[string]interface{}{
			"type":       "compressed",
			"data":       base64.StdEncoding.EncodeToString(compressed),
			"compressed": true,
		})
		return
	}
	_ = c.SendJSON(msg)
}

// compressBytes gzip-compresses data and returns the compressed bytes.
func compressBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(data); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// computeStateDiff returns only the keys where newState differs from prevState,
// plus any keys present in newState but absent from prevState.
func computeStateDiff(prev, next map[string]interface{}) map[string]interface{} {
	diff := make(map[string]interface{})
	for k, nv := range next {
		pv, exists := prev[k]
		if !exists || !deepEqual(pv, nv) {
			diff[k] = nv
		}
	}
	return diff
}

// deepEqual compares two values for equality with optimized paths for common types.
// Uses fast path for primitives and type-specific comparisons, avoiding expensive
// JSON marshaling except as final fallback for complex nested structures.
func deepEqual(a, b interface{}) bool {
	// Fast path: identical pointers or simple equality
	if a == b {
		return true
	}

	// Handle nil cases
	if a == nil || b == nil {
		return a == b
	}

	// Type check - different types can't be equal
	typeA, typeB := fmt.Sprintf("%T", a), fmt.Sprintf("%T", b)
	if typeA != typeB {
		return false
	}

	// Fast paths for common primitive types
	switch av := a.(type) {
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case int:
		bv, ok := b.(int)
		return ok && av == bv
	case int64:
		bv, ok := b.(int64)
		return ok && av == bv
	case float64:
		bv, ok := b.(float64)
		return ok && av == bv
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case []byte:
		bv, ok := b.([]byte)
		return ok && bytes.Equal(av, bv)
	case map[string]interface{}:
		bv, ok := b.(map[string]interface{})
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			if bvVal, exists := bv[k]; !exists || !deepEqual(v, bvVal) {
				return false
			}
		}
		return true
	case []interface{}:
		bv, ok := b.([]interface{})
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !deepEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	}

	// Reflection-based comparison for slices of other types
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	switch av.Kind() {
	case reflect.Slice, reflect.Array:
		if av.Len() != bv.Len() {
			return false
		}
		for i := 0; i < av.Len(); i++ {
			if !deepEqual(av.Index(i).Interface(), bv.Index(i).Interface()) {
				return false
			}
		}
		return true
	case reflect.Map:
		if av.Len() != bv.Len() {
			return false
		}
		for _, key := range av.MapKeys() {
			aVal := av.MapIndex(key)
			bVal := bv.MapIndex(key)
			if !bVal.IsValid() || !deepEqual(aVal.Interface(), bVal.Interface()) {
				return false
			}
		}
		return true
	}

	// Final fallback: JSON comparison for complex nested structures
	aJSON, err1 := json.Marshal(a)
	bJSON, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		// String comparison as last resort
		return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
	}
	return bytes.Equal(aJSON, bJSON)
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
	// CompressState enables gzip compression of outbound state payloads.
	// The client receives { type:'compressed', data: '<base64>', compressed:true }
	// and must decompress using the DecompressionStream browser API.
	CompressState bool
	// StateDiffing enables delta-only 'patch' messages instead of full state syncs.
	// When enabled only changed keys are broadcast after the initial snapshot.
	StateDiffing bool
	// Serializer overrides JSON for outbound state serialization.
	Serializer func(interface{}) ([]byte, error)
	// Deserializer overrides JSON for inbound state deserialization.
	Deserializer func([]byte, interface{}) error
}

// DefaultWebSocketConfig returns default WebSocket configuration.
// NOTE: The caller is responsible for starting the hub with `go hub.Run()` before
// registering the handler. gospa.New() does this automatically when EnableWebSocket is true.
func DefaultWebSocketConfig() WebSocketConfig {
	return WebSocketConfig{
		Hub:        NewWSHub(nil),
		GenerateID: generateComponentID,
	}
}

// WebSocketHandler creates a WebSocket handler.
// IMPORTANT: The Hub in config must already be running (go hub.Run()) before calling this.
// gospa.New() ensures this when EnableWebSocket is true. If you call this directly,
// start the hub yourself: go config.Hub.Run()
func WebSocketHandler(config WebSocketConfig) fiber.Handler {
	// Apply defaults for nil config values
	if config.Hub == nil {
		config.Hub = NewWSHub(nil)
		// When creating a default hub here, start it since caller didn't
		go config.Hub.Run()
	}
	if config.GenerateID == nil {
		config.GenerateID = generateComponentID
	}

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
		// Wire optional features from config
		client.compress = config.CompressState
		client.stateDiffing = config.StateDiffing
		client.serializer = config.Serializer
		client.deserializer = config.Deserializer

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
			var saveMutex sync.Mutex
			var saveTimer *time.Timer
			client.State.OnChange = func(key string, value any) {
				// Save state to persistent store safely, debounced
				saveMutex.Lock()
				if saveTimer != nil {
					saveTimer.Stop()
				}
				saveTimer = time.AfterFunc(100*time.Millisecond, func() {
					globalClientStateStore.Save(sessionID, client.State)
				})
				saveMutex.Unlock()

				// Parse componentId and local key for Svelte updates
				componentID := ""
				localKey := key
				if dotIdx := strings.Index(key, "."); dotIdx > 0 {
					componentID = key[:dotIdx]
					localKey = key[dotIdx+1:]
				}

				// Broadcast state change to all clients sharing this session ID via pubsub
				syncMsg := map[string]interface{}{
					"type":        "sync",
					"componentId": componentID,
					"key":         localKey,
					"value":       value,
					"_sessionID":  sessionID,
				}
				data, err := json.Marshal(syncMsg)
				if err == nil {
					_ = config.Hub.pubsub.Publish("gospa:broadcast", data)
				}
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
		stateStr, _ := client.State.ToJSON()
		sendResponse(map[string]interface{}{
			"type":        "init",
			"componentId": msg.ComponentID,
			"state":       json.RawMessage([]byte(stateStr)),
		})

	case "update":
		var update WSStateUpdate
		var unmarshalErr error
		if client.deserializer != nil {
			unmarshalErr = client.deserializer(msg.Payload, &update)
		} else {
			unmarshalErr = json.Unmarshal(msg.Payload, &update)
		}
		if unmarshalErr != nil {
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
		client.actionMu.Lock()
		now := time.Now()
		elapsed := now.Sub(client.actionLastRefill).Seconds()
		client.actionTokens += elapsed * 5.0 // refill at 5 actions/sec
		if client.actionTokens > 10.0 {
			client.actionTokens = 10.0
		}
		client.actionLastRefill = now

		if client.actionTokens < 1.0 {
			client.actionMu.Unlock()
			sendResponse(map[string]interface{}{
				"type":  "error",
				"error": "Rate limit exceeded",
			})
			return
		}
		client.actionTokens -= 1.0
		client.actionMu.Unlock()

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
// It also enforces per-IP rate limiting to prevent connection DoS attacks.
func WebSocketUpgradeMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if WebSocket upgrade request
		if string(c.Request().Header.Peek("Upgrade")) != "websocket" {
			return c.Next()
		}

		// SECURITY: Apply per-IP rate limiting for WebSocket connections
		clientIP := GetIPFromContext(c)
		if !globalConnRateLimiter.Allow(clientIP) {
			log.Printf("WebSocket connection rate limit exceeded for IP: %s", clientIP)
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded. Please try again later.",
			})
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
