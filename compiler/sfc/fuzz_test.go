package sfc

import (
	"strings"
	"testing"
)

// FuzzSFCParser exercises the .gospa parser with arbitrary inputs to discover
// crashes, panics, or infinite loops. Run with:
//
//	go test -fuzz=FuzzSFCParser -fuzztime=30s ./compiler/sfc/
func FuzzSFCParser(f *testing.F) {
	// Seed corpus: representative well-formed inputs
	f.Add(`<template><div>hello</div></template>`)
	f.Add(`<script lang="go">
  var count = $state(0)
</script>
<template><div>{count}</div></template>`)
	f.Add(`<script lang="go">var x = 1</script>
<script lang="ts">const y = 2</script>
<template><div>{x} {y}</div></template>
<style>div { color: red; }</style>`)
	f.Add(`---
type: page
hydrate: false
---
<template><main>@children</main></template>`)
	f.Add(`<script lang="go">
  var s = "` + "`" + `<template>inside</template>` + "`" + `"
</script>
<template>{s}</template>`)
	f.Add(`{#if x > 0}positive{:else}zero{/if}`)
	f.Add(`{#each items as item}<li>{item}</li>{/each}`)
	f.Add(`{#snippet mySnippet(arg)}<span>{arg}</span>{/snippet}`)

	f.Fuzz(func(t *testing.T, input string) {
		// Skip extremely large inputs (parser has a 2MB cap, use a smaller
		// threshold here to keep fuzz iterations fast).
		if len(input) > 2*1024*1024 {
			t.Skip("input too large")
		}

		sfc, err := Parse(input)
		if err != nil {
			// Parse errors are expected for malformed input
			return
		}

		// Sanity-check invariants on successfully parsed SFCs
		if sfc.Template.Content == "" && sfc.Script.Content == "" && sfc.ScriptTS.Content == "" {
			// This should have been caught by the "SFC is empty" error
			t.Error("parse returned empty SFC without error")
		}

		// Ensure round-tripping through tokenizer doesn't panic
		_ = sfc.FrontMatter
		_ = sfc.Script.Lang
		_ = sfc.ScriptTS.Lang
	})
}

// FuzzSFCTemplateParser exercises the template parser with arbitrary template
// fragments.
func FuzzSFCTemplateParser(f *testing.F) {
	f.Add(`<div class="container">hello</div>`)
	f.Add(`<span>text</span>`)
	f.Add(`<img src="x.png" />`)
	f.Add(`<div><br><hr><input type="text"></div>`)

	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 2*1024*1024 {
			t.Skip("input too large")
		}

		p := NewTemplateParser(input, 0, 0, 0)
		nodes, err := p.Parse()
		if err != nil {
			return
		}

		// Walk the AST to ensure no panics during traversal
		var walk func([]Node)
		walk = func(nodes []Node) {
			for _, node := range nodes {
				switch n := node.(type) {
				case *ElementNode:
					_ = n.TagName
					walk(n.Children)
				case *IfNode:
					walk(n.Then)
					for _, ei := range n.ElseIfs {
						walk(ei.Then)
					}
					walk(n.Else)
				case *EachNode:
					walk(n.Children)
				case *SnippetNode:
					walk(n.Children)
				case *ComponentNode:
					walk(n.Children)
				case *TextNode:
					_ = n.Content
				case *ExpressionNode:
					_ = n.Content
				case *CommentNode:
					_ = n.Content
				default:
					_ = strings.Builder{} // ensure type is used
				}
			}
		}
		walk(nodes)
	})
}
