package sfc

import (
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
