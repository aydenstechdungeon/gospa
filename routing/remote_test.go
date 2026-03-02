package routing

import (
	"context"
	"errors"
	"testing"
)

func TestRegisterRemoteAction(t *testing.T) {
	// Clear the registry before test
	globalRemoteRegistry.mu.Lock()
	globalRemoteRegistry.actions = make(map[string]RemoteActionFunc)
	globalRemoteRegistry.mu.Unlock()

	// Register a simple action
	RegisterRemoteAction("testAction", func(ctx context.Context, input interface{}) (interface{}, error) {
		return "result", nil
	})

	// Verify it was registered
	fn, ok := GetRemoteAction("testAction")
	if !ok {
		t.Error("Expected action to be registered")
	}

	// Verify it works
	result, err := fn(context.Background(), nil)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "result" {
		t.Errorf("Expected 'result', got %v", result)
	}
}

func TestGetRemoteActionNotFound(t *testing.T) {
	// Clear the registry
	globalRemoteRegistry.mu.Lock()
	globalRemoteRegistry.actions = make(map[string]RemoteActionFunc)
	globalRemoteRegistry.mu.Unlock()

	// Try to get non-existent action
	_, ok := GetRemoteAction("nonExistent")
	if ok {
		t.Error("Expected action to not be found")
	}
}

func TestRemoteActionWithInput(t *testing.T) {
	// Clear the registry
	globalRemoteRegistry.mu.Lock()
	globalRemoteRegistry.actions = make(map[string]RemoteActionFunc)
	globalRemoteRegistry.mu.Unlock()

	// Register action that processes input
	RegisterRemoteAction("addOne", func(ctx context.Context, input interface{}) (interface{}, error) {
		if num, ok := input.(float64); ok {
			return num + 1, nil
		}
		if num, ok := input.(int); ok {
			return num + 1, nil
		}
		return nil, errors.New("invalid input type")
	})

	fn, ok := GetRemoteAction("addOne")
	if !ok {
		t.Fatal("Expected action to be registered")
	}

	// Test with float64 (JSON numbers)
	result, err := fn(context.Background(), float64(5))
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != float64(6) {
		t.Errorf("Expected 6, got %v", result)
	}
}

func TestRemoteActionWithError(t *testing.T) {
	// Clear the registry
	globalRemoteRegistry.mu.Lock()
	globalRemoteRegistry.actions = make(map[string]RemoteActionFunc)
	globalRemoteRegistry.mu.Unlock()

	// Register action that returns error
	RegisterRemoteAction("errorAction", func(ctx context.Context, input interface{}) (interface{}, error) {
		return nil, errors.New("something went wrong")
	})

	fn, ok := GetRemoteAction("errorAction")
	if !ok {
		t.Fatal("Expected action to be registered")
	}

	result, err := fn(context.Background(), nil)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if err.Error() != "something went wrong" {
		t.Errorf("Expected specific error message, got %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil result on error, got %v", result)
	}
}

func TestRemoteActionWithContext(t *testing.T) {
	// Clear the registry
	globalRemoteRegistry.mu.Lock()
	globalRemoteRegistry.actions = make(map[string]RemoteActionFunc)
	globalRemoteRegistry.mu.Unlock()

	// Register action that checks context
	RegisterRemoteAction("contextAction", func(ctx context.Context, input interface{}) (interface{}, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return "success", nil
		}
	})

	fn, ok := GetRemoteAction("contextAction")
	if !ok {
		t.Fatal("Expected action to be registered")
	}

	// Test with active context
	result, err := fn(context.Background(), nil)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "success" {
		t.Errorf("Expected 'success', got %v", result)
	}

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = fn(ctx, nil)
	if err == nil {
		t.Error("Expected context cancelled error")
	}
}

func TestMultipleRemoteActions(t *testing.T) {
	// Clear the registry
	globalRemoteRegistry.mu.Lock()
	globalRemoteRegistry.actions = make(map[string]RemoteActionFunc)
	globalRemoteRegistry.mu.Unlock()

	// Register multiple actions
	RegisterRemoteAction("action1", func(ctx context.Context, input interface{}) (interface{}, error) {
		return "result1", nil
	})
	RegisterRemoteAction("action2", func(ctx context.Context, input interface{}) (interface{}, error) {
		return "result2", nil
	})
	RegisterRemoteAction("action3", func(ctx context.Context, input interface{}) (interface{}, error) {
		return "result3", nil
	})

	// Verify all are accessible
	fn1, ok1 := GetRemoteAction("action1")
	fn2, ok2 := GetRemoteAction("action2")
	fn3, ok3 := GetRemoteAction("action3")

	if !ok1 || !ok2 || !ok3 {
		t.Error("Expected all actions to be registered")
	}

	result1, _ := fn1(context.Background(), nil)
	result2, _ := fn2(context.Background(), nil)
	result3, _ := fn3(context.Background(), nil)

	if result1 != "result1" || result2 != "result2" || result3 != "result3" {
		t.Error("Expected correct results from all actions")
	}
}

func TestOverwriteRemoteAction(t *testing.T) {
	// Clear the registry
	globalRemoteRegistry.mu.Lock()
	globalRemoteRegistry.actions = make(map[string]RemoteActionFunc)
	globalRemoteRegistry.mu.Unlock()

	// Register initial action
	RegisterRemoteAction("overwrite", func(ctx context.Context, input interface{}) (interface{}, error) {
		return "first", nil
	})

	// Overwrite with new action
	RegisterRemoteAction("overwrite", func(ctx context.Context, input interface{}) (interface{}, error) {
		return "second", nil
	})

	fn, ok := GetRemoteAction("overwrite")
	if !ok {
		t.Fatal("Expected action to be registered")
	}

	result, _ := fn(context.Background(), nil)
	if result != "second" {
		t.Error("Expected overwritten action to return 'second'")
	}
}

func TestRemoteActionConcurrentAccess(t *testing.T) {
	// Clear the registry
	globalRemoteRegistry.mu.Lock()
	globalRemoteRegistry.actions = make(map[string]RemoteActionFunc)
	globalRemoteRegistry.mu.Unlock()

	// Register action
	RegisterRemoteAction("concurrent", func(ctx context.Context, input interface{}) (interface{}, error) {
		return "ok", nil
	})

	// Concurrent reads
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			fn, ok := GetRemoteAction("concurrent")
			if !ok {
				t.Error("Expected action to be found")
			}
			result, err := fn(context.Background(), nil)
			if err != nil || result != "ok" {
				t.Error("Expected successful execution")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
