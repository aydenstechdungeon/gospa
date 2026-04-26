package compiler

import (
	"regexp"
	"strings"

	"github.com/aydenstechdungeon/gospa/compiler/sfc"
)

type reactiveHint struct {
	pattern    *regexp.Regexp
	message    string
	suggestion string
	snippet    string
}

var reactiveHints = []reactiveHint{
	{
		pattern:    regexp.MustCompile(`(^|[^$\w.])state\s*\(`),
		message:    "invalid reactive rune usage: state(...)",
		suggestion: "Use $state(...) for reactive state.",
		snippet:    "var count = $state(0)",
	},
	{
		pattern:    regexp.MustCompile(`\$derive\s*\(`),
		message:    "unknown rune $derive(...)",
		suggestion: "Use $derived(...).",
		snippet:    "var doubled = $derived(count * 2)",
	},
	{
		pattern:    regexp.MustCompile(`\$effects\s*\(`),
		message:    "unknown rune $effects(...)",
		suggestion: "Use $effect(func() { ... }).",
		snippet:    "$effect(func() {\n  // side effect\n})",
	},
}

var effectInvocationRegex = regexp.MustCompile(`(?s)^\$effect\(\s*func\(\)\s*\{.*?\}\s*\)`)

func validateReactiveUsage(script string, block sfc.Block) error {
	for _, hint := range reactiveHints {
		loc := hint.pattern.FindStringSubmatchIndex(script)
		if len(loc) < 2 {
			continue
		}
		start := loc[0]
		// For patterns with a leading context group, report start of actual token.
		if len(loc) >= 4 && loc[2] != -1 && loc[3] != -1 {
			start = loc[3]
		}
		line, col := localToAbsolutePosition(script, start, block)
		return &sfc.DiagnosticError{
			Line:       line,
			Column:     col,
			Message:    hint.message,
			Suggestion: hint.suggestion,
			Snippet:    hint.snippet,
		}
	}

	for idx := strings.Index(script, "$effect("); idx >= 0; {
		fragment := strings.TrimSpace(script[idx:])
		if !effectInvocationRegex.MatchString(fragment) {
			line, col := localToAbsolutePosition(script, idx, block)
			return &sfc.DiagnosticError{
				Line:       line,
				Column:     col,
				Message:    "invalid $effect usage",
				Suggestion: "Use $effect(func() { ... }) in Go scripts.",
				Snippet:    "$effect(func() {\n  // side effect\n})",
			}
		}
		next := strings.Index(script[idx+8:], "$effect(")
		if next == -1 {
			break
		}
		idx += 8 + next
	}

	return nil
}

func localToAbsolutePosition(script string, offset int, block sfc.Block) (line, col int) {
	relLine, relCol := sfc.OffsetToPosition(script, offset)
	line = block.Line + relLine + 1
	col = relCol + 1
	if relLine == 0 {
		col += block.Column
	}
	return line, col
}
