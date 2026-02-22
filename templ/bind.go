// Package templ provides Templ integration helpers for GoSPA reactive bindings.
package templ

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"strings"

	"github.com/a-h/templ"
	"github.com/aydenstechdungeon/gospa/state"
)

// BindingType represents the type of DOM binding.
type BindingType string

const (
	// TextBind binds to textContent.
	TextBind BindingType = "text"
	// HTMLBind binds to innerHTML.
	HTMLBind BindingType = "html"
	// ValueBind binds to input value.
	ValueBind BindingType = "value"
	// CheckedBind binds to checkbox/radio checked state.
	CheckedBind BindingType = "checked"
	// ClassBind binds to CSS classes.
	ClassBind BindingType = "class"
	// StyleBind binds to inline styles.
	StyleBind BindingType = "style"
	// AttrBind binds to a custom attribute.
	AttrBind BindingType = "attr"
	// PropBind binds to a DOM property.
	PropBind BindingType = "prop"
	// ShowBind shows/hides element based on boolean.
	ShowBind BindingType = "show"
	// IfBind conditionally renders element.
	IfBind BindingType = "if"
)

// Binding represents a reactive DOM binding.
type Binding struct {
	// Type of binding.
	Type BindingType
	// Key is the state key.
	Key string
	// Attr is the attribute name for attr/prop bindings.
	Attr string
	// Transform is an optional transform function name.
	Transform string
}

// Bind creates a data-bind attribute for reactive bindings.
// Usage: <div { templ.Bind("count", templ.TextBind) }>{ count }</div>
func Bind(key string, bindType BindingType) templ.Attributes {
	return templ.Attributes{
		"data-bind": string(bindType) + ":" + key,
	}
}

// BindWithAttr creates a data-bind attribute for attribute-specific bindings.
// Usage: <input { templ.BindWithAttr("value", "placeholder", templ.AttrBind) } />
func BindWithAttr(key string, attr string, bindType BindingType) templ.Attributes {
	return templ.Attributes{
		"data-bind": fmt.Sprintf("%s:%s:%s", bindType, key, attr),
	}
}

// BindWithTransform creates a data-bind attribute with a transform function.
// Usage: <span { templ.BindWithTransform("price", templ.TextBind, "formatCurrency") }></span>
func BindWithTransform(key string, bindType BindingType, transform string) templ.Attributes {
	return templ.Attributes{
		"data-bind": fmt.Sprintf("%s:%s:%s", bindType, key, transform),
	}
}

// TwoWayBind creates a two-way binding for form elements.
// Usage: <input { templ.TwoWayBind("name") } />
func TwoWayBind(key string) templ.Attributes {
	return templ.Attributes{
		"data-bind":     "value:" + key,
		"data-bind-two": "true",
		"data-sync":     "input",
	}
}

// ClassBinding creates a class binding that toggles based on state.
// Usage: <div { templ.ClassBinding("isActive", "active") }></div>
func ClassBinding(key string, className string) templ.Attributes {
	return templ.Attributes{
		"data-bind-class": className + ":" + key,
	}
}

// ClassBindings creates multiple class bindings.
// Usage: <div { templ.ClassBindings(map[string]string{"active": "isActive", "disabled": "isDisabled"}) }></div>
func ClassBindings(classes map[string]string) templ.Attributes {
	var parts []string
	for class, key := range classes {
		parts = append(parts, class+":"+key)
	}
	return templ.Attributes{
		"data-bind-class": strings.Join(parts, ","),
	}
}

// StyleBinding creates a style binding.
// Usage: <div { templ.StyleBinding("color", "textColor") }></div>
func StyleBinding(property string, key string) templ.Attributes {
	return templ.Attributes{
		"data-bind-style": property + ":" + key,
	}
}

// ShowBinding creates a show/hide binding based on boolean state.
// Usage: <div { templ.ShowBinding("isVisible") }></div>
func ShowBinding(key string) templ.Attributes {
	return templ.Attributes{
		"data-bind-show": key,
	}
}

// IfBinding creates a conditional rendering binding.
// Usage: <div { templ.IfBinding("shouldRender") }></div>
func IfBinding(key string) templ.Attributes {
	return templ.Attributes{
		"data-bind-if": key,
	}
}

// ListBinding creates a list rendering binding.
// Usage: <ul { templ.ListBinding("items", "item") }><li>{ item }</li></ul>
func ListBinding(key string, itemName string) templ.Attributes {
	return templ.Attributes{
		"data-bind-list": key,
		"data-item-name": itemName,
	}
}

// ListBindingWithKey creates a list rendering binding with a key field.
// Usage: <ul { templ.ListBindingWithKey("items", "item", "id") }><li>{ item.name }</li></ul>
func ListBindingWithKey(key string, itemName string, keyField string) templ.Attributes {
	return templ.Attributes{
		"data-bind-list": key,
		"data-item-name": itemName,
		"data-item-key":  keyField,
	}
}

// AttrBinding creates an attribute binding.
// Usage: <a { templ.AttrBinding("href", "url") }></a>
func AttrBinding(attr string, key string) templ.Attributes {
	return templ.Attributes{
		"data-bind-attr": attr + ":" + key,
	}
}

// AttrBindings creates multiple attribute bindings.
// Usage: <a { templ.AttrBindings(map[string]string{"href": "url", "title": "tooltip"}) }></a>
func AttrBindings(attrs map[string]string) templ.Attributes {
	var parts []string
	for attr, key := range attrs {
		parts = append(parts, attr+":"+key)
	}
	return templ.Attributes{
		"data-bind-attr": strings.Join(parts, ","),
	}
}

// PropBinding creates a property binding.
// Usage: <input { templ.PropBinding("disabled", "isDisabled") } />
func PropBinding(prop string, key string) templ.Attributes {
	return templ.Attributes{
		"data-bind-prop": prop + ":" + key,
	}
}

// Text creates a text content binding.
// Usage: <span { templ.Text("message") }></span>
func Text(key string) templ.Attributes {
	return Bind(key, TextBind)
}

// HTML creates an HTML content binding.
// Usage: <div { templ.HTML("content") }></div>
func HTML(key string) templ.Attributes {
	return Bind(key, HTMLBind)
}

// Value creates a value binding for form elements.
// Usage: <input { templ.Value("name") } />
func Value(key string) templ.Attributes {
	return Bind(key, ValueBind)
}

// Checked creates a checked binding for checkboxes.
// Usage: <input type="checkbox" { templ.Checked("agree") } />
func Checked(key string) templ.Attributes {
	return Bind(key, CheckedBind)
}

// ComponentState represents state for a component.
type ComponentState struct {
	// ID is the unique component identifier.
	ID string
	// State is the reactive state map.
	State *state.StateMap
	// Bindings are the component's bindings.
	Bindings map[string]Binding
}

// NewComponentState creates a new component state container.
func NewComponentState(id string) *ComponentState {
	return &ComponentState{
		ID:       id,
		State:    state.NewStateMap(),
		Bindings: make(map[string]Binding),
	}
}

// AddRune adds a rune to the component state.
func (cs *ComponentState) AddRune(key string, r *state.Rune[any]) *ComponentState {
	cs.State.Add(key, r)
	return cs
}

// GetRune gets a rune by key.
func (cs *ComponentState) GetRune(key string) (*state.Rune[any], bool) {
	obs, ok := cs.State.Get(key)
	if !ok {
		return nil, false
	}
	r, ok := obs.(*state.Rune[any])
	return r, ok
}

// AddBinding adds a binding to the component.
func (cs *ComponentState) AddBinding(key string, binding Binding) {
	cs.Bindings[key] = binding
}

// ToJSON serializes the component state to JSON.
func (cs *ComponentState) ToJSON() (string, error) {
	return cs.State.ToJSON()
}

// StateAttrs generates data attributes for component state initialization.
// Usage: <div { cs.StateAttrs() }></div>
func (cs *ComponentState) StateAttrs() templ.Attributes {
	return templ.Attributes{
		"data-component": cs.ID,
	}
}

// InitScript generates the component initialization script.
func (cs *ComponentState) InitScript() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		json, err := cs.ToJSON()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, `<script data-component-init="%s">window.__GOSPA_STATE__=%s;</script>`, cs.ID, json)
		return err
	})
}

// RenderBindings renders all bindings as data attributes.
func (cs *ComponentState) RenderBindings() templ.Attributes {
	attrs := make(templ.Attributes)
	for key, binding := range cs.Bindings {
		attrKey := fmt.Sprintf("data-bind-%s", binding.Type)
		if binding.Attr != "" {
			attrs[attrKey] = fmt.Sprintf("%s:%s:%s", binding.Attr, key, binding.Transform)
		} else if binding.Transform != "" {
			attrs[attrKey] = fmt.Sprintf("%s:%s", key, binding.Transform)
		} else {
			attrs[attrKey] = key
		}
	}
	attrs["data-component"] = cs.ID
	return attrs
}

// SafeHTML marks a string as safe HTML.
func SafeHTML(s string) template.HTML {
	return template.HTML(s)
}

// SafeAttr marks a string as a safe attribute value.
func SafeAttr(s string) template.HTMLAttr {
	return template.HTMLAttr(s)
}
