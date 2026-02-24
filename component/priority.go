// Package component provides priority-based hydration for GoSPA islands.
// This enables intelligent loading order based on component importance.
package component

import (
	"encoding/json"
	"sort"
	"sync"
)

// PriorityLevel is a numeric priority for hydration ordering.
// Higher values hydrate first.
type PriorityLevel int

const (
	// PriorityLevelCritical - Above the fold, immediately visible, interactive.
	PriorityLevelCritical PriorityLevel = 100
	// PriorityLevelHigh - Important UI elements, likely to be interacted with.
	PriorityLevelHigh PriorityLevel = 75
	// PriorityLevelNormal - Standard content, below the fold.
	PriorityLevelNormal PriorityLevel = 50
	// PriorityLevelLow - Non-essential content, ads, recommendations.
	PriorityLevelLow PriorityLevel = 25
	// PriorityLevelDeferred - Extremely low priority, may never hydrate.
	PriorityLevelDeferred PriorityLevel = 10
)

// PriorityIsland represents an island with extended priority information.
type PriorityIsland struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Priority     PriorityLevel       `json:"priority"`
	Mode         IslandHydrationMode `json:"mode"`
	Dependencies []string            `json:"dependencies,omitempty"`
	State        json.RawMessage     `json:"state,omitempty"`
	Script       string              `json:"script,omitempty"`
	Position     int                 `json:"position"`
	Metadata     map[string]any      `json:"metadata,omitempty"`
}

// PriorityConfig configures priority-based hydration behavior.
type PriorityConfig struct {
	// MaxConcurrent limits simultaneous hydrations.
	MaxConcurrent int `json:"maxConcurrent"`
	// IdleTimeout is the deadline for idle callbacks (ms).
	IdleTimeout int `json:"idleTimeout"`
	// IntersectionThreshold for visible strategy (0-1).
	IntersectionThreshold float64 `json:"intersectionThreshold"`
	// IntersectionRootMargin for visible strategy.
	IntersectionRootMargin string `json:"intersectionRootMargin"`
	// EnablePreload enables preloading of island scripts.
	EnablePreload bool `json:"enablePreload"`
}

// DefaultPriorityConfig returns sensible defaults.
func DefaultPriorityConfig() PriorityConfig {
	return PriorityConfig{
		MaxConcurrent:          3,
		IdleTimeout:            2000,
		IntersectionThreshold:  0.1,
		IntersectionRootMargin: "50px",
		EnablePreload:          true,
	}
}

// PriorityQueue manages islands ordered by priority.
type PriorityQueue struct {
	mu      sync.RWMutex
	islands []*PriorityIsland
	config  PriorityConfig
}

// NewPriorityQueue creates a new priority queue.
func NewPriorityQueue(config PriorityConfig) *PriorityQueue {
	return &PriorityQueue{
		islands: make([]*PriorityIsland, 0),
		config:  config,
	}
}

// Register adds an island to the priority queue.
func (pq *PriorityQueue) Register(island *PriorityIsland) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	// Check for existing
	for i, existing := range pq.islands {
		if existing.ID == island.ID {
			pq.islands[i] = island
			return
		}
	}

	pq.islands = append(pq.islands, island)
}

// RegisterBatch adds multiple islands at once.
func (pq *PriorityQueue) RegisterBatch(islands []*PriorityIsland) {
	for _, island := range islands {
		pq.Register(island)
	}
}

// GetOrdered returns islands sorted by priority (highest first).
func (pq *PriorityQueue) GetOrdered() []*PriorityIsland {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	// Copy slice
	result := make([]*PriorityIsland, len(pq.islands))
	copy(result, pq.islands)

	// Sort by priority (descending), then by position (ascending)
	sort.Slice(result, func(i, j int) bool {
		if result[i].Priority != result[j].Priority {
			return result[i].Priority > result[j].Priority
		}
		return result[i].Position < result[j].Position
	})

	return result
}

// GetByMode returns islands filtered by hydration mode.
func (pq *PriorityQueue) GetByMode(mode IslandHydrationMode) []*PriorityIsland {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	var result []*PriorityIsland
	for _, island := range pq.islands {
		if island.Mode == mode {
			result = append(result, island)
		}
	}

	// Sort by priority
	sort.Slice(result, func(i, j int) bool {
		return result[i].Priority > result[j].Priority
	})

	return result
}

// GetCritical returns all critical priority islands.
func (pq *PriorityQueue) GetCritical() []*PriorityIsland {
	return pq.GetByMinPriority(PriorityLevelCritical)
}

// GetByMinPriority returns islands at or above the given priority.
func (pq *PriorityQueue) GetByMinPriority(minPriority PriorityLevel) []*PriorityIsland {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	var result []*PriorityIsland
	for _, island := range pq.islands {
		if island.Priority >= minPriority {
			result = append(result, island)
		}
	}

	// Sort by priority
	sort.Slice(result, func(i, j int) bool {
		return result[i].Priority > result[j].Priority
	})

	return result
}

// GetDependencyOrder returns islands in dependency-resolved order.
func (pq *PriorityQueue) GetDependencyOrder() ([]*PriorityIsland, error) {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	// Build dependency graph
	graph := make(map[string][]string)
	islandMap := make(map[string]*PriorityIsland)

	for _, island := range pq.islands {
		islandMap[island.ID] = island
		graph[island.ID] = island.Dependencies
	}

	// Topological sort with priority consideration
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	var result []*PriorityIsland

	// Sort by priority first
	sortedIDs := make([]string, 0, len(pq.islands))
	for _, island := range pq.islands {
		sortedIDs = append(sortedIDs, island.ID)
	}
	sort.Slice(sortedIDs, func(i, j int) bool {
		return islandMap[sortedIDs[i]].Priority > islandMap[sortedIDs[j]].Priority
	})

	var visit func(id string) error
	visit = func(id string) error {
		if visited[id] {
			return nil
		}
		if visiting[id] {
			return &CircularDependencyError{ID: id}
		}

		visiting[id] = true

		for _, dep := range graph[id] {
			if _, exists := islandMap[dep]; exists {
				if err := visit(dep); err != nil {
					return err
				}
			}
		}

		visiting[id] = false
		visited[id] = true

		if island, exists := islandMap[id]; exists {
			result = append(result, island)
		}

		return nil
	}

	for _, id := range sortedIDs {
		if err := visit(id); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// CircularDependencyError indicates a circular dependency was detected.
type CircularDependencyError struct {
	ID string
}

func (e *CircularDependencyError) Error() string {
	return "circular dependency detected involving island: " + e.ID
}

// HydrationPlan represents a complete hydration schedule.
type HydrationPlan struct {
	// Immediate islands to hydrate right away.
	Immediate []*PriorityIsland `json:"immediate"`
	// Idle islands to hydrate during idle time.
	Idle []*PriorityIsland `json:"idle"`
	// Visible islands to hydrate on viewport entry.
	Visible []*PriorityIsland `json:"visible"`
	// Interaction islands to hydrate on user interaction.
	Interaction []*PriorityIsland `json:"interaction"`
	// Lazy islands to hydrate when resources permit.
	Lazy []*PriorityIsland `json:"lazy"`
	// Preload scripts for high-priority islands.
	Preload []string `json:"preload"`
}

// CreatePlan generates a hydration plan from the queue.
func (pq *PriorityQueue) CreatePlan() *HydrationPlan {
	plan := &HydrationPlan{
		Immediate:   make([]*PriorityIsland, 0),
		Idle:        make([]*PriorityIsland, 0),
		Visible:     make([]*PriorityIsland, 0),
		Interaction: make([]*PriorityIsland, 0),
		Lazy:        make([]*PriorityIsland, 0),
		Preload:     make([]string, 0),
	}

	// Get islands in dependency order
	ordered, err := pq.GetDependencyOrder()
	if err != nil {
		// Fall back to priority order on error
		ordered = pq.GetOrdered()
	}

	preloadSet := make(map[string]bool)

	for _, island := range ordered {
		switch island.Mode {
		case HydrationImmediate:
			plan.Immediate = append(plan.Immediate, island)
		case HydrationIdle:
			plan.Idle = append(plan.Idle, island)
		case HydrationVisible:
			plan.Visible = append(plan.Visible, island)
		case HydrationInteraction:
			plan.Interaction = append(plan.Interaction, island)
		case HydrationLazy:
			plan.Lazy = append(plan.Lazy, island)
		}

		// Add to preload list for high priority
		if pq.config.EnablePreload && island.Priority >= PriorityLevelHigh && island.Script != "" {
			if !preloadSet[island.Script] {
				plan.Preload = append(plan.Preload, island.Script)
				preloadSet[island.Script] = true
			}
		}
	}

	return plan
}

// GetConfig returns the current configuration.
func (pq *PriorityQueue) GetConfig() PriorityConfig {
	return pq.config
}

// Clear removes all islands from the queue.
func (pq *PriorityQueue) Clear() {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.islands = make([]*PriorityIsland, 0)
}

// Count returns the number of islands in the queue.
func (pq *PriorityQueue) Count() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	return len(pq.islands)
}

// PriorityFromIslandPriority converts IslandPriority to PriorityLevel.
func PriorityFromIslandPriority(p IslandPriority) PriorityLevel {
	switch p {
	case PriorityHigh:
		return PriorityLevelHigh
	case PriorityLow:
		return PriorityLevelLow
	default:
		return PriorityLevelNormal
	}
}

// IslandPriorityFromLevel converts PriorityLevel to IslandPriority.
func IslandPriorityFromLevel(p PriorityLevel) IslandPriority {
	switch {
	case p >= PriorityLevelCritical:
		return PriorityHigh
	case p <= PriorityLevelLow:
		return PriorityLow
	default:
		return PriorityNormal
	}
}

// PriorityIslandFromIsland converts an Island to a PriorityIsland.
func PriorityIslandFromIsland(island *Island, position int) *PriorityIsland {
	state, _ := island.ToJSON()
	return &PriorityIsland{
		ID:       island.ID,
		Name:     island.Config.Name,
		Priority: PriorityFromIslandPriority(island.Config.Priority),
		Mode:     island.Config.HydrationMode,
		State:    state,
		Position: position,
		Metadata: make(map[string]any),
	}
}

// Global priority queue instance.
var globalPriorityQueue = NewPriorityQueue(DefaultPriorityConfig())

// RegisterPriorityIsland registers an island in the global priority queue.
func RegisterPriorityIsland(island *PriorityIsland) {
	globalPriorityQueue.Register(island)
}

// GetPriorityQueue returns the global priority queue.
func GetPriorityQueue() *PriorityQueue {
	return globalPriorityQueue
}

// CreateHydrationPlan creates a hydration plan from the global queue.
func CreateHydrationPlan() *HydrationPlan {
	return globalPriorityQueue.CreatePlan()
}
