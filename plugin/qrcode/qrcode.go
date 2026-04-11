// Package qrcode provides QR code generation for GoSPA applications.
// It integrates with the auth plugin for TOTP/OTP setup and can be used
// independently for any QR code generation needs.
package qrcode

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"

	"github.com/aydenstechdungeon/gospa/plugin"
	"github.com/skip2/go-qrcode"
	"golang.org/x/image/draw"
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

// Plugin is the QR code generation plugin for GoSPA.
type Plugin struct {
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

var _ plugin.Plugin = (*Plugin)(nil)

// NewPlugin creates a new QR code plugin with default settings.
func NewPlugin() *Plugin {
	return &Plugin{
		DefaultSize:       256,
		DefaultLevel:      LevelMedium,
		DefaultForeground: color.Black,
		DefaultBackground: color.White,
	}
}

// NewWithConfig creates a new QR code plugin with configuration.
func NewWithConfig(cfg Config) *Plugin {
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
func (p *Plugin) Name() string {
	return "qrcode"
}

// Init initializes the plugin.
func (p *Plugin) Init() error {
	return nil
}

// Dependencies returns the plugin dependencies.
func (p *Plugin) Dependencies() []plugin.Dependency {
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
func (p *Plugin) NewQRCode(content string, opts ...Option) *QRCode {
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
func (p *Plugin) Generate(content string, opts ...Option) (image.Image, error) {
	return p.NewQRCode(content, opts...).Generate()
}

// GeneratePNG generates a PNG-encoded QR code.
func (p *Plugin) GeneratePNG(content string, opts ...Option) ([]byte, error) {
	return p.NewQRCode(content, opts...).PNG()
}

// GenerateDataURL generates a data URL for the QR code.
func (p *Plugin) GenerateDataURL(content string, opts ...Option) (string, error) {
	return p.NewQRCode(content, opts...).DataURL()
}

// ForOTP generates a QR code for OTP/TOTP setup.
// The URL should be in the format: otpauth://totp/Issuer:Account?secret=XXX&issuer=Issuer
func (p *Plugin) ForOTP(otpURL string, opts ...Option) (string, error) {
	// Default to larger size for OTP codes
	opts = append([]Option{WithSize(300)}, opts...)
	return p.GenerateDataURL(otpURL, opts...)
}

// toQrcodeLevel converts our Level to go-qrcode's RecoveryLevel.
func (l Level) toQrcodeLevel() qrcode.RecoveryLevel {
	switch l {
	case LevelLow:
		return qrcode.Low
	case LevelMedium:
		return qrcode.Medium
	case LevelQuartile:
		return qrcode.High
	case LevelHigh:
		return qrcode.Highest
	default:
		return qrcode.Medium
	}
}

// Generate generates a QR code image.
func (qr *QRCode) Generate() (image.Image, error) {
	// Use the go-qrcode library which implements proper Reed-Solomon error correction
	q, err := qrcode.New(qr.Content, qr.Level.toQrcodeLevel())
	if err != nil {
		return nil, err
	}
	return q.Image(qr.Size), nil
}

// PNG generates a PNG-encoded QR code.
func (qr *QRCode) PNG() ([]byte, error) {
	// Use the go-qrcode library for proper encoding with Reed-Solomon error correction
	return qrcode.Encode(qr.Content, qr.Level.toQrcodeLevel(), qr.Size)
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
	base64Data, err := qr.Base64()
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64Data, nil
}

// GenerateWithLogo generates a QR code with a logo in the center.
func (p *Plugin) GenerateWithLogo(content string, logo image.Image, opts ...Option) (image.Image, error) {
	qr := p.NewQRCode(content, opts...)

	// If logo is provided, we need higher error correction to maintain scannability
	// A logo covering 20% of the QR code center can interfere with ~15% of modules
	// Upgrade to LevelHigh (30%) if using a logo with lower correction levels
	if logo != nil && qr.Level < LevelHigh {
		qr.Level = LevelHigh
	}

	// Generate base QR code
	baseQR, err := qrcode.New(content, qr.Level.toQrcodeLevel())
	if err != nil {
		return nil, err
	}

	// If logo provided, overlay it
	if logo != nil {
		return overlayLogo(baseQR.Image(qr.Size), logo, qr.Size)
	}

	return baseQR.Image(qr.Size), nil
}

// overlayLogo overlays a logo on the QR code.
func overlayLogo(qr image.Image, logo image.Image, size int) (image.Image, error) {
	if size < 5 || logo == nil {
		return qr, nil
	}

	// Create output image
	output := image.NewRGBA(image.Rect(0, 0, size, size))

	// Scale QR code to output size using CatmullRom interpolation for high quality
	qrBounds := qr.Bounds()
	dstBounds := image.Rect(0, 0, size, size)
	draw.CatmullRom.Scale(output, dstBounds, qr, qrBounds, draw.Over, nil)

	// Calculate logo position (centered, max 20% of QR code size)
	logoSize := size / 5
	logoX := (size - logoSize) / 2
	logoY := (size - logoSize) / 2

	// Scale logo to fit in the center area using high-quality interpolation
	logoBounds := logo.Bounds()
	logoDst := image.Rect(logoX, logoY, logoX+logoSize, logoY+logoSize)
	scaledLogo := image.NewRGBA(logoDst)
	draw.CatmullRom.Scale(scaledLogo, logoDst, logo, logoBounds, draw.Over, nil)

	// Draw scaled logo onto output with alpha blending
	for y := logoY; y < logoY+logoSize; y++ {
		for x := logoX; x < logoX+logoSize; x++ {
			srcX := logoX + (x - logoX)
			srcY := logoY + (y - logoY)
			_, _, _, a := scaledLogo.At(srcX, srcY).RGBA()
			if a > 0 {
				output.Set(x, y, scaledLogo.At(srcX, srcY))
			}
		}
	}

	return output, nil
}

// GeneratePNGWithLogo generates a PNG QR code with a logo.
func (p *Plugin) GeneratePNGWithLogo(content string, logo image.Image, opts ...Option) ([]byte, error) {
	img, err := p.GenerateWithLogo(content, logo, opts...)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
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

// GenerateWithLogo generates a QR code with a logo in the center.
func GenerateWithLogo(content string, logo image.Image, opts ...Option) (image.Image, error) {
	return defaultPlugin.GenerateWithLogo(content, logo, opts...)
}

// GeneratePNGWithLogo generates a PNG QR code with a logo.
func GeneratePNGWithLogo(content string, logo image.Image, opts ...Option) ([]byte, error) {
	return defaultPlugin.GeneratePNGWithLogo(content, logo, opts...)
}

func init() {
	if err := plugin.Register(defaultPlugin); err != nil {
		panic("failed to register qrcode plugin: " + err.Error())
	}
}
