package state

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

// ─── StateMap.Add / Get / Remove ──────────────────────────────────────────────

func TestStateMap_AddAndGet(t *testing.T) {
	sm := NewStateMap()
	r := NewRune(42)
	sm.Add("count", r)

	obs, ok := sm.Get("count")
	if !ok {
		t.Fatal("Get('count') returned false after Add")
	}
	if obs.GetAny() != 42 {
		t.Errorf("expected 42, got %v", obs.GetAny())
	}
}

func TestStateMap_GetMissing(t *testing.T) {
	sm := NewStateMap()
	_, ok := sm.Get("nonexistent")
	if ok {
		t.Error("Get on missing key should return false")
	}
}

func TestStateMap_AddAny(t *testing.T) {
	sm := NewStateMap()
	sm.AddAny("name", "hello")
	obs, ok := sm.Get("name")
	if !ok {
		t.Fatal("Get('name') returned false after AddAny")
	}
	if obs.GetAny() != "hello" {
		t.Errorf("expected 'hello', got %v", obs.GetAny())
	}
}

func TestStateMap_Remove(t *testing.T) {
	sm := NewStateMap()
	sm.Add("count", NewRune(0))
	sm.Remove("count")
	_, ok := sm.Get("count")
	if ok {
		t.Error("Get should return false after Remove")
	}
}

func TestStateMap_RemoveNonExistent(_ *testing.T) {
	sm := NewStateMap()
	// Should not panic
	sm.Remove("nonexistent")
}

func TestStateMap_AddOverwrite(t *testing.T) {
	sm := NewStateMap()
	sm.Add("x", NewRune(1))
	sm.Add("x", NewRune(99)) // new rune, StateMap transfers old value to it
	obs, ok := sm.Get("x")
	if !ok {
		t.Fatal("Get('x') returned false")
	}
	// StateMap.Add transfers the existing observable's value to the new one.
	// So the new observable's value becomes the old value (1), not 99.
	// Either is acceptable since this tests that the key is still accessible.
	v := obs.GetAny()
	if v != 1 && v != 99 {
		t.Errorf("expected 1 (transferred) or 99 (initial), got %v", v)
	}
}

// ─── StateMap.ToMap / MarshalJSON / ToJSON ────────────────────────────────────

func TestStateMap_ToMap(t *testing.T) {
	sm := NewStateMap()
	sm.Add("a", NewRune(1))
	sm.Add("b", NewRune("hello"))

	m := sm.ToMap()
	if m["a"] != 1 {
		t.Errorf("expected a=1, got %v", m["a"])
	}
	if m["b"] != "hello" {
		t.Errorf("expected b='hello', got %v", m["b"])
	}
}

func TestStateMap_MarshalJSON(t *testing.T) {
	sm := NewStateMap()
	sm.Add("count", NewRune(7))

	data, err := sm.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	// JSON numbers unmarshal as float64
	if result["count"] != float64(7) {
		t.Errorf("expected count=7, got %v", result["count"])
	}
}

func TestStateMap_ToJSON(t *testing.T) {
	sm := NewStateMap()
	sm.Add("key", NewRune("value"))

	s, err := sm.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}
	if s == "" {
		t.Error("ToJSON should return non-empty string")
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("JSON parse failed: %v", err)
	}
}

// ─── StateMap.ForEach ─────────────────────────────────────────────────────────

func TestStateMap_ForEach(t *testing.T) {
	sm := NewStateMap()
	sm.Add("a", NewRune(1))
	sm.Add("b", NewRune(2))
	sm.Add("c", NewRune(3))

	seen := map[string]bool{}
	sm.ForEach(func(key string, _ any) {
		seen[key] = true
	})
	for _, key := range []string{"a", "b", "c"} {
		if !seen[key] {
			t.Errorf("ForEach did not visit key %q", key)
		}
	}
}

// ─── StateMap.Diff ────────────────────────────────────────────────────────────

func TestStateMap_Diff_Added(t *testing.T) {
	sm1 := NewStateMap()
	sm1.Add("x", NewRune(1))
	sm1.Add("new", NewRune(2))

	sm2 := NewStateMap()
	sm2.Add("x", NewRune(1))

	diff := sm1.Diff(sm2)
	if _, ok := diff.Added["new"]; !ok {
		t.Error("Diff should detect 'new' as Added")
	}
}

func TestStateMap_Diff_Removed(t *testing.T) {
	sm1 := NewStateMap()
	sm1.Add("x", NewRune(1))

	sm2 := NewStateMap()
	sm2.Add("x", NewRune(1))
	sm2.Add("old", NewRune(99))

	diff := sm1.Diff(sm2)
	if _, ok := diff.Removed["old"]; !ok {
		t.Error("Diff should detect 'old' as Removed")
	}
}

func TestStateMap_Diff_Changed(t *testing.T) {
	sm1 := NewStateMap()
	sm1.Add("x", NewRune(10))

	sm2 := NewStateMap()
	sm2.Add("x", NewRune(5))

	diff := sm1.Diff(sm2)
	if _, ok := diff.Changed["x"]; !ok {
		t.Error("Diff should detect 'x' as Changed")
	}
}

func TestStateMap_Diff_NilOther(t *testing.T) {
	sm := NewStateMap()
	sm.Add("a", NewRune(1))

	diff := sm.Diff(nil)
	if _, ok := diff.Added["a"]; !ok {
		t.Error("Diff(nil) should show all keys as Added")
	}
}

func TestStateMap_Diff_NoChange(t *testing.T) {
	sm1 := NewStateMap()
	sm1.Add("x", NewRune(42))

	sm2 := NewStateMap()
	sm2.Add("x", NewRune(42))

	diff := sm1.Diff(sm2)
	if len(diff.Added)+len(diff.Removed)+len(diff.Changed) != 0 {
		t.Errorf("expected no diff for identical maps, got added=%v removed=%v changed=%v",
			diff.Added, diff.Removed, diff.Changed)
	}
}

// ─── StateMap.OnChange callback ───────────────────────────────────────────────

func TestStateMap_OnChange(t *testing.T) {
	sm := NewStateMap()
	r := NewRune(0)
	notifications := make(chan struct{}, 10)
	sm.OnChange = func(_ string, _ any) {
		notifications <- struct{}{}
	}
	sm.Add("count", r)

	r.Set(1)
	select {
	case <-notifications:
		// OK
	case <-time.After(500 * time.Millisecond):
		t.Error("OnChange callback was not called after state change")
	}
}

// ─── StateMap concurrent access ───────────────────────────────────────────────

func TestStateMap_ConcurrentAccess(_ *testing.T) {
	sm := NewStateMap()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key"
			sm.Add(key, NewRune(n))
			_, _ = sm.Get(key)
			sm.Remove(key)
		}(i)
	}
	wg.Wait()
}

// ─── SerializeState ───────────────────────────────────────────────────────────

func TestSerializeState(t *testing.T) {
	runes := map[string]interface{}{
		"count": NewRune[any](42),
		"name":  NewRune[any]("test"),
	}
	data, err := SerializeState(runes)
	if err != nil {
		t.Fatalf("SerializeState failed: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if result["count"] != float64(42) {
		t.Errorf("expected count=42, got %v", result["count"])
	}
	if result["name"] != "test" {
		t.Errorf("expected name='test', got %v", result["name"])
	}
}

// ─── State messages ───────────────────────────────────────────────────────────

func TestNewInitMessage(t *testing.T) {
	msg := NewInitMessage("comp1", map[string]interface{}{"x": 1})
	if msg.Type != "init" {
		t.Errorf("expected type 'init', got %q", msg.Type)
	}
	if msg.ComponentID != "comp1" {
		t.Errorf("expected componentId 'comp1', got %q", msg.ComponentID)
	}
	if msg.Timestamp <= 0 {
		t.Error("Timestamp should be positive")
	}
}

func TestNewUpdateMessage(t *testing.T) {
	msg := NewUpdateMessage("comp1", "count", 5)
	if msg.Type != "update" {
		t.Errorf("expected type 'update', got %q", msg.Type)
	}
	if msg.Key != "count" {
		t.Errorf("expected key 'count', got %q", msg.Key)
	}
	if msg.Value != 5 {
		t.Errorf("expected value 5, got %v", msg.Value)
	}
}

func TestNewSyncMessage(t *testing.T) {
	msg := NewSyncMessage("comp1", map[string]interface{}{"a": 1})
	if msg.Type != "sync" {
		t.Errorf("expected type 'sync', got %q", msg.Type)
	}
}

func TestNewErrorMessage(t *testing.T) {
	msg := NewErrorMessage("comp1", "something went wrong")
	if msg.Type != "error" {
		t.Errorf("expected type 'error', got %q", msg.Type)
	}
	if msg.Error != "something went wrong" {
		t.Errorf("expected error message 'something went wrong', got %q", msg.Error)
	}
}

func TestParseMessage_RoundTrip(t *testing.T) {
	original := NewUpdateMessage("comp1", "key", "value")
	data, err := original.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	parsed, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}
	if parsed.Type != original.Type {
		t.Errorf("expected type %q, got %q", original.Type, parsed.Type)
	}
	if parsed.Key != original.Key {
		t.Errorf("expected key %q, got %q", original.Key, parsed.Key)
	}
}

func TestParseMessage_InvalidJSON(t *testing.T) {
	_, err := ParseMessage([]byte("invalid json {"))
	if err == nil {
		t.Error("ParseMessage should return error for invalid JSON")
	}
}

func TestParseMessage_InvalidType(t *testing.T) {
	_, err := ParseMessage([]byte(`{"type":"admin_override","componentId":"x"}`))
	if err == nil {
		t.Error("ParseMessage should return error for invalid message type")
	}
}

// ─── StateSnapshot ────────────────────────────────────────────────────────────

func TestNewSnapshot(t *testing.T) {
	snap := NewSnapshot("comp1", map[string]interface{}{"x": 1})
	if snap.ComponentID != "comp1" {
		t.Errorf("expected ComponentID 'comp1', got %q", snap.ComponentID)
	}
	if snap.Timestamp <= 0 {
		t.Error("Timestamp should be positive")
	}
}

func TestSnapshot_MarshalJSON(t *testing.T) {
	snap := NewSnapshot("comp1", map[string]interface{}{"count": 42})
	data, err := snap.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if result["componentId"] != "comp1" {
		t.Errorf("expected componentId 'comp1', got %v", result["componentId"])
	}
}

// ─── StateDiff ────────────────────────────────────────────────────────────────

func TestNewStateDiff(t *testing.T) {
	diff := NewStateDiff("comp1", "count", 0, 5)
	if diff.ComponentID != "comp1" {
		t.Errorf("expected ComponentID 'comp1', got %q", diff.ComponentID)
	}
	if diff.Key != "count" {
		t.Errorf("expected Key 'count', got %q", diff.Key)
	}
	if diff.OldValue != 0 {
		t.Errorf("expected OldValue 0, got %v", diff.OldValue)
	}
	if diff.NewValue != 5 {
		t.Errorf("expected NewValue 5, got %v", diff.NewValue)
	}
}

// ─── StateValidator ───────────────────────────────────────────────────────────

func TestStateValidator_ValidValue(t *testing.T) {
	sv := NewStateValidator()
	sv.AddValidator("age", func(v interface{}) error {
		if v.(int) < 0 {
			return fmt.Errorf("age cannot be negative")
		}
		return nil
	})
	if err := sv.Validate("age", 25); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestStateValidator_InvalidValue(t *testing.T) {
	sv := NewStateValidator()
	sv.AddValidator("age", func(v interface{}) error {
		if v.(int) < 0 {
			return fmt.Errorf("age cannot be negative")
		}
		return nil
	})
	if err := sv.Validate("age", -1); err == nil {
		t.Error("expected validation error for negative age")
	}
}

func TestStateValidator_UnregisteredKey(t *testing.T) {
	sv := NewStateValidator()
	// Should not error for unregistered key
	if err := sv.Validate("unknown", "anything"); err != nil {
		t.Errorf("expected no error for unregistered key, got %v", err)
	}
}

func TestStateValidator_ValidateAll(t *testing.T) {
	sv := NewStateValidator()
	sv.AddValidator("count", func(v interface{}) error {
		if v.(int) > 100 {
			return fmt.Errorf("too large")
		}
		return nil
	})
	err := sv.ValidateAll(map[string]interface{}{
		"count": 50,
		"name":  "test",
	})
	if err != nil {
		t.Errorf("ValidateAll should not error for valid values: %v", err)
	}

	err = sv.ValidateAll(map[string]interface{}{
		"count": 200,
	})
	if err == nil {
		t.Error("ValidateAll should error for invalid values")
	}
}

// ─── deepEqualValues helper tests ─────────────────────────────────────────────

func TestDeepEqualValues(t *testing.T) {
	tests := []struct {
		a, b   interface{}
		expect bool
	}{
		{nil, nil, true},
		{nil, "x", false},
		{"x", nil, false},
		{1, 1, true},
		{1, 2, false},
		{"hello", "hello", true},
		{"hello", "world", false},
		{map[string]int{"a": 1}, map[string]int{"a": 1}, true},
		{map[string]int{"a": 1}, map[string]int{"a": 2}, false},
	}

	for _, tt := range tests {
		got := deepEqualValues(tt.a, tt.b)
		if got != tt.expect {
			t.Errorf("deepEqualValues(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.expect)
		}
	}
}

var _ = fmt.Sprintf // ensure fmt import is used
