package qrcode

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestNewPlugin(t *testing.T) {
	p := NewPlugin()
	if p.DefaultSize != 256 {
		t.Errorf("expected default size 256, got %d", p.DefaultSize)
	}
	if p.DefaultLevel != LevelMedium {
		t.Errorf("expected default level medium, got %v", p.DefaultLevel)
	}
}

func TestNewWithConfig(t *testing.T) {
	cfg := Config{
		DefaultSize:  512,
		DefaultLevel: "high",
	}
	p := NewWithConfig(cfg)
	if p.DefaultSize != 512 {
		t.Errorf("expected size 512, got %d", p.DefaultSize)
	}
	if p.DefaultLevel != LevelHigh {
		t.Errorf("expected level high, got %v", p.DefaultLevel)
	}
}

func TestGenerate(t *testing.T) {
	p := NewPlugin()
	content := "https://gospa.dev"
	img, err := p.Generate(content)
	if err != nil {
		t.Fatalf("failed to generate QR code: %v", err)
	}

	if img.Bounds().Dx() != 256 || img.Bounds().Dy() != 256 {
		t.Errorf("expected 256x256 image, got %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestWithSize(t *testing.T) {
	p := NewPlugin()
	content := "test"
	img, err := p.Generate(content, WithSize(128))
	if err != nil {
		t.Fatalf("failed to generate QR code: %v", err)
	}

	if img.Bounds().Dx() != 128 {
		t.Errorf("expected 128x128 image, got %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestDataURL(t *testing.T) {
	p := NewPlugin()
	content := "test data url"
	dataURL, err := p.GenerateDataURL(content)
	if err != nil {
		t.Fatalf("failed to generate data URL: %v", err)
	}

	if !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		t.Errorf("expected data URL prefix, got %s", dataURL)
	}
}

func TestForOTP(t *testing.T) {
	p := NewPlugin()
	otpURL := "otpauth://totp/Example:alice@google.com?secret=JBSWY3DPEHPK3PXP&issuer=Example"
	dataURL, err := p.ForOTP(otpURL)
	if err != nil {
		t.Fatalf("failed to generate OTP QR code: %v", err)
	}

	if !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		t.Errorf("expected data URL prefix, got %s", dataURL)
	}
}

func TestGenerateWithLogo(t *testing.T) {
	p := NewPlugin()
	content := "test with logo"

	// Create a simple 10x10 red logo
	logo := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			logo.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}

	img, err := p.GenerateWithLogo(content, logo, WithSize(200))
	if err != nil {
		t.Fatalf("failed to generate QR code with logo: %v", err)
	}

	if img.Bounds().Dx() != 200 {
		t.Errorf("expected 200x200 image, got %d", img.Bounds().Dx())
	}

	// Check center pixel of the logo area (200/5 = 40x40 logo area)
	// The logo is centered.
	centerColor := img.At(100, 100)
	r, g, b, _ := centerColor.RGBA()
	// Since color.RGBA{255, 0, 0, 255} is red, r should be high, g and b low.
	if r < 0xF000 || g > 0x1000 || b > 0x1000 {
		t.Errorf("expected red logo at center, got %v", centerColor)
	}
}
