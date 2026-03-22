//go:build cgo

package image

/*
#cgo pkg-config: libwebp libheif
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <webp/encode.h>
#include <libheif/heif.h>

size_t gospa_webp_encode_rgba(const uint8_t* rgba, int width, int height, int stride, float quality, uint8_t** output) {
	return WebPEncodeRGBA(rgba, width, height, stride, quality, output);
}

void gospa_webp_free(uint8_t* output) {
	WebPFree(output);
}

const char* gospa_avif_encode_file(const uint8_t* rgba, int width, int height, int stride, int quality, const char* out_path) {
	struct heif_context* ctx = heif_context_alloc();
	if (!ctx) {
		return "failed to allocate heif context";
	}

	struct heif_encoder* encoder = NULL;
	struct heif_error err = heif_context_get_encoder_for_format(ctx, heif_compression_AV1, &encoder);
	if (err.code != heif_error_Ok || !encoder) {
		heif_context_free(ctx);
		return "failed to get AV1 encoder (libheif/libaom)";
	}

	heif_encoder_set_lossy_quality(encoder, quality);

	struct heif_image* img = NULL;
	err = heif_image_create(width, height, heif_colorspace_RGB, heif_chroma_interleaved_RGBA, &img);
	if (err.code != heif_error_Ok || !img) {
		heif_encoder_release(encoder);
		heif_context_free(ctx);
		return "failed to create heif image";
	}

	err = heif_image_add_plane(img, heif_channel_interleaved, width, height, 32);
	if (err.code != heif_error_Ok) {
		heif_image_release(img);
		heif_encoder_release(encoder);
		heif_context_free(ctx);
		return "failed to add interleaved RGBA plane";
	}

	int dst_stride = 0;
	uint8_t* dst = heif_image_get_plane(img, heif_channel_interleaved, &dst_stride);
	if (!dst) {
		heif_image_release(img);
		heif_encoder_release(encoder);
		heif_context_free(ctx);
		return "failed to access heif image plane";
	}

	for (int y = 0; y < height; y++) {
		memcpy(dst + y * dst_stride, rgba + y * stride, (size_t)width * 4);
	}

	err = heif_context_encode_image(ctx, img, encoder, NULL, NULL);
	if (err.code != heif_error_Ok) {
		heif_image_release(img);
		heif_encoder_release(encoder);
		heif_context_free(ctx);
		return "failed to encode AVIF image";
	}

	err = heif_context_write_to_file(ctx, out_path);
	if (err.code != heif_error_Ok) {
		heif_image_release(img);
		heif_encoder_release(encoder);
		heif_context_free(ctx);
		return "failed to write AVIF file";
	}

	heif_image_release(img);
	heif_encoder_release(encoder);
	heif_context_free(ctx);
	return NULL;
}
*/
import "C"

import (
	"fmt"
	"image"
	"image/draw"
	"os"
	"unsafe"
)

// #nosec G115
func (p *ImagePlugin) saveWebP(img image.Image, path string, quality int) error {
	if quality == 0 {
		quality = p.config.Quality
	}
	rgba := toRGBA(img)
	if len(rgba.Pix) == 0 {
		return fmt.Errorf("empty image data")
	}
	// Check bounds to prevent C-side overflow
	minSize := rgba.Bounds().Dy() * rgba.Stride
	if len(rgba.Pix) < minSize {
		return fmt.Errorf("invalid image data size")
	}

	var out *C.uint8_t

	cdx := C.int(int32(rgba.Bounds().Dx()))
	// #nosec G115
	cdy := C.int(int32(rgba.Bounds().Dy()))
	// #nosec G115
	cstride := C.int(int32(rgba.Stride))
	// #nosec G115
	cquality := C.float(float32(quality))

	size := C.gospa_webp_encode_rgba( //nolint:gocritic // false positive: &out is passing pointer to C function, not a comparison
		(*C.uint8_t)(unsafe.Pointer(&rgba.Pix[0])),
		cdx,
		cdy,
		cstride,
		cquality,
		&out, //nolint:gocritic // passing pointer to C function output parameter
	)
	if size == 0 || out == nil {
		return fmt.Errorf("failed to encode webp")
	}
	defer C.gospa_webp_free(out)

	// #nosec G115
	csize := C.int(int32(size))
	data := C.GoBytes(unsafe.Pointer(out), csize)
	return os.WriteFile(path, data, 0640)
}

// #nosec G115
func (p *ImagePlugin) saveAVIF(img image.Image, path string, quality int) error {
	if quality == 0 {
		quality = p.config.Quality
	}
	rgba := toRGBA(img)
	if len(rgba.Pix) == 0 {
		return fmt.Errorf("empty image data")
	}
	// Check bounds to prevent C-side overflow
	minSize := rgba.Bounds().Dy() * rgba.Stride
	if len(rgba.Pix) < minSize {
		return fmt.Errorf("invalid image data size")
	}

	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	// #nosec G115
	cdx := C.int(int32(rgba.Bounds().Dx()))
	// #nosec G115
	cdy := C.int(int32(rgba.Bounds().Dy()))
	// #nosec G115
	cstride := C.int(int32(rgba.Stride))
	// #nosec G115
	cquality := C.int(int32(quality))

	errMsg := C.gospa_avif_encode_file(
		(*C.uint8_t)(unsafe.Pointer(&rgba.Pix[0])),
		cdx,
		cdy,
		cstride,
		cquality,
		cpath,
	)
	if errMsg != nil {
		return fmt.Errorf("%s", C.GoString(errMsg))
	}

	return nil
}

func toRGBA(img image.Image) *image.RGBA {
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	return rgba
}
