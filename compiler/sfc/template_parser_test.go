package sfc

import (
	"errors"
	"testing"
)

func TestTemplateParser(t *testing.T) {
	input := `<div class="container">
  <h1>{title}</h1>
  {#if count > 0}
    <p>Count is {count}</p>
  {:else}
    <p>Zero</p>
  {/if}
  {#each items as item}
    <li>{item}</li>
  {/each}
  {#await profilePromise}
    <p>Loading profile...</p>
  {:then profile}
    <p>{profile.Name}</p>
  {:catch err}
    <p>{err.Error()}</p>
  {/await}
  <img src="logo.png" />
  <@MyComponent prop={val} />
</div>`
	p := NewTemplateParser(input, 0, 0, 0)
	nodes, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(nodes) != 1 {
		t.Fatalf("Expected 1 root node (div), got %d", len(nodes))
	}

	div := nodes[0].(*ElementNode)
	if div.TagName != "div" {
		t.Errorf("Expected tagName div, got %s", div.TagName)
	}

	// Verify if node
	foundIf := false
	for _, child := range div.Children {
		if _, ok := child.(*IfNode); ok {
			foundIf = true
			break
		}
	}
	if !foundIf {
		t.Error("IfNode not found in div children")
	}

	// Verify each node
	foundEach := false
	for _, child := range div.Children {
		if _, ok := child.(*EachNode); ok {
			foundEach = true
			break
		}
	}
	if !foundEach {
		t.Error("EachNode not found in div children")
	}

	// Verify await node
	foundAwait := false
	for _, child := range div.Children {
		if _, ok := child.(*AwaitNode); ok {
			foundAwait = true
			break
		}
	}
	if !foundAwait {
		t.Error("AwaitNode not found in div children")
	}

	// Verify void tag
	foundImg := false
	for _, child := range div.Children {
		if img, ok := child.(*ElementNode); ok && img.TagName == "img" {
			foundImg = true
			if !img.SelfClosing {
				t.Error("img should be self-closing")
			}
			break
		}
	}
	if !foundImg {
		t.Error("img tag not found")
	}
}

func TestTemplateParser_PreservesQuotedText(t *testing.T) {
	input := "<div>\"hello\" and `code`</div>"
	p := NewTemplateParser(input, 0, 0, 0)
	nodes, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 root node, got %d", len(nodes))
	}
	div, ok := nodes[0].(*ElementNode)
	if !ok {
		t.Fatalf("Expected ElementNode, got %T", nodes[0])
	}
	if len(div.Children) != 1 {
		t.Fatalf("Expected 1 text child, got %d", len(div.Children))
	}
	text, ok := div.Children[0].(*TextNode)
	if !ok {
		t.Fatalf("Expected TextNode, got %T", div.Children[0])
	}
	if text.Content != "\"hello\" and `code`" {
		t.Fatalf("Unexpected text content: %q", text.Content)
	}
}

func TestTemplateParser_TreatsInvalidAtComponentAsText(t *testing.T) {
	input := `@("oops")`
	p := NewTemplateParser(input, 0, 0, 0)
	nodes, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse should keep invalid @ forms as text: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 text node, got %d", len(nodes))
	}
	text, ok := nodes[0].(*TextNode)
	if !ok {
		t.Fatalf("Expected TextNode, got %T", nodes[0])
	}
	if text.Content != `@("oops")` {
		t.Fatalf("Unexpected text content: %q", text.Content)
	}
}

func TestTemplateParser_RejectsUnclosedAtComponentArgs(t *testing.T) {
	input := `@components.CodeBlock("unterminated"`
	p := NewTemplateParser(input, 0, 0, 0)
	_, err := p.Parse()
	if err == nil {
		t.Fatal("Expected parse error for unterminated component arguments")
	}
}

func TestTemplateParser_PreservesAtSymbolInText(t *testing.T) {
	input := "<div>email me at user@example.com</div>"
	p := NewTemplateParser(input, 0, 0, 0)
	nodes, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 root node, got %d", len(nodes))
	}
	div := nodes[0].(*ElementNode)
	if len(div.Children) != 1 {
		t.Fatalf("Expected one text child, got %d", len(div.Children))
	}
	text := div.Children[0].(*TextNode)
	if text.Content != "email me at user@example.com" {
		t.Fatalf("Unexpected text content: %q", text.Content)
	}
}

func TestTemplateParser_RejectsAtComponentWithoutParens(t *testing.T) {
	input := `@components.CodeBlock`
	p := NewTemplateParser(input, 0, 0, 0)
	nodes, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse should treat bare @identifier as text, got error: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 text node, got %d", len(nodes))
	}
	text, ok := nodes[0].(*TextNode)
	if !ok {
		t.Fatalf("Expected TextNode, got %T", nodes[0])
	}
	if text.Content != "@components.CodeBlock" {
		t.Fatalf("Unexpected text content: %q", text.Content)
	}
}

func TestTemplateParser_ComponentTagRequiresClosingTag(t *testing.T) {
	input := `<@Card><div>Body</div>`
	p := NewTemplateParser(input, 0, 0, 0)
	_, err := p.Parse()
	if err == nil {
		t.Fatal("expected parse error for unclosed <@Card> tag")
	}
}

func TestTemplateParser_ComplexExpressions(t *testing.T) {
	input := `<div>{ "prop": { "key": "value" } }</div>`
	p := NewTemplateParser(input, 0, 0, 0)
	nodes, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 root node, got %d", len(nodes))
	}
	div := nodes[0].(*ElementNode)
	if len(div.Children) != 1 {
		t.Fatalf("Expected 1 child, got %d", len(div.Children))
	}
	expr, ok := div.Children[0].(*ExpressionNode)
	if !ok {
		t.Fatalf("Expected ExpressionNode, got %T", div.Children[0])
	}
	if expr.Content != ` "prop": { "key": "value" } ` {
		t.Fatalf("Unexpected expression content: %q", expr.Content)
	}
}

func TestTemplateParser_ExpressionWithBraceInString(t *testing.T) {
	input := `<div>{ map[string]string{"k": "}"} }</div>`
	p := NewTemplateParser(input, 0, 0, 0)
	nodes, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 root node, got %d", len(nodes))
	}
	div := nodes[0].(*ElementNode)
	if len(div.Children) != 1 {
		t.Fatalf("Expected 1 child, got %d", len(div.Children))
	}
	expr, ok := div.Children[0].(*ExpressionNode)
	if !ok {
		t.Fatalf("Expected ExpressionNode, got %T", div.Children[0])
	}
	if expr.Content != ` map[string]string{"k": "}"} ` {
		t.Fatalf("Unexpected expression content: %q", expr.Content)
	}
}

func TestTemplateParser_ExpressionWithCommentContainingDelimiter(t *testing.T) {
	input := `<div>{ value /* } */ + 1 }</div>`
	p := NewTemplateParser(input, 0, 0, 0)
	nodes, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	div := nodes[0].(*ElementNode)
	expr, ok := div.Children[0].(*ExpressionNode)
	if !ok {
		t.Fatalf("Expected ExpressionNode, got %T", div.Children[0])
	}
	if expr.Content != ` value /* } */ + 1 ` {
		t.Fatalf("Unexpected expression content: %q", expr.Content)
	}
}

func TestTemplateParser_InvalidEventDirectiveHint(t *testing.T) {
	input := `<button @click={inc}>+</button>`
	p := NewTemplateParser(input, 0, 0, 0)
	_, err := p.Parse()
	if err == nil {
		t.Fatal("expected parse error for @click syntax")
	}
	var diag *DiagnosticError
	if !errors.As(err, &diag) {
		t.Fatalf("expected DiagnosticError, got %T (%v)", err, err)
	}
	if diag.Line != 1 || diag.Column == 0 {
		t.Fatalf("expected line/column, got %d:%d", diag.Line, diag.Column)
	}
	if diag.Suggestion == "" || diag.Snippet == "" {
		t.Fatalf("expected suggestion/snippet, got %#v", diag)
	}
}

func TestTemplateParser_UnknownDirectiveHint(t *testing.T) {
	input := `{#iff ok}<div/> {/iff}`
	p := NewTemplateParser(input, 0, 0, 0)
	_, err := p.Parse()
	if err == nil {
		t.Fatal("expected parse error for unknown directive")
	}
	var diag *DiagnosticError
	if !errors.As(err, &diag) {
		t.Fatalf("expected DiagnosticError, got %T (%v)", err, err)
	}
	if diag.Suggestion == "" {
		t.Fatalf("expected suggestion for unknown directive, got %#v", diag)
	}
}
