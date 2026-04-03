# Image Optimization Plugin

Optimize images for production with responsive sizes.

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
```

## CLI Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `image:optimize` | `io` | Optimize all images |
| `image:clean` | `ic` | Clean optimized images |
| `image:sizes` | `is` | List available image sizes |

## Requirements

`cgo` must be enabled with `libwebp` and `libheif` installed on the system.
