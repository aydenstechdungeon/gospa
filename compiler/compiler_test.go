package compiler

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aydenstechdungeon/gospa/compiler/sfc"
)

func TestParseSFC(t *testing.T) {
	input := `
<script lang="go">
  var count = 0
</script>

<template>
  <div>{count}</div>
</template>

<style>
  div { color: red; }
</style>
`
	parsed, err := sfc.Parse(input)
	if err != nil {
		t.Fatalf("Failed to parse SFC: %v", err)
	}

	if parsed.Script.Content != "var count = 0" {
		t.Errorf("Unexpected script content: %q", parsed.Script.Content)
	}

	if parsed.Template.Content != "<div>{count}</div>" {
		t.Errorf("Unexpected template content: %q", parsed.Template.Content)
	}

	if parsed.Style.Content != "div { color: red; }" {
		t.Errorf("Unexpected style content: %q", parsed.Style.Content)
	}

	fmt.Println("SFC Parse test passed")
}

func TestCompileCounter(t *testing.T) {
	c := NewCompiler()
	input := `
<script lang="go">
  var count = $state(0)
  var doubled = $derived(count * 2)
  
  $effect(func() {
    fmt.Printf("Count: %d\n", count)
  })

  func increment() {
    count++
  }
</script>

<template>
  <button on:click={increment}>{count}</button>
</template>
`
	templ, ts, err := c.Compile("Counter", input)
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	if !strings.Contains(templ, "data-gospa-island=\"Counter\"") {
		t.Errorf("Templ missing island attribute")
	}

	if !strings.Contains(ts, "state.$state(0)") {
		t.Errorf("TS missing reactive state")
	}

	if !strings.Contains(ts, "state.$derived(() => count * 2)") {
		t.Errorf("TS missing derived state")
	}

	if !strings.Contains(ts, "console.log(\"Count: %d\\n\", count)") {
		t.Errorf("TS missing effect/console.log: %q", ts)
	}

	fmt.Println("Counter compilation test passed")
}

func TestSanitizeName(t *testing.T) {
	c := NewCompiler()
	rawName := "Counter'); alert(1); //"
	_, ts, err := c.Compile(rawName, "<template><div>Test</div></template>")
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	if strings.Contains(ts, "alert(1)") {
		t.Errorf("Sanitization failed: TS still contains alert(1)")
	}

	if strings.Contains(ts, "name: 'Counteralert1'") {
		// nameSafeRegex: [^a-zA-Z0-9]
		// 'Counter' + '); alert(1); //' -> 'Counteralert1'
	} else if !strings.Contains(ts, "name: 'Counter") {
		t.Errorf("Unexpected sanitized name in TS: %v", ts)
	}
}
