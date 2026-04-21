package seo

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gospatempl "github.com/aydenstechdungeon/gospa/templ"
)

func TestMeta(t *testing.T) {
	params := MetaParams{
		Title:       "Test Title",
		Description: "Test Description",
		Keywords:    []string{"key1", "key2"},
		Canonical:   "https://example.com/test",
		OGImage:     "https://example.com/image.png",
	}

	component := Meta(params)
	w := httptest.NewRecorder()
	err := component.Render(context.Background(), w)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	out := w.Body.String()
	expected := []string{
		"<title>Test Title</title>",
		"<meta name=\"description\" content=\"Test Description\">",
		"<meta name=\"keywords\" content=\"key1, key2\">",
		"<link rel=\"canonical\" href=\"https://example.com/test\">",
		"<meta property=\"og:image\" content=\"https://example.com/image.png\">",
	}

	for _, e := range expected {
		if !strings.Contains(out, e) {
			t.Errorf("expected %q to be in output, but it wasn't: %s", e, out)
		}
	}
}

func TestOpenGraph(t *testing.T) {
	params := OGParams{
		Type:        "website",
		Title:       "OG Title",
		Description: "OG Description",
		URL:         "https://example.com/og",
		Image:       "https://example.com/og.png",
	}

	component := OpenGraph(params)
	w := httptest.NewRecorder()
	err := component.Render(context.Background(), w)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	out := w.Body.String()
	expected := []string{
		"<meta property=\"og:type\" content=\"website\">",
		"<meta property=\"og:title\" content=\"OG Title\">",
		"<meta property=\"og:description\" content=\"OG Description\">",
		"<meta property=\"og:url\" content=\"https://example.com/og\">",
		"<meta property=\"og:image\" content=\"https://example.com/og.png\">",
	}

	for _, e := range expected {
		if !strings.Contains(out, e) {
			t.Errorf("expected %q to be in output, but it wasn't: %s", e, out)
		}
	}
}

func TestTwitterCard(t *testing.T) {
	params := TwitterParams{
		Card:        "summary_large_image",
		Title:       "Twitter Title",
		Description: "Twitter Description",
		Image:       "https://example.com/twitter.png",
		Site:        "@test",
	}

	component := TwitterCard(params)
	w := httptest.NewRecorder()
	err := component.Render(context.Background(), w)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	out := w.Body.String()
	expected := []string{
		"<meta name=\"twitter:card\" content=\"summary_large_image\">",
		"<meta name=\"twitter:title\" content=\"Twitter Title\">",
		"<meta name=\"twitter:description\" content=\"Twitter Description\">",
		"<meta name=\"twitter:image\" content=\"https://example.com/twitter.png\">",
		"<meta name=\"twitter:site\" content=\"@test\">",
	}

	for _, e := range expected {
		if !strings.Contains(out, e) {
			t.Errorf("expected %q to be in output, but it wasn't: %s", e, out)
		}
	}
}

func TestStructuredData(t *testing.T) {
	data := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "Organization",
		"name":     "Test Org",
	}

	component := StructuredData(data)
	w := httptest.NewRecorder()
	err := component.Render(context.Background(), w)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	out := w.Body.String()
	if !strings.Contains(out, "<script type=\"application/ld+json\">") {
		t.Errorf("expected script tag, but it wasn't in output: %s", out)
	}
	if !strings.Contains(out, `"name": "Test Org"`) {
		t.Errorf("expected JSON data, but it wasn't in output: %s", out)
	}
}

func TestStructuredDataWithNonce(t *testing.T) {
	data := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "Organization",
		"name":     "Test Org",
	}

	component := StructuredData(data)
	w := httptest.NewRecorder()
	ctx := gospatempl.WithNonce(context.Background(), "test-nonce-123")
	err := component.Render(ctx, w)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	out := w.Body.String()
	if !strings.Contains(out, `<script type="application/ld+json" nonce="test-nonce-123">`) {
		t.Errorf("expected nonce on script tag, but it wasn't in output: %s", out)
	}
}

func TestGenerateSitemap(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "seo-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cfg := &Config{
		SiteURL:   "https://example.com",
		OutputDir: tmpDir,
	}
	p := New(cfg)

	pages := []PageSEO{
		{Path: "/", Modified: "2023-01-01", ChangeFreq: "daily", Priority: 1.0},
		{Path: "/about", Modified: "2023-01-01", ChangeFreq: "weekly", Priority: 0.8},
		{Path: "/hidden", NoIndex: true},
	}

	err = p.generateSitemap(pages)
	if err != nil {
		t.Fatalf("failed to generate sitemap: %v", err)
	}

	sitemapPath := filepath.Join(tmpDir, "sitemap.xml")
	// #nosec G304
	data, err := os.ReadFile(sitemapPath)
	if err != nil {
		t.Fatalf("failed to read sitemap: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "<loc>https://example.com/</loc>") {
		t.Errorf("missing home URL in sitemap")
	}
	if !strings.Contains(content, "<loc>https://example.com/about</loc>") {
		t.Errorf("missing about URL in sitemap")
	}
	if strings.Contains(content, "https://example.com/hidden") {
		t.Errorf("found hidden URL in sitemap")
	}
}

func TestGenerateRobots(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "seo-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cfg := &Config{
		SiteURL:   "https://example.com",
		SiteName:  "Test Site",
		OutputDir: tmpDir,
	}
	p := New(cfg)

	err = p.generateRobots()
	if err != nil {
		t.Fatalf("failed to generate robots: %v", err)
	}

	robotsPath := filepath.Join(tmpDir, "robots.txt")
	// #nosec G304
	data, err := os.ReadFile(robotsPath)
	if err != nil {
		t.Fatalf("failed to read robots: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Sitemap: https://example.com/sitemap.xml") {
		t.Errorf("missing sitemap URL in robots.txt")
	}
}

func TestDiscoverPages(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "seo-discover-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	routesDir := filepath.Join(tmpDir, "routes")
	err = os.MkdirAll(filepath.Join(routesDir, "about"), 0750)
	if err != nil {
		t.Fatalf("failed to create about dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(routesDir, "page.templ"), []byte(""), 0600); err != nil {
		t.Fatalf("failed to write page.templ: %v", err)
	}
	if err := os.WriteFile(filepath.Join(routesDir, "about", "page.templ"), []byte(""), 0600); err != nil {
		t.Fatalf("failed to write about/page.templ: %v", err)
	}

	cfg := &Config{
		RoutesDir: "routes",
	}
	p := New(cfg)

	pages, err := p.discoverPages(tmpDir)
	if err != nil {
		t.Fatalf("failed to discover pages: %v", err)
	}

	if len(pages) != 2 {
		t.Errorf("expected 2 pages, got %d", len(pages))
	}

	foundHome := false
	foundAbout := false
	for _, p := range pages {
		if p.Path == "/" {
			foundHome = true
		}
		if p.Path == "/about" {
			foundAbout = true
		}
	}

	if !foundHome || !foundAbout {
		t.Errorf("missing discovered pages: home=%v, about=%v", foundHome, foundAbout)
	}
}
