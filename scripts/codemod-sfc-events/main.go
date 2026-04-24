// Package main rewrites legacy SFC event attribute syntaxes to `on:<event>=`.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	write := flag.Bool("write", false, "Rewrite files in place")
	flag.Parse()

	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"routes", "components"}
	}

	total := 0
	changed := 0
	for _, root := range paths {
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}
		rootFS, err := os.OpenRoot(root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open root error for %s: %v\n", root, err)
			continue
		}
		err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				name := info.Name()
				if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" || name == "generated" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".gospa") {
				return nil
			}

			total++
			relPath, err := filepath.Rel(root, path)
			if err != nil {
				return nil
			}
			before, err := rootFS.ReadFile(relPath)
			if err != nil {
				return nil
			}
			after := migrateEventSyntax(before)
			if bytes.Equal(before, after) {
				return nil
			}
			changed++
			if *write {
				if err := rootFS.WriteFile(relPath, after, 0600); err != nil {
					return err
				}
				fmt.Printf("updated %s\n", path)
			} else {
				fmt.Printf("would update %s\n", path)
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "scan error for %s: %v\n", root, err)
		}
		if closeErr := rootFS.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "close root error for %s: %v\n", root, closeErr)
		}
	}

	mode := "dry-run"
	if *write {
		mode = "write"
	}
	fmt.Printf("codemod complete (%s): scanned=%d changed=%d\n", mode, total, changed)
}

func migrateEventSyntax(input []byte) []byte {
	// Keep codemod intentionally conservative: transform attribute syntax only.
	atEvent := regexp.MustCompile(`(^|[\s<])@([a-zA-Z][a-zA-Z0-9:_-]*)\s*=`)
	onDashEvent := regexp.MustCompile(`(^|[\s<])on-([a-zA-Z][a-zA-Z0-9:_-]*)\s*=`)
	xOnEvent := regexp.MustCompile(`(^|[\s<])x-on:([a-zA-Z][a-zA-Z0-9:_-]*)\s*=`)

	scanner := bufio.NewScanner(bytes.NewReader(input))
	var out strings.Builder
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		line = atEvent.ReplaceAllString(line, "${1}on:${2}=")
		line = onDashEvent.ReplaceAllString(line, "${1}on:${2}=")
		line = xOnEvent.ReplaceAllString(line, "${1}on:${2}=")
		if !first {
			out.WriteByte('\n')
		}
		first = false
		out.WriteString(line)
	}
	if err := scanner.Err(); err != nil {
		return input
	}
	if len(input) > 0 && input[len(input)-1] == '\n' {
		out.WriteByte('\n')
	}
	return []byte(out.String())
}
