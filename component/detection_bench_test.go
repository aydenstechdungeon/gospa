package component

import (
	"fmt"
	"testing"
)

func BenchmarkDetectBoundaries(b *testing.B) {
	rd := NewReactiveDetector()
	source := `<script lang="go">
	var count = $state(0)
	func increment() { count++ }
	var double = $derived(count * 2)
	$effect(func() {
		fmt.Println("Count changed:", count)
	})
</script>
<template><div>Benchmark</div></template>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := rd.Detect(source, "benchmark.gospa")
		if result == nil {
			b.Fatal("result is nil")
		}
	}
}

func BenchmarkDetectBoundaries_Large(b *testing.B) {
	rd := NewReactiveDetector()

	// Generate a larger SFC with 100+ runes
	var scriptContent string
	for i := 0; i < 100; i++ {
		scriptContent += fmt.Sprintf("var state%d = $state(%d); ", i, i)
		scriptContent += fmt.Sprintf("var derived%d = $derived(state%d * 2); ", i, i)
	}

	source := fmt.Sprintf("<script lang=\"go\">\n%s\n</script>\n<template><div>Large</div></template>", scriptContent)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := rd.Detect(source, "large.gospa")
		if result == nil {
			b.Fatal("result is nil")
		}
	}
}

func BenchmarkDetectBoundaries_ManyLines(b *testing.B) {
	rd := NewReactiveDetector()

	// Generate a SFC with 1000 lines to test newline indexing performance
	var content string
	for i := 0; i < 1000; i++ {
		content += fmt.Sprintf("// line %d\n", i)
	}
	content += "var count = $state(0)\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := rd.Detect(content, "many_lines.gospa")
		if result == nil {
			b.Fatal("result is nil")
		}
	}
}
