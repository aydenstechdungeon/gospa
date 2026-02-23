package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	_ "website/routes" // Import routes to trigger init()

	"github.com/aydenstechdungeon/gospa"
	"github.com/gofiber/fiber/v2"
)

func main() {
	// Production config with performance optimizations
	devMode := getEnvBool("GOSPA_DEV", false)

	app := gospa.New(gospa.Config{
		RoutesDir:        "./routes",
		DevMode:          devMode,
		AppName:          "GoSPA Documentation",
		CacheTemplates:   !devMode, // Enable template caching in production
		CompressState:    true,     // Compress WebSocket messages
		StateDiffing:     true,     // Only send state diffs
		EnableWebSocket:  true,
		WSHeartbeat:      30 * time.Second,
		WSReconnectDelay: 1 * time.Second,
		WSMaxReconnect:   5,
		HydrationMode:    "immediate",
	})

	// Add cache headers middleware for static assets
	if !devMode {
		app.Use(staticCacheMiddleware)
	}

	port := getEnvString("PORT", ":3000")
	if err := app.Run(port); err != nil {
		log.Fatal(err)
	}
}

// staticCacheMiddleware adds Cache-Control headers for static assets
func staticCacheMiddleware(c *fiber.Ctx) error {
	path := c.Path()

	// Apply cache headers for static assets
	if strings.HasPrefix(path, "/static/") {
		// Static assets with content hash: 1 year cache
		if hasContentHash(path) {
			c.Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			// Static assets without hash: 1 day cache, revalidate
			c.Set("Cache-Control", "public, max-age=86400, stale-while-revalidate=604800")
		}
	}

	return c.Next()
}

// hasContentHash checks if filename contains a content hash pattern
func hasContentHash(path string) bool {
	// Check for common hash patterns: -abc123, .abc123, _abc123
	// Matches patterns like: gospa1-64.webp, app.a1b2c3.js, etc.
	for _, pattern := range []string{"-", ".", "_"} {
		parts := strings.Split(path, pattern)
		if len(parts) >= 2 {
			// Check if last part looks like a hash (alphanumeric, 4+ chars before extension)
			last := parts[len(parts)-1]
			if idx := strings.Index(last, "."); idx > 3 {
				hash := last[:idx]
				if isAlphanumeric(hash) && len(hash) >= 4 {
					return true
				}
			}
		}
	}
	return false
}

func isAlphanumeric(s string) bool {
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

func getEnvString(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		b, err := strconv.ParseBool(val)
		if err != nil {
			return defaultVal
		}
		return b
	}
	return defaultVal
}
