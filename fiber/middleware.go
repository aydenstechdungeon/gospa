package fiber

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	stdjson "encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/aydenstechdungeon/gospa/embed"
	"github.com/aydenstechdungeon/gospa/state"
	gospatempl "github.com/aydenstechdungeon/gospa/templ"
	json "github.com/goccy/go-json"
	gofiber "github.com/gofiber/fiber/v3"
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
	// Logger is the structured logger. Defaults to slog.Default().
	Logger *slog.Logger
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		RuntimeScript:  "/_gospa/runtime.js",
		StateKey:       "gospa.state",
		ComponentIDKey: "gospa.componentID",
		DevMode:        false,
		DefaultState:   make(map[string]interface{}),
		Logger:         slog.Default(),
	}
}

// SPAMiddleware creates a Fiber middleware for SPA support.
func SPAMiddleware(config Config) gofiber.Handler {
	return func(c gofiber.Ctx) error {
		// Initialize state for this request
		stateMap := state.NewStateMap()
		if config.DefaultState != nil {
			for k, v := range config.DefaultState {
				r := state.NewRune(v)
				stateMap.Add(k, r)
			}
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
	return func(c gofiber.Ctx) error {
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

		// Use encoding/json with SetEscapeHTML for robust HTML-safe JSON encoding.
		// This escapes <, >, &, U+2028, and U+2029 which can break out of <script> contexts.
		var buf bytes.Buffer
		encoder := stdjson.NewEncoder(&buf)
		encoder.SetEscapeHTML(true)
		if err := encoder.Encode(stateMap.ToMap()); err != nil {
			return err
		}
		// Encode appends a trailing newline; trim it for inline embedding.
		escapedJSON := strings.TrimRight(buf.String(), "\n")
		stateScript := `<script>window.__GOSPA_STATE__ = ` + escapedJSON + `;</script>`

		// Always inject the runtime script if not already present in the HTML.
		// The runtime is required for island hydration — without it, client-side
		// TypeScript in <script lang="ts"> blocks never executes.
		bodyStr := string(body)
		if config.RuntimeScript != "" && !strings.Contains(bodyStr, config.RuntimeScript) {
			stateScript += `<script src="` + config.RuntimeScript + `" type="module"></script>`
		}

		// In dev mode, also inject islands.js if not already present and the file exists
		if config.DevMode && !strings.Contains(bodyStr, "/static/js/islands.js") {
			if _, err := os.Stat("static/js/islands.js"); err == nil {
				stateScript += `<script src="/static/js/islands.js"></script>`
			}
		}

		// Inject before </body>
		body = bytes.Replace(body, []byte("</body>"), append([]byte(stateScript), []byte("</body>")...), 1)
		c.Response().SetBody(body)

		return err
	}
}

// RuntimeMiddleware serves the client runtime script.
// Uses the embedded runtime from the embed package.
func RuntimeMiddleware(simple bool) gofiber.Handler {
	runtimeJS, err := embed.RuntimeJS(simple)
	if err != nil {
		// Return a middleware that serves an error if runtime is not available
		return func(c gofiber.Ctx) error {
			return c.Status(gofiber.StatusInternalServerError).SendString("Runtime not available")
		}
	}
	return func(c gofiber.Ctx) error {
		c.Set("Content-Type", "application/javascript")
		c.Set("Cache-Control", "public, max-age=31536000, immutable")
		return c.Send(runtimeJS)
	}
}

// RuntimeMiddlewareWithContent serves a custom runtime script.
func RuntimeMiddlewareWithContent(runtimeContent []byte) gofiber.Handler {
	return func(c gofiber.Ctx) error {
		c.Set("Content-Type", "application/javascript")
		c.Set("Cache-Control", "public, max-age=31536000")
		return c.Send(runtimeContent)
	}
}

// isHTTPS returns true if the request was made over HTTPS, even when behind
// a TLS-terminating reverse proxy. It checks both the direct protocol and the
// X-Forwarded-Proto header.
func isHTTPS(c gofiber.Ctx) bool {
	return c.Protocol() == "https" || c.Get("X-Forwarded-Proto") == "https"
}

// CSRFSetTokenMiddleware issues and rotates the CSRF cookie on safe HTTP methods.
func CSRFSetTokenMiddleware() gofiber.Handler {
	return func(c gofiber.Ctx) error {
		// Only issue/rotate tokens on safe methods
		if c.Method() != "GET" && c.Method() != "HEAD" {
			return c.Next()
		}

		// Keep token stable across tabs/requests unless it doesn't exist.
		if existing := c.Cookies("csrf_token"); existing != "" {
			c.Locals("gospa.csrf_token", existing)
			return c.Next()
		}

		token, err := generateCSRFToken()
		if err != nil {
			return c.Next()
		}
		c.Cookie(&gofiber.Cookie{
			Name:     "csrf_token",
			Value:    token,
			HTTPOnly: true, // SECURITY FIX: Protected from XSS extraction.
			SameSite: "Strict",
			Secure:   isHTTPS(c),
			Path:     "/", // Protect global endpoints
		})

		// Set in Locals so the renderer can inject it into the page config.
		c.Locals("gospa.csrf_token", token)

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

// SessionMiddleware ensures a session token exists in an HttpOnly cookie.
// This mitigates XSS risks compared to storing tokens in sessionStorage.
func SessionMiddleware() gofiber.Handler {
	return func(c gofiber.Ctx) error {
		cookie := c.Cookies("gospa_session")
		if cookie != "" {
			// Validate existing session
			if _, ok := globalSessionStore.ValidateSession(cookie); ok {
				c.Locals("gospa.session", cookie)
				return c.Next()
			}
		}

		// Create new session
		clientID := generateComponentID()
		token, err := globalSessionStore.CreateSession(clientID)
		if err != nil {
			return c.Next()
		}

		c.Cookie(&gofiber.Cookie{
			Name:     "gospa_session",
			Value:    token,
			HTTPOnly: true,
			SameSite: "Lax",
			Secure:   isHTTPS(c),
			Path:     "/",
			Expires:  time.Now().Add(SessionTTL),
		})

		c.Locals("gospa.session", token)
		return c.Next()
	}
}

// CSRFTokenMiddleware validates CSRF tokens on mutating requests.
func CSRFTokenMiddleware() gofiber.Handler {
	return func(c gofiber.Ctx) error {
		// Skip for GET, HEAD, OPTIONS
		if c.Method() == "GET" || c.Method() == "HEAD" || c.Method() == "OPTIONS" {
			return c.Next()
		}

		token := c.Get("X-CSRF-Token")
		cookie := c.Cookies("csrf_token")

		if token == "" || cookie == "" || subtle.ConstantTimeCompare([]byte(token), []byte(cookie)) != 1 {
			return c.Status(gofiber.StatusForbidden).JSON(gofiber.Map{
				"error": "CSRF token mismatch",
			})
		}

		return c.Next()
	}
}

// PreloadConfig configures preload headers for critical resources.
type PreloadConfig struct {
	RuntimeScript    string
	NavigationScript string
	WebSocketScript  string
	CoreScript       string
	MicroScript      string
	Enabled          bool
}

// DefaultPreloadConfig returns the default preload configuration.
func DefaultPreloadConfig() PreloadConfig {
	return PreloadConfig{
		RuntimeScript: "/_gospa/runtime.js",
		CoreScript:    "/_gospa/runtime-core.js",
		Enabled:       true,
	}
}

// PreloadHeadersMiddleware adds HTTP Link headers for preloading critical resources.
func PreloadHeadersMiddleware(config PreloadConfig) gofiber.Handler {
	return func(c gofiber.Ctx) error {
		err := c.Next()
		if err != nil {
			return err
		}

		contentType := string(c.Response().Header.ContentType())
		if !strings.Contains(contentType, "text/html") {
			return nil
		}

		if !config.Enabled {
			return nil
		}

		var links []string
		// Preload explicit core files only if they are set and not empty
		if config.CoreScript != "" {
			links = append(links, fmt.Sprintf("<%s>; rel=modulepreload", config.CoreScript))
		}
		if config.RuntimeScript != "" {
			links = append(links, fmt.Sprintf("<%s>; rel=modulepreload", config.RuntimeScript))
		}
		if config.NavigationScript != "" {
			links = append(links, fmt.Sprintf("<%s>; rel=modulepreload", config.NavigationScript))
		}
		if config.WebSocketScript != "" {
			links = append(links, fmt.Sprintf("<%s>; rel=modulepreload", config.WebSocketScript))
		}

		// Automatically discover and preload GoSPA internal runtime chunks if they aren't already included.
		// We filter these to avoid preloading large optional assets like DOMPurify or unused
		// runtime variants (e.g. not preloading runtime-simple if using the full runtime).
		for _, chunk := range embed.RuntimeChunks() {
			chunkPath := fmt.Sprintf("/_gospa/%s", chunk)

			// Skip purification chunks by default - they are large and usually lazy-loaded
			// by runtime-secure only when actually needed (often during idle time).
			if strings.HasPrefix(chunk, "purify") {
				continue
			}

			// Skip other runtime entry points that aren't the one currently configured.
			// This prevents preloading runtime-simple when using the full runtime, etc.
			// We keep runtime-core and shared helper chunks (sm, qx).
			if (strings.HasPrefix(chunk, "runtime-") || chunk == "runtime.js") &&
				chunk != "runtime-core.js" &&
				!strings.HasPrefix(chunk, "runtime-sm") &&
				!strings.HasPrefix(chunk, "runtime-qx") {

				base := strings.TrimSuffix(chunk, ".js")
				if !strings.Contains(config.RuntimeScript, "/"+base+".") &&
					!strings.HasSuffix(config.RuntimeScript, "/"+chunk) {
					continue
				}
			}

			// Skip if we already added it explicitly (prevents duplicates)
			alreadyAdded := false
			for _, link := range links {
				if strings.Contains(link, chunkPath) {
					alreadyAdded = true
					break
				}
			}
			if !alreadyAdded {
				links = append(links, fmt.Sprintf("<%s>; rel=modulepreload", chunkPath))
			}
		}

		if len(links) > 0 {
			// Limit the number of links to prevent oversized headers (capped at 12 for safety)
			if len(links) > 12 {
				links = links[:12]
			}
			c.Set("Link", strings.Join(links, ", "))
		}

		return nil
	}
}

// PreloadHeadersMiddlewareMinimal adds minimal preload headers for micro-runtime.
func PreloadHeadersMiddlewareMinimal(config PreloadConfig) gofiber.Handler {
	return func(c gofiber.Ctx) error {
		err := c.Next()
		if err != nil {
			return err
		}

		contentType := string(c.Response().Header.ContentType())
		if !strings.Contains(contentType, "text/html") {
			return nil
		}

		if !config.Enabled {
			return nil
		}

		if config.MicroScript != "" {
			c.Set("Link", fmt.Sprintf("<%s>; rel=preload; as=script", config.MicroScript))
		}

		return nil
	}
}

// DefaultContentSecurityPolicy is the CSP used when gospa.Config.ContentSecurityPolicy is empty.
// Uses a nonce-compatible strict policy. Prefer StrictContentSecurityPolicy for apps
// that don't require any inline scripts. For full compatibility with inline scripts,
// use LegacyContentSecurityPolicy.
const DefaultContentSecurityPolicy = "default-src 'self'; base-uri 'self'; frame-ancestors 'none'; object-src 'none'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob: https:; font-src 'self' data:; connect-src 'self' ws: wss:; form-action 'self'"

// StrictContentSecurityPolicy is a hardened CSP preset for applications that do
// not rely on inline scripts or inline styles.
const StrictContentSecurityPolicy = "default-src 'self'; base-uri 'self'; frame-ancestors 'none'; object-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data: blob: https:; font-src 'self' data:; connect-src 'self' ws: wss:; form-action 'self'"

// LegacyContentSecurityPolicy allows unsafe-inline for script-src. Use only when
// the application requires inline event handlers or eval-based script execution.
const LegacyContentSecurityPolicy = "default-src 'self'; base-uri 'self'; frame-ancestors 'none'; object-src 'none'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob: https:; font-src 'self' data:; connect-src 'self' ws: wss:; form-action 'self'"

// SecurityHeadersMiddleware adds security headers.
func SecurityHeadersMiddleware(policy string) gofiber.Handler {
	if strings.TrimSpace(policy) == "" {
		policy = DefaultContentSecurityPolicy
	}
	return func(c gofiber.Ctx) error {
		if isHTTPS(c) {
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}
		c.Set("Content-Security-Policy", policy)
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "0")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		return c.Next()
	}
}

// SPANavigationMiddleware detects SPA navigation requests and modifies response.
func SPANavigationMiddleware() gofiber.Handler {
	return func(c gofiber.Ctx) error {
		isSPANavigate := c.Get("X-Requested-With") == "GoSPA-Navigate"
		c.Locals("gospa.spa_navigate", isSPANavigate)

		err := c.Next()
		if err != nil {
			return err
		}

		if !isSPANavigate {
			return nil
		}

		contentType := string(c.Response().Header.ContentType())
		if !strings.Contains(contentType, "text/html") {
			return nil
		}

		body := c.Response().Body()
		if len(body) == 0 {
			return nil
		}
		c.Set("X-GoSPA-Partial", "true")

		return nil
	}
}

// IsSPANavigation returns true if the current request is an SPA navigation.
func IsSPANavigation(c gofiber.Ctx) bool {
	if isSPA, ok := c.Locals("gospa.spa_navigate").(bool); ok {
		return isSPA
	}
	return false
}

// CORSMiddleware handles CORS for API routes.
func CORSMiddleware(allowedOrigins []string) gofiber.Handler {
	return func(c gofiber.Ctx) error {
		origin := c.Get("Origin")

		if origin == "" {
			return c.Next()
		}

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

		c.Set("Vary", "Origin")

		if exactMatch {
			c.Set("Access-Control-Allow-Origin", origin)
			c.Set("Access-Control-Allow-Credentials", "true")
			c.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			c.Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-CSRF-Token")
			c.Set("Access-Control-Expose-Headers", "X-GoSPA-Partial")
		} else if wildcard {
			// SECURITY: Do NOT allow wildcard origin if Credentials (Auth header or Session cookie) are present.
			// This prevents credential leakage when allowedOrigins contains "*".
			if c.Get("Authorization") != "" || c.Cookies("gospa_session") != "" || c.Get("X-CSRF-Token") != "" {
				return c.Next()
			}
			c.Set("Access-Control-Allow-Origin", "*")
			c.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			c.Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-CSRF-Token")
			c.Set("Access-Control-Expose-Headers", "X-GoSPA-Partial")
		}

		if c.Method() == "OPTIONS" {
			return c.SendStatus(gofiber.StatusNoContent)
		}

		return c.Next()
	}
}

// RequestLoggerMiddleware logs requests with method, path, status code, and duration.
func RequestLoggerMiddleware() gofiber.Handler {
	logger := slog.Default()
	return func(c gofiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		logger.Info("request",
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"duration", time.Since(start),
		)
		return err
	}
}

// RecoveryMiddleware recovers from panics.
func RecoveryMiddleware() gofiber.Handler {
	return func(c gofiber.Ctx) error {
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
func GetComponentID(c gofiber.Ctx, config Config) string {
	if id, ok := c.Locals(config.ComponentIDKey).(string); ok {
		return id
	}
	return ""
}

// GetState extracts the state map from context.
func GetState(c gofiber.Ctx, config Config) *state.StateMap {
	if s, ok := c.Locals(config.StateKey).(*state.StateMap); ok {
		return s
	}
	return nil
}

// RenderComponent renders a Templ component with state.
func RenderComponent(c gofiber.Ctx, config Config, component templ.Component, componentName string) error {
	stateMap := GetState(c, config)

	opts := []gospatempl.ComponentOption{}
	if stateMap != nil {
		stateData := make(map[string]any)
		if jsonData, err := stateMap.ToJSON(); err == nil {
			_ = json.Unmarshal([]byte(jsonData), &stateData)
		}
		opts = append(opts, gospatempl.WithProps(stateData))
	}

	wrapper := gospatempl.NewComponent(componentName, opts...)
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

	if _, err := rand.Read(randomBytes); err != nil {
		panic(fmt.Sprintf("failed to generate secure random: %v", err))
	}

	for i := 0; i < length; {
		idx := int(randomBytes[i])
		if idx < 248 {
			b[i] = charset[idx%len(charset)]
			i++
		} else {
			if _, err := rand.Read(randomBytes[i : i+1]); err != nil {
				panic(fmt.Sprintf("failed to generate secure random: %v", err))
			}
		}
	}
	return string(b)
}

// JSONResponse sends a JSON response.
func JSONResponse(c gofiber.Ctx, status int, data interface{}) error {
	return c.Status(status).JSON(data)
}

// JSONError sends a JSON error response.
func JSONError(c gofiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(gofiber.Map{
		"error": message,
	})
}

// ParseBody parses request body into a struct.
func ParseBody(c gofiber.Ctx, v interface{}) error {
	return json.Unmarshal(c.Body(), v)
}

// GetSessionState gets or creates session state.
func GetSessionState(c gofiber.Ctx, config Config) map[string]interface{} {
	stateMap := GetState(c, config)
	if stateMap == nil {
		return make(map[string]interface{})
	}

	result := make(map[string]interface{})
	jsonData, err := stateMap.ToJSON()
	if err != nil {
		return result
	}
	_ = json.Unmarshal([]byte(jsonData), &result)
	return result
}

// SetSessionState sets session state.
func SetSessionState(c gofiber.Ctx, config Config, key string, value interface{}) {
	stateMap := GetState(c, config)
	if stateMap == nil {
		return
	}
	r := state.NewRune(value)
	stateMap.Add(key, r)
}
