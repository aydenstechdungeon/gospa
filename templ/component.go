// Package templ provides component helpers for GoSPA.
package templ

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/a-h/templ"
	"github.com/aydenstechdungeon/gospa/state"
)

// Snippet defines a reusable template chunk with typed parameters
type Snippet[T any] func(params T) templ.Component

// ErrorBoundary catches rendering errors from content and renders a fallback UI instead
func ErrorBoundary(content templ.Component, fallback func(error) templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		var buf bytes.Buffer
		err := content.Render(ctx, &buf)
		if err != nil {
			if fallback != nil {
				return fallback(err).Render(ctx, w)
			}
			// if no fallback, we still return the err
			return err
		}
		_, writeErr := io.Copy(w, &buf)
		return writeErr
	})
}

// componentIDCounter is a global counter for generating unique component IDs.
var componentIDCounter atomic.Uint64

// Component represents a GoSPA component with reactive state.
type Component struct {
	// ID is the unique component identifier.
	ID string
	// Name is the component name for debugging.
	Name string
	// State holds the reactive state.
	State *ComponentState
	// Props are the component's input properties.
	Props map[string]any
	// Children is the component's children snippet.
	Children templ.Component
	// Slots are named slots for composition.
	Slots map[string]templ.Component
	// Lifecycle hooks.
	onMount   func()
	onUpdate  func()
	onDestroy func()
}

// ComponentOption is a function that configures a component.
type ComponentOption func(*Component)

// WithProps sets the component props.
func WithProps(props map[string]any) ComponentOption {
	return func(c *Component) {
		c.Props = props
	}
}

// WithChildren sets the component children.
func WithChildren(children templ.Component) ComponentOption {
	return func(c *Component) {
		c.Children = children
	}
}

// WithSlot sets a named slot.
func WithSlot(name string, content templ.Component) ComponentOption {
	return func(c *Component) {
		if c.Slots == nil {
			c.Slots = make(map[string]templ.Component)
		}
		c.Slots[name] = content
	}
}

// WithSlots sets multiple named slots.
func WithSlots(slots map[string]templ.Component) ComponentOption {
	return func(c *Component) {
		if c.Slots == nil {
			c.Slots = make(map[string]templ.Component)
		}
		for name, content := range slots {
			c.Slots[name] = content
		}
	}
}

// OnMount sets the mount lifecycle hook.
func OnMount(fn func()) ComponentOption {
	return func(c *Component) {
		c.onMount = fn
	}
}

// OnUpdate sets the update lifecycle hook.
func OnUpdate(fn func()) ComponentOption {
	return func(c *Component) {
		c.onUpdate = fn
	}
}

// OnDestroy sets the destroy lifecycle hook.
func OnDestroy(fn func()) ComponentOption {
	return func(c *Component) {
		c.onDestroy = fn
	}
}

// NewComponent creates a new component with the given name and options.
func NewComponent(name string, opts ...ComponentOption) *Component {
	id := fmt.Sprintf("%s-%d", name, componentIDCounter.Add(1))
	c := &Component{
		ID:    id,
		Name:  name,
		State: NewComponentState(id),
		Props: make(map[string]any),
		Slots: make(map[string]templ.Component),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// AddState adds a reactive state value to the component.
func (c *Component) AddState(key string, value any) *state.Rune[any] {
	r := state.NewRune(value)
	c.State.AddRune(key, r)
	return r
}

// GetState gets a state value by key.
func (c *Component) GetState(key string) (any, bool) {
	if r, ok := c.State.GetRune(key); ok {
		return r.Get(), true
	}
	return nil, false
}

// SetState sets a state value by key.
func (c *Component) SetState(key string, value any) {
	if r, ok := c.State.GetRune(key); ok {
		r.Set(value)
	}
}

// UpdateState updates a state value using a function.
func (c *Component) UpdateState(key string, fn func(any) any) {
	if r, ok := c.State.GetRune(key); ok {
		r.Update(fn)
	}
}

// GetProp gets a prop value by key with a default.
func (c *Component) GetProp(key string, defaultValue any) any {
	if v, ok := c.Props[key]; ok {
		return v
	}
	return defaultValue
}

// GetSlot gets a slot by name.
func (c *Component) GetSlot(name string) templ.Component {
	if c.Slots != nil {
		return c.Slots[name]
	}
	return nil
}

// HasSlot checks if a slot exists.
func (c *Component) HasSlot(name string) bool {
	_, ok := c.Slots[name]
	return ok
}

// Render renders the component's children.
func (c *Component) Render(ctx context.Context, w io.Writer) error {
	if c.Children != nil {
		return c.Children.Render(ctx, w)
	}
	return nil
}

// RenderSlot renders a named slot.
func (c *Component) RenderSlot(ctx context.Context, w io.Writer, name string) error {
	if slot, ok := c.Slots[name]; ok {
		return slot.Render(ctx, w)
	}
	return nil
}

// Attrs returns the component's data attributes.
func (c *Component) Attrs() templ.Attributes {
	return c.State.StateAttrs()
}

// InitScript returns the component initialization script.
func (c *Component) InitScript() templ.Component {
	return c.State.InitScript()
}

// ToJSON serializes the component state to JSON.
func (c *Component) ToJSON() (string, error) {
	return c.State.ToJSON()
}

// Mount triggers the mount lifecycle hook.
func (c *Component) Mount() {
	if c.onMount != nil {
		c.onMount()
	}
}

// Update triggers the update lifecycle hook.
func (c *Component) Update() {
	if c.onUpdate != nil {
		c.onUpdate()
	}
}

// Destroy triggers the destroy lifecycle hook.
func (c *Component) Destroy() {
	if c.onDestroy != nil {
		c.onDestroy()
	}
}

// ComponentFunc is a function that creates a templ.Component.
type ComponentFunc func(*Component) templ.Component

// DefineComponent defines a new component with a render function.
func DefineComponent(name string, render ComponentFunc, opts ...ComponentOption) *Component {
	c := NewComponent(name, opts...)
	return c
}

// RenderComponent renders a component with its state.
func RenderComponent(c *Component, content templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		// Write component wrapper with data attributes
		attrs := c.Attrs()
		attrStr := ""
		for k, v := range attrs {
			attrStr += fmt.Sprintf(` %s="%v"`, k, v)
		}

		// Write opening tag
		if _, err := fmt.Fprintf(w, `<div data-gospa-component="%s"%s>`, c.ID, attrStr); err != nil {
			return err
		}

		// Write initialization script
		if err := c.InitScript().Render(ctx, w); err != nil {
			return err
		}

		// Write content
		if content != nil {
			if err := content.Render(ctx, w); err != nil {
				return err
			}
		}

		// Write closing tag
		if _, err := fmt.Fprint(w, `</div>`); err != nil {
			return err
		}

		return nil
	})
}

// PropsFromJSON parses JSON into a props map.
func PropsFromJSON(data string) (map[string]any, error) {
	var props map[string]any
	if err := json.Unmarshal([]byte(data), &props); err != nil {
		return nil, err
	}
	return props, nil
}

// PropsToJSON serializes props to JSON.
func PropsToJSON(props map[string]any) (string, error) {
	data, err := json.Marshal(props)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// MergeProps merges multiple props maps.
func MergeProps(props ...map[string]any) map[string]any {
	result := make(map[string]any)
	for _, p := range props {
		for k, v := range p {
			result[k] = v
		}
	}
	return result
}

// SpreadProps spreads props as attributes.
func SpreadProps(props map[string]any) templ.Attributes {
	attrs := make(templ.Attributes)
	for k, v := range props {
		attrs[k] = v
	}
	return attrs
}

// ComponentContext provides context for components.
type ComponentContext struct {
	// Component is the current component.
	Component *Component
	// Parent is the parent component context.
	Parent *ComponentContext
	// Context is the Go context.
	Context context.Context
}

// componentContextKey is the context key for component context.
type componentContextKey struct{}

// WithComponentContext sets the component context in the Go context.
func WithComponentContext(ctx context.Context, cc *ComponentContext) context.Context {
	return context.WithValue(ctx, componentContextKey{}, cc)
}

// GetContext gets the component context from the Go context.
func GetContext(ctx context.Context) *ComponentContext {
	if cc, ok := ctx.Value(componentContextKey{}).(*ComponentContext); ok {
		return cc
	}
	return nil
}

// GetParentComponent gets the parent component from context.
func GetParentComponent(ctx context.Context) *Component {
	cc := GetContext(ctx)
	if cc != nil && cc.Parent != nil {
		return cc.Parent.Component
	}
	return nil
}

// GetCurrentComponent gets the current component from context.
func GetCurrentComponent(ctx context.Context) *Component {
	cc := GetContext(ctx)
	if cc != nil {
		return cc.Component
	}
	return nil
}

// ProvideState provides state to child components via context.
func ProvideState(ctx context.Context, key string, value any) context.Context {
	cc := GetContext(ctx)
	if cc == nil {
		cc = &ComponentContext{
			Context: ctx,
		}
	}
	if cc.Component == nil {
		cc.Component = &Component{
			State: NewComponentState("context"),
		}
	}
	cc.Component.AddState(key, value)
	return WithComponentContext(ctx, cc)
}

// InjectState injects state from parent components.
func InjectState(ctx context.Context, key string) (any, bool) {
	cc := GetContext(ctx)
	if cc == nil {
		return nil, false
	}

	// Check current component first
	if cc.Component != nil {
		if v, ok := cc.Component.GetState(key); ok {
			return v, true
		}
	}

	// Check parent components
	if cc.Parent != nil {
		return InjectState(cc.Parent.Context, key)
	}

	return nil, false
}

// Slot is a helper for defining slot content.
type Slot struct {
	Name    string
	Content templ.Component
}

// Slot creates a new slot.
func NewSlot(name string, content templ.Component) Slot {
	return Slot{Name: name, Content: content}
}

// Slots is a collection of slots.
type Slots []Slot

// ToMap converts slots to a map.
func (s Slots) ToMap() map[string]templ.Component {
	m := make(map[string]templ.Component)
	for _, slot := range s {
		m[slot.Name] = slot.Content
	}
	return m
}

// RenderChildren renders the children slot.
func RenderChildren(c *Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if c.Children != nil {
			return c.Children.Render(ctx, w)
		}
		return nil
	})
}

// RenderSlot renders a named slot with fallback.
func RenderSlot(c *Component, name string, fallback templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if slot := c.GetSlot(name); slot != nil {
			return slot.Render(ctx, w)
		}
		if fallback != nil {
			return fallback.Render(ctx, w)
		}
		return nil
	})
}
