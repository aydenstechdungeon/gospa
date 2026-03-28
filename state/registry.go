package state

import (
	"context"
	"sync"
)

// RegistryKey identifies a value in a context.
type RegistryKey string

// RegistryContextKey is the key used to store the registry in a context.
const RegistryContextKey RegistryKey = "gospa_state_registry"

// IslandData holds the state and props for a specific island instance.
type IslandData struct {
	ID    string                 `json:"id"`
	Props map[string]interface{} `json:"props"`
	State map[string]interface{} `json:"state"`
}

// Registry collects all island data during a single render request.
type Registry struct {
	mu      sync.Mutex
	islands []IslandData
}

// NewRegistry creates a new Registry.
func NewRegistry() *Registry {
	return &Registry{
		islands: make([]IslandData, 0),
	}
}

// Register adds island data to the registry.
func (r *Registry) Register(id string, props, state map[string]interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.islands = append(r.islands, IslandData{
		ID:    id,
		Props: props,
		State: state,
	})
}

// GetData returns all registered island data.
func (r *Registry) GetData() []IslandData {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.islands
}

// FromContext retrieves the registry from a context.
func FromContext(ctx context.Context) *Registry {
	if r, ok := ctx.Value(RegistryContextKey).(*Registry); ok {
		return r
	}
	return nil
}
