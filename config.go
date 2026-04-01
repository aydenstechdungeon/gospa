package gospa

import (
	"io/fs"
	"log/slog"
	"os"
	"time"

	"github.com/aydenstechdungeon/gospa/fiber"
	"github.com/aydenstechdungeon/gospa/routing"
	"github.com/aydenstechdungeon/gospa/store"
	fiberpkg "github.com/gofiber/fiber/v3"
)

// Version is the current version of GoSPA.
const Version = "0.1.32"

// Serialization formats
const (
	SerializationJSON    = "json"
	SerializationMsgPack = "msgpack"
)

// NavigationSpeculativePrefetchingConfig configures speculative prefetching
type NavigationSpeculativePrefetchingConfig struct {
	Enabled        *bool `json:"enabled,omitempty"`
	TTL            *int  `json:"ttl,omitempty"`
	HoverDelay     *int  `json:"hoverDelay,omitempty"`
	ViewportMargin *int  `json:"viewportMargin,omitempty"`
}

// NavigationURLParsingCacheConfig configures the URL parsing cache
type NavigationURLParsingCacheConfig struct {
	Enabled *bool `json:"enabled,omitempty"`
	MaxSize *int  `json:"maxSize,omitempty"`
	TTL     *int  `json:"ttl,omitempty"`
}

// NavigationIdleCallbackBatchUpdatesConfig configures idle callback batching
type NavigationIdleCallbackBatchUpdatesConfig struct {
	Enabled             *bool `json:"enabled,omitempty"`
	FallbackToMicrotask *bool `json:"fallbackToMicrotask,omitempty"`
}

// NavigationLazyRuntimeInitializationConfig configures lazy runtime init
type NavigationLazyRuntimeInitializationConfig struct {
	Enabled       *bool `json:"enabled,omitempty"`
	DeferBindings *bool `json:"deferBindings,omitempty"`
}

// NavigationServiceWorkerCachingConfig configures service worker caching
type NavigationServiceWorkerCachingConfig struct {
	Enabled   *bool  `json:"enabled,omitempty"`
	CacheName string `json:"cacheName,omitempty"`
	Path      string `json:"path,omitempty"`
}

// NavigationViewTransitionsConfig configures view transitions
type NavigationViewTransitionsConfig struct {
	Enabled           *bool `json:"enabled,omitempty"`
	FallbackToClassic *bool `json:"fallbackToClassic,omitempty"`
}

// NavigationProgressBarConfig configures the navigation progress bar
type NavigationProgressBarConfig struct {
	Enabled *bool   `json:"enabled,omitempty"`
	Color   *string `json:"color,omitempty"`
	Height  *string `json:"height,omitempty"`
}

// NavigationOptions configures client-side navigation
type NavigationOptions struct {
	SpeculativePrefetching         *NavigationSpeculativePrefetchingConfig    `json:"speculativePrefetching,omitempty"`
	URLParsingCache                *NavigationURLParsingCacheConfig           `json:"urlParsingCache,omitempty"`
	IdleCallbackBatchUpdates       *NavigationIdleCallbackBatchUpdatesConfig  `json:"idleCallbackBatchUpdates,omitempty"`
	LazyRuntimeInitialization      *NavigationLazyRuntimeInitializationConfig `json:"lazyRuntimeInitialization,omitempty"`
	ServiceWorkerNavigationCaching *NavigationServiceWorkerCachingConfig      `json:"serviceWorkerNavigationCaching,omitempty"`
	ViewTransitions                *NavigationViewTransitionsConfig           `json:"viewTransitions,omitempty"`
	ProgressBar                    *NavigationProgressBarConfig               `json:"progressBar,omitempty"`
}

// StateSerializerFunc defines a function for state serialization
type StateSerializerFunc func(interface{}) ([]byte, error)

// StateDeserializerFunc defines a function for state deserialization
type StateDeserializerFunc func([]byte, interface{}) error

// Config holds the application configuration.
type Config struct {
	// RoutesDir is the directory containing route files.
	RoutesDir string
	// RoutesFS is the filesystem containing route files (optional). Takes precedence over RoutesDir if provided.
	RoutesFS fs.FS
	// DevMode enables development features.
	DevMode bool
	// RuntimeScript is the path to the client runtime script.
	RuntimeScript string
	// StaticDir is the directory for static files.
	StaticDir string
	// StaticPrefix is the URL prefix for static files.
	StaticPrefix string
	// AppName is the application name.
	AppName string
	// DefaultState is the initial state for new sessions.
	DefaultState map[string]interface{}
	// EnableWebSocket enables WebSocket support.
	EnableWebSocket bool
	// WebSocketPath is the WebSocket endpoint path.
	WebSocketPath string
	// WebSocketMiddleware allows injecting session/auth middleware before WebSocket upgrade.
	WebSocketMiddleware fiberpkg.Handler
	// Logger is the structured logger. Defaults to slog.Default().
	Logger *slog.Logger

	// Performance Options
	// CompressState enables gzip compression of outbound WebSocket state payloads.
	CompressState bool
	// StateDiffing enables delta-only "patch" WebSocket messages for state syncs.
	StateDiffing   bool
	CacheTemplates bool // Cache compiled templates (SSG only)
	SimpleRuntime  bool // Use lightweight runtime without DOMPurify (~6KB smaller)
	// SimpleRuntimeSVGs allows SVG elements in the simple runtime sanitizer.
	SimpleRuntimeSVGs bool
	// DisableSanitization disables client-side HTML sanitization for SPA navigation.
	DisableSanitization bool
	// NotificationBufferSize sets the size of the state change notification queue (default 1024).
	NotificationBufferSize int

	// WebSocket Options — these values are passed directly to the client runtime's init() call.
	WSReconnectDelay time.Duration // Initial reconnect delay (default 1s)
	WSMaxReconnect   int           // Max reconnect attempts (default 10)
	WSHeartbeat      time.Duration // Heartbeat ping interval (default 30s)

	// WSMaxMessageSize limits the maximum payload size for WebSocket messages (default 64KB).
	WSMaxMessageSize int
	// WSConnRateLimit sets the refilling rate in connections per second for WebSocket upgrades (default 1.5).
	WSConnRateLimit float64
	// WSConnBurst sets the burst capacity for WebSocket connection upgrades (default 15.0).
	WSConnBurst float64

	// Hydration Options
	HydrationMode    string
	HydrationTimeout int // ms before force hydrate

	// Serialization Options
	SerializationFormat string
	StateSerializer     StateSerializerFunc
	StateDeserializer   StateDeserializerFunc

	// Routing Options
	DisableSPA bool // Disable SPA navigation completely

	// Rendering Strategy Defaults
	DefaultRenderStrategy  routing.RenderStrategy
	DefaultRevalidateAfter time.Duration

	// Remote Action Options
	MaxRequestBodySize                int              // Maximum allowed size for remote action request bodies
	RemotePrefix                      string           // Prefix for remote action endpoints (default "/_gospa/remote")
	RemoteActionMiddleware            fiberpkg.Handler // Optional middleware
	AllowUnauthenticatedRemoteActions bool             // Default false

	// Security Options
	AllowedOrigins        []string
	EnableCSRF            bool
	ContentSecurityPolicy string
	PublicOrigin          string
	// AllowInsecureWS allows unsecure ws:// connections even on https:// pages.
	// This is useful for development setups with reverse proxies that don't support wss://.
	AllowInsecureWS bool
	// AllowPortsWithInsecureWS allows unsecure ws:// connections for these specific ports, even on https:// pages.
	// This is useful for development setups with reverse proxies that don't support wss://.
	// Defaults to []int{3000}.
	AllowPortsWithInsecureWS []int
	SSGCacheMaxEntries       int           // Default: 500
	SSGCacheTTL              time.Duration // Default: 0 (no expiry)

	// Prefork enables Fiber's prefork mode.
	Prefork bool

	// Storage defines the external storage backend for sessions and state.
	Storage store.Storage

	// PubSub defines the messaging backend for multi-process broadcasting.
	PubSub store.PubSub

	// NavigationOptions configures optional client-side navigation behavior.
	NavigationOptions NavigationOptions

	// ISR Options
	// ISRSemaphoreLimit limits concurrent ISR background revalidations.
	ISRSemaphoreLimit int
	// ISRTimeout sets the maximum time for a background ISR revalidation.
	ISRTimeout time.Duration
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		RoutesDir:                "./routes",
		DevMode:                  false,
		RuntimeScript:            "/_gospa/runtime.js",
		StaticDir:                "./static",
		StaticPrefix:             "/static",
		AppName:                  "GoSPA App",
		DefaultState:             make(map[string]interface{}),
		EnableWebSocket:          true,
		WebSocketPath:            "/_gospa/ws",
		RemotePrefix:             "/_gospa/remote",
		MaxRequestBodySize:       4 * 1024 * 1024,
		SerializationFormat:      SerializationJSON,
		EnableCSRF:               true,
		ContentSecurityPolicy:    fiber.DefaultContentSecurityPolicy,
		ISRSemaphoreLimit:        10,
		ISRTimeout:               60 * time.Second,
		NotificationBufferSize:   1024,
		AllowInsecureWS:          os.Getenv("GOSPA_WS_INSECURE") == "1",
		AllowPortsWithInsecureWS: []int{3000},
	}
}

// ProductionConfig returns an opinionated production-ready baseline.
func ProductionConfig() Config {
	config := DefaultConfig()
	config.DevMode = false
	config.CacheTemplates = true
	config.WSReconnectDelay = time.Second
	config.WSMaxReconnect = 10
	config.WSHeartbeat = 30 * time.Second
	config.SSGCacheMaxEntries = 500
	return config
}

// MinimalConfig returns a smaller baseline.
func MinimalConfig() Config {
	config := DefaultConfig()
	config.EnableWebSocket = false
	config.CompressState = false
	config.StateDiffing = false
	config.WSReconnectDelay = 0
	config.WSMaxReconnect = 0
	config.WSHeartbeat = 0
	return config
}
