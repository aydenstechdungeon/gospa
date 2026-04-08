package state

import (
	"context"
	"testing"
)

func BenchmarkGetGID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getGID()
	}
}

func BenchmarkBatchFastPath(b *testing.B) {
	// No batch active
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getBatchState(context.Background())
	}
}

func BenchmarkBatchInsideBatch(b *testing.B) {
	Batch(func() {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = getBatchState(context.Background())
		}
	})
}
