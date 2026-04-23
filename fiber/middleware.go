package fiber

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	stdjson "encoding/json"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/aydenstechdungeon/gospa/compiler"
	"github.com/aydenstechdungeon/gospa/embed"
	"github.com/aydenstechdungeon/gospa/routing"
	"github.com/aydenstechdungeon/gospa/state"
	gospatempl "github.com/aydenstechdungeon/gospa/templ"
	json "github.com/goccy/go-json"
	gofiber "github.com/gofiber/fiber/v3"
)

var csrfTokenPattern = regexp.MustCompile(`^[A-Fa-f0-9]{64}$`)

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
	// BuildManifest is the loaded manifest.json (optional)
	BuildManifest map[string]string
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

		// Recover state from context
		stateMap, ok := c.Locals(config.StateKey).(*state.StateMap)
		if !ok {
			return err
		}

		// Inject state as a script tag before </body>
		body := c.Response().Body()

		// Retrieve CSRF token for possible forms and AJAX setup.
		csrfToken, _ := c.Locals("gospa.csrf_token").(string)

		// Retrieve CSP Nonce from locals
		nonce, _ := c.Locals("gospa.csp_nonce").(string)
		nonceAttr := ""
		if nonce != "" {
			nonceAttr = ` nonce="` + nonce + `"`
		}

		// Use encoding/json with SetEscapeHTML for robust HTML-safe JSON encoding.
		var buf bytes.Buffer
		encoder := stdjson.NewEncoder(&buf)
		encoder.SetEscapeHTML(true)
		if err := encoder.Encode(stateMap.ToMap()); err != nil {
			return err
		}
		// Encode appends a trailing newline; trim it for inline embedding.
		escapedJSON := strings.TrimRight(buf.String(), "\n")
		stateScript := `<script` + nonceAttr + `>window.__GOSPA_STATE__ = ` + escapedJSON + `;`
		if isValidCSRFToken(csrfToken) {
			csrfJSON, err := stdjson.Marshal(csrfToken)
			if err != nil {
				return err
			}
			stateScript += `window.__GOSPA_CSRF_TOKEN__ = ` + string(csrfJSON) + `;`
		}
		stateScript += `</script>`

		// Always inject the runtime script if not already present in the HTML.
		if config.RuntimeScript != "" && !bytes.Contains(body, []byte(config.RuntimeScript)) {
			runtimePath := config.RuntimeScript
			if strings.HasPrefix(runtimePath, "/_gospa/runtime.js") {
				opts := routing.GetRouteOptions(c.Path())
				if opts.RuntimeTier != "" && opts.RuntimeTier != "full" {
					runtimePath = "/_gospa/runtime-" + opts.RuntimeTier + ".js"
				}
			}
			stateScript += `<script src="` + runtimePath + `" type="module"` + nonceAttr + `></script>`
		}

		// In dev mode, also inject islands.js if not already present and the file exists
		if config.DevMode && !bytes.Contains(body, []byte("/static/js/islands.js")) {
			if _, err := os.Stat("static/js/islands.js"); err == nil {
				stateScript += `<script src="/static/js/islands.js" type="module"` + nonceAttr + `></script>`
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
func RuntimeMiddleware(tier compiler.RuntimeTier) gofiber.Handler {
	runtimeJS, err := embed.RuntimeJS(tier)
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

// isHTTPS returns true if the request was made over HTTPS.
// It relies on Fiber's Ctx.Protocol() which respects TrustedProxies and ProxyHeader
// configuration on the Fiber App.
func isHTTPS(c gofiber.Ctx) bool {
	return c.Protocol() == "https"
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
			if isValidCSRFToken(existing) {
				c.Locals("gospa.csrf_token", existing)
				return c.Next()
			}
			c.Cookie(&gofiber.Cookie{
				Name:     "csrf_token",
				Value:    "",
				HTTPOnly: true,
				SameSite: "Strict",
				Secure:   isHTTPS(c),
				Path:     "/",
				Expires:  time.Unix(0, 0),
				MaxAge:   -1,
			})
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

func isValidCSRFToken(token string) bool {
	return csrfTokenPattern.MatchString(token)
}

// generateCSRFToken creates a random CSRF token.
func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// generateCSPNonce creates a base64 nonce for Content Security Policy.
// CSP nonces are expected to use a base64-compatible value.
func generateCSPNonce() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
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
		if token == "" {
			// Fallback to form field for standard HTML submissions.
			// This enables progressive enhancement without mandatory JS.
			token = c.FormValue("_csrf")
		}
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
	// CSSLinks contains stylesheets to preload with high priority
	CSSLinks []string
	Enabled  bool
	// BuildManifest is the loaded manifest.json (optional)
	BuildManifest map[string]string
}

// DefaultPreloadConfig returns the default preload configuration.
func DefaultPreloadConfig() PreloadConfig {
	return PreloadConfig{
		RuntimeScript: "/_gospa/runtime.js",
		Enabled:       true,
	}
}

// PreloadHeadersMiddleware adds HTTP Link headers for preloading critical resources.
// Link headers are set before downstream handlers run so they arrive in the response
// headers rather than after the body已经开始解析.
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
		// 1. Prioritize CSS preloads with high fetchpriority
		for _, css := range config.CSSLinks {
			links = append(links, fmt.Sprintf("<%s>; rel=preload; as=style", css))
		}

		// 2. Preload explicit core files
		if config.CoreScript != "" {
			links = append(links, fmt.Sprintf("<%s>; rel=modulepreload", config.CoreScript))
		}
		if config.RuntimeScript != "" {
			runtimePath := config.RuntimeScript
			if strings.HasPrefix(runtimePath, "/_gospa/runtime.js") {
				opts := routing.GetRouteOptions(c.Path())
				if opts.RuntimeTier != "" && opts.RuntimeTier != "full" {
					runtimePath = "/_gospa/runtime-" + opts.RuntimeTier + ".js"
				}
			}
			links = append(links, fmt.Sprintf("<%s>; rel=modulepreload", runtimePath))
		}

		// 3. Automatically discover and preload GoSPA internal runtime chunks or manifest entries
		// We limit this based on the protocol to avoid saturating connections.
		// HTTP/1.1 usually has a 6-connection limit per host, while H2/H3 handle many more.
		limit := 6
		if isHTTPS(c) {
			limit = 12 // Safe increase for H2/H3
		}

		alreadyAdded := func(link string) bool {
			for _, l := range links {
				if strings.Contains(l, link) {
					return true
				}
			}
			return false
		}

		// Discovery from manifest (prioritize hashed assets)
		count := 0
		if config.BuildManifest != nil {
			for relPath := range config.BuildManifest {
				if len(links) >= limit {
					break
				}
				// Preload JS/CSS from manifest that looks like core runtime or islands
				if (strings.HasPrefix(relPath, "static/js/runtime-") || strings.HasPrefix(relPath, "static/js/islands-")) && strings.HasSuffix(relPath, ".js") {
					linkPath := "/" + relPath
					if !alreadyAdded(linkPath) {
						links = append(links, fmt.Sprintf("<%s>; rel=modulepreload", linkPath))
					}
				}
			}
		}

		// Fallback to embedded runtime chunks if manifest discovery didn't fill the limit
		for _, chunk := range embed.RuntimeChunks() {
			if len(links) >= limit || count >= 4 {
				break
			}
			chunkPath := fmt.Sprintf("/_gospa/%s", chunk)

			// Skip heavy/optional chunks
			if strings.HasPrefix(chunk, "purify") || strings.HasPrefix(chunk, "runtime-micro") {
				continue
			}

			// Preload core-related chunks only
			if !strings.HasPrefix(chunk, "runtime-") || strings.HasPrefix(chunk, "runtime-secure") {
				continue
			}

			if !alreadyAdded(chunkPath) {
				links = append(links, fmt.Sprintf("<%s>; rel=modulepreload", chunkPath))
				count++
			}
		}

		if len(links) > 0 {
			if len(links) > limit {
				links = links[:limit]
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

// DefaultContentSecurityPolicy is the default CSP policy for GoSPA.
// Uses a {nonce} placeholder that SecurityHeadersMiddleware replaces per request.
const DefaultContentSecurityPolicy = "default-src 'self'; base-uri 'none'; frame-ancestors 'none'; object-src 'none'; script-src 'nonce-{nonce}' 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob: https:; font-src 'self' data:; connect-src 'self' ws: wss:; form-action 'self'"

// StrictContentSecurityPolicy is a strict CSP policy that disallows unsafe-inline.
const StrictContentSecurityPolicy = "default-src 'self'; base-uri 'none'; frame-ancestors 'none'; object-src 'none'; script-src 'nonce-{nonce}' 'self'; style-src 'self'; img-src 'self' data: blob: https:; font-src 'self' data:; connect-src 'self' ws: wss:; form-action 'self'"

// LegacyContentSecurityPolicy allows unsafe-inline for script-src. Use only when
// the application requires inline event handlers or eval-based script execution.
const LegacyContentSecurityPolicy = "default-src 'self'; base-uri 'self'; frame-ancestors 'none'; object-src 'none'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob: https:; font-src 'self' data:; connect-src 'self' ws: wss:; form-action 'self'"

// SecurityHeadersMiddleware adds security headers and handles the CSP nonce.
func SecurityHeadersMiddleware(policy string) gofiber.Handler {
	if strings.TrimSpace(policy) == "" {
		policy = DefaultContentSecurityPolicy
	}
	basePolicy := policy
	return func(c gofiber.Ctx) error {
		currentPolicy := basePolicy
		// Generate an unpredictable nonce for every request to harden CSP
		nonce, err := generateCSPNonce()
		if err == nil {
			c.Locals("gospa.csp_nonce", nonce)
			// Inject nonce into the CSP policy only when the policy explicitly
			// opts in via the {nonce} placeholder.
			// When a nonce is present in script-src, browsers ignore 'unsafe-inline'
			// for scripts per spec, so strip it in that explicit nonce mode.
			if strings.Contains(currentPolicy, "{nonce}") {
				currentPolicy = strings.ReplaceAll(currentPolicy, "{nonce}", nonce)
				currentPolicy = removeUnsafeInlineFromScriptSrc(currentPolicy)
			}
		}

		if isHTTPS(c) {
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}
		c.Set("Content-Security-Policy", currentPolicy)
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

	ctx := c.Context()
	if nonce, ok := c.Locals("gospa.csp_nonce").(string); ok && nonce != "" {
		ctx = gospatempl.WithNonce(ctx, nonce)
	}

	c.Set("Content-Type", "text/html")
	return rendered.Render(ctx, c.Response().BodyWriter())
}

// removeUnsafeInlineFromScriptSrc strips 'unsafe-inline' from the script-src
// directive of a CSP string.  When a nonce is present in script-src, browsers
// already ignore 'unsafe-inline' per spec — removing it explicitly keeps the
// policy unambiguous and avoids browser console warnings.
// Other directives (e.g. style-src) are left untouched.
func removeUnsafeInlineFromScriptSrc(policy string) string {
	directives := strings.Split(policy, ";")
	for i, d := range directives {
		trimmed := strings.TrimSpace(d)
		if strings.HasPrefix(trimmed, "script-src") {
			directives[i] = " " + strings.TrimSpace(strings.ReplaceAll(trimmed, "'unsafe-inline'", ""))
		}
	}
	return strings.Join(directives, ";")
}

// generateComponentID generates a unique component ID using timestamp + random.
// Timestamp ensures ordering/uniqueness, random provides entropy.
func generateComponentID() string {
	return fmt.Sprintf("gospa_%d_%s", time.Now().UnixNano(), randomString(8))
}

// randomString generates a random string of given length.
// It uses crypto/rand when available and falls back to a time-based value if entropy fails.
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	randomBytes := make([]byte, length)

	if _, err := rand.Read(randomBytes); err != nil {
		seed := time.Now().UnixNano()
		for i := 0; i < length; i++ {
			b[i] = charset[int((seed+int64(i*17))%int64(len(charset)))]
		}
		return string(b)
	}

	for i := 0; i < length; {
		idx := int(randomBytes[i])
		if idx < 248 {
			b[i] = charset[idx%len(charset)]
			i++
		} else {
			if _, err := rand.Read(randomBytes[i : i+1]); err != nil {
				seed := time.Now().UnixNano()
				b[i] = charset[int((seed+int64(i*31))%int64(len(charset)))]
				i++
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

// SetFlash sets a flash message in the current session.
func SetFlash(c gofiber.Ctx, key string, value interface{}) {
	token, ok := c.Locals("gospa.session").(string)
	if !ok || token == "" {
		return
	}
	_ = globalSessionStore.SetFlash(token, key, value)
}

// GetFlashes retrieves and clears all flash messages from the current session.
func GetFlashes(c gofiber.Ctx) map[string]interface{} {
	token, ok := c.Locals("gospa.session").(string)
	if !ok || token == "" {
		return nil
	}
	return globalSessionStore.GetFlashes(token)
}
