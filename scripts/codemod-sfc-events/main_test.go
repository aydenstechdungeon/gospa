package main

import "testing"

func TestMigrateEventSyntax(t *testing.T) {
	in := []byte(`<button @click={inc} on-input={h} x-on:change={c}>x</button>
<div>user@example.com</div>
`)
	out := string(migrateEventSyntax(in))
	if got, want := out, `<button on:click={inc} on:input={h} on:change={c}>x</button>
<div>user@example.com</div>
`; got != want {
		t.Fatalf("unexpected codemod output\nwant:\n%s\ngot:\n%s", want, got)
	}
}
