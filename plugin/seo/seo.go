// Package seo provides SEO optimization for GoSPA projects.
// Includes meta tags, sitemap generation, structured data (JSON-LD), and Open Graph.
package seo

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/aydenstechdungeon/gospa/plugin"
)

// MetaParams represents parameters for the Meta component.
type MetaParams struct {
	Title       string
	Description string
	Keywords    []string
	Canonical   string
	OGImage     string
}

// OGParams represents parameters for the OpenGraph component.
type OGParams struct {
	Type        string
	Title       string
	Description string
	Image       string
	URL         string
}

// TwitterParams represents parameters for the TwitterCard component.
type TwitterParams struct {
	Card        string
	Title       string
	Description string
	Image       string
	Site        string
}

// Organization represents JSON-LD Organization data.
type Organization struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Logo string `json:"logo"`
}

// Plugin provides SEO optimization capabilities.
type Plugin struct {
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

// MetaConfig represents SEO metadata for a page.
type MetaConfig struct {
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

// PageSEO is an alias for MetaConfig.
type PageSEO = MetaConfig

// ArticleData represents article-specific metadata for JSON-LD.
type ArticleData struct {
	Headline      string `json:"headline"`
	Author        string `json:"author"`
	DatePublished string `json:"datePublished"`
	Image         string `json:"image"`
}

// RawStructuredData represents JSON-LD structured data.
type RawStructuredData struct {
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

// Meta returns a component with meta tags.
func Meta(p MetaParams) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		var sb strings.Builder
		fmt.Fprintf(&sb, "<title>%s</title>\n", html.EscapeString(p.Title))
		fmt.Fprintf(&sb, "<meta name=\"description\" content=\"%s\">\n", html.EscapeString(p.Description))
		if len(p.Keywords) > 0 {
			fmt.Fprintf(&sb, "<meta name=\"keywords\" content=\"%s\">\n", html.EscapeString(strings.Join(p.Keywords, ", ")))
		}
		if p.Canonical != "" {
			fmt.Fprintf(&sb, "<link rel=\"canonical\" href=\"%s\">\n", p.Canonical)
		}
		if p.OGImage != "" {
			fmt.Fprintf(&sb, "<meta property=\"og:image\" content=\"%s\">\n", html.EscapeString(p.OGImage))
		}
		_, err := w.Write([]byte(sb.String()))
		return err
	})
}

// OpenGraph returns a component with Open Graph tags.
func OpenGraph(p OGParams) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		var sb strings.Builder
		fmt.Fprintf(&sb, "<meta property=\"og:type\" content=\"%s\">\n", html.EscapeString(p.Type))
		fmt.Fprintf(&sb, "<meta property=\"og:title\" content=\"%s\">\n", html.EscapeString(p.Title))
		fmt.Fprintf(&sb, "<meta property=\"og:description\" content=\"%s\">\n", html.EscapeString(p.Description))
		fmt.Fprintf(&sb, "<meta property=\"og:url\" content=\"%s\">\n", p.URL)
		if p.Image != "" {
			fmt.Fprintf(&sb, "<meta property=\"og:image\" content=\"%s\">\n", html.EscapeString(p.Image))
		}
		_, err := w.Write([]byte(sb.String()))
		return err
	})
}

// TwitterCard returns a component with Twitter Card tags.
func TwitterCard(p TwitterParams) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		var sb strings.Builder
		fmt.Fprintf(&sb, "<meta name=\"twitter:card\" content=\"%s\">\n", html.EscapeString(p.Card))
		fmt.Fprintf(&sb, "<meta name=\"twitter:title\" content=\"%s\">\n", html.EscapeString(p.Title))
		fmt.Fprintf(&sb, "<meta name=\"twitter:description\" content=\"%s\">\n", html.EscapeString(p.Description))
		if p.Image != "" {
			fmt.Fprintf(&sb, "<meta name=\"twitter:image\" content=\"%s\">\n", html.EscapeString(p.Image))
		}
		if p.Site != "" {
			fmt.Fprintf(&sb, "<meta name=\"twitter:site\" content=\"%s\">\n", html.EscapeString(p.Site))
		}
		_, err := w.Write([]byte(sb.String()))
		return err
	})
}

var defaultPlugin = New(DefaultConfig())

// StructuredData generates JSON-LD structured data.
func StructuredData(data any) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "<script type=\"application/ld+json\">\n%s\n</script>\n", string(jsonData))
		return err
	})
}

// MetaTags generates meta tags using the default plugin.
func MetaTags(config MetaConfig) string {
	return defaultPlugin.GeneratePageMeta(config)
}

// New creates a new SEO plugin.
func New(cfg *Config) *Plugin {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	p := &Plugin{config: cfg}
	return p
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "seo"
}

// Init initializes the SEO plugin.
func (p *Plugin) Init() error {
	// Create output directory
	if err := os.MkdirAll(p.config.OutputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	return nil
}

// Dependencies returns required dependencies.
func (p *Plugin) Dependencies() []plugin.Dependency {
	return []plugin.Dependency{
		// No external dependencies required
	}
}

// OnHook handles lifecycle hooks.
func (p *Plugin) OnHook(hook plugin.Hook, ctx map[string]interface{}) error {
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
func (p *Plugin) Commands() []plugin.Command {
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
func (p *Plugin) generateSEOFiles(projectDir string) error {
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
func (p *Plugin) discoverPages(projectDir string) ([]PageSEO, error) {
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
			relPath := strings.TrimPrefix(filepath.ToSlash(path), filepath.ToSlash(routesDir))
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
func (p *Plugin) generateSitemap(pages []PageSEO) error {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")

	for _, page := range pages {
		if page.NoIndex {
			continue
		}

		url := p.config.SiteURL + page.Path
		sb.WriteString("  <url>\n")
		fmt.Fprintf(&sb, "    <loc>%s</loc>\n", url)
		fmt.Fprintf(&sb, "    <lastmod>%s</lastmod>\n", page.Modified)
		fmt.Fprintf(&sb, "    <changefreq>%s</changefreq>\n", page.ChangeFreq)
		fmt.Fprintf(&sb, "    <priority>%.1f</priority>\n", page.Priority)
		sb.WriteString("  </url>\n")
	}

	sb.WriteString("</urlset>\n")

	sitemapPath := filepath.Join(p.config.OutputDir, "sitemap.xml")
	return os.WriteFile(sitemapPath, []byte(sb.String()), 0600)
}

// generateRobots generates robots.txt.
func (p *Plugin) generateRobots() error {
	var sb strings.Builder
	sb.WriteString("# robots.txt for " + p.config.SiteName + "\n")
	sb.WriteString("User-agent: *\n")
	sb.WriteString("Allow: /\n")
	sb.WriteString("\n")
	sb.WriteString("Sitemap: " + p.config.SiteURL + "/sitemap.xml\n")

	robotsPath := filepath.Join(p.config.OutputDir, "robots.txt")
	return os.WriteFile(robotsPath, []byte(sb.String()), 0600)
}

// generateMetaTags generates meta tags for a page.
func (p *Plugin) generateMetaTags(pagePath string) error {
	page := PageSEO{
		Path:        pagePath,
		Title:       p.config.DefaultTitle,
		Description: p.config.DefaultDescription,
		Image:       p.config.DefaultImage,
	}

	var sb strings.Builder
	sb.WriteString("<!-- Meta Tags -->\n")
	fmt.Fprintf(&sb, "<title>%s | %s</title>\n", html.EscapeString(page.Title), html.EscapeString(p.config.SiteName))
	fmt.Fprintf(&sb, "<meta name=\"description\" content=\"%s\">\n", html.EscapeString(page.Description))
	fmt.Fprintf(&sb, "<meta name=\"language\" content=\"%s\">\n", html.EscapeString(p.config.Language))

	// Open Graph
	sb.WriteString("\n<!-- Open Graph -->\n")
	sb.WriteString("<meta property=\"og:type\" content=\"website\">\n")
	fmt.Fprintf(&sb, "<meta property=\"og:url\" content=\"%s%s\">\n", p.config.SiteURL, page.Path) // URLs usually don't need HTML escaping in the same way, but good practice if queried
	fmt.Fprintf(&sb, "<meta property=\"og:title\" content=\"%s\">\n", html.EscapeString(page.Title))
	fmt.Fprintf(&sb, "<meta property=\"og:description\" content=\"%s\">\n", html.EscapeString(page.Description))
	fmt.Fprintf(&sb, "<meta property=\"og:site_name\" content=\"%s\">\n", html.EscapeString(p.config.SiteName))
	if page.Image != "" {
		fmt.Fprintf(&sb, "<meta property=\"og:image\" content=\"%s\">\n", html.EscapeString(page.Image))
	}

	// Twitter Card
	sb.WriteString("\n<!-- Twitter Card -->\n")
	sb.WriteString("<meta name=\"twitter:card\" content=\"summary_large_image\">\n")
	if p.config.TwitterHandle != "" {
		fmt.Fprintf(&sb, "<meta name=\"twitter:site\" content=\"%s\">\n", html.EscapeString(p.config.TwitterHandle))
	}
	fmt.Fprintf(&sb, "<meta name=\"twitter:title\" content=\"%s\">\n", html.EscapeString(page.Title))
	fmt.Fprintf(&sb, "<meta name=\"twitter:description\" content=\"%s\">\n", html.EscapeString(page.Description))
	if page.Image != "" {
		fmt.Fprintf(&sb, "<meta name=\"twitter:image\" content=\"%s\">\n", html.EscapeString(page.Image))
	}

	// Canonical
	sb.WriteString("\n<!-- Canonical -->\n")
	fmt.Fprintf(&sb, "<link rel=\"canonical\" href=\"%s%s\">\n", p.config.SiteURL, page.Path)

	fmt.Println(sb.String())
	return nil
}

// generateStructuredData generates JSON-LD structured data.
func (p *Plugin) generateStructuredData(typeName string) error {
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
func (p *Plugin) GeneratePageMeta(page PageSEO) string {
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
	fmt.Fprintf(&sb, "<title>%s | %s</title>\n", html.EscapeString(title), html.EscapeString(p.config.SiteName))
	fmt.Fprintf(&sb, "<meta name=\"description\" content=\"%s\">\n", html.EscapeString(description))

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
		fmt.Fprintf(&sb, "<meta name=\"robots\" content=\"%s\">\n", robots)
	}

	// Canonical
	canonical := page.Canonical
	if canonical == "" {
		canonical = p.config.SiteURL + page.Path
	}
	fmt.Fprintf(&sb, "<link rel=\"canonical\" href=\"%s\">\n", canonical)

	// Open Graph
	fmt.Fprintf(&sb, "<meta property=\"og:url\" content=\"%s\">\n", p.config.SiteURL+page.Path)
	fmt.Fprintf(&sb, "<meta property=\"og:title\" content=\"%s\">\n", html.EscapeString(title))
	fmt.Fprintf(&sb, "<meta property=\"og:description\" content=\"%s\">\n", html.EscapeString(description))
	if image != "" {
		fmt.Fprintf(&sb, "<meta property=\"og:image\" content=\"%s\">\n", html.EscapeString(image))
	}

	// Twitter
	fmt.Fprintf(&sb, "<meta name=\"twitter:title\" content=\"%s\">\n", html.EscapeString(title))
	fmt.Fprintf(&sb, "<meta name=\"twitter:description\" content=\"%s\">\n", html.EscapeString(description))
	if image != "" {
		fmt.Fprintf(&sb, "<meta name=\"twitter:image\" content=\"%s\">\n", html.EscapeString(image))
	}

	return sb.String()
}

// GetConfig returns the current configuration.
func (p *Plugin) GetConfig() *Config {
	return p.config
}

// Ensure Plugin implements CLIPlugin interface.
var _ plugin.CLIPlugin = (*Plugin)(nil)
