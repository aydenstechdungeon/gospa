// Package seo provides SEO optimization for GoSPA projects.
// Includes meta tags, sitemap generation, structured data (JSON-LD), and Open Graph.
package seo

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aydenstechdungeon/gospa/plugin"
)

// SEOPlugin provides SEO optimization capabilities.
type SEOPlugin struct {
	config *Config
}

// Config holds SEO plugin configuration.
type Config struct {
	// SiteURL is the base URL of the website.
	SiteURL string `yaml:"site_url" json:"siteUrl"`

	// SiteName is the name of the website.
	SiteName string `yaml:"site_name" json:"siteName"`

	// DefaultTitle is the default page title.
	DefaultTitle string `yaml:"default_title" json:"defaultTitle"`

	// DefaultDescription is the default meta description.
	DefaultDescription string `yaml:"default_description" json:"defaultDescription"`

	// DefaultImage is the default Open Graph image.
	DefaultImage string `yaml:"default_image" json:"defaultImage"`

	// TwitterHandle is the Twitter/X handle (@username).
	TwitterHandle string `yaml:"twitter_handle" json:"twitterHandle"`

	// Language is the default language code.
	Language string `yaml:"language" json:"language"`

	// OutputDir is where generated SEO files are written.
	OutputDir string `yaml:"output_dir" json:"outputDir"`

	// GenerateSitemap enables sitemap.xml generation.
	GenerateSitemap bool `yaml:"generate_sitemap" json:"generateSitemap"`

	// GenerateRobots enables robots.txt generation.
	GenerateRobots bool `yaml:"generate_robots" json:"generateRobots"`

	// RoutesDir is where route files are located.
	RoutesDir string `yaml:"routes_dir" json:"routesDir"`
}

// PageSEO represents SEO metadata for a page.
type PageSEO struct {
	Path        string   `json:"path"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Image       string   `json:"image"`
	Keywords    []string `json:"keywords"`
	NoIndex     bool     `json:"noIndex"`
	NoFollow    bool     `json:"noFollow"`
	Canonical   string   `json:"canonical"`
	Modified    string   `json:"modified"`
	ChangeFreq  string   `json:"changeFreq"`
	Priority    float64  `json:"priority"`
}

// StructuredData represents JSON-LD structured data.
type StructuredData struct {
	Type       string                 `json:"@type"`
	Context    string                 `json:"@context"`
	Properties map[string]interface{} `json:"-"`
}

// DefaultConfig returns the default SEO configuration.
func DefaultConfig() *Config {
	return &Config{
		SiteURL:            "https://example.com",
		SiteName:           "My GoSPA Site",
		DefaultTitle:       "Home",
		DefaultDescription: "A GoSPA application",
		Language:           "en",
		OutputDir:          "static",
		GenerateSitemap:    true,
		GenerateRobots:     true,
		RoutesDir:          "routes",
	}
}

// New creates a new SEO plugin.
func New(cfg *Config) *SEOPlugin {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &SEOPlugin{config: cfg}
}

// Name returns the plugin name.
func (p *SEOPlugin) Name() string {
	return "seo"
}

// Init initializes the SEO plugin.
func (p *SEOPlugin) Init() error {
	// Create output directory
	if err := os.MkdirAll(p.config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	return nil
}

// Dependencies returns required dependencies.
func (p *SEOPlugin) Dependencies() []plugin.Dependency {
	return []plugin.Dependency{
		// No external dependencies required
	}
}

// OnHook handles lifecycle hooks.
func (p *SEOPlugin) OnHook(hook plugin.Hook, ctx map[string]interface{}) error {
	switch hook {
	case plugin.AfterBuild, plugin.AfterGenerate:
		projectDir, _ := ctx["project_dir"].(string)
		if projectDir == "" {
			projectDir = "."
		}
		return p.generateSEOFiles(projectDir)
	}
	return nil
}

// Commands returns custom CLI commands.
func (p *SEOPlugin) Commands() []plugin.Command {
	return []plugin.Command{
		{
			Name:        "seo:generate",
			Alias:       "sg",
			Description: "Generate SEO files (sitemap, robots.txt)",
			Action: func(args []string) error {
				projectDir := "."
				if len(args) > 0 {
					projectDir = args[0]
				}
				return p.generateSEOFiles(projectDir)
			},
		},
		{
			Name:        "seo:meta",
			Alias:       "sm",
			Description: "Generate meta tags for a page",
			Action: func(args []string) error {
				if len(args) == 0 {
					return fmt.Errorf("page path required")
				}
				return p.generateMetaTags(args[0])
			},
		},
		{
			Name:        "seo:structured",
			Alias:       "ss",
			Description: "Generate structured data (JSON-LD)",
			Action: func(args []string) error {
				if len(args) == 0 {
					return fmt.Errorf("type required (e.g., Organization, Article, Product)")
				}
				return p.generateStructuredData(args[0])
			},
		},
	}
}

// generateSEOFiles generates all SEO files.
func (p *SEOPlugin) generateSEOFiles(projectDir string) error {
	// Load pages from routes
	pages, err := p.discoverPages(projectDir)
	if err != nil {
		fmt.Printf("Warning: could not discover pages: %v\n", err)
		pages = []PageSEO{}
	}

	// Generate sitemap.xml
	if p.config.GenerateSitemap {
		if err := p.generateSitemap(pages); err != nil {
			return fmt.Errorf("failed to generate sitemap: %w", err)
		}
		fmt.Println("Generated sitemap.xml")
	}

	// Generate robots.txt
	if p.config.GenerateRobots {
		if err := p.generateRobots(); err != nil {
			return fmt.Errorf("failed to generate robots.txt: %w", err)
		}
		fmt.Println("Generated robots.txt")
	}

	return nil
}

// discoverPages discovers pages from the routes directory.
func (p *SEOPlugin) discoverPages(projectDir string) ([]PageSEO, error) {
	routesDir := filepath.Join(projectDir, p.config.RoutesDir)
	var pages []PageSEO

	err := filepath.Walk(routesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check for page.templ files
		if strings.HasSuffix(path, "page.templ") {
			relPath := strings.TrimPrefix(path, routesDir)
			relPath = strings.TrimSuffix(relPath, "page.templ")
			relPath = strings.TrimSuffix(relPath, "/")
			if relPath == "" {
				relPath = "/"
			}

			page := PageSEO{
				Path:        relPath,
				Title:       p.config.DefaultTitle,
				Description: p.config.DefaultDescription,
				Image:       p.config.DefaultImage,
				ChangeFreq:  "weekly",
				Priority:    0.5,
				Modified:    time.Now().Format(time.RFC3339),
			}

			// Adjust priority for important pages
			if relPath == "/" {
				page.Priority = 1.0
				page.ChangeFreq = "daily"
			}

			pages = append(pages, page)
		}

		return nil
	})

	return pages, err
}

// generateSitemap generates sitemap.xml.
func (p *SEOPlugin) generateSitemap(pages []PageSEO) error {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")

	for _, page := range pages {
		if page.NoIndex {
			continue
		}

		url := p.config.SiteURL + page.Path
		sb.WriteString("  <url>\n")
		sb.WriteString(fmt.Sprintf("    <loc>%s</loc>\n", url))
		sb.WriteString(fmt.Sprintf("    <lastmod>%s</lastmod>\n", page.Modified))
		sb.WriteString(fmt.Sprintf("    <changefreq>%s</changefreq>\n", page.ChangeFreq))
		sb.WriteString(fmt.Sprintf("    <priority>%.1f</priority>\n", page.Priority))
		sb.WriteString("  </url>\n")
	}

	sb.WriteString("</urlset>\n")

	sitemapPath := filepath.Join(p.config.OutputDir, "sitemap.xml")
	return os.WriteFile(sitemapPath, []byte(sb.String()), 0644)
}

// generateRobots generates robots.txt.
func (p *SEOPlugin) generateRobots() error {
	var sb strings.Builder
	sb.WriteString("# robots.txt for " + p.config.SiteName + "\n")
	sb.WriteString("User-agent: *\n")
	sb.WriteString("Allow: /\n")
	sb.WriteString("\n")
	sb.WriteString("Sitemap: " + p.config.SiteURL + "/sitemap.xml\n")

	robotsPath := filepath.Join(p.config.OutputDir, "robots.txt")
	return os.WriteFile(robotsPath, []byte(sb.String()), 0644)
}

// generateMetaTags generates meta tags for a page.
func (p *SEOPlugin) generateMetaTags(pagePath string) error {
	page := PageSEO{
		Path:        pagePath,
		Title:       p.config.DefaultTitle,
		Description: p.config.DefaultDescription,
		Image:       p.config.DefaultImage,
	}

	var sb strings.Builder
	sb.WriteString("<!-- Meta Tags -->\n")
	sb.WriteString(fmt.Sprintf("<title>%s | %s</title>\n", html.EscapeString(page.Title), html.EscapeString(p.config.SiteName)))
	sb.WriteString(fmt.Sprintf("<meta name=\"description\" content=\"%s\">\n", html.EscapeString(page.Description)))
	sb.WriteString(fmt.Sprintf("<meta name=\"language\" content=\"%s\">\n", html.EscapeString(p.config.Language)))

	// Open Graph
	sb.WriteString("\n<!-- Open Graph -->\n")
	sb.WriteString("<meta property=\"og:type\" content=\"website\">\n")
	sb.WriteString(fmt.Sprintf("<meta property=\"og:url\" content=\"%s%s\">\n", p.config.SiteURL, page.Path)) // URLs usually don't need HTML escaping in the same way, but good practice if queried
	sb.WriteString(fmt.Sprintf("<meta property=\"og:title\" content=\"%s\">\n", html.EscapeString(page.Title)))
	sb.WriteString(fmt.Sprintf("<meta property=\"og:description\" content=\"%s\">\n", html.EscapeString(page.Description)))
	sb.WriteString(fmt.Sprintf("<meta property=\"og:site_name\" content=\"%s\">\n", html.EscapeString(p.config.SiteName)))
	if page.Image != "" {
		sb.WriteString(fmt.Sprintf("<meta property=\"og:image\" content=\"%s\">\n", html.EscapeString(page.Image)))
	}

	// Twitter Card
	sb.WriteString("\n<!-- Twitter Card -->\n")
	sb.WriteString("<meta name=\"twitter:card\" content=\"summary_large_image\">\n")
	if p.config.TwitterHandle != "" {
		sb.WriteString(fmt.Sprintf("<meta name=\"twitter:site\" content=\"%s\">\n", html.EscapeString(p.config.TwitterHandle)))
	}
	sb.WriteString(fmt.Sprintf("<meta name=\"twitter:title\" content=\"%s\">\n", html.EscapeString(page.Title)))
	sb.WriteString(fmt.Sprintf("<meta name=\"twitter:description\" content=\"%s\">\n", html.EscapeString(page.Description)))
	if page.Image != "" {
		sb.WriteString(fmt.Sprintf("<meta name=\"twitter:image\" content=\"%s\">\n", html.EscapeString(page.Image)))
	}

	// Canonical
	sb.WriteString("\n<!-- Canonical -->\n")
	sb.WriteString(fmt.Sprintf("<link rel=\"canonical\" href=\"%s%s\">\n", p.config.SiteURL, page.Path))

	fmt.Println(sb.String())
	return nil
}

// generateStructuredData generates JSON-LD structured data.
func (p *SEOPlugin) generateStructuredData(typeName string) error {
	var data map[string]interface{}

	switch typeName {
	case "Organization":
		data = map[string]interface{}{
			"@context": "https://schema.org",
			"@type":    "Organization",
			"name":     p.config.SiteName,
			"url":      p.config.SiteURL,
		}
		if p.config.DefaultImage != "" {
			data["logo"] = p.config.DefaultImage
		}

	case "WebSite":
		data = map[string]interface{}{
			"@context": "https://schema.org",
			"@type":    "WebSite",
			"name":     p.config.SiteName,
			"url":      p.config.SiteURL,
		}

	case "Article":
		data = map[string]interface{}{
			"@context":    "https://schema.org",
			"@type":       "Article",
			"headline":    p.config.DefaultTitle,
			"description": p.config.DefaultDescription,
			"author": map[string]interface{}{
				"@type": "Organization",
				"name":  p.config.SiteName,
			},
			"publisher": map[string]interface{}{
				"@type": "Organization",
				"name":  p.config.SiteName,
			},
		}

	default:
		data = map[string]interface{}{
			"@context": "https://schema.org",
			"@type":    typeName,
			"name":     p.config.SiteName,
			"url":      p.config.SiteURL,
		}
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	fmt.Printf("<script type=\"application/ld+json\">\n%s\n</script>\n", string(jsonData))
	return nil
}

// GeneratePageMeta generates meta tags for a specific page.
func (p *SEOPlugin) GeneratePageMeta(page PageSEO) string {
	var sb strings.Builder

	title := page.Title
	if title == "" {
		title = p.config.DefaultTitle
	}
	description := page.Description
	if description == "" {
		description = p.config.DefaultDescription
	}
	image := page.Image
	if image == "" {
		image = p.config.DefaultImage
	}

	// Basic meta
	sb.WriteString(fmt.Sprintf("<title>%s | %s</title>\n", html.EscapeString(title), html.EscapeString(p.config.SiteName)))
	sb.WriteString(fmt.Sprintf("<meta name=\"description\" content=\"%s\">\n", html.EscapeString(description)))

	// Robots
	if page.NoIndex || page.NoFollow {
		robots := ""
		if page.NoIndex {
			robots += "noindex"
		}
		if page.NoFollow {
			if robots != "" {
				robots += ", "
			}
			robots += "nofollow"
		}
		sb.WriteString(fmt.Sprintf("<meta name=\"robots\" content=\"%s\">\n", robots))
	}

	// Canonical
	canonical := page.Canonical
	if canonical == "" {
		canonical = p.config.SiteURL + page.Path
	}
	sb.WriteString(fmt.Sprintf("<link rel=\"canonical\" href=\"%s\">\n", canonical))

	// Open Graph
	sb.WriteString(fmt.Sprintf("<meta property=\"og:url\" content=\"%s\">\n", p.config.SiteURL+page.Path))
	sb.WriteString(fmt.Sprintf("<meta property=\"og:title\" content=\"%s\">\n", html.EscapeString(title)))
	sb.WriteString(fmt.Sprintf("<meta property=\"og:description\" content=\"%s\">\n", html.EscapeString(description)))
	if image != "" {
		sb.WriteString(fmt.Sprintf("<meta property=\"og:image\" content=\"%s\">\n", html.EscapeString(image)))
	}

	// Twitter
	sb.WriteString(fmt.Sprintf("<meta name=\"twitter:title\" content=\"%s\">\n", html.EscapeString(title)))
	sb.WriteString(fmt.Sprintf("<meta name=\"twitter:description\" content=\"%s\">\n", html.EscapeString(description)))
	if image != "" {
		sb.WriteString(fmt.Sprintf("<meta name=\"twitter:image\" content=\"%s\">\n", html.EscapeString(image)))
	}

	return sb.String()
}

// GetConfig returns the current configuration.
func (p *SEOPlugin) GetConfig() *Config {
	return p.config
}

// Ensure SEOPlugin implements CLIPlugin interface.
var _ plugin.CLIPlugin = (*SEOPlugin)(nil)
