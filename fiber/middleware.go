package fiber

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/aydenstechdungeon/gospa/embed"
	"github.com/aydenstechdungeon/gospa/state"
	gospatempl "github.com/aydenstechdungeon/gospa/templ"
	gofiber "github.com/gofiber/fiber/v2"
)

// Config holds the SPA middleware configuration.
type Config struct {
	// RuntimeScript is the path to the client runtime script
	RuntimeScript string
	// StateKey is the context key for storing state
	StateKey string
	// ComponentIDKey is the context key for component IDs
	ComponentIDKey string
	// DevMode enables development features
	DevMode bool
	// DefaultState is the initial state for new sessions
	DefaultState map[string]interface{}
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		RuntimeScript:  "/_gospa/runtime.js",
		StateKey:       "gospa.state",
		ComponentIDKey: "gospa.componentID",
		DevMode:        false,
		DefaultState:   make(map[string]interface{}),
	}
}

// SPAMiddleware creates a Fiber middleware for SPA support.
func SPAMiddleware(config Config) gofiber.Handler {
	return func(c *gofiber.Ctx) error {
		// Initialize state for this request
		stateMap := state.NewStateMap()
		for k, v := range config.DefaultState {
			r := state.NewRune(v)
			stateMap.Add(k, r)
		}

		// Store state in context
		c.Locals(config.StateKey, stateMap)

		// Generate component ID for this request
		componentID := generateComponentID()
		c.Locals(config.ComponentIDKey, componentID)

		return c.Next()
	}
}

// StateMiddleware creates middleware that injects state into responses.
func StateMiddleware(config Config) gofiber.Handler {
	return func(c *gofiber.Ctx) error {
		err := c.Next()

		// Only inject state for HTML responses
		contentType := string(c.Response().Header.ContentType())
		if !strings.Contains(contentType, "text/html") {
			return err
		}

		// Get state from context
		stateMap, ok := c.Locals(config.StateKey).(*state.StateMap)
		if !ok {
			return err
		}

		// Inject state as a script tag before </body>
		body := c.Response().Body()
		stateJSON, err := stateMap.ToJSON()
		if err != nil {
			return err
		}

		// Escape the JSON for safe injection into JavaScript to prevent XSS
		escapedStateJSON := template.JSEscapeString(stateJSON)
		stateScript := `<script>window.__GOSPA_STATE__ = ` + escapedStateJSON + `;</script>`
		if config.DevMode {
			stateScript += `<script src="` + config.RuntimeScript + `" type="module"></script>`
		}

		// Replace </body> with state script + </body>
		bodyStr := string(body)
		bodyStr = strings.Replace(bodyStr, "</body>", stateScript+"</body>", 1)
		c.Response().SetBodyString(bodyStr)

		return err
	}
}

// RuntimeMiddleware serves the client runtime script.
// Uses the embedded runtime from the embed package.
func RuntimeMiddleware(simple bool) gofiber.Handler {
	runtimeJS, err := embed.RuntimeJS(simple)
	if err != nil {
		// Return a middleware that serves an error if runtime is not available
		return func(c *gofiber.Ctx) error {
			return c.Status(gofiber.StatusInternalServerError).SendString("Runtime not available")
		}
	}
	return func(c *gofiber.Ctx) error {
		c.Set("Content-Type", "application/javascript")
		c.Set("Cache-Control", "public, max-age=31536000, immutable")
		return c.Send(runtimeJS)
	}
}

// RuntimeMiddlewareWithContent serves a custom runtime script.
// This is provided for advanced use cases where a custom runtime is needed.
func RuntimeMiddlewareWithContent(runtimeContent []byte) gofiber.Handler {
	return func(c *gofiber.Ctx) error {
		c.Set("Content-Type", "application/javascript")
		c.Set("Cache-Control", "public, max-age=31536000")
		return c.Send(runtimeContent)
	}
}

// CSRFSetTokenMiddleware issues and rotates the CSRF cookie on safe HTTP methods.
// Use this alongside CSRFTokenMiddleware: the setter runs on GETs to plant the token,
// the validator runs on mutating methods to verify it.
func CSRFSetTokenMiddleware() gofiber.Handler {
	return func(c *gofiber.Ctx) error {
		// Only issue tokens on safe methods
		if c.Method() != "GET" && c.Method() != "HEAD" {
			return c.Next()
		}

		// Issue a token if one doesn't exist yet
		if c.Cookies("csrf_token") == "" {
			token, err := generateCSRFToken()
			if err != nil {
				// Non-critical: skip token issuance, don't block the request
				return c.Next()
			}
			c.Cookie(&gofiber.Cookie{
				Name:     "csrf_token",
				Value:    token,
				HTTPOnly: false, // Must be readable by JS to set the X-CSRF-Token header
				SameSite: "Strict",
				Secure:   c.Protocol() == "https",
			})
		}

		return c.Next()
	}
}

// generateCSRFToken creates a random CSRF token.
func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CSRFTokenMiddleware validates CSRF tokens on mutating requests.
// IMPORTANT: Use CSRFSetTokenMiddleware() as well so the token is actually issued.
func CSRFTokenMiddleware() gofiber.Handler {
	return func(c *gofiber.Ctx) error {
		// Skip for GET, HEAD, OPTIONS
		if c.Method() == "GET" || c.Method() == "HEAD" || c.Method() == "OPTIONS" {
			return c.Next()
		}

		// Check for CSRF token in header
		token := c.Get("X-CSRF-Token")
		cookie := c.Cookies("csrf_token")

		if token == "" || cookie == "" || token != cookie {
			return c.Status(gofiber.StatusForbidden).JSON(gofiber.Map{
				"error": "CSRF token mismatch",
			})
		}

		return c.Next()
	}
}

// PreloadConfig configures preload headers for critical resources.
type PreloadConfig struct {
	// RuntimeScript is the path to the runtime JS
	RuntimeScript string
	// NavigationScript is the path to the navigation module
	NavigationScript string
	// WebSocketScript is the path to the WebSocket module
	WebSocketScript string
	// CoreScript is the path to the core runtime
	CoreScript string
	// MicroScript is the path to the micro runtime
	MicroScript string
	// Enabled determines if preload headers are added
	Enabled bool
}

// DefaultPreloadConfig returns the default preload configuration.
func DefaultPreloadConfig() PreloadConfig {
	return PreloadConfig{
		RuntimeScript:    "/_gospa/runtime.js",
		NavigationScript: "/_gospa/navigation.js",
		WebSocketScript:  "/_gospa/websocket.js",
		CoreScript:       "/_gospa/runtime-core.js",
		MicroScript:      "/_gospa/runtime-micro.js",
		Enabled:          true,
	}
}

// PreloadHeadersMiddleware adds HTTP Link headers for preloading critical resources.
// This allows browsers to start fetching critical scripts before parsing HTML,
// reducing time-to-interactive (TTI).
func PreloadHeadersMiddleware(config PreloadConfig) gofiber.Handler {
	return func(c *gofiber.Ctx) error {
		err := c.Next()
		if err != nil {
			return err
		}

		// Only add preload headers for HTML responses
		contentType := string(c.Response().Header.ContentType())
		if !strings.Contains(contentType, "text/html") {
			return nil
		}

		if !config.Enabled {
			return nil
		}

		// Build Link header with preload hints
		// Priority: runtime-core (smallest), then full runtime
		// Navigation and WebSocket are lazy-loaded, so we use preconnect instead
		var links []string

		// Preload the core runtime (smallest, essential for hydration)
		if config.CoreScript != "" {
			links = append(links, fmt.Sprintf("<%s>; rel=preload; as=script", config.CoreScript))
		}

		// Preload full runtime if needed (for full SPA experience)
		if config.RuntimeScript != "" {
			links = append(links, fmt.Sprintf("<%s>; rel=preload; as=script", config.RuntimeScript))
		}

		// Preconnect for lazy-loaded modules (starts DNS + TCP + TLS early)
		// These are fetched on-demand, so preconnect is more efficient than preload
		if config.NavigationScript != "" {
			// Extract origin for preconnect if it's an external URL
			links = append(links, fmt.Sprintf("<%s>; rel=modulepreload", config.NavigationScript))
		}

		if len(links) > 0 {
			c.Set("Link", strings.Join(links, ", "))
		}

		return nil
	}
}

// PreloadHeadersMiddlewareMinimal adds minimal preload headers for micro-runtime.
// Use this for pages that only need basic reactivity without full SPA features.
func PreloadHeadersMiddlewareMinimal(config PreloadConfig) gofiber.Handler {
	return func(c *gofiber.Ctx) error {
		err := c.Next()
		if err != nil {
			return err
		}

		// Only add preload headers for HTML responses
		contentType := string(c.Response().Header.ContentType())
		if !strings.Contains(contentType, "text/html") {
			return nil
		}

		if !config.Enabled {
			return nil
		}

		// Only preload the micro runtime for minimal pages
		if config.MicroScript != "" {
			c.Set("Link", fmt.Sprintf("<%s>; rel=preload; as=script", config.MicroScript))
		}

		return nil
	}
}

// SecurityHeadersMiddleware adds security headers.
func SecurityHeadersMiddleware() gofiber.Handler {
	return func(c *gofiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		return c.Next()
	}
}

// SPANavigationMiddleware detects SPA navigation requests and modifies response.
// When a request has the X-Requested-With: GoSPA-Navigate header, it strips the
// full HTML shell and returns only the main content for partial page updates.
func SPANavigationMiddleware() gofiber.Handler {
	return func(c *gofiber.Ctx) error {
		// Check if this is an SPA navigation request
		isSPANavigate := c.Get("X-Requested-With") == "GoSPA-Navigate"

		// Store in locals for handlers to check
		c.Locals("gospa.spa_navigate", isSPANavigate)

		err := c.Next()
		if err != nil {
			return err
		}

		// Only process HTML responses for SPA navigation
		if !isSPANavigate {
			return nil
		}

		contentType := string(c.Response().Header.ContentType())
		if !strings.Contains(contentType, "text/html") {
			return nil
		}

		// For SPA navigation, we need to ensure the response contains
		// the main content, title, and head elements with proper attributes
		// The client-side will parse and extract these
		body := c.Response().Body()

		// Guard: streaming responses have empty body at this point — skip
		if len(body) == 0 {
			return nil
		}
		bodyStr := string(body)

		// Set a custom header to indicate this is a partial response
		c.Set("X-GoSPA-Partial", "true")

		// The client expects full HTML and will parse out main, title, and head elements
		c.Response().SetBodyString(bodyStr)

		return nil
	}
}

// IsSPANavigation returns true if the current request is an SPA navigation.
func IsSPANavigation(c *gofiber.Ctx) bool {
	if isSPA, ok := c.Locals("gospa.spa_navigate").(bool); ok {
		return isSPA
	}
	return false
}

// CORSMiddleware handles CORS for API routes.
// SECURITY: When "*" is in allowedOrigins, Access-Control-Allow-Origin is set to "*"
// WITHOUT Allow-Credentials (the two are mutually exclusive per the CORS spec).
// Credentialed access is only enabled for explicitly named origins.
func CORSMiddleware(allowedOrigins []string) gofiber.Handler {
	return func(c *gofiber.Ctx) error {
		origin := c.Get("Origin")

		// Handle null Origin header (e.g., from file:// URLs or some mobile apps)
		// Security: null Origin is treated as a distinct origin, not a wildcard match
		if origin == "" {
			// No Origin header present (same-origin request or non-browser client)
			return c.Next()
		}

		// Determine if wildcard is configured and check for exact match
		wildcard := false
		exactMatch := false
		for _, o := range allowedOrigins {
			if o == "*" {
				wildcard = true
			} else if o == origin {
				exactMatch = true
				break
			}
		}

		// Always set Vary: Origin to prevent CDN caching issues with CORS
		// This ensures caches store separate responses per origin
		c.Set("Vary", "Origin")

		if exactMatch {
			// Explicit origin match: allow credentials
			c.Set("Access-Control-Allow-Origin", origin)
			c.Set("Access-Control-Allow-Credentials", "true")
			c.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			c.Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-CSRF-Token")
			c.Set("Access-Control-Expose-Headers", "X-GoSPA-Partial")
		} else if wildcard {
			// Wildcard: cannot combine with Allow-Credentials per CORS spec
			c.Set("Access-Control-Allow-Origin", "*")
			c.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			c.Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-CSRF-Token")
			c.Set("Access-Control-Expose-Headers", "X-GoSPA-Partial")
		}
		// If no match and no wildcard, don't set CORS headers (browser will block)

		// Handle preflight
		if c.Method() == "OPTIONS" {
			return c.SendStatus(gofiber.StatusNoContent)
		}

		return c.Next()
	}
}

// RequestLoggerMiddleware logs requests with method, path, status code, and duration.
// Output format: [METHOD] /path STATUS duration
// For structured logging, use Fiber's built-in logger middleware instead.
func RequestLoggerMiddleware() gofiber.Handler {
	return func(c *gofiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		log.Printf("[%s] %s %d %v",
			c.Method(),
			c.Path(),
			c.Response().StatusCode(),
			time.Since(start),
		)
		return err
	}
}

// RecoveryMiddleware recovers from panics.
func RecoveryMiddleware() gofiber.Handler {
	return func(c *gofiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				_ = c.Status(gofiber.StatusInternalServerError).JSON(gofiber.Map{
					"error": "Internal server error",
				})
			}
		}()
		return c.Next()
	}
}

// GetComponentID extracts the component ID from context.
func GetComponentID(c *gofiber.Ctx, config Config) string {
	if id, ok := c.Locals(config.ComponentIDKey).(string); ok {
		return id
	}
	return ""
}

// GetState extracts the state map from context.
func GetState(c *gofiber.Ctx, config Config) *state.StateMap {
	if s, ok := c.Locals(config.StateKey).(*state.StateMap); ok {
		return s
	}
	return nil
}

// RenderComponent renders a Templ component with state.
func RenderComponent(c *gofiber.Ctx, config Config, component templ.Component, componentName string) error {
	stateMap := GetState(c, config)

	// Create component with options
	opts := []gospatempl.ComponentOption{}
	if stateMap != nil {
		// Add state values to component
		stateData := make(map[string]any)
		if jsonData, err := stateMap.ToJSON(); err == nil {
			_ = json.Unmarshal([]byte(jsonData), &stateData)
		}
		opts = append(opts, gospatempl.WithProps(stateData))
	}

	wrapper := gospatempl.NewComponent(componentName, opts...)

	// Render component with wrapper
	rendered := gospatempl.RenderComponent(wrapper, component)

	c.Set("Content-Type", "text/html")
	return rendered.Render(c.Context(), c.Response().BodyWriter())
}

// generateComponentID generates a unique component ID.
func generateComponentID() string {
	return "gospa_" + randomString(8)
}

// randomString generates a cryptographically secure random string of given length.
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	randomBytes := make([]byte, length)

	// Use crypto/rand for cryptographically secure random generation
	if _, err := rand.Read(randomBytes); err != nil {
		// This should never happen with crypto/rand on modern systems
		// If it does, we panic as this is a critical security function
		panic(fmt.Sprintf("failed to generate secure random: %v", err))
	}

	for i := range b {
		b[i] = charset[int(randomBytes[i])%len(charset)]
	}
	return string(b)
}

// JSONResponse sends a JSON response.
func JSONResponse(c *gofiber.Ctx, status int, data interface{}) error {
	return c.Status(status).JSON(data)
}

// JSONError sends a JSON error response.
func JSONError(c *gofiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(gofiber.Map{
		"error": message,
	})
}

// ParseBody parses request body into a struct.
func ParseBody(c *gofiber.Ctx, v interface{}) error {
	return json.Unmarshal(c.Body(), v)
}

// GetSessionState gets or creates session state.
func GetSessionState(c *gofiber.Ctx, config Config) map[string]interface{} {
	stateMap := GetState(c, config)
	if stateMap == nil {
		return make(map[string]interface{})
	}

	result := make(map[string]interface{})
	// Extract values from state map
	jsonData, err := stateMap.ToJSON()
	if err != nil {
		return result
	}
	_ = json.Unmarshal([]byte(jsonData), &result)
	return result
}

// SetSessionState sets session state.
func SetSessionState(c *gofiber.Ctx, config Config, key string, value interface{}) {
	stateMap := GetState(c, config)
	if stateMap == nil {
		return
	}
	r := state.NewRune(value)
	stateMap.Add(key, r)
}
