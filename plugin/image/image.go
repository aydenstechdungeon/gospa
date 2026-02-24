// Package image provides image optimization for GoSPA projects.
// Supports build-time optimization with optional on-the-fly processing.
package image

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/aydenstechdungeon/gospa/plugin"
	"golang.org/x/image/draw"
)

// ImagePlugin provides image optimization capabilities.
type ImagePlugin struct {
	config *Config
}

// Config holds image optimization configuration.
type Config struct {
	// SourceDir is where source images are located.
	SourceDir string `yaml:"source_dir" json:"sourceDir"`

	// OutputDir is where optimized images are written.
	OutputDir string `yaml:"output_dir" json:"outputDir"`

	// Quality is the compression quality (1-100).
	Quality int `yaml:"quality" json:"quality"`

	// Formats to convert to.
	Formats FormatConfig `yaml:"formats" json:"formats"`

	// Sizes for responsive images.
	Sizes []SizeConfig `yaml:"sizes" json:"sizes"`

	// OnTheFly enables runtime image optimization.
	OnTheFly bool `yaml:"on_the_fly" json:"onTheFly"`

	// OnTheFlyCacheDir is the cache directory for on-the-fly images.
	OnTheFlyCacheDir string `yaml:"on_the_fly_cache_dir" json:"onTheFlyCacheDir"`

	// PreserveOriginals keeps original files alongside optimized versions.
	PreserveOriginals bool `yaml:"preserve_originals" json:"preserveOriginals"`

	// LazyLoadThreshold sets the threshold for lazy loading (in bytes).
	LazyLoadThreshold int64 `yaml:"lazy_load_threshold" json:"lazyLoadThreshold"`
}

// FormatConfig configures output formats.
type FormatConfig struct {
	// WebP enables WebP conversion.
	WebP bool `yaml:"webp" json:"webP"`

	// AVIF enables AVIF conversion (requires external tool).
	AVIF bool `yaml:"avif" json:"avif"`

	// JPEG enables JPEG optimization.
	JPEG bool `yaml:"jpeg" json:"jpeg"`

	// PNG enables PNG optimization.
	PNG bool `yaml:"png" json:"png"`
}

// SizeConfig defines a responsive image size.
type SizeConfig struct {
	// Name is the size identifier (e.g., "thumbnail", "medium", "large").
	Name string `yaml:"name" json:"name"`

	// Width is the target width in pixels.
	Width int `yaml:"width" json:"width"`

	// Height is the target height in pixels (0 for auto).
	Height int `yaml:"height" json:"height"`

	// Quality override for this size (0 uses default).
	Quality int `yaml:"quality" json:"quality"`
}

// DefaultConfig returns the default image optimization configuration.
func DefaultConfig() *Config {
	return &Config{
		SourceDir:         "static/images/src",
		OutputDir:         "static/images/dist",
		Quality:           85,
		OnTheFly:          false,
		PreserveOriginals: true,
		LazyLoadThreshold: 10240, // 10KB
		Formats: FormatConfig{
			WebP: false, // Disabled by default - requires external library for encoding
			AVIF: false, // Requires external tool
			JPEG: true,
			PNG:  true,
		},
		Sizes: []SizeConfig{
			{Name: "thumbnail", Width: 150, Height: 150, Quality: 80},
			{Name: "small", Width: 320, Height: 0, Quality: 85},
			{Name: "medium", Width: 640, Height: 0, Quality: 85},
			{Name: "large", Width: 1024, Height: 0, Quality: 85},
			{Name: "xlarge", Width: 1920, Height: 0, Quality: 90},
		},
	}
}

// New creates a new Image optimization plugin.
func New(cfg *Config) *ImagePlugin {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &ImagePlugin{config: cfg}
}

// Name returns the plugin name.
func (p *ImagePlugin) Name() string {
	return "image"
}

// Init initializes the image plugin.
func (p *ImagePlugin) Init() error {
	// Create directories
	dirs := []string{p.config.SourceDir, p.config.OutputDir}
	if p.config.OnTheFly && p.config.OnTheFlyCacheDir != "" {
		dirs = append(dirs, p.config.OnTheFlyCacheDir)
	}

	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// Dependencies returns required dependencies.
func (p *ImagePlugin) Dependencies() []plugin.Dependency {
	deps := []plugin.Dependency{
		{Type: plugin.DepGo, Name: "golang.org/x/image", Version: "latest"},
	}

	// If AVIF is enabled, we need the external tool
	if p.config.Formats.AVIF {
		deps = append(deps, plugin.Dependency{
			Type: plugin.DepBun, Name: "sharp", Version: "latest",
		})
	}

	// If on-the-fly optimization is enabled, we need server-side packages
	if p.config.OnTheFly {
		deps = append(deps, plugin.Dependency{
			Type: plugin.DepGo, Name: "github.com/disintegration/imaging", Version: "latest",
		})
	}

	return deps
}

// OnHook handles lifecycle hooks.
func (p *ImagePlugin) OnHook(hook plugin.Hook, ctx map[string]interface{}) error {
	switch hook {
	case plugin.BeforeBuild:
		projectDir, _ := ctx["project_dir"].(string)
		if projectDir == "" {
			projectDir = "."
		}
		return p.optimizeAllImages(projectDir)

	case plugin.BeforeDev:
		projectDir, _ := ctx["project_dir"].(string)
		if projectDir == "" {
			projectDir = "."
		}
		// In dev mode, only optimize changed images
		return p.optimizeChangedImages(projectDir)
	}
	return nil
}

// Commands returns custom CLI commands.
func (p *ImagePlugin) Commands() []plugin.Command {
	return []plugin.Command{
		{
			Name:        "image:optimize",
			Alias:       "io",
			Description: "Optimize all images in the source directory",
			Action: func(args []string) error {
				projectDir := "."
				if len(args) > 0 {
					projectDir = args[0]
				}
				return p.optimizeAllImages(projectDir)
			},
		},
		{
			Name:        "image:clean",
			Alias:       "ic",
			Description: "Clean optimized images cache",
			Action: func(args []string) error {
				projectDir := "."
				if len(args) > 0 {
					projectDir = args[0]
				}
				return p.cleanCache(projectDir)
			},
		},
		{
			Name:        "image:sizes",
			Alias:       "is",
			Description: "List configured image sizes",
			Action: func(args []string) error {
				fmt.Println("Configured image sizes:")
				for _, size := range p.config.Sizes {
					quality := size.Quality
					if quality == 0 {
						quality = p.config.Quality
					}
					fmt.Printf("  %s: %dx%d (quality: %d%%)\n", size.Name, size.Width, size.Height, quality)
				}
				return nil
			},
		},
	}
}

// optimizeAllImages optimizes all images in the source directory.
func (p *ImagePlugin) optimizeAllImages(projectDir string) error {
	srcDir := filepath.Join(projectDir, p.config.SourceDir)
	outDir := filepath.Join(projectDir, p.config.OutputDir)

	// Ensure output directory exists
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Walk source directory
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Check if it's an image
		ext := strings.ToLower(filepath.Ext(path))
		if !isImageFile(ext) {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// Optimize the image
		return p.optimizeImage(path, filepath.Join(outDir, relPath))
	})
}

// optimizeChangedImages optimizes only images that have changed.
func (p *ImagePlugin) optimizeChangedImages(projectDir string) error {
	// For dev mode, we could implement a file watcher or mtime check
	// For now, just optimize all images
	return p.optimizeAllImages(projectDir)
}

// optimizeImage optimizes a single image.
func (p *ImagePlugin) optimizeImage(srcPath, outPath string) error {
	// Read source image
	file, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	// Decode image
	var img image.Image
	ext := strings.ToLower(filepath.Ext(srcPath))

	switch ext {
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(file)
	case ".png":
		img, err = png.Decode(file)
	default:
		img, _, err = image.Decode(file)
	}

	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}

	// Generate responsive sizes
	for _, size := range p.config.Sizes {
		resized := p.resizeImage(img, size.Width, size.Height)
		sizeOutPath := p.addSizeSuffix(outPath, size.Name)

		// Save in each enabled format
		// Note: WebP encoding is not supported by Go's standard library.
		// For WebP support, use an external tool like sharp (npm) or cwebp.
		// The WebP config option is reserved for future implementation.

		if p.config.Formats.JPEG && ext != ".png" {
			if err := p.saveJPEG(resized, sizeOutPath+".jpg", size.Quality); err != nil {
				return err
			}
		}

		if p.config.Formats.PNG && ext == ".png" {
			if err := p.savePNG(resized, sizeOutPath+".png"); err != nil {
				return err
			}
		}
	}

	// Preserve original if configured
	if p.config.PreserveOriginals {
		origPath := outPath + ".original" + ext
		if err := copyFile(srcPath, origPath); err != nil {
			return err
		}
	}

	return nil
}

// resizeImage resizes an image to the specified dimensions using high-quality scaling.
// Uses golang.org/x/image/draw for proper image resampling.
func (p *ImagePlugin) resizeImage(img image.Image, width, height int) image.Image {
	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// Calculate aspect ratio if one dimension is 0
	if height == 0 && width > 0 {
		ratio := float64(width) / float64(srcWidth)
		height = int(float64(srcHeight) * ratio)
	}
	if width == 0 && height > 0 {
		ratio := float64(height) / float64(srcHeight)
		width = int(float64(srcWidth) * ratio)
	}

	// If target size matches source, return original
	if width == srcWidth && height == srcHeight {
		return img
	}

	// Create new image with target dimensions
	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	// Use CatmullRom interpolation for high-quality downscaling
	// This provides excellent quality for responsive image variants
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	return dst
}

// saveJPEG saves an image as JPEG format.
func (p *ImagePlugin) saveJPEG(img image.Image, path string, quality int) error {
	if quality == 0 {
		quality = p.config.Quality
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return jpeg.Encode(file, img, &jpeg.Options{Quality: quality})
}

// savePNG saves an image as PNG format.
func (p *ImagePlugin) savePNG(img image.Image, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

// addSizeSuffix adds a size suffix to a filename.
func (p *ImagePlugin) addSizeSuffix(path, size string) string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	return base + "-" + size
}

// cleanCache removes all optimized images.
func (p *ImagePlugin) cleanCache(projectDir string) error {
	outDir := filepath.Join(projectDir, p.config.OutputDir)
	return os.RemoveAll(outDir)
}

// isImageFile checks if a file extension is an image.
func isImageFile(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff", ".svg":
		return true
	default:
		return false
	}
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// GetConfig returns the current configuration.
func (p *ImagePlugin) GetConfig() *Config {
	return p.config
}

// Ensure ImagePlugin implements CLIPlugin interface.
var _ plugin.CLIPlugin = (*ImagePlugin)(nil)
