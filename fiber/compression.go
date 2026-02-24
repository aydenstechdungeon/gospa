package fiber

import (
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"sync"

	"github.com/andybalholm/brotli"
	gofiber "github.com/gofiber/fiber/v2"
)

// CompressionConfig configures response compression.
type CompressionConfig struct {
	// EnableBrotli enables Brotli compression (better compression ratio)
	EnableBrotli bool
	// EnableGzip enables Gzip compression (wider browser support)
	EnableGzip bool
	// BrotliLevel compression level (0-11, default 4 for balance)
	BrotliLevel int
	// GzipLevel compression level (1-9, default 6 for balance)
	GzipLevel int
	// MinSize minimum response size to compress (default 1024 bytes)
	MinSize int
	// CompressibleTypes content types that should be compressed
	CompressibleTypes []string
	// SkipCompression paths to skip compression for
	SkipPaths []string
}

// DefaultCompressionConfig returns default compression configuration.
func DefaultCompressionConfig() CompressionConfig {
	return CompressionConfig{
		EnableBrotli: true,
		EnableGzip:   true,
		BrotliLevel:  4, // Good balance between compression and speed
		GzipLevel:    6, // Default gzip level
		MinSize:      1024,
		CompressibleTypes: []string{
			"text/html",
			"text/css",
			"text/javascript",
			"text/xml",
			"text/plain",
			"application/javascript",
			"application/json",
			"application/xml",
			"application/xhtml+xml",
			"image/svg+xml",
		},
		SkipPaths: []string{},
	}
}

// BrotliGzipMiddleware creates a compression middleware with Brotli and Gzip support.
// Brotli is preferred when supported by the client, falling back to Gzip.
func BrotliGzipMiddleware(config CompressionConfig) gofiber.Handler {
	// Validate compression levels
	if config.BrotliLevel < 0 {
		config.BrotliLevel = 0
	}
	if config.BrotliLevel > 11 {
		config.BrotliLevel = 11
	}
	if config.GzipLevel < 1 {
		config.GzipLevel = 1
	}
	if config.GzipLevel > 9 {
		config.GzipLevel = 9
	}

	// Store compression level for pool writers
	brotliLevel := config.BrotliLevel
	gzipLevel := config.GzipLevel

	// Pools for reusing compression writers
	brotliWriterPool := sync.Pool{
		New: func() interface{} {
			return brotli.NewWriterLevel(nil, brotliLevel)
		},
	}

	gzipWriterPool := sync.Pool{
		New: func() interface{} {
			w, _ := gzip.NewWriterLevel(nil, gzipLevel)
			return w
		},
	}

	return func(c *gofiber.Ctx) error {
		// Skip compression for certain paths
		path := c.Path()
		for _, skipPath := range config.SkipPaths {
			if strings.HasPrefix(path, skipPath) {
				return c.Next()
			}
		}

		// Skip if client doesn't accept encoding
		acceptEncoding := c.Get("Accept-Encoding")
		if acceptEncoding == "" {
			return c.Next()
		}

		// Determine compression method
		var useBrotli, useGzip bool
		acceptEncodingLower := strings.ToLower(acceptEncoding)

		// Prefer Brotli if available (better compression)
		if config.EnableBrotli && strings.Contains(acceptEncodingLower, "br") {
			useBrotli = true
		} else if config.EnableGzip && strings.Contains(acceptEncodingLower, "gzip") {
			useGzip = true
		}

		// No supported compression
		if !useBrotli && !useGzip {
			return c.Next()
		}

		// Continue with request
		err := c.Next()
		if err != nil {
			return err
		}

		// Check if response should be compressed
		body := c.Response().Body()
		if len(body) < config.MinSize {
			return nil
		}

		// Check content type
		contentType := string(c.Response().Header.ContentType())
		shouldCompress := false
		for _, ct := range config.CompressibleTypes {
			if strings.Contains(contentType, ct) {
				shouldCompress = true
				break
			}
		}

		if !shouldCompress {
			return nil
		}

		// Skip if already encoded
		if c.Get("Content-Encoding") != "" {
			return nil
		}

		// Compress the response
		var compressed []byte
		var encoding string

		if useBrotli {
			compressed = compressBrotli(body, brotliWriterPool)
			encoding = "br"
		} else if useGzip {
			compressed = compressGzip(body, gzipWriterPool)
			encoding = "gzip"
		}

		if len(compressed) == 0 {
			return nil
		}

		// Only use compression if it actually reduces size
		if len(compressed) >= len(body) {
			return nil
		}

		// Set headers
		c.Set("Content-Encoding", encoding)
		c.Set("Vary", "Accept-Encoding")
		c.Response().SetBody(compressed)

		return nil
	}
}

// compressBrotli compresses data using Brotli with writer pool.
func compressBrotli(data []byte, pool sync.Pool) []byte {
	writer := pool.Get().(*brotli.Writer)
	defer pool.Put(writer)

	var buf bytes.Buffer
	writer.Reset(&buf)

	_, err := writer.Write(data)
	if err != nil {
		return nil
	}

	err = writer.Close()
	if err != nil {
		return nil
	}

	return buf.Bytes()
}

// compressGzip compresses data using Gzip with writer pool.
func compressGzip(data []byte, pool sync.Pool) []byte {
	writer := pool.Get().(*gzip.Writer)
	defer pool.Put(writer)

	var buf bytes.Buffer
	writer.Reset(&buf)

	_, err := writer.Write(data)
	if err != nil {
		return nil
	}

	err = writer.Close()
	if err != nil {
		return nil
	}

	return buf.Bytes()
}

// StaticCompressionMiddleware serves pre-compressed static files.
// This is more efficient than on-the-fly compression for static assets.
type StaticCompressionMiddleware struct {
	// Cache stores compressed versions of files
	cache sync.Map
	// Config compression configuration
	config CompressionConfig
}

// NewStaticCompressionMiddleware creates a new static compression middleware.
func NewStaticCompressionMiddleware(config CompressionConfig) *StaticCompressionMiddleware {
	return &StaticCompressionMiddleware{
		config: config,
	}
}

// CompressStatic pre-compresses static content and caches it.
// Use this for embedding static assets that don't change.
func (s *StaticCompressionMiddleware) CompressStatic(content []byte, contentType string) *CompressedContent {
	result := &CompressedContent{
		Original: content,
	}

	// Compress with Brotli
	if s.config.EnableBrotli {
		var brotliBuf bytes.Buffer
		writer := brotli.NewWriterLevel(&brotliBuf, s.config.BrotliLevel)
		_, err := writer.Write(content)
		if err == nil {
			err = writer.Close()
			if err == nil && brotliBuf.Len() < len(content) {
				result.Brotli = brotliBuf.Bytes()
			}
		}
	}

	// Compress with Gzip
	if s.config.EnableGzip {
		var gzipBuf bytes.Buffer
		writer, _ := gzip.NewWriterLevel(&gzipBuf, s.config.GzipLevel)
		_, err := writer.Write(content)
		if err == nil {
			err = writer.Close()
			if err == nil && gzipBuf.Len() < len(content) {
				result.Gzip = gzipBuf.Bytes()
			}
		}
	}

	return result
}

// CompressedContent holds original and compressed versions of content.
type CompressedContent struct {
	Original []byte
	Brotli   []byte
	Gzip     []byte
}

// ServeCompressed serves the best compression for the client.
func (s *StaticCompressionMiddleware) ServeCompressed(c *gofiber.Ctx, content *CompressedContent, contentType string) error {
	acceptEncoding := strings.ToLower(c.Get("Accept-Encoding"))

	// Try Brotli first (best compression)
	if content.Brotli != nil && strings.Contains(acceptEncoding, "br") {
		c.Set("Content-Encoding", "br")
		c.Set("Vary", "Accept-Encoding")
		c.Set("Content-Type", contentType)
		return c.Send(content.Brotli)
	}

	// Fall back to Gzip
	if content.Gzip != nil && strings.Contains(acceptEncoding, "gzip") {
		c.Set("Content-Encoding", "gzip")
		c.Set("Vary", "Accept-Encoding")
		c.Set("Content-Type", contentType)
		return c.Send(content.Gzip)
	}

	// No compression
	c.Set("Content-Type", contentType)
	return c.Send(content.Original)
}

// StreamingCompressionWriter wraps a writer with compression support.
// Useful for streaming responses like SSE.
type StreamingCompressionWriter struct {
	writer     io.Writer
	compressor interface{}
	compressed bool
}

// NewBrotliStreamingWriter creates a streaming Brotli writer.
func NewBrotliStreamingWriter(w io.Writer, level int) *StreamingCompressionWriter {
	return &StreamingCompressionWriter{
		writer:     w,
		compressor: brotli.NewWriterLevel(w, level),
		compressed: true,
	}
}

// NewGzipStreamingWriter creates a streaming Gzip writer.
func NewGzipStreamingWriter(w io.Writer, level int) *StreamingCompressionWriter {
	writer, _ := gzip.NewWriterLevel(w, level)
	return &StreamingCompressionWriter{
		writer:     w,
		compressor: writer,
		compressed: true,
	}
}

// Write writes data to the compressed stream.
func (w *StreamingCompressionWriter) Write(p []byte) (n int, err error) {
	if !w.compressed {
		return w.writer.Write(p)
	}

	switch c := w.compressor.(type) {
	case *brotli.Writer:
		return c.Write(p)
	case *gzip.Writer:
		return c.Write(p)
	default:
		return w.writer.Write(p)
	}
}

// Close closes the compression writer.
func (w *StreamingCompressionWriter) Close() error {
	if !w.compressed {
		return nil
	}

	switch c := w.compressor.(type) {
	case *brotli.Writer:
		return c.Close()
	case *gzip.Writer:
		return c.Close()
	default:
		return nil
	}
}

// Flush flushes the compression buffer.
func (w *StreamingCompressionWriter) Flush() error {
	if !w.compressed {
		return nil
	}

	switch c := w.compressor.(type) {
	case *brotli.Writer:
		return c.Flush()
	case *gzip.Writer:
		return c.Flush()
	default:
		return nil
	}
}
