// Package component provides lifecycle management for components.
// Lifecycle hooks allow components to react to mount, update, and destroy events.
package component

import (
	"sync"
)

// LifecyclePhase represents the current phase of a component's lifecycle
type LifecyclePhase int

const (
	// PhaseCreated is the initial state after component creation
	PhaseCreated LifecyclePhase = iota
	// PhaseMounting is when the component is being mounted
	PhaseMounting
	// PhaseMounted is when the component is fully mounted
	PhaseMounted
	// PhaseUpdating is when the component is updating
	PhaseUpdating
	// PhaseUpdated is when the component has finished updating
	PhaseUpdated
	// PhaseDestroying is when the component is being destroyed
	PhaseDestroying
	// PhaseDestroyed is when the component is fully destroyed
	PhaseDestroyed
)

// String returns the string representation of the lifecycle phase
func (p LifecyclePhase) String() string {
	switch p {
	case PhaseCreated:
		return "created"
	case PhaseMounting:
		return "mounting"
	case PhaseMounted:
		return "mounted"
	case PhaseUpdating:
		return "updating"
	case PhaseUpdated:
		return "updated"
	case PhaseDestroying:
		return "destroying"
	case PhaseDestroyed:
		return "destroyed"
	default:
		return "unknown"
	}
}

// Hook is a function that runs during a lifecycle phase
type Hook func()

// CleanupHook is a function that runs during cleanup
type CleanupHook func()

// Lifecycle manages component lifecycle hooks and state
type Lifecycle struct {
	mu sync.RWMutex

	// Current phase
	phase LifecyclePhase

	// Hooks for each phase
	onBeforeMount   []Hook
	onMount         []Hook
	onBeforeUpdate  []Hook
	onUpdate        []Hook
	onBeforeDestroy []Hook
	onDestroy       []Hook

	// Cleanup hooks (run on destroy)
	cleanupHooks []CleanupHook

	// Track if mounted
	mounted bool
}

// NewLifecycle creates a new lifecycle manager
func NewLifecycle() *Lifecycle {
	return &Lifecycle{
		phase:           PhaseCreated,
		onBeforeMount:   make([]Hook, 0),
		onMount:         make([]Hook, 0),
		onBeforeUpdate:  make([]Hook, 0),
		onUpdate:        make([]Hook, 0),
		onBeforeDestroy: make([]Hook, 0),
		onDestroy:       make([]Hook, 0),
		cleanupHooks:    make([]CleanupHook, 0),
	}
}

// Phase returns the current lifecycle phase
func (l *Lifecycle) Phase() LifecyclePhase {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.phase
}

// IsMounted returns true if the component is mounted
func (l *Lifecycle) IsMounted() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.mounted
}

// OnBeforeMount registers a hook to run before mounting
func (l *Lifecycle) OnBeforeMount(hook Hook) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.onBeforeMount = append(l.onBeforeMount, hook)
}

// OnMount registers a hook to run after mounting
func (l *Lifecycle) OnMount(hook Hook) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.onMount = append(l.onMount, hook)
}

// OnBeforeUpdate registers a hook to run before updating
func (l *Lifecycle) OnBeforeUpdate(hook Hook) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.onBeforeUpdate = append(l.onBeforeUpdate, hook)
}

// OnUpdate registers a hook to run after updating
func (l *Lifecycle) OnUpdate(hook Hook) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.onUpdate = append(l.onUpdate, hook)
}

// OnBeforeDestroy registers a hook to run before destroying
func (l *Lifecycle) OnBeforeDestroy(hook Hook) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.onBeforeDestroy = append(l.onBeforeDestroy, hook)
}

// OnDestroy registers a hook to run after destroying
func (l *Lifecycle) OnDestroy(hook Hook) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.onDestroy = append(l.onDestroy, hook)
}

// OnCleanup registers a cleanup hook to run on destroy
func (l *Lifecycle) OnCleanup(hook CleanupHook) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cleanupHooks = append(l.cleanupHooks, hook)
}

// Mount triggers the mount lifecycle
func (l *Lifecycle) Mount() {
	l.mu.Lock()

	// Check if already mounted
	if l.mounted {
		l.mu.Unlock()
		return
	}

	// Set phase to mounting
	l.phase = PhaseMounting
	beforeMountHooks := make([]Hook, len(l.onBeforeMount))
	copy(beforeMountHooks, l.onBeforeMount)
	l.mu.Unlock()

	// Run before mount hooks
	for _, hook := range beforeMountHooks {
		if hook != nil {
			hook()
		}
	}

	l.mu.Lock()
	l.phase = PhaseMounted
	l.mounted = true
	mountHooks := make([]Hook, len(l.onMount))
	copy(mountHooks, l.onMount)
	l.mu.Unlock()

	// Run mount hooks
	for _, hook := range mountHooks {
		if hook != nil {
			hook()
		}
	}
}

// Update triggers the update lifecycle
func (l *Lifecycle) Update() {
	l.mu.Lock()

	// Must be mounted to update
	if !l.mounted {
		l.mu.Unlock()
		return
	}

	// Set phase to updating
	l.phase = PhaseUpdating
	beforeUpdateHooks := make([]Hook, len(l.onBeforeUpdate))
	copy(beforeUpdateHooks, l.onBeforeUpdate)
	l.mu.Unlock()

	// Run before update hooks
	for _, hook := range beforeUpdateHooks {
		if hook != nil {
			hook()
		}
	}

	l.mu.Lock()
	l.phase = PhaseUpdated
	updateHooks := make([]Hook, len(l.onUpdate))
	copy(updateHooks, l.onUpdate)
	l.mu.Unlock()

	// Run update hooks
	for _, hook := range updateHooks {
		if hook != nil {
			hook()
		}
	}
}

// Destroy triggers the destroy lifecycle
func (l *Lifecycle) Destroy() {
	l.mu.Lock()

	// Check if already destroyed
	if l.phase == PhaseDestroyed {
		l.mu.Unlock()
		return
	}

	// Set phase to destroying
	l.phase = PhaseDestroying
	beforeDestroyHooks := make([]Hook, len(l.onBeforeDestroy))
	copy(beforeDestroyHooks, l.onBeforeDestroy)
	cleanupHooks := make([]CleanupHook, len(l.cleanupHooks))
	copy(cleanupHooks, l.cleanupHooks)
	l.mu.Unlock()

	// Run before destroy hooks
	for _, hook := range beforeDestroyHooks {
		if hook != nil {
			hook()
		}
	}

	// Run cleanup hooks
	for _, hook := range cleanupHooks {
		if hook != nil {
			hook()
		}
	}

	l.mu.Lock()
	l.phase = PhaseDestroyed
	l.mounted = false
	destroyHooks := make([]Hook, len(l.onDestroy))
	copy(destroyHooks, l.onDestroy)
	l.mu.Unlock()

	// Run destroy hooks
	for _, hook := range destroyHooks {
		if hook != nil {
			hook()
		}
	}
}

// ClearHooks removes all hooks
func (l *Lifecycle) ClearHooks() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.onBeforeMount = make([]Hook, 0)
	l.onMount = make([]Hook, 0)
	l.onBeforeUpdate = make([]Hook, 0)
	l.onUpdate = make([]Hook, 0)
	l.onBeforeDestroy = make([]Hook, 0)
	l.onDestroy = make([]Hook, 0)
	l.cleanupHooks = make([]CleanupHook, 0)
}

// LifecycleAware is an interface for components with lifecycle
type LifecycleAware interface {
	// Lifecycle returns the lifecycle manager
	Lifecycle() *Lifecycle
}

// MountComponent mounts a component and its children
func MountComponent(c Component) {
	if c == nil {
		return
	}

	// Mount children first (depth-first)
	for _, child := range c.Children() {
		MountComponent(child)
	}

	// Mount self
	if lc, ok := c.(LifecycleAware); ok {
		lc.Lifecycle().Mount()
	}
}

// DestroyComponent destroys a component and its children
func DestroyComponent(c Component) {
	if c == nil {
		return
	}

	// Destroy self first
	if lc, ok := c.(LifecycleAware); ok {
		lc.Lifecycle().Destroy()
	}

	// Destroy children
	for _, child := range c.Children() {
		DestroyComponent(child)
	}
}

// UpdateComponent updates a component
func UpdateComponent(c Component) {
	if c == nil {
		return
	}

	if lc, ok := c.(LifecycleAware); ok {
		lc.Lifecycle().Update()
	}
}
