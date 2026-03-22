package gospa

import (
	stdjson "encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestValidateJSONMaxNesting_AcceptsTypicalDepth(t *testing.T) {
	data := []byte(`{"a":1,"b":[2,{"c":3}]}`)
	if err := validateJSONMaxNesting(data, remoteJSONMaxNesting); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateJSONMaxNesting_RejectsDeepNesting(t *testing.T) {
	n := remoteJSONMaxNesting + 1
	data := []byte(strings.Repeat(`{"a":`, n) + `0` + strings.Repeat(`}`, n))
	err := validateJSONMaxNesting(data, remoteJSONMaxNesting)
	if err == nil {
		t.Fatal("expected nesting error")
	}
	if !errors.Is(err, ErrJSONTooDeep) {
		t.Fatalf("expected ErrJSONTooDeep, got %v", err)
	}
}

func TestDecodeRemoteActionBody_UseNumber(t *testing.T) {
	v, err := decodeRemoteActionBody([]byte(`{"n":9007199254740993}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", v)
	}
	if _, ok := m["n"].(stdjson.Number); !ok {
		t.Fatalf("expected json.Number, got %T", m["n"])
	}
}
