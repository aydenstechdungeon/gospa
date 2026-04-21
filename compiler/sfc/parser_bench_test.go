package sfc

import (
	"strings"
	"testing"
)

func BenchmarkOffsetToPosition(b *testing.B) {
	input := strings.Repeat("line\n", 4096)
	offset := len(input) - 3

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = OffsetToPosition(input, offset)
	}
}
