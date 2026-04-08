// Package sfc provides a parser for GoSPA Single File Components (.gospa).
package sfc

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

// Block represents a section of a .gospa file.
type Block struct {
	Type       string // "script", "template", "style"
	Lang       string // e.g., "go", "ts", "css"
	Content    string
	ByteOffset int // Start of the content block in the original source
	Line       int // 0-indexed line number
	Column     int // 0-indexed column number
}

// SFC represents the parsed structure of a .gospa file.
type SFC struct {
	FrontMatter map[string]string
	Script      Block
	ScriptTS    Block
	Template    Block
	Style       Block
}

// maxSFCInputBytes is the hard ceiling on .gospa source files.
// Inputs larger than this are rejected to prevent memory exhaustion during
// compilation and to limit the attack surface for resource-abuse scenarios.
const maxSFCInputBytes = 2 * 1024 * 1024 // 2 MB

// Parse splits a .gospa file into its component blocks.
func Parse(input string) (*SFC, error) {
	if len(input) > maxSFCInputBytes {
		return nil, fmt.Errorf("SFC input exceeds maximum allowed size of %d bytes", maxSFCInputBytes)
	}

	sfc := &SFC{}
	var offset int

	// 1. Handle Front Matter
	if strings.HasPrefix(input, "---") {
		endIdx := strings.Index(input[3:], "---")
		if endIdx != -1 {
			fmContent := input[3 : endIdx+3]
			sfc.FrontMatter = parseFrontMatter(fmContent)
			offset = endIdx + 6 // "---" + content + "---"
			// Skip newline if any
			if offset < len(input) && input[offset] == '\n' {
				offset++
			}
			if offset < len(input) && input[offset] == '\n' {
				offset++
			}
		}
	}

	// 2. Tokenize top-level blocks
	rawInput := []byte(input)
	var topLevelBlocks []Block
	var implicitContent strings.Builder
	var implicitStartOffset = -1 // Tracks the start of the first implicit content block
	var baseOffset = offset

	// Pre-compute string literal ranges to skip false-positive tags
	stringRanges := findStringLiteralRanges(input)

	for baseOffset < len(input) {
		z := html.NewTokenizer(strings.NewReader(input[baseOffset:]))
		tt := z.Next()

		if tt == html.ErrorToken {
			if z.Err() == io.EOF {
				break // End of input
			}
			return nil, z.Err()
		}

		raw := z.Raw() // The raw bytes of the current token
		token := z.Token()

		if tt == html.StartTagToken {
			tagName := token.DataAtom.String()
			if tagName == "" {
				tagName = token.Data
			}

			if tagName == "script" || tagName == "style" || tagName == "template" {
				// Skip tags found inside Go string literals (e.g., backtick strings in CodeBlock calls)
				if isInsideStringLiteral(baseOffset, stringRanges) {
					// Treat the entire raw token as implicit content
					if implicitStartOffset == -1 {
						implicitStartOffset = baseOffset
					}
					implicitContent.Write(raw)
					if len(raw) == 0 {
						baseOffset++
					} else {
						baseOffset += len(raw)
					}
					continue
				}
				startTagRawLen := len(raw)
				contentOffset := baseOffset + startTagRawLen

				// Find end tag, skipping over string literals to avoid false matches
				endTagBytes := []byte("</" + tagName + ">")
				endIdx := findEndTagSkippingStrings(rawInput, contentOffset, endTagBytes, stringRanges)
				if endIdx == -1 {
					return nil, fmt.Errorf("unclosed <%s> block starting at offset %d", tagName, baseOffset)
				}

				content := string(rawInput[contentOffset : contentOffset+endIdx])

				// Extract lang attribute from the current token
				lang := ""
				for _, attr := range token.Attr {
					if attr.Key == "lang" {
						lang = attr.Val
						break
					}
				}
				if lang == "" {
					lang = extractLangFromToken(token, tagName)
				}

				block := Block{
					Type:       tagName,
					Lang:       lang,
					Content:    strings.TrimSpace(content), // Trim whitespace
					ByteOffset: contentOffset,
				}
				block.Line, block.Column = OffsetToPosition(input, contentOffset)
				topLevelBlocks = append(topLevelBlocks, block)

				baseOffset += startTagRawLen + endIdx + len(endTagBytes)
				continue
			}
		}

		// Any other token type or non-top-level StartTag is considered part of the implicit template
		if implicitStartOffset == -1 {
			implicitStartOffset = baseOffset
		}
		implicitContent.Write(raw)
		if len(raw) == 0 {
			baseOffset++
		} else {
			baseOffset += len(raw)
		}
	}

	// 3. Process blocks
	var explicitTemplate bool
	for _, b := range topLevelBlocks {
		switch b.Type {
		case "script":
			lang := normalizeScriptLang(b.Lang)
			b.Lang = lang
			switch lang {
			case "go":
				if sfc.Script.Content != "" {
					return nil, fmt.Errorf("multiple <script lang=\"go\"> blocks are not supported")
				}
				sfc.Script = b
			case "ts":
				if sfc.ScriptTS.Content != "" {
					return nil, fmt.Errorf("multiple <script lang=\"ts\"> blocks are not supported")
				}
				sfc.ScriptTS = b
			}
		case "style":
			if sfc.Style.Content != "" {
				return nil, fmt.Errorf("multiple <style> blocks are not supported")
			}
			sfc.Style = b
		case "template":
			if explicitTemplate {
				return nil, fmt.Errorf("multiple <template> blocks are not supported")
			}
			explicitTemplate = true
			sfc.Template = b
		}
	}

	// 4. Handle implicit template
	if !explicitTemplate && implicitContent.Len() > 0 {
		sfc.Template = Block{
			Type:       "template",
			Content:    strings.TrimSpace(implicitContent.String()),
			ByteOffset: implicitStartOffset,
		}
		sfc.Template.Line, sfc.Template.Column = OffsetToPosition(input, implicitStartOffset)
	}

	if sfc.Template.Content == "" && sfc.Script.Content == "" && sfc.ScriptTS.Content == "" {
		return nil, fmt.Errorf("SFC is empty")
	}

	return sfc, nil
}

func extractLangFromToken(t html.Token, tagName string) string {
	for _, attr := range t.Attr {
		if attr.Key == "lang" {
			return attr.Val
		}
	}
	switch tagName {
	case "script":
		return "go"
	case "style":
		return "css"
	default:
		return ""
	}
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

// OffsetToPosition converts a byte offset to a (line, column) coordinate.
func OffsetToPosition(input string, offset int) (int, int) {
	line := 0
	col := 0
	for i := 0; i < offset && i < len(input); i++ {
		if input[i] == '\n' {
			line++
			col = 0
		} else {
			col++
		}
	}
	return line, col
}

// stringLiteralRange represents the [start, end) byte range of a Go string literal.
type stringLiteralRange struct {
	start int
	end   int
}

// findStringLiteralRanges returns all Go string literal ranges (backtick only)
// in the input. This is used to skip false-positive HTML tags found inside string literals.
// We only track backtick strings because double-quoted strings are ambiguous with HTML attributes.
func findStringLiteralRanges(input string) []stringLiteralRange {
	var ranges []stringLiteralRange
	i := 0
	for i < len(input) {
		switch input[i] {
		case '`':
			start := i
			i++ // skip opening `
			for i < len(input) && input[i] != '`' {
				i++
			}
			if i < len(input) {
				i++ // skip closing `
				ranges = append(ranges, stringLiteralRange{start, i})
			}
		case '"':
			start := i
			i++ // skip opening "
			for i < len(input) && input[i] != '"' {
				if input[i] == '\\' && i+1 < len(input) {
					i += 2
					continue
				}
				i++
			}
			if i < len(input) {
				i++ // skip closing "
				ranges = append(ranges, stringLiteralRange{start, i})
			}
		default:
			i++
		}
	}
	return ranges
}

// isInsideStringLiteral returns true if the given byte position falls within
// any of the provided string literal ranges.
func isInsideStringLiteral(pos int, ranges []stringLiteralRange) bool {
	for _, r := range ranges {
		if pos >= r.start && pos < r.end {
			return true
		}
		if r.start > pos {
			break // ranges are ordered, no need to check further
		}
	}
	return false
}

// findEndTagSkippingStrings searches for endTagBytes in data starting from offset,
// skipping over positions that fall within string literal ranges.
// Returns the index RELATIVE to offset (like bytes.Index), or -1 if not found.
func findEndTagSkippingStrings(data []byte, offset int, endTagBytes []byte, stringRanges []stringLiteralRange) int {
	searchStart := offset
	for searchStart < len(data) {
		idx := bytes.Index(data[searchStart:], endTagBytes)
		if idx == -1 {
			return -1
		}
		absIdx := searchStart + idx
		// Check if this match is inside a string literal
		if !isInsideStringLiteral(absIdx, stringRanges) {
			return absIdx - offset // Return relative index
		}
		// Skip past this match and continue searching
		searchStart = absIdx + len(endTagBytes)
	}
	return -1
}
