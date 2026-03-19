//go:build cgo

package image

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

func sampleImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 16), G: uint8(y * 16), B: 128, A: 255})
		}
	}
	return img
}

func TestSaveWebP(t *testing.T) {
	p := New(DefaultConfig())
	path := filepath.Join(t.TempDir(), "test.webp")

	if err := p.saveWebP(sampleImage(), path, 80); err != nil {
		t.Fatalf("saveWebP failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("webp output is empty")
	}
}

func TestSaveAVIF(t *testing.T) {
	p := New(DefaultConfig())
	path := filepath.Join(t.TempDir(), "test.avif")

	if err := p.saveAVIF(sampleImage(), path, 80); err != nil {
		if err.Error() == "failed to get AV1 encoder (libheif/libaom)" {
			t.Skipf("AV1 encoder unavailable in environment: %v", err)
		}
		t.Fatalf("saveAVIF failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("avif output is empty")
	}
}
