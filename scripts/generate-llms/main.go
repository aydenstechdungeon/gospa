//go:build ignore

// Package main provides a script to generate LLM-friendly documentation assets.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileInfo holds information about a documentation file
type FileInfo struct {
	Path     string
	Size     int64
	Category string
	Order    int
}

// Category info for organizing docs
type Category struct {
	Name   string
	Order  int
	Prefix string
}

var categories = []Category{
	{"Getting Started", 1, "getstarted"},
	{"SFC Components", 2, "gospasfc"},
	{"Routing", 3, "routing"},
	{"State Management", 4, "state-management"},
	{"Client Runtime", 5, "client-runtime"},
	{"Configuration", 6, "configuration"},
	{"Plugins", 7, "plugins"},
	{"Reactive Primitives", 8, "reactive-primitives"},
	{"API Reference", 9, "api"},
	{"Troubleshooting", 10, "troubleshooting"},
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run scripts/generate-llms.go <output-dir>")
		fmt.Println("Example: go run scripts/generate-llms.go docs/llms")
		os.Exit(1)
	}

	outputDir := os.Args[1]
	docsDir := "docs"

	// Collect all markdown files
	files, err := collectDocs(docsDir)
	if err != nil {
		fmt.Printf("Error collecting docs: %v\n", err)
		os.Exit(1)
	}

	// Generate llms.txt
	llmsTxt := generateLLMsTxt(files)
	txtPath := filepath.Join(outputDir, "llms.txt")
	if err := os.WriteFile(txtPath, []byte(llmsTxt), 0644); err != nil {
		fmt.Printf("Error writing llms.txt: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated: %s\n", txtPath)

	// Generate llms-full.md
	llmsFull, err := generateLLMsFull(files)
	if err != nil {
		fmt.Printf("Error generating llms-full.md: %v\n", err)
		os.Exit(1)
	}
	fullPath := filepath.Join(outputDir, "llms-full.md")
	if err := os.WriteFile(fullPath, []byte(llmsFull), 0644); err != nil {
		fmt.Printf("Error writing llms-full.md: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated: %s\n", fullPath)
}

func collectDocs(docsDir string) ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		// Skip the llms directory
		if strings.Contains(path, "llms/") {
			return nil
		}

		relPath, err := filepath.Rel(docsDir, path)
		if err != nil {
			return err
		}

		// Determine category and order
		category := "Other"
		order := 99
		for _, cat := range categories {
			if strings.HasPrefix(relPath, cat.Prefix) ||
				strings.Contains(relPath, "/"+cat.Prefix+"/") {
				category = cat.Name
				order = cat.Order
				break
			}
		}

		// Extract file order from filename (e.g., "01-quick-start.md" -> 1)
		fileOrder := 99
		baseName := filepath.Base(path)
		if len(baseName) >= 3 && baseName[0] >= '0' && baseName[0] <= '9' {
			fileOrder = int(baseName[0]-'0')*10 + int(baseName[1]-'0')
		}

		files = append(files, FileInfo{
			Path:     relPath,
			Size:     info.Size(),
			Category: category,
			Order:    order*100 + fileOrder,
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort files by order
	sort.Slice(files, func(i, j int) bool {
		return files[i].Order < files[j].Order
	})

	return files, nil
}

func generateLLMsTxt(files []FileInfo) string {
	var sb strings.Builder

	sb.WriteString("# GoSPA Documentation Index for LLMs\n")
	sb.WriteString("# Repository: https://github.com/aydenstechdungeon/gospa/\n")
	sb.WriteString("# Official website: https://gospa.onrender.com/\n")
	sb.WriteString("# Website's Docs: https://gospa.onrender.com/docs\n")
	sb.WriteString("# Framework: Go-based SPA with Svelte-like reactivity, Templ SSR, TypeScript client runtime\n\n")

	sb.WriteString("## Documentation Files\n\n")

	currentCategory := ""
	for _, file := range files {
		if file.Category != currentCategory {
			sb.WriteString(fmt.Sprintf("\n### %s\n\n", file.Category))
			currentCategory = file.Category
		}

		displayName := strings.TrimSuffix(file.Path, ".md")
		sb.WriteString(fmt.Sprintf("[%s](docs/%s)\n", filepath.Base(displayName), file.Path))
		sb.WriteString(fmt.Sprintf("  - Size: %d bytes\n\n", file.Size))
	}

	sb.WriteString("\n## File Summary\n\n")
	sb.WriteString(fmt.Sprintf("Total documentation files: %d\n", len(files)))

	return sb.String()
}

func generateLLMsFull(files []FileInfo) (string, error) {
	var sb strings.Builder

	sb.WriteString("# Repository: https://github.com/aydenstechdungeon/gospa/\n")
	sb.WriteString("# Official website: https://gospa.onrender.com/\n")
	sb.WriteString("# Website's Docs: https://gospa.onrender.com/docs\n\n")

	// First include the main README
	readmeContent, err := os.ReadFile("README.md")
	if err == nil {
		sb.WriteString("<!-- FILE: README.md -->\n")
		sb.WriteString("================================================================================\n\n")
		sb.WriteString(string(readmeContent))
		sb.WriteString("\n\n")
	}

	// Include docs README if it exists and is different
	docsReadme, err := os.ReadFile("docs/README.md")
	if err == nil && string(docsReadme) != string(readmeContent) {
		sb.WriteString("<!-- FILE: docs/README.md -->\n")
		sb.WriteString("================================================================================\n\n")
		sb.WriteString(string(docsReadme))
		sb.WriteString("\n\n")
	}

	// Include all documentation files
	for _, file := range files {
		content, err := os.ReadFile(filepath.Join("docs", file.Path))
		if err != nil {
			return "", fmt.Errorf("error reading %s: %w", file.Path, err)
		}

		sb.WriteString(fmt.Sprintf("<!-- FILE: docs/%s -->\n", file.Path))
		sb.WriteString("================================================================================\n\n")
		sb.WriteString(string(content))
		sb.WriteString("\n\n")
	}

	return sb.String(), nil
}
