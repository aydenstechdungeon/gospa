// Package component provides island architecture support for GoSPA.
// Islands are independently hydratable components with their own state.
package component

import (
	"encoding/json"
	"fmt"
	"sync"
)

// IslandHydrationMode defines when an island should hydrate.
type IslandHydrationMode string

const (
	// HydrationImmediate hydrates as soon as the script loads.
	HydrationImmediate IslandHydrationMode = "immediate"
	// HydrationVisible hydrates when the island enters the viewport.
	HydrationVisible IslandHydrationMode = "visible"
	// HydrationIdle hydrates during browser idle time.
	HydrationIdle IslandHydrationMode = "idle"
	// HydrationInteraction hydrates on first user interaction.
	HydrationInteraction IslandHydrationMode = "interaction"
	// HydrationLazy hydrates when explicitly triggered.
	HydrationLazy IslandHydrationMode = "lazy"
)

// IslandPriority defines the loading priority for an island.
type IslandPriority string

const (
	// PriorityHigh for above-fold critical content.
	PriorityHigh IslandPriority = "high"
	// PriorityNormal for standard content.
	PriorityNormal IslandPriority = "normal"
	// PriorityLow for below-fold or deferred content.
	PriorityLow IslandPriority = "low"
)

// IslandConfig configures an island's behavior.
type IslandConfig struct {
	// Name is the unique identifier for this island type.
	Name string
	// HydrationMode determines when the island hydrates.
	HydrationMode IslandHydrationMode
	// Priority affects loading order.
	Priority IslandPriority
	// ClientOnly skips SSR entirely.
	ClientOnly bool
	// ServerOnly renders HTML without client JS.
	ServerOnly bool
	// LazyThreshold for visible mode - margin in pixels.
	LazyThreshold int
	// DeferDelay for idle mode - max delay in ms.
	DeferDelay int
}

// Island represents a registered island component.
type Island struct {
	ID       string
	Config   IslandConfig
	Props    map[string]any
	State    map[string]any
	Children string // HTML content from SSR
}

// IslandRegistry manages all registered islands.
type IslandRegistry struct {
	mu      sync.RWMutex
	islands map[string]*Island
	configs map[string]IslandConfig
}

// NewIslandRegistry creates a new island registry.
func NewIslandRegistry() *IslandRegistry {
	return &IslandRegistry{
		islands: make(map[string]*Island),
		configs: make(map[string]IslandConfig),
	}
}

// Register registers an island configuration.
func (r *IslandRegistry) Register(config IslandConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if config.Name == "" {
		return fmt.Errorf("island name cannot be empty")
	}

	if _, exists := r.configs[config.Name]; exists {
		return fmt.Errorf("island %q already registered", config.Name)
	}

	// Set defaults
	if config.HydrationMode == "" {
		config.HydrationMode = HydrationImmediate
	}
	if config.Priority == "" {
		config.Priority = PriorityNormal
	}

	r.configs[config.Name] = config
	return nil
}

// Create creates a new island instance.
func (r *IslandRegistry) Create(name string, props map[string]any) (*Island, error) {
	r.mu.RLock()
	config, exists := r.configs[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("island %q not registered", name)
	}

	id := generateIslandID(name)

	island := &Island{
		ID:     id,
		Config: config,
		Props:  props,
		State:  make(map[string]any),
	}

	r.mu.Lock()
	r.islands[id] = island
	r.mu.Unlock()

	return island, nil
}

// Get retrieves an island by ID.
func (r *IslandRegistry) Get(id string) (*Island, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	island, exists := r.islands[id]
	return island, exists
}

// Remove removes an island from the registry.
func (r *IslandRegistry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.islands, id)
}

// GetAll returns all active islands.
func (r *IslandRegistry) GetAll() []*Island {
	r.mu.RLock()
	defer r.mu.RUnlock()

	islands := make([]*Island, 0, len(r.islands))
	for _, island := range r.islands {
		islands = append(islands, island)
	}
	return islands
}

// GetByPriority returns islands grouped by priority.
func (r *IslandRegistry) GetByPriority() map[IslandPriority][]*Island {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := map[IslandPriority][]*Island{
		PriorityHigh:   {},
		PriorityNormal: {},
		PriorityLow:    {},
	}

	for _, island := range r.islands {
		result[island.Config.Priority] = append(result[island.Config.Priority], island)
	}
	return result
}

// SerializeState returns the serialized state for all islands.
func (r *IslandRegistry) SerializeState() (map[string]map[string]any, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	state := make(map[string]map[string]any)
	for id, island := range r.islands {
		state[id] = island.State
	}
	return state, nil
}

// IslandData is the data structure sent to the client.
type IslandData struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Props    map[string]any `json:"props,omitempty"`
	State    map[string]any `json:"state,omitempty"`
	HTML     string         `json:"html,omitempty"`
	Mode     string         `json:"mode"`
	Priority string         `json:"priority"`
}

// ToData converts an island to client-transferable data.
func (i *Island) ToData() IslandData {
	return IslandData{
		ID:       i.ID,
		Name:     i.Config.Name,
		Props:    i.Props,
		State:    i.State,
		HTML:     i.Children,
		Mode:     string(i.Config.HydrationMode),
		Priority: string(i.Config.Priority),
	}
}

// ToJSON serializes island data to JSON.
func (i *Island) ToJSON() ([]byte, error) {
	return json.Marshal(i.ToData())
}

// generateIslandID generates a unique ID for an island instance.
func generateIslandID(name string) string {
	return fmt.Sprintf("island-%s-%d", name, getNextID())
}

var idCounter int64
var idMu sync.Mutex

func getNextID() int64 {
	idMu.Lock()
	defer idMu.Unlock()
	idCounter++
	return idCounter
}

// Global registry instance.
var globalRegistry = NewIslandRegistry()

// RegisterIsland registers an island configuration globally.
func RegisterIsland(config IslandConfig) error {
	return globalRegistry.Register(config)
}

// CreateIsland creates a new island instance in the global registry.
func CreateIsland(name string, props map[string]any) (*Island, error) {
	return globalRegistry.Create(name, props)
}

// GetIsland retrieves an island from the global registry.
func GetIsland(id string) (*Island, bool) {
	return globalRegistry.Get(id)
}

// GetAllIslands returns all islands from the global registry.
func GetAllIslands() []*Island {
	return globalRegistry.GetAll()
}

// GetIslandsByPriority returns islands grouped by priority.
func GetIslandsByPriority() map[IslandPriority][]*Island {
	return globalRegistry.GetByPriority()
}
