# Image Optimization Plugin

Optimize images for production with responsive sizes and built-in parallel processing.

## Installation

```bash
gospa add image
```

## Configuration

```yaml
plugins:
  image:
    input: ./static/images
    output: ./static/images/optimized
    formats: [webp, avif, jpeg]
    widths: [320, 640, 1280]
    quality: 85
    max_image_size: 20971520 # 20MB
    max_dimensions: 8192
    concurrency: 4
```

## Performance & Security
- **Parallel Processing:** Images are optimized in parallel using a worker pool.
- **Safety Bounds:** The plugin enforces maximum file size and dimensions to prevent decompression bombs and memory exhaustion.

## CLI Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `image:optimize` | `io` | Optimize all images |
| `image:clean` | `ic` | Clean optimized images |
| `image:sizes` | `is` | List available image sizes |

## Requirements

`cgo` must be enabled with `libwebp` and `libheif` installed on the system for WebP and AVIF support.
