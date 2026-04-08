package compiler

import (
	"fmt"
	"testing"
)

func BenchmarkCompileSFC(b *testing.B) {
	c := NewCompiler()
	opts := CompileOptions{
		Type: ComponentTypeIsland,
		Name: "BenchmarkIsland",
	}

	source := `<script lang="go">
	var count = $state(0)
	func increment() { count++ }
	var double = $derived(count * 2)
	$effect(func() {
		fmt.Println("Count changed:", count)
	})
</script>

<template>
	<div>
		<h1>Benchmark</h1>
		<button on:click={increment}>Increment</button>
		<p>Count: {count}</p>
		<p>Double: {double}</p>
	</div>
</template>

<style>
	h1 { color: blue; }
</style>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := c.Compile(opts, source)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompileSFC_Large(b *testing.B) {
	c := NewCompiler()
	opts := CompileOptions{
		Type: ComponentTypeIsland,
		Name: "LargeIsland",
	}

	// Generate a larger SFC
	var scriptContent string
	for i := 0; i < 50; i++ {
		scriptContent += fmt.Sprintf("var state%d = $state(%d); ", i, i)
		scriptContent += fmt.Sprintf("var derived%d = $derived(state%d * 2); ", i, i)
	}

	source := fmt.Sprintf("<script lang=\"go\">\n%s\n</script>\n<template><div>Large</div></template>", scriptContent)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := c.Compile(opts, source)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompileSFC_SafeMode(b *testing.B) {
	c := NewCompiler()
	opts := CompileOptions{
		Type:     ComponentTypeIsland,
		Name:     "SafeIsland",
		SafeMode: true,
	}

	source := `<script lang="go">
	var count = $state(0)
	func increment() { count++ }
</script>
<template><div>Safe</div></template>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := c.Compile(opts, source)
		if err != nil {
			b.Fatal(err)
		}
	}
}
