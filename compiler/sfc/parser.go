// Package sfc provides a parser for GoSPA Single File Components (.gospa).
package sfc

import (
	"fmt"
	"regexp"
	"sort"
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
	FrontMatter map[string]string
	Script      Block
	ScriptTS    Block
	Template    Block
	Style       Block
}

var (
	scriptRegex   = regexp.MustCompile(`(?is)<script(.*?)>(.*?)</script>`)
	templateRegex = regexp.MustCompile(`(?is)<template(.*?)>(.*?)</template>`)
	styleRegex    = regexp.MustCompile(`(?is)<style(.*?)>(.*?)</style>`)
	langRegex     = regexp.MustCompile(`(?i)lang="([^"]*)"`)
)

// Parse splits a .gospa file into its component blocks.
func Parse(input string) (*SFC, error) {
	sfc := &SFC{}
	trimmed := strings.TrimSpace(input)

	frontMatterRegex := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n?`)
	if matches := frontMatterRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
		sfc.FrontMatter = parseFrontMatter(matches[1])
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, matches[0]))
	}
	input = trimmed

	// 1. Identify top-level blocks using masked input to ignore tags in strings/comments
	maskedInput := maskForParsing(input)

	type rawBlock struct {
		typ     string // "script", "template", "style"
		start   int
		end     int
		attr    string
		content string
	}
	var candidates []rawBlock

	for _, m := range scriptRegex.FindAllStringSubmatchIndex(maskedInput, -1) {
		candidates = append(candidates, rawBlock{"script", m[0], m[1], input[m[2]:m[3]], input[m[4]:m[5]]})
	}
	for _, m := range templateRegex.FindAllStringSubmatchIndex(maskedInput, -1) {
		candidates = append(candidates, rawBlock{"template", m[0], m[1], input[m[2]:m[3]], input[m[4]:m[5]]})
	}
	for _, m := range styleRegex.FindAllStringSubmatchIndex(maskedInput, -1) {
		candidates = append(candidates, rawBlock{"style", m[0], m[1], input[m[2]:m[3]], input[m[4]:m[5]]})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].start < candidates[j].start
	})

	var topLevel []rawBlock
	lastEnd := 0
	for _, b := range candidates {
		if b.start >= lastEnd {
			topLevel = append(topLevel, b)
			lastEnd = b.end
		}
	}

	// 2. Process top-level blocks
	var explicitTemplate bool
	for _, b := range topLevel {
		switch b.typ {
		case "script":
			lang := normalizeScriptLang(extractLang(b.attr, "go"))
			block := Block{
				Type:    "script",
				Lang:    lang,
				Content: strings.TrimSpace(b.content),
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
		case "style":
			if sfc.Style.Content != "" {
				return nil, fmt.Errorf("multiple <style> blocks are not supported")
			}
			sfc.Style = Block{
				Type:    "style",
				Lang:    extractLang(b.attr, "css"),
				Content: strings.TrimSpace(b.content),
			}
		case "template":
			if explicitTemplate {
				return nil, fmt.Errorf("multiple <template> blocks are not supported")
			}
			explicitTemplate = true
			sfc.Template = Block{
				Type:    "template",
				Content: strings.TrimSpace(b.content),
			}
		}
	}

	// 3. Handle implicit template if needed
	if !explicitTemplate {
		var builder strings.Builder
		lastPos := 0
		for _, b := range topLevel {
			builder.WriteString(input[lastPos:b.start])
			lastPos = b.end
		}
		builder.WriteString(input[lastPos:])
		sfc.Template = Block{
			Type:    "template",
			Content: strings.TrimSpace(builder.String()),
		}
	}

	if sfc.Template.Content == "" {
		return nil, fmt.Errorf("missing template content")
	}

	// 4. Final safety check: ensure no unclosed tags remain in the discarded or implicit content
	remainingMasked := maskedInput
	for i := len(topLevel) - 1; i >= 0; i-- {
		b := topLevel[i]
		remainingMasked = remainingMasked[:b.start] + remainingMasked[b.end:]
	}
	if regexp.MustCompile(`(?i)<(?:script|style|template)[\s/>]`).MatchString(remainingMasked) {
		return nil, fmt.Errorf("detected unclosed or malformed <script>, <style> or <template> block")
	}

	return sfc, nil
}

func extractLang(attr, defaultLang string) string {
	if matches := langRegex.FindStringSubmatch(attr); len(matches) > 1 {
		return matches[1]
	}
	return defaultLang
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

func parseFrontMatter(content string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key != "" {
			result[key] = val
		}
	}
	return result
}

var maskRegex = regexp.MustCompile("(?s)`[^`]*`|\"(?:\\\\.|[^\"\\\\])*\"|//.*|/\\*.*?\\*/")

func maskForParsing(input string) string {
	return maskRegex.ReplaceAllStringFunc(input, func(s string) string {
		res := make([]byte, len(s))
		for i := 0; i < len(s); i++ {
			if s[i] == '\n' {
				res[i] = '\n'
			} else {
				res[i] = ' '
			}
		}
		return string(res)
	})
}
