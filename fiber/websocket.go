package fiber

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/aydenstechdungeon/gospa/state"
	"github.com/aydenstechdungeon/gospa/store"
	json "github.com/goccy/go-json"
	websocket "github.com/gofiber/contrib/v3/websocket"
	fiberpkg "github.com/gofiber/fiber/v3"
	"github.com/vmihailenco/msgpack/v5"
)

const (
	// Time allowed to keep an idle connection alive.
	pongWait = 60 * time.Second
	// Send pings to peer with this period.
	pingPeriod = (pongWait * 9) / 10
)

// ConnectionRateLimiter implements per-IP rate limiting for WebSocket connections
// to prevent DoS attacks. Uses token bucket algorithm with burst capacity.
// It can optionally use a store.Storage backend (like Redis) for multi-process environments.
type ConnectionRateLimiter struct {
	mu      sync.RWMutex
	buckets map[string]*rateBucket // IP -> bucket (for in-memory fallback)
	storage store.Storage          // Optional storage backend (e.g., Redis)
	// Configurable limits
	maxTokens       float64       // Maximum burst (default: 5)
	refillRate      float64       // Tokens per second (default: 0.2 = 1 per 5 sec)
	cleanupInterval time.Duration // How often to clean stale entries (in-memory only, default: 1 min)
	stop            chan struct{}
	stopOnce        sync.Once
}

type rateBucket struct {
	Tokens     float64   `json:"tokens"`
	LastRefill time.Time `json:"lastRefill"`
}

// SetConnectionRateLimiter configures the global connection rate limiter limits.
func SetConnectionRateLimiter(maxTokens float64, refillRate float64) {
	globalConnRateLimiter.mu.Lock()
	defer globalConnRateLimiter.mu.Unlock()
	globalConnRateLimiter.maxTokens = maxTokens
	globalConnRateLimiter.refillRate = refillRate
}

// SetStorage configures the global connection rate limiter to use an external storage backend.
func (rl *ConnectionRateLimiter) SetStorage(storage store.Storage) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.storage = storage
}

// SetLimits configures the burst capacity and refill rate for this specific limiter instance.
func (rl *ConnectionRateLimiter) SetLimits(maxTokens float64, refillRate float64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.maxTokens = maxTokens
	rl.refillRate = refillRate
}

// AtomicRateLimiterStorage allows storage backends to implement atomic distributed
// rate limiting semantics for shared deployments.
type AtomicRateLimiterStorage interface {
	ConsumeRateLimitToken(key string, now time.Time, maxTokens float64, refillRate float64, ttl time.Duration) (bool, error)
}

// Global connection rate limiter (singleton)
var globalConnRateLimiter = NewConnectionRateLimiter(store.NewMemoryStorage())

// Global remote action rate limiter (singleton, separate from websocket connections)
// Uses more aggressive rate limits (burst=50, refill=20/sec) suitable for API calls.
var globalRemoteActionRateLimiter = func() *ConnectionRateLimiter {
	rl := NewConnectionRateLimiter(store.NewMemoryStorage())
	rl.maxTokens = 50.0
	rl.refillRate = 20.0
	return rl
}()

// SetRemoteActionRateLimiter configures the global remote action rate limiter limits.
func SetRemoteActionRateLimiter(maxTokens float64, refillRate float64) {
	globalRemoteActionRateLimiter.mu.Lock()
	defer globalRemoteActionRateLimiter.mu.Unlock()
	globalRemoteActionRateLimiter.maxTokens = maxTokens
	globalRemoteActionRateLimiter.refillRate = refillRate
}

// NewConnectionRateLimiter creates a rate limiter with sensible defaults.
func NewConnectionRateLimiter(storage store.Storage) *ConnectionRateLimiter {
	rl := &ConnectionRateLimiter{
		buckets:         make(map[string]*rateBucket),
		storage:         storage,
		maxTokens:       5.0,         // Allow burst of 5 connections
		refillRate:      0.2,         // 1 connection per 5 seconds sustained
		cleanupInterval: time.Minute, // Cleanup stale entries every minute
		stop:            make(chan struct{}),
	}
	// Start cleanup goroutine
	go rl.cleanupLoop()
	return rl
}

// Allow checks if a connection from the given IP is allowed.
// Returns true if the connection should be accepted.
func (rl *ConnectionRateLimiter) Allow(ip string) bool {
	now := time.Now()

	rl.mu.RLock()
	storage := rl.storage
	maxTokens := rl.maxTokens
	refillRate := rl.refillRate
	rl.mu.RUnlock()

	var bucket *rateBucket

	if storage != nil {
		key := "rate:" + ip
		if atomicStorage, ok := storage.(AtomicRateLimiterStorage); ok {
			allowed, err := atomicStorage.ConsumeRateLimitToken(key, now, maxTokens, refillRate, 10*time.Minute)
			if err == nil {
				return allowed
			}
		}

		data, err := storage.Get(key)
		if err == nil {
			var b rateBucket
			if json.Unmarshal(data, &b) == nil {
				bucket = &b
			}
		}

		if bucket == nil {
			bucket = &rateBucket{
				Tokens:     maxTokens - 1,
				LastRefill: now,
			}
			newBytes, _ := json.Marshal(bucket)
			_ = storage.Set(key, newBytes, 10*time.Minute)
			return true
		}

		elapsed := now.Sub(bucket.LastRefill).Seconds()
		bucket.Tokens += elapsed * refillRate
		if bucket.Tokens > maxTokens {
			bucket.Tokens = maxTokens
		}
		bucket.LastRefill = now

		allowed := false
		if bucket.Tokens >= 1.0 {
			bucket.Tokens -= 1.0
			allowed = true
		}

		newBytes, _ := json.Marshal(bucket)
		_ = storage.Set(key, newBytes, 10*time.Minute)

		return allowed
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	maxTokens = rl.maxTokens
	refillRate = rl.refillRate

	// In-memory fallback
	bucket, exists := rl.buckets[ip]

	if !exists {
		// First connection from this IP
		rl.buckets[ip] = &rateBucket{
			Tokens:     maxTokens - 1, // Consume 1 token
			LastRefill: now,
		}
		return true
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(bucket.LastRefill).Seconds()
	bucket.Tokens += elapsed * refillRate
	if bucket.Tokens > maxTokens {
		bucket.Tokens = maxTokens
	}
	bucket.LastRefill = now

	// Check if we can consume a token
	if bucket.Tokens >= 1.0 {
		bucket.Tokens -= 1.0
		return true
	}

	return false
}

// GetIPFromContext extracts the client IP from the Fiber context.
// Uses Fiber's built-in IP extraction which handles X-Forwarded-For
// based on the app's Proxy settings (TrustedProxies, etc).
func GetIPFromContext(c fiberpkg.Ctx) string {
	return c.IP()
}

// cleanupLoop periodically removes stale rate limit entries to prevent memory leaks.
// Note: only cleans up the memory fallback map.
func (rl *ConnectionRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, bucket := range rl.buckets {
				// Remove entries that haven't been used in 10 minutes
				if now.Sub(bucket.LastRefill) > 10*time.Minute {
					delete(rl.buckets, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stop:
			return
		}
	}
}

// Close explicitly stops the rate limiter's cleanup goroutine.
func (rl *ConnectionRateLimiter) Close() {
	rl.stopOnce.Do(func() {
		close(rl.stop)
	})
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
// Returns the session token and an error if token generation or persistence fails.
func (s *SessionStore) CreateSession(clientID string) (string, error) {
	token := generateSecureToken()
	if token == "" {
		return "", fmt.Errorf("failed to generate secure session token")
	}
	entry := sessionEntry{
		ClientID:  clientID,
		ExpiresAt: time.Now().Add(SessionTTL),
	}
	bytes, err := json.Marshal(entry)
	if err != nil {
		return "", err
	}
	if err := s.storage.Set(token, bytes, SessionTTL); err != nil {
		return "", err
	}
	return token, nil
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
func (s *SessionStore) RemoveClientSessions(_ string) {
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
		slog.Default().Error("CRITICAL: crypto/rand.Read failed", "err", err)
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
	globalConnRateLimiter.SetStorage(storage)
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
	// maxMessageSize is the per-connection inbound frame size limit.
	maxMessageSize int64
	// optional features wired from WebSocketConfig at creation time
	compress     bool
	stateDiffing bool
	// lastSentState holds the snapshot used for StateDiffing
	lastSentStateMu sync.Mutex
	lastSentState   map[string]interface{}
	// Rate limiting
	actionMu         sync.Mutex
	actionTokens     float64
	actionLastRefill time.Time
	// transport
	format string
	// Custom serializer/deserializer from config
	serializer   func(interface{}) ([]byte, error)
	deserializer func([]byte, interface{}) error
}

// WSMessage represents a WebSocket message.
type WSMessage struct {
	Type         string                 `json:"type" msgpack:"type"`
	ComponentID  string                 `json:"componentId,omitempty" msgpack:"componentId,omitempty"`
	Action       string                 `json:"action,omitempty" msgpack:"action,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty" msgpack:"data,omitempty"`
	Payload      interface{}            `json:"payload,omitempty" msgpack:"payload,omitempty"`
	SessionToken string                 `json:"sessionToken,omitempty" msgpack:"sessionToken,omitempty"`
	ClientID     string                 `json:"clientId,omitempty" msgpack:"clientId,omitempty"`
}

// WSStateUpdate represents a state update message.
type WSStateUpdate struct {
	Key   string      `json:"key" msgpack:"key"`
	Value interface{} `json:"value" msgpack:"value"`
}

// WSHub maintains the set of active clients and broadcasts messages.
type WSHub struct {
	Clients          map[string]*WSClient
	ClientsBySession map[string]map[string]*WSClient // SessionID -> {ClientID -> *WSClient}
	Register         chan *WSClient
	Unregister       chan *WSClient
	Broadcast        chan []byte
	mu               sync.RWMutex
	pubsub           store.PubSub
	stop             chan struct{}
	// stopOnce ensures Close() is idempotent and never panics on double-call.
	stopOnce sync.Once
}

// NewWSHub creates a new WebSocket hub.
func NewWSHub(pubsub store.PubSub) *WSHub {
	if pubsub == nil {
		pubsub = store.NewMemoryPubSub()
	}
	h := &WSHub{
		Clients:          make(map[string]*WSClient),
		ClientsBySession: make(map[string]map[string]*WSClient),
		Register:         make(chan *WSClient),
		Unregister:       make(chan *WSClient),
		Broadcast:        make(chan []byte, 256),
		pubsub:           pubsub,
		stop:             make(chan struct{}),
	}

	// Subscribe to a global broadcast channel for state syncing across processes
	_, _ = h.pubsub.Subscribe("gospa:broadcast", func(message []byte) {
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

		if sessionID != "" {
			// Targeted broadcast to specific session (O(1) session lookup)
			if clients, ok := h.ClientsBySession[sessionID]; ok {
				for _, client := range clients {
					select {
					case client.Send <- message:
					default:
					}
				}
			}
			return
		}

		// Global broadcast (O(N) iteration unavoidable here)
		for _, client := range h.Clients {
			select {
			case client.Send <- message:
			default:
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
				// Cleanup existing indexing
				if oldClient.SessionID != "" {
					if clients, ok := h.ClientsBySession[oldClient.SessionID]; ok {
						delete(clients, oldClient.ID)
						if len(clients) == 0 {
							delete(h.ClientsBySession, oldClient.SessionID)
						}
					}
				}
				_ = oldClient.Conn.Close()
			}

			h.Clients[client.ID] = client
			if client.SessionID != "" {
				if h.ClientsBySession[client.SessionID] == nil {
					h.ClientsBySession[client.SessionID] = make(map[string]*WSClient)
				}
				h.ClientsBySession[client.SessionID][client.ID] = client
			}
			h.mu.Unlock()
			slog.Default().Info("client connected", "id", client.ID)

		case client := <-h.Unregister:
			h.mu.Lock()
			if existing, ok := h.Clients[client.ID]; ok && existing == client {
				delete(h.Clients, client.ID)
				if client.SessionID != "" {
					if clients, ok := h.ClientsBySession[client.SessionID]; ok {
						delete(clients, client.ID)
						if len(clients) == 0 {
							delete(h.ClientsBySession, client.SessionID)
						}
					}
				}
				// Use guarded Close() to prevent double-close panics
				client.Close()
			}
			h.mu.Unlock()
			slog.Default().Info("client disconnected", "id", client.ID)

		case message := <-h.Broadcast:
			// Instead of directly sending to local clients, publish to the PubSub system.
			// The PubSub subscription handler will broadcast it locally.
			_ = h.pubsub.Publish("gospa:broadcast", message)
		case <-h.stop:
			return
		}
	}
}

// Close explicitly stops the WSHub loop.
// It is safe to call Close multiple times.
func (h *WSHub) Close() {
	h.stopOnce.Do(func() {
		close(h.stop)
	})
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
func NewWSClient(id string, conn *websocket.Conn, config WebSocketConfig) *WSClient {
	return &WSClient{
		ID:               id,
		Conn:             conn,
		Send:             make(chan []byte, 256),
		State:            state.NewStateMap(),
		closed:           false,
		maxMessageSize:   maxWSMessageSize,
		actionTokens:     10.0,
		actionLastRefill: time.Now(),
		lastSentState:    make(map[string]interface{}),
		compress:         config.CompressState,
		stateDiffing:     config.StateDiffing,
		format:           config.SerializationFormat,
		serializer:       config.Serializer,
		deserializer:     config.Deserializer,
	}
}

// maxWSMessageSize is the maximum WebSocket message size we accept (64KB).
const maxWSMessageSize = 64 * 1024

// maxActionNameLen is the maximum length of an action name field.
const maxActionNameLen = 256

// maxJSONDepth is the maximum nesting depth allowed for WebSocket JSON messages.
const maxJSONDepth = 64

// validateJSONDepth checks that JSON data doesn't exceed the maximum nesting depth.
func validateJSONDepth(data []byte, maxDepth int) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	depth := 0
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("invalid JSON: %w", err)
		}
		switch token {
		case json.Delim('{'), json.Delim('['):
			depth++
			if depth > maxDepth {
				return fmt.Errorf("JSON nesting depth exceeds %d", maxDepth)
			}
		case json.Delim('}'), json.Delim(']'):
			depth--
		}
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub.
func (c *WSClient) ReadPump(hub *WSHub, onMessage func(*WSClient, WSMessage)) {
	defer func() {
		hub.Unregister <- c
		c.Close()
	}()

	// Limit inbound message size to prevent DoS attacks
	c.Conn.SetReadLimit(c.maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure, websocket.CloseAbnormalClosure) {
				slog.Default().Warn("ws disconnect", "client", c.ID, "err", err)
			}
			break
		}

		// Reset read deadline on every message received to keep the connection alive
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))

		// Validate JSON nesting depth to prevent stack overflow attacks
		if c.format != "msgpack" {
			if err := validateJSONDepth(message, maxJSONDepth); err != nil {
				c.SendError("JSON nesting too deep")
				continue
			}
		}

		var msg WSMessage
		if err := c.Unmarshal(message, &msg); err != nil {
			c.SendError("Invalid message format")
			continue
		}

		// Sanitize field lengths to prevent injection via long strings
		if len(msg.Action) > maxActionNameLen {
			c.SendError("Action name too long")
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
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.mu.Lock()
			if c.closed || !ok {
				c.mu.Unlock()
				return
			}
			messageType := websocket.TextMessage
			if c.format == "msgpack" {
				messageType = websocket.BinaryMessage
			}
			err := c.Conn.WriteMessage(messageType, message)
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

// Marshal marshals a value using the client's configured format.
func (c *WSClient) Marshal(v interface{}) ([]byte, error) {
	if c.serializer != nil {
		return c.serializer(v)
	}
	if c.format == "msgpack" {
		return msgpack.Marshal(v)
	}
	return json.Marshal(v)
}

// Unmarshal unmarshals a value using the client's configured format.
func (c *WSClient) Unmarshal(data []byte, v interface{}) error {
	if c.deserializer != nil {
		return c.deserializer(data, v)
	}
	if c.format == "msgpack" {
		// Only allow unmarshaling into known safe types to prevent type confusion attacks.
		if !isSafeMsgpackTarget(v) {
			return fmt.Errorf("msgpack: unmarshal into unsupported type %T", v)
		}
		return msgpack.Unmarshal(data, v)
	}
	return json.Unmarshal(data, v)
}

// isSafeMsgpackTarget returns true if the target type is a known safe struct type
// for msgpack deserialization, preventing arbitrary type instantiation.
func isSafeMsgpackTarget(v interface{}) bool {
	switch v.(type) {
	case *WSMessage, *WSStateUpdate, *sessionEntry, *rateBucket:
		return true
	}
	// Allow maps and slices of interface{} (common pattern)
	switch v.(type) {
	case *map[string]interface{}, *[]interface{}:
		return true
	}
	return false
}

// SendJSON sends a message to the client using the configured format.
func (c *WSClient) SendJSON(v interface{}) error {
	data, err := c.Marshal(v)
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

	var stateData interface{}
	var err error
	if c.serializer != nil {
		stateData, err = c.serializer(stateMap)
		// If using JSON format, custom serializer output ([]byte) must be RawMessage
		if err == nil && c.format != "msgpack" {
			if b, ok := stateData.([]byte); ok {
				stateData = json.RawMessage(b)
			}
		}
	} else {
		if c.format == "msgpack" {
			stateData = stateMap
		} else {
			s, _ := c.State.ToJSON()
			stateData = json.RawMessage(s)
		}
	}
	if err != nil {
		c.SendError("Failed to serialize state")
		return
	}
	c.sendEncodedPayload(map[string]interface{}{
		"type":  "init",
		"state": stateData,
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

	var stateData interface{}
	var err error
	if c.serializer != nil {
		stateData, err = c.serializer(stateMap)
		if err == nil && c.format != "msgpack" {
			if b, ok := stateData.([]byte); ok {
				stateData = json.RawMessage(b)
			}
		}
	} else {
		if c.format == "msgpack" {
			stateData = stateMap
		} else {
			s, _ := c.State.ToJSON()
			stateData = json.RawMessage(s)
		}
	}
	if err != nil {
		c.SendError("Failed to serialize state")
		return
	}
	c.sendEncodedPayload(map[string]interface{}{
		"type":         "init",
		"state":        stateData,
		"sessionToken": sessionToken,
		"clientId":     c.SessionID,
	})
}

// sendEncodedPayload marshals msg and optionally gzip-compresses it before
// queueing on the Send channel.
func (c *WSClient) sendEncodedPayload(payload interface{}) {
	if c.compress {
		data, err := c.Marshal(payload)
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
	_ = c.SendJSON(payload)
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
	typeA, typeB := reflect.TypeOf(a), reflect.TypeOf(b)
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

	// Reflection-based comparison for slices, arrays, maps, and structs
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
	case reflect.Struct:
		// Compare struct fields recursively
		if av.NumField() != bv.NumField() {
			return false
		}
		for i := 0; i < av.NumField(); i++ {
			fieldA := av.Field(i)
			fieldB := bv.Field(i)
			// Skip unexported fields (they can't be compared reliably)
			if !fieldA.CanInterface() || !fieldB.CanInterface() {
				continue
			}
			if !deepEqual(fieldA.Interface(), fieldB.Interface()) {
				return false
			}
		}
		return true
	case reflect.Ptr:
		// Handle pointer comparison
		if av.IsNil() && bv.IsNil() {
			return true
		}
		if av.IsNil() || bv.IsNil() {
			return false
		}
		return deepEqual(av.Elem().Interface(), bv.Elem().Interface())
	case reflect.Interface:
		// Handle interface comparison
		if av.IsNil() && bv.IsNil() {
			return true
		}
		if av.IsNil() || bv.IsNil() {
			return false
		}
		return deepEqual(av.Elem().Interface(), bv.Elem().Interface())
	}

	// Final fallback: use pure reflection to handle complex nested structures without JSON allocations
	return reflect.DeepEqual(a, b)
}

// Close closes the client connection.
func (c *WSClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed {
		c.closed = true
		close(c.Send)
		_ = c.Conn.Close()
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
	// SerializationFormat sets the underlying format for all WebSocket communications.
	SerializationFormat string
	// WSMaxMessageSize limits the maximum payload size for WebSocket messages.
	WSMaxMessageSize int
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
func WebSocketHandler(config WebSocketConfig) fiberpkg.Handler {
	// Apply defaults for nil config values
	if config.Hub == nil {
		config.Hub = NewWSHub(nil)
		go config.Hub.Run()
	}
	if config.GenerateID == nil {
		config.GenerateID = generateComponentID
	}

	// Fiber v3: websocket.New returns a fiber.Handler (func(Ctx) error)
	// The websocket upgrade check is performed inside websocket.New.
	return websocket.New(func(c *websocket.Conn) {
		if config.WSMaxMessageSize > 0 {
			c.SetReadLimit(int64(config.WSMaxMessageSize))
		}
		var sessionID string
		var sessionToken string
		var restoredState *state.StateMap

		// Session token is received via the first WebSocket message (preferred method).
		// This avoids URL-based token leakage in logs/referer headers.

		// Generate unique connection ID so tabs don't kick each other off
		connID := "conn_" + generateSecureToken()[:8]

		// Create client with placeholder session (will be updated after auth)
		client := NewWSClient(connID, c, config)
		client.SessionID = "" // Will be set after session validation
		if config.WSMaxMessageSize > 0 {
			client.maxMessageSize = int64(config.WSMaxMessageSize)
		}
		// Wire optional features from config
		client.compress = config.CompressState
		client.stateDiffing = config.StateDiffing
		client.serializer = config.Serializer
		client.deserializer = config.Deserializer

		// Register client with timeout to prevent blocking if hub is slow
		select {
		case config.Hub.Register <- client:
		case <-time.After(5 * time.Second):
			slog.Default().Warn("ws register timeout", "client", connID)
			_ = c.Close()
			return
		}

		// Set up read deadline for initial auth message
		_ = c.SetReadDeadline(time.Now().Add(10 * time.Second))

		// Wait for first message (should be init with session token)
		_, firstMsg, err := c.ReadMessage()
		if err != nil {
			slog.Default().Warn("failed to read initial ws message", "client", connID, "err", err)
			config.Hub.Unregister <- client
			_ = c.Close()
			return
		}

		var initMsg WSMessage
		if err := client.Unmarshal(firstMsg, &initMsg); err != nil {
			slog.Default().Warn("invalid initial ws message format", "client", connID, "err", err)
			client.SendError("Invalid initial message format")
			config.Hub.Unregister <- client
			_ = c.Close()
			return
		}

		// Handle session authentication
		// 1. Try cookie from middleware locals or direct header (most secure)
		cookieToken := c.Cookies("gospa_session")
		if cookieToken == "" {
			// Fallback: check if it was set in locals by middleware
			if l, ok := c.Locals("gospa.session").(string); ok {
				cookieToken = l
			}
		}

		if cookieToken != "" {
			if prevSessionID, ok := globalSessionStore.ValidateSession(cookieToken); ok {
				if savedState, hasState := globalClientStateStore.Get(prevSessionID); hasState {
					sessionID = prevSessionID
					restoredState = savedState
					sessionToken = cookieToken
				}
			}
		}

		// 2. Fallback: Removed for security (token-in-body anti-pattern)
		// To maintain backward compatibility, we still allow the sessionID to be empty
		// here, and it will be generated below if cookieToken was missing.

		// If no valid session, generate new session ID
		if sessionID == "" {
			sessionID = config.GenerateID()
			token, err := globalSessionStore.CreateSession(sessionID)
			if err != nil {
				slog.Default().Error("failed to create websocket session", "session_id", sessionID, "err", err)
				client.SendError("Failed to create session")
				_ = c.Close()
				return
			}
			sessionToken = token
		}

		// Update client with session ID
		client.SessionID = sessionID

		// Set up state change handler BEFORE sending initial state
		// This ensures we don't miss the first state change for new sessions
		var saveMutex sync.Mutex
		var saveTimer *time.Timer

		// Clean up saveTimer and OnChange on disconnect to prevent
		// the callback from firing after the client is gone
		defer func() {
			saveMutex.Lock()
			if saveTimer != nil {
				saveTimer.Stop()
			}
			saveMutex.Unlock()
			client.State.OnChange = nil
		}()

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

		// Restore previous state if available, passing pointer
		if restoredState != nil {
			client.State = restoredState
		} else {
			// Save initial state for new sessions
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

		// Call init function with the first message received (so it's not lost)
		onMessage(client, initMsg)

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
		var stateData interface{}
		if client.format == "msgpack" {
			stateData = client.State.ToMap()
		} else {
			stateStr, _ := client.State.ToJSON()
			stateData = json.RawMessage([]byte(stateStr))
		}
		sendResponse(map[string]interface{}{
			"type":        "init",
			"componentId": msg.ComponentID,
			"state":       stateData,
		})

	case "update":
		var update WSStateUpdate
		var unmarshalErr error
		payloadBytes, payloadIsBytes := msg.Payload.([]byte)
		if client.deserializer != nil {
			if payloadIsBytes {
				unmarshalErr = client.deserializer(payloadBytes, &update)
			} else {
				// Try to re-marshal if it's already a map (JSON case)
				b, _ := json.Marshal(msg.Payload)
				unmarshalErr = client.deserializer(b, &update)
			}
		} else {
			if payloadIsBytes {
				unmarshalErr = json.Unmarshal(payloadBytes, &update)
			} else {
				// Already unmarshaled by json into interface{}
				b, _ := json.Marshal(msg.Payload)
				unmarshalErr = json.Unmarshal(b, &update)
			}
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
		slog.Default().Debug("ws action received", "action", action, "client", client.ID)

		if handler, ok := GetActionHandler(action); ok {
			var payload interface{}
			if b, ok := msg.Payload.([]byte); ok {
				// If it's a byte slice, it's either RawMessage or direct binary
				if client.format == "msgpack" {
					payload = b // Keep as bytes for msgpack (handler might decode it)
				} else {
					payload = json.RawMessage(b)
				}
			} else {
				payload = msg.Payload
			}
			handler(client, payload)
			sendResponse(map[string]interface{}{
				"type": "action_ack",
			})
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
type ActionHandler func(client *WSClient, payload interface{})

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
// Returns a function to unregister the handler.
func RegisterOnConnectHandler(handler ConnectHandler) func() {
	connectMu.Lock()
	defer connectMu.Unlock()
	connectHandlers = append(connectHandlers, handler)
	idx := len(connectHandlers) - 1
	return func() {
		connectMu.Lock()
		defer connectMu.Unlock()
		if idx < len(connectHandlers) {
			connectHandlers = append(connectHandlers[:idx], connectHandlers[idx+1:]...)
		}
	}
}

// callConnectHandlers calls all registered connect handlers.
func callConnectHandlers(client *WSClient) {
	connectMu.RLock()
	defer connectMu.RUnlock()
	for _, handler := range connectHandlers {
		handler(client)
	}
}

// WebSocketUpgradeMiddleware enforces per-IP rate limiting before WebSocket upgrade.
func WebSocketUpgradeMiddleware() fiberpkg.Handler {
	return func(c fiberpkg.Ctx) error {
		// Check if WebSocket upgrade request
		if !c.IsWebSocket() {
			return c.Next()
		}

		// SECURITY: Apply per-IP rate limiting for WebSocket connections
		clientIP := GetIPFromContext(c)
		if !globalConnRateLimiter.Allow(clientIP) {
			slog.Default().Warn("ws rate limit exceeded", "ip", clientIP)
			return c.Status(fiberpkg.StatusTooManyRequests).JSON(fiberpkg.Map{
				"error": "Rate limit exceeded. Please try again later.",
			})
		}

		return c.Next()
	}
}

// RemoteActionRateLimitMiddleware enforces per-IP rate limiting for the HTTP remote action endpoint.
func RemoteActionRateLimitMiddleware() fiberpkg.Handler {
	return func(c fiberpkg.Ctx) error {
		clientIP := GetIPFromContext(c)
		if !globalRemoteActionRateLimiter.Allow(clientIP) {
			slog.Default().Warn("remote action rate limit exceeded", "ip", clientIP)
			return c.Status(fiberpkg.StatusTooManyRequests).JSON(fiberpkg.Map{
				"error": "Rate limit exceeded. Please try again later.",
			})
		}
		return c.Next()
	}
}

// StateSyncHandler creates a handler for state synchronization.
func StateSyncHandler(_ *WSHub) fiberpkg.Handler {
	return func(c fiberpkg.Ctx) error {
		sessionToken := c.Get("X-Session-Token")

		if sessionToken == "" {
			return c.Status(fiberpkg.StatusUnauthorized).JSON(fiberpkg.Map{
				"error": "Session token required",
			})
		}

		sessionID, ok := globalSessionStore.ValidateSession(sessionToken)
		if !ok {
			return c.Status(fiberpkg.StatusUnauthorized).JSON(fiberpkg.Map{
				"error": "Invalid session",
			})
		}

		stateMap, ok := globalClientStateStore.Get(sessionID)
		if !ok {
			return c.Status(fiberpkg.StatusNotFound).JSON(fiberpkg.Map{
				"error": "Session state not found",
			})
		}

		var update WSStateUpdate
		if err := json.Unmarshal(c.Body(), &update); err != nil {
			return c.Status(fiberpkg.StatusBadRequest).JSON(fiberpkg.Map{
				"error": "Invalid update payload",
			})
		}

		if obs, ok := stateMap.Get(update.Key); ok {
			if settable, isSettable := obs.(state.Settable); isSettable {
				_ = settable.SetAny(update.Value)
			}
		} else {
			r := state.NewRune(update.Value)
			stateMap.Add(update.Key, r)
		}

		return c.JSON(fiberpkg.Map{
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
