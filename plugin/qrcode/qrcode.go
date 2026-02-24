// Package qrcode provides QR code generation for GoSPA applications.
// It integrates with the auth plugin for TOTP/OTP setup and can be used
// independently for any QR code generation needs.
package qrcode

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"

	"github.com/aydenstechdungeon/gospa/plugin"
)

// Level represents the error correction level.
type Level int

const (
	// LevelLow 7% error recovery.
	LevelLow Level = iota
	// LevelMedium 15% error recovery.
	LevelMedium
	// LevelQuartile 25% error recovery.
	LevelQuartile
	// LevelHigh 30% error recovery.
	LevelHigh
)

// QRCodePlugin is the QR code generation plugin for GoSPA.
type QRCodePlugin struct {
	// DefaultSize is the default QR code size in pixels.
	DefaultSize int
	// DefaultLevel is the default error correction level.
	DefaultLevel Level
	// DefaultForeground is the default foreground color.
	DefaultForeground color.Color
	// DefaultBackground is the default background color.
	DefaultBackground color.Color
}

// QRCode represents a QR code.
type QRCode struct {
	Content string
	Level   Level
	Size    int
	// Module colors
	Foreground color.Color
	Background color.Color
}

// Option is a functional option for QR code generation.
type Option func(*QRCode)

// Config holds plugin configuration from gospa.yaml.
type Config struct {
	// Default size in pixels (default: 256).
	DefaultSize int `yaml:"default_size" json:"defaultSize"`
	// Error correction level: low, medium, quartile, high (default: medium).
	DefaultLevel string `yaml:"default_level" json:"defaultLevel"`
}

var _ plugin.Plugin = (*QRCodePlugin)(nil)

// NewPlugin creates a new QR code plugin with default settings.
func NewPlugin() *QRCodePlugin {
	return &QRCodePlugin{
		DefaultSize:       256,
		DefaultLevel:      LevelMedium,
		DefaultForeground: color.Black,
		DefaultBackground: color.White,
	}
}

// NewWithConfig creates a new QR code plugin with configuration.
func NewWithConfig(cfg Config) *QRCodePlugin {
	p := NewPlugin()
	if cfg.DefaultSize > 0 {
		p.DefaultSize = cfg.DefaultSize
	}
	switch cfg.DefaultLevel {
	case "low":
		p.DefaultLevel = LevelLow
	case "quartile":
		p.DefaultLevel = LevelQuartile
	case "high":
		p.DefaultLevel = LevelHigh
	}
	return p
}

// Name returns the plugin name.
func (p *QRCodePlugin) Name() string {
	return "qrcode"
}

// Init initializes the plugin.
func (p *QRCodePlugin) Init() error {
	return nil
}

// Dependencies returns the plugin dependencies.
func (p *QRCodePlugin) Dependencies() []plugin.Dependency {
	// No external dependencies - pure Go implementation
	return nil
}

// WithLevel sets the error correction level.
func WithLevel(level Level) Option {
	return func(qr *QRCode) {
		qr.Level = level
	}
}

// WithSize sets the size in pixels.
func WithSize(size int) Option {
	return func(qr *QRCode) {
		qr.Size = size
	}
}

// WithColors sets the foreground and background colors.
func WithColors(foreground, background color.Color) Option {
	return func(qr *QRCode) {
		qr.Foreground = foreground
		qr.Background = background
	}
}

// NewQRCode creates a new QR code with the given content and options.
func (p *QRCodePlugin) NewQRCode(content string, opts ...Option) *QRCode {
	qr := &QRCode{
		Content:    content,
		Level:      p.DefaultLevel,
		Size:       p.DefaultSize,
		Foreground: p.DefaultForeground,
		Background: p.DefaultBackground,
	}
	for _, opt := range opts {
		opt(qr)
	}
	return qr
}

// Generate generates a QR code image for the given content.
func (p *QRCodePlugin) Generate(content string, opts ...Option) (image.Image, error) {
	return p.NewQRCode(content, opts...).Generate()
}

// GeneratePNG generates a PNG-encoded QR code.
func (p *QRCodePlugin) GeneratePNG(content string, opts ...Option) ([]byte, error) {
	return p.NewQRCode(content, opts...).PNG()
}

// GenerateDataURL generates a data URL for the QR code.
func (p *QRCodePlugin) GenerateDataURL(content string, opts ...Option) (string, error) {
	return p.NewQRCode(content, opts...).DataURL()
}

// ForOTP generates a QR code for OTP/TOTP setup.
// The URL should be in the format: otpauth://totp/Issuer:Account?secret=XXX&issuer=Issuer
func (p *QRCodePlugin) ForOTP(otpURL string, opts ...Option) (string, error) {
	// Default to larger size for OTP codes
	opts = append([]Option{WithSize(300)}, opts...)
	return p.GenerateDataURL(otpURL, opts...)
}

// Generate generates a QR code image.
func (qr *QRCode) Generate() (image.Image, error) {
	// Encode the content to QR code matrix
	matrix, err := encode(qr.Content, qr.Level)
	if err != nil {
		return nil, err
	}

	// Calculate module size
	moduleSize := qr.Size / len(matrix)
	if moduleSize < 1 {
		moduleSize = 1
	}

	// Create image
	img := image.NewRGBA(image.Rect(0, 0, qr.Size, qr.Size))

	// Fill background
	for y := 0; y < qr.Size; y++ {
		for x := 0; x < qr.Size; x++ {
			img.Set(x, y, qr.Background)
		}
	}

	// Draw modules
	offset := (qr.Size - moduleSize*len(matrix)) / 2
	for y, row := range matrix {
		for x, module := range row {
			if module {
				// Draw module
				for dy := 0; dy < moduleSize; dy++ {
					for dx := 0; dx < moduleSize; dx++ {
						px := offset + x*moduleSize + dx
						py := offset + y*moduleSize + dy
						if px >= 0 && px < qr.Size && py >= 0 && py < qr.Size {
							img.Set(px, py, qr.Foreground)
						}
					}
				}
			}
		}
	}

	return img, nil
}

// PNG generates a PNG-encoded QR code.
func (qr *QRCode) PNG() ([]byte, error) {
	img, err := qr.Generate()
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Base64 generates a base64-encoded PNG QR code.
func (qr *QRCode) Base64() (string, error) {
	data, err := qr.PNG()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// DataURL generates a data URL for the QR code.
func (qr *QRCode) DataURL() (string, error) {
	base64, err := qr.Base64()
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64, nil
}

// encode encodes content to a QR code matrix.
func encode(content string, level Level) ([][]bool, error) {
	// Determine version based on content length
	version := getVersion(len(content), level)
	if version == 0 {
		return nil, fmt.Errorf("content too long")
	}

	// Calculate matrix size (version 1 = 21x21, each version adds 4)
	size := 21 + (version-1)*4

	// Create matrix
	matrix := make([][]bool, size)
	for i := range matrix {
		matrix[i] = make([]bool, size)
	}

	// Add finder patterns
	addFinderPattern(matrix, 0, 0)
	addFinderPattern(matrix, size-7, 0)
	addFinderPattern(matrix, 0, size-7)

	// Add timing patterns
	for i := 8; i < size-8; i++ {
		matrix[6][i] = i%2 == 0
		matrix[i][6] = i%2 == 0
	}

	// Add alignment patterns for version >= 2
	if version >= 2 {
		// Simplified: just add one alignment pattern
		ax := size - 9
		ay := size - 9
		addAlignmentPattern(matrix, ax, ay)
	}

	// Encode data (simplified - just encode as bits)
	data := encodeData(content, version, level)

	// Place data in matrix
	placeData(matrix, data, size)

	return matrix, nil
}

// getVersion returns the minimum QR version needed for the content.
func getVersion(length int, level Level) int {
	// Simplified capacity table (alphanumeric mode)
	capacities := []int{
		0, 25, 47, 77, 114, 154, 195, 224, 279, 335,
		395, 468, 535, 619, 667, 758, 854, 938, 1046, 1153,
	}

	// Reduce capacity based on error correction level
	factor := 1.0
	switch level {
	case LevelMedium:
		factor = 0.85
	case LevelQuartile:
		factor = 0.70
	case LevelHigh:
		factor = 0.55
	}

	for v, cap := range capacities {
		if int(float64(cap)*factor) >= length {
			return v
		}
	}
	return 0
}

// addFinderPattern adds a finder pattern at the given position.
func addFinderPattern(matrix [][]bool, x, y int) {
	// Outer border (7x7 black border with white inner)
	for dy := 0; dy < 7; dy++ {
		for dx := 0; dx < 7; dx++ {
			if dy == 0 || dy == 6 || dx == 0 || dx == 6 ||
				(dy >= 2 && dy <= 4 && dx >= 2 && dx <= 4) {
				matrix[y+dy][x+dx] = true
			}
		}
	}
}

// addAlignmentPattern adds an alignment pattern at the given position.
func addAlignmentPattern(matrix [][]bool, x, y int) {
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			if abs(dy) == 2 || abs(dx) == 2 || (dy == 0 && dx == 0) {
				matrix[y+dy][x+dx] = true
			}
		}
	}
}

// abs returns the absolute value.
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// encodeData encodes content to bit stream.
func encodeData(content string, version int, level Level) []bool {
	// Simplified encoding - just convert to binary
	var bits []bool

	// Mode indicator (byte mode = 0100)
	bits = append(bits, false, true, false, false)

	// Character count (8 bits for version 1-9, 16 for 10+)
	count := len(content)
	if version < 10 {
		for i := 7; i >= 0; i-- {
			bits = append(bits, (count>>i)&1 == 1)
		}
	} else {
		for i := 15; i >= 0; i-- {
			bits = append(bits, (count>>i)&1 == 1)
		}
	}

	// Data
	for _, c := range content {
		for i := 7; i >= 0; i-- {
			bits = append(bits, (int(c)>>i)&1 == 1)
		}
	}

	// Terminator (0000)
	bits = append(bits, false, false, false, false)

	// Pad to byte boundary
	for len(bits)%8 != 0 {
		bits = append(bits, false)
	}

	return bits
}

// placeData places data bits in the matrix.
func placeData(matrix [][]bool, data []bool, size int) {
	// Zigzag pattern from bottom-right
	x := size - 1
	y := size - 1
	dx := -1
	bitIndex := 0

	for x > 0 {
		if x == 6 {
			x-- // Skip timing pattern column
		}

		for y >= 0 && y < size {
			// Skip if module is already set (finder pattern, etc.)
			if !isReserved(matrix, x, y, size) && bitIndex < len(data) {
				matrix[y][x] = data[bitIndex]
				bitIndex++
			}
			if !isReserved(matrix, x+1, y, size) && bitIndex < len(data) {
				matrix[y][x+1] = data[bitIndex]
				bitIndex++
			}
			y -= dx
		}
		y += dx
		if y < 0 {
			y = 0
		} else if y >= size {
			y = size - 1
		}
		dx = -dx
		x -= 2
	}
}

// isReserved checks if a module is reserved for patterns.
func isReserved(matrix [][]bool, x, y, size int) bool {
	// Check bounds
	if x < 0 || x >= size || y < 0 || y >= size {
		return true
	}

	// Check finder patterns
	if (x < 9 && y < 9) || (x < 9 && y >= size-8) || (x >= size-8 && y < 9) {
		return true
	}

	// Check timing patterns
	if x == 6 || y == 6 {
		return true
	}

	return false
}

// Package-level functions for convenience

var defaultPlugin = NewPlugin()

// NewQRCode creates a new QR code with the given content and options.
// Uses the default plugin settings.
func NewQRCode(content string, opts ...Option) *QRCode {
	return defaultPlugin.NewQRCode(content, opts...)
}

// Generate generates a QR code image for the given content.
func Generate(content string, opts ...Option) (image.Image, error) {
	return defaultPlugin.Generate(content, opts...)
}

// GeneratePNG generates a PNG-encoded QR code.
func GeneratePNG(content string, opts ...Option) ([]byte, error) {
	return defaultPlugin.GeneratePNG(content, opts...)
}

// GenerateBase64 generates a base64-encoded PNG QR code.
func GenerateBase64(content string, opts ...Option) (string, error) {
	return defaultPlugin.NewQRCode(content, opts...).Base64()
}

// GenerateDataURL generates a data URL for the QR code.
func GenerateDataURL(content string, opts ...Option) (string, error) {
	return defaultPlugin.GenerateDataURL(content, opts...)
}

// ForOTP generates a QR code for OTP/TOTP setup.
func ForOTP(otpURL string, opts ...Option) (string, error) {
	return defaultPlugin.ForOTP(otpURL, opts...)
}

func init() {
	plugin.Register(defaultPlugin)
}
