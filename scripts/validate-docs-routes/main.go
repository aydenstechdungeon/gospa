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

	syncFailures, err := checkGospaDocsSync(routes, "docs/gospasfc", "/docs/gospasfc")
	if err != nil {
		fmt.Fprintf(os.Stderr, "SFC docs sync check failed: %v\n", err)
		os.Exit(1)
	}
	failures = append(failures, syncFailures...)

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
		if d.IsDir() || (d.Name() != "page.templ" && d.Name() != "page.gospa") {
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

func checkGospaDocsSync(routes map[string]bool, docsRoot, routePrefix string) ([]string, error) {
	files := make(map[string]bool)
	err := filepath.WalkDir(docsRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		rel, relErr := filepath.Rel(docsRoot, path)
		if relErr != nil {
			return relErr
		}
		name := strings.TrimSuffix(filepath.ToSlash(rel), ".md")
		files[name] = true
		return nil
	})
	if err != nil {
		return nil, err
	}

	var failures []string
	for rel := range files {
		expectedRoute := routePrefix
		if rel != "." && rel != "README" {
			expectedRoute = routePrefix + "/" + rel
		}
		if !routes[expectedRoute] {
			failures = append(failures, fmt.Sprintf("missing website docs route for %s (%s)", rel, expectedRoute))
		}
	}

	for route := range routes {
		if route != routePrefix && !strings.HasPrefix(route, routePrefix+"/") {
			continue
		}
		rel := strings.TrimPrefix(route, routePrefix)
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			continue
		}
		if !files[rel] {
			failures = append(failures, fmt.Sprintf("missing markdown doc for route %s (expected %s.md)", route, rel))
		}
	}

	return failures, nil
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
