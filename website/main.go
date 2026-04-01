// Package main provides the entry point for the GoSPA documentation website.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/aydenstechdungeon/gospa/website/routes" // Import routes to trigger init()

	"github.com/aydenstechdungeon/gospa"
	"github.com/aydenstechdungeon/gospa/routing"
	"github.com/gofiber/fiber/v3"

	// Register built-in plugins for runtime features (if any)
	_ "github.com/aydenstechdungeon/gospa/plugin/image"
	_ "github.com/aydenstechdungeon/gospa/plugin/postcss"
	_ "github.com/aydenstechdungeon/gospa/plugin/qrcode"
	_ "github.com/aydenstechdungeon/gospa/plugin/tailwind"
)

// Cached file hashes for ETags - computed once at startup for static files
var (
	fileHashCache = make(map[string]string)
	hashCacheMu   sync.RWMutex
)

func main() {
	// Production config with performance optimizations
	devMode := getEnvBool("GOSPA_DEV", false)

	app := gospa.New(gospa.Config{
		RoutesDir:             "./routes",
		DevMode:               devMode,
		AppName:               "GoSPA Documentation",
		CacheTemplates:        !devMode,            // Enable template caching in production
		DefaultRenderStrategy: routing.StrategySSG, // Make the entire docs site static by default
		SSGCacheMaxEntries:    -1,                  // Cache all pages without eviction
		CompressState:         true,                // Compress WebSocket messages
		StateDiffing:          true,                // Only send state diffs
		EnableWebSocket:       true,
		SerializationFormat:   gospa.SerializationMsgPack,
		WSHeartbeat:           30 * time.Second,
		WSReconnectDelay:      1 * time.Second,
		WSMaxReconnect:        5,
		AllowInsecureWS:       devMode,
		PublicOrigin:          "https://gospa.onrender.com",
		HydrationMode:         "idle",
		NavigationOptions: gospa.NavigationOptions{
			IdleCallbackBatchUpdates: &gospa.NavigationIdleCallbackBatchUpdatesConfig{
				Enabled: boolPtr(true),
			},
			URLParsingCache: &gospa.NavigationURLParsingCacheConfig{
				Enabled: boolPtr(true),
			},
			SpeculativePrefetching: &gospa.NavigationSpeculativePrefetchingConfig{
				Enabled:        boolPtr(true),
				TTL:            intPtr(45000),
				HoverDelay:     intPtr(100),
				ViewportMargin: intPtr(150),
			},
			ServiceWorkerNavigationCaching: &gospa.NavigationServiceWorkerCachingConfig{
				Enabled:   boolPtr(true),
				CacheName: "gospa-docs-navigation-cache",
				Path:      "/gospa-navigation-sw.js",
			},
		},
	})

	// Add middleware for performance (Link headers, caching)
	// IMPORTANT: This must come BEFORE routes to catch static assets
	// We've updated the framework to register static routes later so this works as expected.
	app.Fiber.Use(cacheMiddleware)

	// Legacy redirects after documentation restructuring
	app.Fiber.Get("/docs/getstarted", func(c fiber.Ctx) error {
		return c.Redirect().Status(301).To("/docs/getstarted/installation")
	})
	app.Fiber.Get("/docs/client-runtime", func(c fiber.Ctx) error {
		return c.Redirect().Status(301).To("/docs/client-runtime/overview")
	})

	// LLM support routes
	app.Fiber.Get("/llms.txt", func(c fiber.Ctx) error {
		return c.SendFile("./static/llms.txt")
	})
	app.Fiber.Get("/llms-full.md", func(c fiber.Ctx) error {
		return c.SendFile("./static/llms-full.md")
	})

	// Service Worker route
	app.Fiber.Get("/gospa-navigation-sw.js", func(c fiber.Ctx) error {
		c.Set("Content-Type", "application/javascript")
		return c.SendFile("./static/gospa-navigation-sw.js")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = ":3000"
	} else if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	if err := app.Run(port); err != nil {
		log.Fatal(err)
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func intPtr(v int) *int {
	return &v
}

// cacheMiddleware adds Link headers for preloading and Cache-Control headers in production
func cacheMiddleware(c fiber.Ctx) error {
	path := c.Path()
	devMode := getEnvBool("GOSPA_DEV", false)

	// Send Link headers for critical assets on every HTML page request (Early Discovery)
	if isHTMLPage(path) {
		// Preload fonts and images
		links := []string{
			"</static/fonts/InterVariable.woff2>; rel=preload; as=font; type=font/woff2; crossorigin=anonymous; fetchpriority=high",
			"</static/gospa1-64.webp>; rel=preload; as=image; fetchpriority=high",
		}

		c.Set("Link", strings.Join(links, ", "))
	}

	// Skip caching logic in development mode
	if devMode && !strings.Contains(path, "/static/") {
		return c.Next()
	}

	// Apply cache headers for static assets
	isStatic := strings.HasPrefix(path, "/static/")
	isImage := isImageFile(path)
	isFont := isFontFile(path)

	if isStatic || isImage || isFont {
		// Special handling for docs search index - aggressive caching with immutable
		if strings.HasSuffix(path, "/docs_search_index.json") {
			c.Set("Cache-Control", "public, max-age=31536000, immutable")
			c.Set("Vary", "Accept-Encoding")

			// Generate content-based ETag
			etag := generateFileETag(path)
			if etag != "" {
				c.Set("ETag", etag)
				if match := c.Get("If-None-Match"); match != "" {
					if strings.Contains(match, etag) {
						return c.SendStatus(304)
					}
				}
			}
			return c.Next()
		}

		// Fonts and static assets with content hash: 1 year cache, immutable
		// Set Cache-Control based on file type
		switch {
		case isFont || hasContentHash(path):
			// Critical assets and hashed assets: 1 year cache, immutable
			c.Set("Cache-Control", "public, max-age=31536000, immutable")
		case isImage:
			// Image files without hash: 30 days cache with revalidation
			c.Set("Cache-Control", "public, max-age=2592000, stale-while-revalidate=31536000")
		default:
			// Other static assets (JS/CSS) without hash: 7 days cache, revalidate for 1 year
			c.Set("Cache-Control", "public, max-age=604800, stale-while-revalidate=31536000")
		}

		// Generate ETag for conditional requests
		etag := generateETag(path)
		if etag != "" {
			c.Set("ETag", etag)

			// Check If-None-Match for 304 response
			if match := c.Get("If-None-Match"); match != "" {
				if strings.Contains(match, etag) {
					return c.SendStatus(304)
				}
			}
		}
	} else if isHTMLPage(path) {
		// HTML pages: short cache with revalidation (60 seconds)
		c.Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
	}

	return c.Next()
}

// isHTMLPage checks if the path is an HTML page request
func isHTMLPage(path string) bool {
	// Skip API routes and static files
	if strings.HasPrefix(path, "/api/") ||
		strings.HasPrefix(path, "/_") ||
		strings.Contains(path, ".") {
		return false
	}
	return true
}

// isImageFile checks if the path is an image file
func isImageFile(path string) bool {
	lower := strings.ToLower(path)
	for _, ext := range []string{".webp", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// isFontFile checks if the path is a font file
func isFontFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".woff2")
}

// hasContentHash checks if filename contains a content hash pattern
// Supports patterns like: name-64.ext, name-a1b2c3.ext, name.abc123.ext
func hasContentHash(path string) bool {
	// Extract filename from path
	idx := strings.LastIndex(path, "/")
	if idx >= 0 {
		path = path[idx+1:]
	}

	// Check for common hash patterns: -abc123, .abc123, _abc123
	for _, pattern := range []string{"-", ".", "_"} {
		parts := strings.Split(path, pattern)
		if len(parts) >= 2 {
			// Check if last part looks like a hash (alphanumeric before extension)
			last := parts[len(parts)-1]
			if dotIdx := strings.Index(last, "."); dotIdx > 0 {
				hash := last[:dotIdx]
				// Accept 2+ character hashes for logo files like gospa1-64.webp
				if isAlphanumeric(hash) && len(hash) >= 2 {
					return true
				}
			}
		}
	}
	return false
}

// generateETag creates a weak ETag based on file info (modtime and size)
func generateETag(path string) string {
	filePath := "." + path
	if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
		// Stronger weak ETag using size and modtime
		h := sha256.New()
		_, _ = fmt.Fprintf(h, "%d-%d-%s", info.Size(), info.ModTime().Unix(), path)
		return `W/"` + hex.EncodeToString(h.Sum(nil)[:8]) + `"`
	}
	// Fallback to path-based ETag if file not found (unlikely for static middleware)
	h := sha256.Sum256([]byte(path))
	return `W/"` + hex.EncodeToString(h[:8]) + `"`
}

// generateFileETag creates a strong ETag based on file content hash (cached)
func generateFileETag(path string) string {
	// Check cache first
	hashCacheMu.RLock()
	if hash, ok := fileHashCache[path]; ok {
		hashCacheMu.RUnlock()
		return `"` + hash + `"`
	}
	hashCacheMu.RUnlock()

	// Try to read file and compute hash
	filePath := "." + path // Convert URL path to file path
	if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
		filePath = filepath.Clean(filePath)
		if data, err := os.ReadFile(filePath); err == nil {
			h := sha256.Sum256(data)
			hash := hex.EncodeToString(h[:8])

			// Cache the hash
			hashCacheMu.Lock()
			fileHashCache[path] = hash
			hashCacheMu.Unlock()

			return `"` + hash + `"`
		}
	}

	// Fallback to path-based ETag
	return generateETag(path)
}

func isAlphanumeric(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
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
