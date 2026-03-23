// Package sfc provides a parser for GoSPA Single File Components (.gospa).
package sfc

import (
	"fmt"
	"regexp"
	"strings"
)

// Block represents a section of a .gospa file.
type Block struct {
	Type    string // "script", "template", "style"
	Lang    string // e.g., "go", "ts", "css"
	Content string
}

// SFC represents the parsed structure of a .gospa file.
type SFC struct {
	Script   Block
	Template Block
	Style    Block
}

var (
	scriptRegex   = regexp.MustCompile(`(?s)<script(.*?)>(.*?)</script>`)
	templateRegex = regexp.MustCompile(`(?s)<template(.*?)>(.*?)</template>`)
	styleRegex    = regexp.MustCompile(`(?s)<style(.*?)>(.*?)</style>`)
	langRegex     = regexp.MustCompile(`lang="([^"]*)"`)
)

// Parse splits a .gospa file into its component blocks.
func Parse(input string) (*SFC, error) {
	sfc := &SFC{}

	// Extract Script
	if matches := scriptRegex.FindStringSubmatch(input); len(matches) > 2 {
		sfc.Script = Block{
			Type:    "script",
			Lang:    extractLang(matches[1], "go"),
			Content: strings.TrimSpace(matches[2]),
		}
	}

	// Extract Template
	if matches := templateRegex.FindStringSubmatch(input); len(matches) > 2 {
		sfc.Template = Block{
			Type:    "template",
			Content: strings.TrimSpace(matches[2]),
		}
	}

	// Extract Style
	if matches := styleRegex.FindStringSubmatch(input); len(matches) > 2 {
		sfc.Style = Block{
			Type:    "style",
			Lang:    extractLang(matches[1], "css"),
			Content: strings.TrimSpace(matches[2]),
		}
	}

	if sfc.Template.Content == "" {
		return nil, fmt.Errorf("missing <template> block")
	}

	return sfc, nil
}

func extractLang(attr, defaultLang string) string {
	if matches := langRegex.FindStringSubmatch(attr); len(matches) > 1 {
		return matches[1]
	}
	return defaultLang
}
