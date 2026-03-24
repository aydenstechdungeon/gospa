// Package main provides a tool to generate the documentation search index.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SearchIndexEntry represents a single searchable document
type SearchIndexEntry struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Content     string    `json:"content"`
	Section     string    `json:"section,omitempty"`
	Sections    []Section `json:"sections,omitempty"`
}

// Section represents a section within a page
type Section struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

func main() {
	if err := generateSearchIndex(); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating search index: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Search index generated successfully")
}

func generateSearchIndex() error {
	entries := []SearchIndexEntry{}

	// Walk through routes/docs directory
	err := filepath.Walk("routes/docs", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-page.templ files
		if info.IsDir() || info.Name() != "page.templ" {
			return nil
		}

		path = filepath.Clean(path)
		content, err := os.ReadFile(path) // #nosec G122 - Build script, symlink TOCTOU not a risk for this project structure.
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		entry := parsePageTempl(path, string(content))
		if entry != nil {
			entries = append(entries, *entry)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	// Write to file
	outputPath := "static/docs_search_index.json"
	if err := os.WriteFile(outputPath, jsonData, 0600); err != nil {
		return fmt.Errorf("writing %s: %w", outputPath, err)
	}

	fmt.Printf("Generated %d entries\n", len(entries))
	return nil
}

func parsePageTempl(path string, content string) *SearchIndexEntry {
	// Extract URL from path
	url := pathToURL(path)

	// Extract title from h1 or Page() template
	title := extractTitle(content)
	if title == "" {
		title = filepath.Base(filepath.Dir(path))
	}

	// Extract description
	description := extractDescription(content)

	// Clean content for search
	cleanContent := cleanTemplContent(content)

	// Extract sections
	sections := extractSections(content)

	return &SearchIndexEntry{
		Title:       title,
		Description: description,
		URL:         url,
		Content:     cleanContent,
		Sections:    sections,
	}
}

func pathToURL(path string) string {
	// Convert routes/docs/path/page.templ to /docs/path
	path = strings.TrimPrefix(path, "routes/docs/")
	path = strings.TrimSuffix(path, "/page.templ")

	if path == "" {
		return "/docs"
	}

	return "/docs/" + path
}

func extractTitle(content string) string {
	// Try to find <h1> tag
	h1Regex := regexp.MustCompile(`<h1[^>]*>(.*?)</h1>`)
	if match := h1Regex.FindStringSubmatch(content); match != nil {
		return stripHTML(match[1])
	}

	// Try to find h1 class with title
	h1ClassRegex := regexp.MustCompile(`class="[^"]*text-[^"]*"[^>]*>(.*?)</`)
	if match := h1ClassRegex.FindStringSubmatch(content); match != nil {
		return stripHTML(match[1])
	}

	// Try to find Page() call with string literal
	pageRegex := regexp.MustCompile(`Page\(\)[^{]*\{[^}]*"([^"]+)"`)
	if match := pageRegex.FindStringSubmatch(content); match != nil {
		return match[1]
	}

	return ""
}

func extractDescription(content string) string {
	// Look for description paragraph after h1
	descRegex := regexp.MustCompile(`<p class="text-xl[^"]*"[^>]*>(.*?)</p>`)
	if match := descRegex.FindStringSubmatch(content); match != nil {
		return stripHTML(match[1])
	}

	// Try to find first paragraph
	pRegex := regexp.MustCompile(`<p[^>]*>(.*?)</p>`)
	if match := pRegex.FindStringSubmatch(content); match != nil {
		text := stripHTML(match[1])
		if len(text) > 200 {
			return text[:200] + "..."
		}
		return text
	}

	return ""
}

func extractSections(content string) []Section {
	sections := []Section{}

	// Find all h2 elements with id
	h2Regex := regexp.MustCompile(`<h2[^>]*id="([^"]+)"[^>]*>(.*?)</h2>`)
	matches := h2Regex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			sections = append(sections, Section{
				ID:    match[1],
				Title: stripHTML(match[2]),
				Text:  "",
			})
		}
	}

	return sections
}

func cleanTemplContent(content string) string {
	// Remove templ code
	content = regexp.MustCompile(`package\s+\w+`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`import\s*\([^)]*\)`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`import\s+"[^"]+"`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`templ\s+\w+\([^)]*\)\s*\{`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`@\w+\([^)]*\)`).ReplaceAllString(content, "")

	// Remove Go code blocks from content
	content = regexp.MustCompile(`@components\.CodeBlock\([^)]+\)`).ReplaceAllString(content, "")

	// Remove HTML tags but keep content
	content = stripHTML(content)

	// Clean up whitespace
	content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")

	return strings.TrimSpace(content)
}

func stripHTML(html string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]+>`)
	text := re.ReplaceAllString(html, " ")

	// Clean up entities
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", `"`)
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")

	// Clean up whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}
