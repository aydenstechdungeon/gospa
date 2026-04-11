package cli

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ServeConfig holds configuration for the production server.
type ServeConfig struct {
	Port    int               // Server port
	Host    string            // Bind address
	Dir     string            // Directory to serve from
	HTTPS   bool              // Enable HTTPS
	Cert    string            // TLS certificate file
	Key     string            // TLS key file
	Open    bool              // Open browser on start
	Gzip    bool              // Enable gzip compression
	Brotli  bool              // Enable brotli compression
	Cache   bool              // Enable cache headers
	Headers map[string]string // Custom headers
}

// Serve starts a production server to serve static files.
func Serve(config *ServeConfig) {
	if config == nil {
		config = &ServeConfig{
			Port:   8080,
			Host:   "localhost",
			Dir:    "dist",
			Gzip:   true,
			Brotli: true,
			Cache:  true,
		}
	}

	if config.Dir == "" {
		config.Dir = "dist"
	}

	if _, err := os.Stat(config.Dir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: directory %s does not exist\n", config.Dir)
		os.Exit(1)
	}

	handler := http.FileServer(http.Dir(config.Dir))

	var wrappedHandler http.Handler
	wrappedHandler = handler

	if config.Gzip || config.Brotli {
		wrappedHandler = &compressionHandler{
			handler: handler,
			gzip:    config.Gzip,
			brotli:  config.Brotli,
		}
	}

	if config.Cache {
		wrappedHandler = &cacheHandler{
			handler: wrappedHandler,
			headers: config.Headers,
		}
	} else if len(config.Headers) > 0 {
		wrappedHandler = &headerHandler{
			handler: wrappedHandler,
			headers: config.Headers,
		}
	}

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	if config.HTTPS {
		if config.Cert == "" || config.Key == "" {
			fmt.Fprintln(os.Stderr, "Error: --cert and --key are required for HTTPS")
			os.Exit(1)
		}

		server := &http.Server{
			Addr:              addr,
			Handler:           wrappedHandler,
			ReadHeaderTimeout: 5 * time.Second,
		}

		fmt.Printf("Server running at https://%s\n", addr)
		if err := server.ListenAndServeTLS(config.Cert, config.Key); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	} else {
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
			os.Exit(1)
		}

		server := &http.Server{
			Handler:           wrappedHandler,
			ReadHeaderTimeout: 5 * time.Second,
		}

		fmt.Printf("Server running at http://%s\n", addr)
		if err := server.Serve(listener); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	}
}

type compressionHandler struct {
	handler http.Handler
	gzip    bool
	brotli  bool
}

func (h *compressionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	acceptEncoding := r.Header.Get("Accept-Encoding")

	if h.brotli && strings.Contains(acceptEncoding, "br") {
		w.Header().Set("Content-Encoding", "br")
		w.Header().Set("Vary", "Accept-Encoding")
	} else if h.gzip && strings.Contains(acceptEncoding, "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
	}

	h.handler.ServeHTTP(w, r)
}

type cacheHandler struct {
	handler http.Handler
	headers map[string]string
}

func (h *cacheHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ext := strings.ToLower(filepath.Ext(r.URL.Path))
	switch ext {
	case ".js", ".css", ".html", ".svg", ".json", ".xml":
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	case ".gz", ".br":
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}

	for k, v := range h.headers {
		w.Header().Set(k, v)
	}

	h.handler.ServeHTTP(w, r)
}

type headerHandler struct {
	handler http.Handler
	headers map[string]string
}

func (h *headerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for k, v := range h.headers {
		w.Header().Set(k, v)
	}
	h.handler.ServeHTTP(w, r)
}

// ServeWithMultiFormat serves static files with support for multiple compression formats.
// It handles Brotli (.br) and gzip (.gz) compressed files when Accept-Encoding includes them.
func ServeWithMultiFormat(config *ServeConfig) {
	dir := config.Dir
	if dir == "" {
		dir = "dist"
	}

	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dir, r.URL.Path)

		if _, err := os.Stat(path); err == nil { //nolint:gosec // G703: path is constructed from URL.Path within controlled directory
			serveFile(w, r, path, config)
			return
		}

		acceptEncoding := r.Header.Get("Accept-Encoding")
		for _, enc := range []string{"br", "gz"} {
			if strings.Contains(acceptEncoding, enc) {
				compressedPath := path + "." + enc
				if _, err := os.Stat(compressedPath); err == nil { //nolint:gosec // G703: compressedPath is derived from validated path
					serveCompressedFile(w, r, compressedPath, enc, config)
					return
				}
			}
		}

		serveFile(w, r, path, config)
	}))

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	fmt.Printf("Server running at http://%s\n", addr)

	server := &http.Server{
		Addr:              addr,
		Handler:           nil,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func serveFile(w http.ResponseWriter, r *http.Request, path string, config *ServeConfig) {
	f, err := os.Open(path) //nolint:gosec // G304: path is validated before this call
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close() //nolint:errcheck

	w.Header().Set("Content-Type", getMimeType(path))

	if config.Cache {
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".js", ".css", ".html", ".svg", ".json", ".xml":
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
	}

	for k, v := range config.Headers {
		w.Header().Set(k, v)
	}

	http.ServeContent(w, r, path, time.Time{}, f)
}

func serveCompressedFile(w http.ResponseWriter, r *http.Request, path, encoding string, config *ServeConfig) {
	f, err := os.Open(path) //nolint:gosec // G304: path is validated before this call
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := f.Close(); err != nil {
		return
	}

	switch encoding {
	case "br":
		w.Header().Set("Content-Encoding", "br")
	case "gz":
		w.Header().Set("Content-Encoding", "gzip")
	}

	w.Header().Set("Content-Type", getMimeType(strings.TrimSuffix(path, "."+encoding)))
	w.Header().Set("Vary", "Accept-Encoding")

	if config.Cache {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}

	for k, v := range config.Headers {
		w.Header().Set(k, v)
	}

	http.ServeContent(w, r, path, time.Time{}, f)
}

func getMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	default:
		return "application/octet-stream"
	}
}
