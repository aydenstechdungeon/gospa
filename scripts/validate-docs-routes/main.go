// Package main validates docs route and search index integrity.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	json "github.com/goccy/go-json"
)

type searchEntry struct {
	URL string `json:"url"`
}

func main() {
	routes, err := collectDocsRoutes("website/routes/docs")
	if err != nil {
		fmt.Fprintf(os.Stderr, "route scan failed: %v\n", err)
		os.Exit(1)
	}

	entries, err := loadSearchEntries("website/static/docs_search_index.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "search index read failed: %v\n", err)
		os.Exit(1)
	}

	var failures []string
	for _, e := range entries {
		if e.URL == "" {
			failures = append(failures, "empty URL entry in search index")
			continue
		}
		if strings.Contains(e.URL, ".templ") {
			failures = append(failures, fmt.Sprintf("templ artifact URL in index: %s", e.URL))
			continue
		}
		if !routes[e.URL] {
			failures = append(failures, fmt.Sprintf("index URL has no route: %s", e.URL))
		}
	}

	if len(failures) > 0 {
		fmt.Fprintln(os.Stderr, "docs route validation failed:")
		for _, f := range failures {
			fmt.Fprintf(os.Stderr, "- %s\n", f)
		}
		os.Exit(1)
	}

	fmt.Printf("validated %d docs routes and %d search entries\n", len(routes), len(entries))
}

func collectDocsRoutes(root string) (map[string]bool, error) {
	routes := map[string]bool{"/docs": true}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "page.templ" {
			return nil
		}

		relDir, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}

		if relDir == "." {
			routes["/docs"] = true
			return nil
		}

		url := "/docs/" + filepath.ToSlash(relDir)
		routes[url] = true
		return nil
	})

	return routes, err
}

func loadSearchEntries(path string) ([]searchEntry, error) {
	cleanPath := filepath.Clean(path)
	if filepath.IsAbs(cleanPath) || strings.HasPrefix(cleanPath, "..") {
		return nil, fmt.Errorf("invalid search index path %q", path)
	}

	data, err := os.ReadFile(cleanPath) // #nosec G304 -- path constrained to project-relative safe path
	if err != nil {
		return nil, err
	}

	var entries []searchEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	return entries, nil
}
