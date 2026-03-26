package routing

import (
	"context"
	"sync"
)

// RemoteContext provides HTTP request details to a remote action.
type RemoteContext struct {
	IP        string
	UserAgent string
	RequestID string
	SessionID string
	// Headers is an allowlist of tracing headers only (e.g. X-Request-Id, Traceparent).
	// It does not include Authorization, Cookie, or other sensitive request headers.
	Headers map[string]string
}

// RemoteActionFunc is a type-safe server function that can be called remotely from the client.
type RemoteActionFunc func(ctx context.Context, rc RemoteContext, input interface{}) (interface{}, error)

// RemoteRegistry is a registry for remote actions.
type RemoteRegistry struct {
	mu      sync.RWMutex
	actions map[string]RemoteActionFunc
}

var globalRemoteRegistry = &RemoteRegistry{
	actions: make(map[string]RemoteActionFunc),
}

// RegisterRemoteAction registers a remote server function.
func RegisterRemoteAction(name string, action RemoteActionFunc) {
	globalRemoteRegistry.mu.Lock()
	defer globalRemoteRegistry.mu.Unlock()
	globalRemoteRegistry.actions[name] = action
}

// GetRemoteAction retrieves a registered remote server function.
func GetRemoteAction(name string) (RemoteActionFunc, bool) {
	globalRemoteRegistry.mu.RLock()
	defer globalRemoteRegistry.mu.RUnlock()
	fn, ok := globalRemoteRegistry.actions[name]
	return fn, ok
}

// GetAllActions returns all registered action names.
func GetAllActions() []string {
	globalRemoteRegistry.mu.RLock()
	defer globalRemoteRegistry.mu.RUnlock()
	actions := make([]string, 0, len(globalRemoteRegistry.actions))
	for name := range globalRemoteRegistry.actions {
		actions = append(actions, name)
	}
	return actions
}
