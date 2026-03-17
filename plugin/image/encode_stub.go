//go:build !cgo

package image

import (
	"fmt"
	"image"
)

func (p *ImagePlugin) saveWebP(_ image.Image, _ string, _ int) error {
	return fmt.Errorf("webp encoding requires cgo with libwebp")
}

func (p *ImagePlugin) saveAVIF(_ image.Image, _ string, _ int) error {
	return fmt.Errorf("avif encoding requires cgo with libheif/libaom")
}
