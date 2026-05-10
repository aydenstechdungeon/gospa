package sfc

import (
	"testing"
)

func TestParseEdgeCases(t *testing.T) {
	input := `
<script lang="go">
  var s = "<template>inside string</template>"
</script>
<template>
  <div>{s}</div>
</template>
`
	parsed, err := Parse(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if parsed.Script.Content != "var s = \"<template>inside string</template>\"" {
		t.Errorf("Unexpected script content: %q", parsed.Script.Content)
	}

	if parsed.Template.Content != "<div>{s}</div>" {
		t.Errorf("Unexpected template content: %q", parsed.Template.Content)
	}
}

func TestParseOffsets(t *testing.T) {
	input := "<template>hello</template>"
	parsed, err := Parse(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	// Content starts after "<template>" (10 bytes)
	if parsed.Template.ByteOffset != 10 {
		t.Errorf("Expected offset 10, got %d", parsed.Template.ByteOffset)
	}
}

func TestParseFrontMatterRequiresDelimiterOnOwnLine(t *testing.T) {
	input := `---
title: A --- B
---
<template><h1>Hello</h1></template>`

	parsed, err := Parse(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	if got := parsed.FrontMatter["title"]; got != "A --- B" {
		t.Fatalf("unexpected frontmatter title: %q", got)
	}
	if parsed.Template.Content != "<h1>Hello</h1>" {
		t.Fatalf("unexpected template content: %q", parsed.Template.Content)
	}
}
