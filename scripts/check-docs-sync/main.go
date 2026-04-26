// Package main provides a script to audit documentation consistency between website routes and authoritative markdown files.
package main

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

func main() {
	docsDir := "./docs"
	websiteDocsDir := "./website/routes/docs"

	fmt.Println("=== Documentation Consistency Audit ===")

	// Map of expected docs based on website routes
	websiteRoutes := make(map[string]string)
	err := filepath.WalkDir(websiteDocsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && (d.Name() == "page.templ" || d.Name() == "page.gospa") {
			rel, _ := filepath.Rel(websiteDocsDir, filepath.Dir(path))
			if rel == "." {
				rel = "root"
			}
			websiteRoutes[rel] = path
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error walking website docs: %v\n", err)
	}

	// Map of authoritative docs
	authDocs := make(map[string]string)
	err = filepath.WalkDir(docsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			rel, _ := filepath.Rel(docsDir, path)
			authDocs[rel] = path
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error walking authoritative docs: %v\n", err)
	}

	fmt.Printf("\nFound %d Website doc pages\n", len(websiteRoutes))
	fmt.Printf("Found %d Authoritative Markdown docs\n", len(authDocs))

	fmt.Println("\n--- Missing Authoritative Docs (Website routes with no MD) ---")
	for r := range websiteRoutes {
		found := false
		for d := range authDocs {
			if strings.TrimSuffix(d, ".md") == r {
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("  - %s (Expected: %s.md)\n", r, r)
		}
	}

	fmt.Println("\n--- Extra Authoritative Docs (MD files with no Website route) ---")
	for d := range authDocs {
		found := false
		for r := range websiteRoutes {
			if strings.TrimSuffix(d, ".md") == r {
				found = true
				break
			}
		}
		if !found && d != "README.md" && !strings.HasPrefix(d, "llms/") {
			fmt.Printf("  - %s\n", d)
		}
	}
}
