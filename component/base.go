package component

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/aydenstechdungeon/gospa/state"
)

// ComponentID is a unique identifier for a component.
type ComponentID string

// ComponentName is the name of a component.
type ComponentName string

// BaseComponent is the base implementation of a component.
type BaseComponent struct {
	id       ComponentID
	name     ComponentName
	state    *state.StateMap
	props    Props
	children []Component
	slots    map[string]Slot
	parent   Component
	ctx      context.Context
	mu       sync.RWMutex
}

// Component is the interface for a component.
type Component interface {
	// ID returns the component's unique identifier.
	ID() ComponentID
	// Name returns the component's name.
	Name() ComponentName
	// State returns the component's state.
	State() *state.StateMap
	// Props returns the component's props.
	Props() Props
	// Children returns the component's children.
	Children() []Component
	// Parent returns the component's parent.
	Parent() Component
	// AddChild adds a child component.
	AddChild(child Component)
	// RemoveChild removes a child component.
	RemoveChild(id ComponentID)
	// GetSlot returns a slot by name.
	GetSlot(name string) Slot
	// SetSlot sets a slot.
	SetSlot(name string, slot Slot)
	// Context returns the component's context.
	Context() context.Context
	// SetContext sets the component's context.
	SetContext(ctx context.Context)
	// ToJSON returns the component's state as JSON.
	ToJSON() (string, error)
	// Clone creates a copy of the component.
	Clone() Component
}

// Slot is a function that renders content.
type Slot func() string

// NewBaseComponent creates a new base component.
func NewBaseComponent(id ComponentID, name ComponentName, opts ...Option) *BaseComponent {
	c := &BaseComponent{
		id:       id,
		name:     name,
		state:    state.NewStateMap(),
		props:    make(Props),
		children: make([]Component, 0),
		slots:    make(map[string]Slot),
		ctx:      context.Background(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// ID returns the component's unique identifier.
func (c *BaseComponent) ID() ComponentID {
	return c.id
}

// Name returns the component's name.
func (c *BaseComponent) Name() ComponentName {
	return c.name
}

// State returns the component's state.
func (c *BaseComponent) State() *state.StateMap {
	return c.state
}

// Props returns the component's props.
func (c *BaseComponent) Props() Props {
	return c.props
}

// Children returns the component's children.
func (c *BaseComponent) Children() []Component {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.children
}

// Parent returns the component's parent.
func (c *BaseComponent) Parent() Component {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.parent
}

// AddChild adds a child component.
func (c *BaseComponent) AddChild(child Component) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.children = append(c.children, child)
	if bc, ok := child.(*BaseComponent); ok {
		bc.parent = c
	}
}

// RemoveChild removes a child component.
func (c *BaseComponent) RemoveChild(id ComponentID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, child := range c.children {
		if child.ID() == id {
			c.children = append(c.children[:i], c.children[i+1:]...)
			break
		}
	}
}

// GetSlot returns a slot by name.
func (c *BaseComponent) GetSlot(name string) Slot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.slots[name]
}

// SetSlot sets a slot.
func (c *BaseComponent) SetSlot(name string, slot Slot) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.slots[name] = slot
}

// Context returns the component's context.
func (c *BaseComponent) Context() context.Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ctx
}

// SetContext sets the component's context.
func (c *BaseComponent) SetContext(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ctx = ctx
}

// ToJSON returns the component's state as JSON.
func (c *BaseComponent) ToJSON() (string, error) {
	return c.state.ToJSON()
}

// Clone creates a copy of the component.
func (c *BaseComponent) Clone() Component {
	c.mu.RLock()
	defer c.mu.RUnlock()

	clone := &BaseComponent{
		id:       c.id + "_clone",
		name:     c.name,
		state:    c.state,
		props:    c.props.Clone(),
		children: make([]Component, len(c.children)),
		slots:    make(map[string]Slot),
		ctx:      c.ctx,
	}

	for i, child := range c.children {
		clone.children[i] = child.Clone()
	}

	for name, slot := range c.slots {
		clone.slots[name] = slot
	}

	return clone
}

// Option is a functional option for creating components.
type Option func(*BaseComponent)

// WithProps sets the component's props.
func WithProps(props Props) Option {
	return func(c *BaseComponent) {
		c.props = props
	}
}

// WithState sets the component's initial state.
func WithState(initialState map[string]interface{}) Option {
	return func(c *BaseComponent) {
		for key, value := range initialState {
			c.state.AddAny(key, value)
		}
	}
}

// WithChildren sets the component's children.
func WithChildren(children ...Component) Option {
	return func(c *BaseComponent) {
		for _, child := range children {
			c.AddChild(child)
		}
	}
}

// WithParent sets the component's parent.
func WithParent(parent Component) Option {
	return func(c *BaseComponent) {
		c.parent = parent
	}
}

// WithContext sets the component's context.
func WithContext(ctx context.Context) Option {
	return func(c *BaseComponent) {
		c.ctx = ctx
	}
}

// WithSlots sets the component's slots.
func WithSlots(slots map[string]Slot) Option {
	return func(c *BaseComponent) {
		c.slots = slots
	}
}

// ComponentTree represents a tree of components.
type ComponentTree struct {
	root      Component
	lookup    map[ComponentID]Component
	mu        sync.RWMutex
	onMount   map[ComponentID]LifecycleHook
	onUpdate  map[ComponentID]LifecycleHook
	onDestroy map[ComponentID]LifecycleHook
}

// LifecycleHook is a function called during component lifecycle.
type LifecycleHook func(Component)

// NewComponentTree creates a new component tree.
func NewComponentTree(root Component) *ComponentTree {
	tree := &ComponentTree{
		root:      root,
		lookup:    make(map[ComponentID]Component),
		onMount:   make(map[ComponentID]LifecycleHook),
		onUpdate:  make(map[ComponentID]LifecycleHook),
		onDestroy: make(map[ComponentID]LifecycleHook),
	}
	tree.buildLookup(root)
	return tree
}

// buildLookup builds the component lookup map.
func (t *ComponentTree) buildLookup(c Component) {
	t.lookup[c.ID()] = c
	for _, child := range c.Children() {
		t.buildLookup(child)
	}
}

// Root returns the root component.
func (t *ComponentTree) Root() Component {
	return t.root
}

// Get returns a component by ID.
func (t *ComponentTree) Get(id ComponentID) Component {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lookup[id]
}

// Add adds a component to the tree.
func (t *ComponentTree) Add(parentID ComponentID, child Component) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	parent, ok := t.lookup[parentID]
	if !ok {
		return fmt.Errorf("parent component not found: %s", parentID)
	}

	parent.AddChild(child)
	t.lookup[child.ID()] = child
	return nil
}

// Remove removes a component from the tree.
func (t *ComponentTree) Remove(id ComponentID) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	component, ok := t.lookup[id]
	if !ok {
		return fmt.Errorf("component not found: %s", id)
	}

	// Call onDestroy hook
	if hook, exists := t.onDestroy[id]; exists {
		hook(component)
	}

	// Remove from parent
	if parent := component.Parent(); parent != nil {
		parent.RemoveChild(id)
	}

	// Remove from lookup
	delete(t.lookup, id)

	// Remove children recursively
	t.removeChildren(component)

	return nil
}

// removeChildren removes all children of a component.
func (t *ComponentTree) removeChildren(c Component) {
	for _, child := range c.Children() {
		// Call onDestroy hook
		if hook, exists := t.onDestroy[child.ID()]; exists {
			hook(child)
		}
		delete(t.lookup, child.ID())
		t.removeChildren(child)
	}
}

// OnMount registers a mount hook for a component.
func (t *ComponentTree) OnMount(id ComponentID, hook LifecycleHook) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onMount[id] = hook
}

// OnUpdate registers an update hook for a component.
func (t *ComponentTree) OnUpdate(id ComponentID, hook LifecycleHook) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onUpdate[id] = hook
}

// OnDestroy registers a destroy hook for a component.
func (t *ComponentTree) OnDestroy(id ComponentID, hook LifecycleHook) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onDestroy[id] = hook
}

// Mount calls the mount hook for a component.
func (t *ComponentTree) Mount(id ComponentID) {
	t.mu.RLock()
	hook, exists := t.onMount[id]
	component := t.lookup[id]
	t.mu.RUnlock()

	if exists && component != nil {
		hook(component)
	}
}

// Update calls the update hook for a component.
func (t *ComponentTree) Update(id ComponentID) {
	t.mu.RLock()
	hook, exists := t.onUpdate[id]
	component := t.lookup[id]
	t.mu.RUnlock()

	if exists && component != nil {
		hook(component)
	}
}

// Walk walks the component tree.
func (t *ComponentTree) Walk(fn func(Component) bool) {
	t.walk(t.root, fn)
}

// walk recursively walks the component tree.
func (t *ComponentTree) walk(c Component, fn func(Component) bool) {
	if !fn(c) {
		return
	}
	for _, child := range c.Children() {
		t.walk(child, fn)
	}
}

// ToJSON returns the entire tree's state as JSON.
func (t *ComponentTree) ToJSON() (string, error) {
	states := make(map[string]interface{})
	t.Walk(func(c Component) bool {
		if state := c.State(); state != nil {
			var data map[string]interface{}
			if jsonData, err := state.ToJSON(); err == nil {
				_ = json.Unmarshal([]byte(jsonData), &data)
				states[string(c.ID())] = data
			}
		}
		return true
	})

	result, err := json.Marshal(states)
	return string(result), err
}

// Find finds a component by predicate.
func (t *ComponentTree) Find(predicate func(Component) bool) Component {
	var found Component
	t.Walk(func(c Component) bool {
		if predicate(c) {
			found = c
			return false
		}
		return true
	})
	return found
}

// FindAll finds all components matching a predicate.
func (t *ComponentTree) FindAll(predicate func(Component) bool) []Component {
	var found []Component
	t.Walk(func(c Component) bool {
		if predicate(c) {
			found = append(found, c)
		}
		return true
	})
	return found
}

// FindByName finds components by name.
func (t *ComponentTree) FindByName(name ComponentName) []Component {
	return t.FindAll(func(c Component) bool {
		return c.Name() == name
	})
}

// FindByProp finds components by prop value.
func (t *ComponentTree) FindByProp(key string, value interface{}) []Component {
	return t.FindAll(func(c Component) bool {
		return c.Props().Get(key) == value
	})
}
