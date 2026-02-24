package search

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type DocPage struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Content     string    `json:"content"`
	Sections    []Section `json:"sections"`
}

type Section struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

var (
	titleRegex       = regexp.MustCompile(`<h1[^>]*>(.*?)</h1>`)
	descriptionRegex = regexp.MustCompile(`<p[^>]*class="text-xl[^>]*>(.*?)</p>`)
	headingRegex     = regexp.MustCompile(`<h[23][^>]*id="(.*?)"[^>]*>(.*?)</h[23]>`)
)

func GenerateIndex(routesDir string, outputDir string) error {
	var index []DocPage

	err := filepath.WalkDir(routesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".templ") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		sContent := string(content)

		// Map file path to URL
		relPath, _ := filepath.Rel(routesDir, path)
		urlPath := strings.TrimSuffix(relPath, ".templ")
		urlPath = strings.TrimSuffix(urlPath, "page")
		urlPath = strings.TrimSuffix(urlPath, "/")

		if urlPath == "" {
			urlPath = "/docs"
		} else {
			urlPath = "/docs/" + urlPath
		}
		urlPath = strings.TrimSuffix(urlPath, "/")
		if urlPath == "" {
			urlPath = "/docs"
		}

		titleMatch := titleRegex.FindStringSubmatch(sContent)
		title := "Documentation"
		if len(titleMatch) > 1 {
			title = stripTags(titleMatch[1])
		}

		descMatch := descriptionRegex.FindStringSubmatch(sContent)
		description := ""
		if len(descMatch) > 1 {
			description = stripTags(descMatch[1])
		}

		// Extract sections
		var sections []Section
		headingMatches := headingRegex.FindAllStringSubmatch(sContent, -1)
		for _, m := range headingMatches {
			if len(m) > 2 {
				sections = append(sections, Section{
					ID:    m[1],
					Title: stripTags(m[2]),
					Text:  "", // Optional: Extract text after heading
				})
			}
		}

		index = append(index, DocPage{
			Title:       title,
			Description: description,
			URL:         urlPath,
			Content:     stripTags(sContent), // Simple strip for now
			Sections:    sections,
		})

		return nil
	})

	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(outputDir, "docs_search_index.json"), data, 0644)
}

func stripTags(s string) string {
	// Simple regex to strip HTML tags and Templ calls
	re := regexp.MustCompile(`<[^>]*>|@components\.[A-Za-z0-9]+\(.*\)|templ\.[A-Za-z0-9]+|{[ "]*"}|{ "on" }`)
	s = re.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")

	// Remove extra spaces
	reSpace := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(reSpace.ReplaceAllString(s, " "))
}
