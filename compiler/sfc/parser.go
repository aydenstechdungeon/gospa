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
	ScriptTS Block
	Template Block
	Style    Block
}

var (
	scriptRegex   = regexp.MustCompile(`(?ism)^<script(.*?)>(.*?)^</script>`)
	templateRegex = regexp.MustCompile(`(?ism)^<template(.*?)>(.*?)^</template>`)
	styleRegex    = regexp.MustCompile(`(?ism)^<style(.*?)>(.*?)^</style>`)
	langRegex     = regexp.MustCompile(`(?i)lang="([^"]*)"`)
)

// Parse splits a .gospa file into its component blocks.
func Parse(input string) (*SFC, error) {
	sfc := &SFC{}

	// 1. Extract Script Blocks
	for _, matches := range scriptRegex.FindAllStringSubmatch(input, -1) {
		if len(matches) <= 2 {
			continue
		}
		lang := normalizeScriptLang(extractLang(matches[1], "go"))
		block := Block{
			Type:    "script",
			Lang:    lang,
			Content: strings.TrimSpace(matches[2]),
		}
		switch lang {
		case "go":
			if sfc.Script.Content != "" {
				return nil, fmt.Errorf("multiple <script lang=\"go\"> blocks are not supported")
			}
			sfc.Script = block
		case "ts":
			if sfc.ScriptTS.Content != "" {
				return nil, fmt.Errorf("multiple <script lang=\"ts\"> (or js/typescript/javascript) blocks are not supported")
			}
			sfc.ScriptTS = block
		default:
			return nil, fmt.Errorf("unsupported <script> language %q: supported languages are go, ts, js, typescript, javascript", lang)
		}
	}

	// 2. Extract Style Block
	if matches := styleRegex.FindStringSubmatch(input); len(matches) > 2 {
		sfc.Style = Block{
			Type:    "style",
			Lang:    extractLang(matches[1], "css"),
			Content: strings.TrimSpace(matches[2]),
		}
	}

	// 3. Extract Template Block (Explicit or Implicit)
	templateMatches := templateRegex.FindStringSubmatch(input)
	if len(templateMatches) > 2 {
		if countMatches(templateRegex, input) > 1 {
			return nil, fmt.Errorf("multiple <template> blocks are not supported")
		}
		sfc.Template = Block{
			Type:    "template",
			Content: strings.TrimSpace(templateMatches[2]),
		}
	} else {
		// Implicit template: everything that isn't script or style
		temp := input
		temp = scriptRegex.ReplaceAllString(temp, "")
		temp = styleRegex.ReplaceAllString(temp, "")
		sfc.Template = Block{
			Type:    "template",
			Content: strings.TrimSpace(temp),
		}
	}

	if sfc.Template.Content == "" {
		return nil, fmt.Errorf("missing template content")
	}

	return sfc, nil
}

func extractLang(attr, defaultLang string) string {
	if matches := langRegex.FindStringSubmatch(attr); len(matches) > 1 {
		return matches[1]
	}
	return defaultLang
}

func countMatches(re *regexp.Regexp, input string) int {
	return len(re.FindAllStringSubmatch(input, -1))
}

func normalizeScriptLang(lang string) string {
	l := strings.ToLower(strings.TrimSpace(lang))
	switch l {
	case "", "go", "golang":
		return "go"
	case "ts", "typescript", "js", "javascript":
		return "ts"
	default:
		return l
	}
}
