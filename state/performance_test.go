package state

import (
	"reflect"
	"testing"
)

func BenchmarkEqualPrimitive(b *testing.B) {
	a, res := 1, 1
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = equal(a, res)
	}
}

func BenchmarkEqualComplex(b *testing.B) {
	a := map[string]interface{}{"foo": "bar", "baz": 123}
	res := map[string]interface{}{"foo": "bar", "baz": 123}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = equal(a, res)
	}
}

func BenchmarkDeepEqualValuesPrimitive(b *testing.B) {
	a, res := 1, 1
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = deepEqualValues(a, res)
	}
}

func BenchmarkDeepEqualValuesComplex(b *testing.B) {
	a := map[string]interface{}{"foo": "bar", "baz": 123}
	res := map[string]interface{}{"foo": "bar", "baz": 123}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = deepEqualValues(a, res)
	}
}

func BenchmarkReflectDeepEqualComplex(b *testing.B) {
	a := map[string]interface{}{"foo": "bar", "baz": 123}
	res := map[string]interface{}{"foo": "bar", "baz": 123}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = reflect.DeepEqual(a, res)
	}
}
