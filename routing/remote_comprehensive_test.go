package routing

import (
	"context"
	"testing"
)

// ─── RemoteAction registration ────────────────────────────────────────────────

func TestRegisterAndGetRemoteAction(t *testing.T) {
	name := "testAction_unique_7a3f"
	RegisterRemoteAction(name, func(ctx context.Context, input interface{}) (interface{}, error) {
		return "result", nil
	})
	fn, ok := GetRemoteAction(name)
	if !ok {
		t.Errorf("GetRemoteAction(%q) should return true after registration", name)
	}
	if fn == nil {
		t.Errorf("GetRemoteAction(%q) should return non-nil function", name)
	}
}

func TestGetRemoteAction_NotFound(t *testing.T) {
	_, ok := GetRemoteAction("nonexistent_action_xyz")
	if ok {
		t.Error("GetRemoteAction should return false for unregistered action")
	}
}

func TestRemoteAction_Invocation(t *testing.T) {
	name := "addAction_unique_7b4f"
	RegisterRemoteAction(name, func(ctx context.Context, input interface{}) (interface{}, error) {
		x := input.(float64)
		return x + 10, nil
	})

	fn, ok := GetRemoteAction(name)
	if !ok {
		t.Fatalf("action %q should be registered", name)
	}

	result, err := fn(context.Background(), float64(5))
	if err != nil {
		t.Fatalf("action invocation failed: %v", err)
	}
	if result != float64(15) {
		t.Errorf("expected 15.0, got %v", result)
	}
}

func TestRemoteAction_WithNilInput(t *testing.T) {
	name := "nilInputAction_unique_9c3e"
	RegisterRemoteAction(name, func(ctx context.Context, input interface{}) (interface{}, error) {
		if input != nil {
			return nil, nil
		}
		return "nil-handled", nil
	})

	fn, _ := GetRemoteAction(name)
	result, err := fn(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "nil-handled" {
		t.Errorf("expected 'nil-handled', got %v", result)
	}
}

func TestRemoteAction_OverwriteExisting(t *testing.T) {
	name := "overwriteAction_unique_1d2e"
	RegisterRemoteAction(name, func(ctx context.Context, input interface{}) (interface{}, error) {
		return "first", nil
	})
	RegisterRemoteAction(name, func(ctx context.Context, input interface{}) (interface{}, error) {
		return "second", nil
	})

	fn, ok := GetRemoteAction(name)
	if !ok {
		t.Fatal("action should exist")
	}
	result, _ := fn(context.Background(), nil)
	if result != "second" {
		t.Errorf("expected overwritten 'second', got %v", result)
	}
}

func TestRemoteAction_ConcurrentRegistration(t *testing.T) {
	done := make(chan struct{})
	go func() {
		for i := 0; i < 50; i++ {
			RegisterRemoteAction("concurrent_remote_action", func(ctx context.Context, input interface{}) (interface{}, error) {
				return nil, nil
			})
		}
		close(done)
	}()

	for i := 0; i < 50; i++ {
		_, _ = GetRemoteAction("concurrent_remote_action")
	}
	<-done
}
