// Package templ provides layout helpers for GoSPA.
package templ

import (
	"context"
	"fmt"
	"io"

	"github.com/a-h/templ"
)

// LayoutProps represents props passed to a layout component.
// Similar to Svelte's $props() for layouts.
type LayoutProps struct {
	// Data is the layout data from loader functions.
	Data map[string]any
	// Children is the content to render inside the layout.
	Children templ.Component
	// Slots contains named slots for composition.
	Slots map[string]templ.Component
	// Params are the route parameters.
	Params map[string]string
	// Path is the current route path.
	Path string
}

// LayoutFunc is a function that renders a layout with props.
type LayoutFunc func(LayoutProps) templ.Component

// LayoutLoader is an interface for loading layout data.
// Similar to SvelteKit's +layout.server.js load function.
type LayoutLoader interface {
	// Load loads data for the layout.
	Load(ctx context.Context, params map[string]string) (map[string]any, error)
}

// LayoutLoaderFunc is a function adapter for LayoutLoader.
type LayoutLoaderFunc func(ctx context.Context, params map[string]string) (map[string]any, error)

// Load implements LayoutLoader.
func (f LayoutLoaderFunc) Load(ctx context.Context, params map[string]string) (map[string]any, error) {
	return f(ctx, params)
}

// Layout represents a layout with its renderer and optional data loader.
type Layout struct {
	// Name is the layout name for debugging.
	Name string
	// Render is the layout render function.
	Render LayoutFunc
	// Loader is the optional data loader.
	Loader LayoutLoader
}

// LayoutChain represents a chain of nested layouts.
type LayoutChain struct {
	// Layouts are ordered from root to leaf.
	Layouts []*Layout
	// Page is the final page component.
	Page templ.Component
	// Error is the optional error component.
	Error templ.Component
}

// Render renders the layout chain with the given context.
func (lc *LayoutChain) Render(ctx context.Context, w io.Writer, params map[string]string, path string) error {
	// Start with the page as the innermost content
	var current = lc.Page

	// Wrap layouts from leaf to root (reverse order)
	for i := len(lc.Layouts) - 1; i >= 0; i-- {
		layout := lc.Layouts[i]

		// Load layout data if loader exists
		var data map[string]any
		if layout.Loader != nil {
			var err error
			data, err = layout.Loader.Load(ctx, params)
			if err != nil {
				return fmt.Errorf("layout %s load error: %w", layout.Name, err)
			}
		}

		// Create props with current children
		props := LayoutProps{
			Data:     data,
			Children: current,
			Params:   params,
			Path:     path,
		}

		// Render layout to get new current component
		current = layout.Render(props)
	}

	// Render the final component chain
	if current != nil {
		return current.Render(ctx, w)
	}
	return nil
}

// NewLayoutChain creates a new layout chain.
func NewLayoutChain(page templ.Component, layouts ...*Layout) *LayoutChain {
	return &LayoutChain{
		Layouts: layouts,
		Page:    page,
	}
}

// WithError sets the error component for the layout chain.
func (lc *LayoutChain) WithError(error templ.Component) *LayoutChain {
	lc.Error = error
	return lc
}

// RenderLayout renders a single layout with props.
func RenderLayout(layout LayoutFunc, props LayoutProps) templ.Component {
	return layout(props)
}

// LayoutComponent creates a layout component from a render function.
func LayoutComponent(name string, render LayoutFunc, loader ...LayoutLoader) *Layout {
	l := &Layout{
		Name:   name,
		Render: render,
	}
	if len(loader) > 0 {
		l.Loader = loader[0]
	}
	return l
}

// WithLayoutData adds data to layout props.
func (p LayoutProps) WithData(data map[string]any) LayoutProps {
	if p.Data == nil {
		p.Data = make(map[string]any)
	}
	for k, v := range data {
		p.Data[k] = v
	}
	return p
}

// WithSlot adds a named slot to layout props.
func (p LayoutProps) WithSlot(name string, content templ.Component) LayoutProps {
	if p.Slots == nil {
		p.Slots = make(map[string]templ.Component)
	}
	p.Slots[name] = content
	return p
}

// GetSlot gets a slot by name with optional fallback.
func (p LayoutProps) GetSlot(name string, fallback ...templ.Component) templ.Component {
	if p.Slots != nil {
		if slot, ok := p.Slots[name]; ok {
			return slot
		}
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return nil
}

// HasSlot checks if a slot exists.
func (p LayoutProps) HasSlot(name string) bool {
	_, ok := p.Slots[name]
	return ok
}

// GetProp gets a data prop by key with optional default.
func (p LayoutProps) GetProp(key string, defaultValue ...any) any {
	if p.Data != nil {
		if v, ok := p.Data[key]; ok {
			return v
		}
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return nil
}

// GetParam gets a route parameter by key with optional default.
func (p LayoutProps) GetParam(key string, defaultValue ...string) string {
	if p.Params != nil {
		if v, ok := p.Params[key]; ok {
			return v
		}
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// RenderChildren renders the children content.
func (p LayoutProps) RenderChildren() templ.Component {
	return p.Children
}

// LayoutContext provides context for layout rendering.
type LayoutContext struct {
	// Chain is the current layout chain.
	Chain *LayoutChain
	// Params are the route parameters.
	Params map[string]string
	// Path is the current route path.
	Path string
	// Data is accumulated layout data.
	Data map[string]any
}

// layoutContextKey is the context key for layout context.
type layoutContextKey struct{}

// WithLayoutContext sets the layout context in the Go context.
func WithLayoutContext(ctx context.Context, lc *LayoutContext) context.Context {
	return context.WithValue(ctx, layoutContextKey{}, lc)
}

// GetLayoutContext gets the layout context from the Go context.
func GetLayoutContext(ctx context.Context) *LayoutContext {
	if lc, ok := ctx.Value(layoutContextKey{}).(*LayoutContext); ok {
		return lc
	}
	return nil
}

// GetLayoutData gets layout data from context.
func GetLayoutData(ctx context.Context) map[string]any {
	lc := GetLayoutContext(ctx)
	if lc != nil {
		return lc.Data
	}
	return nil
}

// GetLayoutParam gets a route parameter from layout context.
func GetLayoutParam(ctx context.Context, key string) string {
	lc := GetLayoutContext(ctx)
	if lc != nil {
		return lc.Params[key]
	}
	return ""
}

// GetLayoutPath gets the current path from layout context.
func GetLayoutPath(ctx context.Context) string {
	lc := GetLayoutContext(ctx)
	if lc != nil {
		return lc.Path
	}
	return ""
}

// NestedLayout creates a nested layout structure.
// This is useful for building layouts programmatically.
type NestedLayout struct {
	outer *Layout
	inner *NestedLayout
}

// NewNestedLayout creates a new nested layout.
func NewNestedLayout(outer *Layout) *NestedLayout {
	return &NestedLayout{outer: outer}
}

// Nest adds an inner layout.
func (n *NestedLayout) Nest(inner *Layout) *NestedLayout {
	n.inner = NewNestedLayout(inner)
	return n
}

// Build builds the layout chain from the nested structure.
func (n *NestedLayout) Build(page templ.Component) *LayoutChain {
	var layouts []*Layout
	n.collectLayouts(&layouts)
	return NewLayoutChain(page, layouts...)
}

func (n *NestedLayout) collectLayouts(layouts *[]*Layout) {
	if n.outer != nil {
		*layouts = append(*layouts, n.outer)
	}
	if n.inner != nil {
		n.inner.collectLayouts(layouts)
	}
}

// RootLayout creates a root layout with common HTML structure.
func RootLayout(title string, head, body templ.Component) *Layout {
	return LayoutComponent("root", func(props LayoutProps) templ.Component {
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			// DOCTYPE and html
			if _, err := fmt.Fprint(w, `<!DOCTYPE html><html>`); err != nil {
				return err
			}

			// Head section
			if _, err := fmt.Fprint(w, `<head>`); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, `<meta charset="UTF-8">`); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, `<meta name="viewport" content="width=device-width, initial-scale=1.0">`); err != nil {
				return err
			}
			if title != "" {
				if _, err := fmt.Fprintf(w, `<title>%s</title>`, title); err != nil {
					return err
				}
			}
			if head != nil {
				if err := head.Render(ctx, w); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprint(w, `</head>`); err != nil {
				return err
			}

			// Body section
			if _, err := fmt.Fprint(w, `<body>`); err != nil {
				return err
			}
			if body != nil {
				if err := body.Render(ctx, w); err != nil {
					return err
				}
			}
			if props.Children != nil {
				if err := props.Children.Render(ctx, w); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprint(w, `</body></html>`); err != nil {
				return err
			}

			return nil
		})
	})
}

// WrapLayout wraps an existing templ.Component as a layout.
func WrapLayout(name string, component templ.Component) *Layout {
	return LayoutComponent(name, func(props LayoutProps) templ.Component {
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			// Store props in context for nested access
			ctx = WithLayoutContext(ctx, &LayoutContext{
				Data:   props.Data,
				Params: props.Params,
				Path:   props.Path,
			})
			// Render the component first
			if err := component.Render(ctx, w); err != nil {
				return err
			}
			// Then render children
			if props.Children != nil {
				return props.Children.Render(ctx, w)
			}
			return nil
		})
	})
}
