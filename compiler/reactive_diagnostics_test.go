package compiler

import (
	"errors"
	"testing"

	"github.com/aydenstechdungeon/gospa/compiler/sfc"
)

func TestCompileReactiveDiagnostic_StateMissingDollar(t *testing.T) {
	c := NewCompiler()
	input := `<script lang="go">
var count = state(0)
</script>
<template><div>{count}</div></template>`

	_, _, err := c.Compile(CompileOptions{
		Type:     ComponentTypeIsland,
		Name:     "Counter",
		IslandID: "counter",
		Hydrate:  true,
	}, input)
	if err == nil {
		t.Fatal("expected diagnostic error")
	}
	var diag *sfc.DiagnosticError
	if !errors.As(err, &diag) {
		t.Fatalf("expected DiagnosticError, got %T (%v)", err, err)
	}
	if diag.Line <= 0 || diag.Column <= 0 {
		t.Fatalf("expected exact position, got %d:%d", diag.Line, diag.Column)
	}
	if diag.Suggestion == "" || diag.Snippet == "" {
		t.Fatalf("expected remediation hint, got %#v", diag)
	}
}

func TestCompileReactiveDiagnostic_InvalidEffectForm(t *testing.T) {
	c := NewCompiler()
	input := `<script lang="go">
$effect(count)
</script>
<template><div>ok</div></template>`

	_, _, err := c.Compile(CompileOptions{
		Type:     ComponentTypeIsland,
		Name:     "Counter",
		IslandID: "counter",
		Hydrate:  true,
	}, input)
	if err == nil {
		t.Fatal("expected diagnostic error")
	}
	var diag *sfc.DiagnosticError
	if !errors.As(err, &diag) {
		t.Fatalf("expected DiagnosticError, got %T (%v)", err, err)
	}
	if diag.Message == "" || diag.Suggestion == "" {
		t.Fatalf("expected diagnostic message and suggestion, got %#v", diag)
	}
}

func TestCompileReactiveDiagnostic_EffectAllowsWhitespaceForm(t *testing.T) {
	c := NewCompiler()
	input := `<script lang="go">
$effect( func() {
    return
})
</script>
<template><div>ok</div></template>`

	_, _, err := c.Compile(CompileOptions{
		Type:     ComponentTypeIsland,
		Name:     "Counter",
		IslandID: "counter",
		Hydrate:  true,
	}, input)
	if err != nil {
		t.Fatalf("expected compile success, got %v", err)
	}
}
