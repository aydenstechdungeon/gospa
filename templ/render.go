// Package templ provides rendering utilities for GoSPA.
package templ

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/a-h/templ"
)

// RuntimeScript returns the script tag for the GoSPA client runtime.
func RuntimeScript(src string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<script src="%s" type="module"></script>`, src)
		return err
	})
}

// RuntimeScriptInline returns an inline script tag with the runtime code.
func RuntimeScriptInline(code string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<script>%s</script>`, code)
		return err
	})
}

// CSS returns a link tag for a stylesheet.
func CSS(href string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<link rel="stylesheet" href="%s">`, href)
		return err
	})
}

// CSSInline returns an inline style tag.
func CSSInline(css string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<style>%s</style>`, css)
		return err
	})
}

// Meta returns a meta tag.
func Meta(name, content string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<meta name="%s" content="%s">`, name, content)
		return err
	})
}

// MetaProperty returns a meta property tag (for Open Graph, etc.).
func MetaProperty(property, content string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<meta property="%s" content="%s">`, property, content)
		return err
	})
}

// Title returns a title tag.
func Title(title string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<title>%s</title>`, title)
		return err
	})
}

// Favicon returns a link tag for a favicon.
func Favicon(href string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<link rel="icon" href="%s">`, href)
		return err
	})
}

// Head returns a component that renders content in the head.
func Head(components ...templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		for _, c := range components {
			if err := c.Render(ctx, w); err != nil {
				return err
			}
		}
		return nil
	})
}

// HTMLPage returns a complete HTML page.
func HTMLPage(lang string, head, body templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if _, err := fmt.Fprintf(w, `<!DOCTYPE html><html lang="%s">`, lang); err != nil {
			return err
		}
		if _, err := fmt.Fprint(w, `<head>`); err != nil {
			return err
		}
		if head != nil {
			if err := head.Render(ctx, w); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(w, `</head>`); err != nil {
			return err
		}
		if _, err := fmt.Fprint(w, `<body>`); err != nil {
			return err
		}
		if body != nil {
			if err := body.Render(ctx, w); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(w, `</body></html>`); err != nil {
			return err
		}
		return nil
	})
}

// SPAPage returns an HTML page configured for SPA mode.
func SPAPage(config SPAConfig) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		// DOCTYPE and html
		if _, err := fmt.Fprintf(w, `<!DOCTYPE html><html lang="%s">`, config.Lang); err != nil {
			return err
		}

		// Head
		if _, err := fmt.Fprint(w, `<head>`); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, `<meta charset="UTF-8">`); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, `<meta name="viewport" content="width=device-width, initial-scale=1.0">`); err != nil {
			return err
		}
		if config.Title != "" {
			if _, err := fmt.Fprintf(w, `<title>%s</title>`, config.Title); err != nil {
				return err
			}
		}
		for _, meta := range config.Meta {
			if _, err := fmt.Fprintf(w, `<meta name="%s" content="%s">`, meta.Name, meta.Content); err != nil {
				return err
			}
		}
		for _, link := range config.Stylesheets {
			if _, err := fmt.Fprintf(w, `<link rel="stylesheet" href="%s">`, link); err != nil {
				return err
			}
		}
		if config.Head != nil {
			if err := config.Head.Render(ctx, w); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(w, `</head>`); err != nil {
			return err
		}

		// Body
		if _, err := fmt.Fprint(w, `<body>`); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, `<div id="%s" data-gospa-root>`, config.RootID); err != nil {
			return err
		}
		if config.Body != nil {
			if err := config.Body.Render(ctx, w); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(w, `</div>`); err != nil {
			return err
		}

		// Runtime script
		if config.RuntimeSrc != "" {
			if _, err := fmt.Fprintf(w, `<script src="%s" type="module"></script>`, config.RuntimeSrc); err != nil {
				return err
			}
		}

		// Auto-init script
		if config.AutoInit {
			if _, err := fmt.Fprintf(w, `<script data-gospa-auto></script>`); err != nil {
				return err
			}
		}

		if _, err := fmt.Fprint(w, `</body></html>`); err != nil {
			return err
		}
		return nil
	})
}

// MetaTag represents an HTML meta tag.
type MetaTag struct {
	Name    string
	Content string
}

// SPAConfig configures an SPA page.
type SPAConfig struct {
	Lang        string
	Title       string
	Meta        []MetaTag
	Stylesheets []string
	Head        templ.Component
	Body        templ.Component
	RootID      string
	RuntimeSrc  string
	AutoInit    bool
}

// Raw renders raw HTML content.
func Raw(html string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := w.Write([]byte(html))
		return err
	})
}

// HTMLContent renders HTML content safely (already escaped).
func HTMLContent(html string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := templ.JoinStringErrs(html)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(html))
		return err
	})
}

// TextContent renders text content (HTML escaped).
func TextContent(text string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, templ.EscapeString(text))
		return err
	})
}

// Attrs renders multiple attributes.
func Attrs(attrs ...templ.Attributes) templ.Attributes {
	result := make(templ.Attributes)
	for _, a := range attrs {
		for k, v := range a {
			result[k] = v
		}
	}
	return result
}

// Class generates a class attribute from multiple class names.
func Class(classes ...string) templ.Attributes {
	return templ.Attributes{
		"class": strings.Join(classes, " "),
	}
}

// ClassIf generates a class attribute with conditional classes.
func ClassIf(classes map[string]bool) templ.Attributes {
	var active []string
	for class, isActive := range classes {
		if isActive {
			active = append(active, class)
		}
	}
	return templ.Attributes{
		"class": strings.Join(active, " "),
	}
}

// Style generates a style attribute from a map.
func Style(styles map[string]string) templ.Attributes {
	var parts []string
	for prop, value := range styles {
		parts = append(parts, fmt.Sprintf("%s: %s", prop, value))
	}
	return templ.Attributes{
		"style": strings.Join(parts, "; "),
	}
}

// DataAttrs generates data attributes from a map.
func DataAttrs(data map[string]any) templ.Attributes {
	attrs := make(templ.Attributes)
	for k, v := range data {
		attrs["data-"+k] = v
	}
	return attrs
}

// ID generates an id attribute.
func ID(id string) templ.Attributes {
	return templ.Attributes{
		"id": id,
	}
}

// Name generates a name attribute.
func Name(name string) templ.Attributes {
	return templ.Attributes{
		"name": name,
	}
}

// Type generates a type attribute.
func Type(t string) templ.Attributes {
	return templ.Attributes{
		"type": t,
	}
}

// ValueAttr generates a value attribute.
func ValueAttr(v string) templ.Attributes {
	return templ.Attributes{
		"value": v,
	}
}

// Placeholder generates a placeholder attribute.
func Placeholder(p string) templ.Attributes {
	return templ.Attributes{
		"placeholder": p,
	}
}

// Disabled generates a disabled attribute.
func Disabled(disabled bool) templ.Attributes {
	if disabled {
		return templ.Attributes{
			"disabled": "",
		}
	}
	return nil
}

// Readonly generates a readonly attribute.
func Readonly(readonly bool) templ.Attributes {
	if readonly {
		return templ.Attributes{
			"readonly": "",
		}
	}
	return nil
}

// Required generates a required attribute.
func Required(required bool) templ.Attributes {
	if required {
		return templ.Attributes{
			"required": "",
		}
	}
	return nil
}

// CheckedAttr generates a checked attribute.
func CheckedAttr(checked bool) templ.Attributes {
	if checked {
		return templ.Attributes{
			"checked": "",
		}
	}
	return nil
}

// Selected generates a selected attribute.
func Selected(selected bool) templ.Attributes {
	if selected {
		return templ.Attributes{
			"selected": "",
		}
	}
	return nil
}

// Hidden generates a hidden attribute.
func Hidden(hidden bool) templ.Attributes {
	if hidden {
		return templ.Attributes{
			"hidden": "",
		}
	}
	return nil
}

// Href generates an href attribute.
func Href(href string) templ.Attributes {
	return templ.Attributes{
		"href": href,
	}
}

// Src generates a src attribute.
func Src(src string) templ.Attributes {
	return templ.Attributes{
		"src": src,
	}
}

// Alt generates an alt attribute.
func Alt(alt string) templ.Attributes {
	return templ.Attributes{
		"alt": alt,
	}
}

// Target generates a target attribute.
func Target(target string) templ.Attributes {
	return templ.Attributes{
		"target": target,
	}
}

// Rel generates a rel attribute.
func Rel(rel string) templ.Attributes {
	return templ.Attributes{
		"rel": rel,
	}
}

// Aria generates an aria-* attribute.
func Aria(name string, value any) templ.Attributes {
	return templ.Attributes{
		"aria-" + name: value,
	}
}

// Role generates a role attribute.
func Role(role string) templ.Attributes {
	return templ.Attributes{
		"role": role,
	}
}

// TabIndex generates a tabindex attribute.
func TabIndex(index int) templ.Attributes {
	return templ.Attributes{
		"tabindex": index,
	}
}

// Fragment renders a fragment of components.
func Fragment(components ...templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		for _, c := range components {
			if err := c.Render(ctx, w); err != nil {
				return err
			}
		}
		return nil
	})
}

// Empty renders nothing.
func Empty() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return nil
	})
}

// When renders a component conditionally.
func When(condition bool, component templ.Component) templ.Component {
	if condition {
		return component
	}
	return Empty()
}

// WhenElse renders one of two components based on a condition.
func WhenElse(condition bool, ifTrue, ifFalse templ.Component) templ.Component {
	if condition {
		return ifTrue
	}
	return ifFalse
}

// For renders a list of items.
func For[T any](items []T, render func(T, int) templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		for i, item := range items {
			if err := render(item, i).Render(ctx, w); err != nil {
				return err
			}
		}
		return nil
	})
}

// ForKey renders a list of items with keys.
func ForKey[T any, K comparable](items []T, keyFn func(T) K, render func(T, int) templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		for i, item := range items {
			// Key is used for reconciliation on the client
			k := keyFn(item)
			if _, err := fmt.Fprintf(w, `<template data-key="%v">`, k); err != nil {
				return err
			}
			if err := render(item, i).Render(ctx, w); err != nil {
				return err
			}
			if _, err := fmt.Fprint(w, `</template>`); err != nil {
				return err
			}
		}
		return nil
	})
}

// Switch renders the first matching case.
func Switch(cases ...SwitchCase) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		for _, c := range cases {
			if c.condition {
				return c.component.Render(ctx, w)
			}
		}
		return nil
	})
}

// SwitchCase represents a case in a switch.
type SwitchCase struct {
	condition bool
	component templ.Component
}

// Case creates a switch case.
func Case(condition bool, component templ.Component) SwitchCase {
	return SwitchCase{condition: condition, component: component}
}

// Default creates a default switch case.
func Default(component templ.Component) SwitchCase {
	return SwitchCase{condition: true, component: component}
}

// HeadManager manages head elements for SPA navigation.
// It renders elements with data-gospa-head attribute for client-side updates.
type HeadManager struct {
	elements []HeadElement
}

// HeadElement represents an element in the document head.
type HeadElement struct {
	Tag      string            // e.g., "title", "meta", "link", "script", "style"
	Attrs    map[string]string // HTML attributes
	Content  string            // Inner content (for title, script, style)
	Key      string            // Unique key for deduplication (optional)
	Priority int               // Higher priority renders first
}

// NewHeadManager creates a new head manager.
func NewHeadManager() *HeadManager {
	return &HeadManager{
		elements: make([]HeadElement, 0),
	}
}

// SetHeadTitle sets the page title.
func (h *HeadManager) SetHeadTitle(title string) *HeadManager {
	h.elements = append(h.elements, HeadElement{
		Tag:      "title",
		Content:  title,
		Key:      "title",
		Priority: 100,
	})
	return h
}

// AddHeadMeta adds a meta tag.
func (h *HeadManager) AddHeadMeta(name, content string) *HeadManager {
	h.elements = append(h.elements, HeadElement{
		Tag: "meta",
		Key: "meta-" + name,
		Attrs: map[string]string{
			"name":    name,
			"content": content,
		},
		Priority: 50,
	})
	return h
}

// AddHeadMetaProperty adds a meta property tag (Open Graph).
func (h *HeadManager) AddHeadMetaProperty(property, content string) *HeadManager {
	h.elements = append(h.elements, HeadElement{
		Tag: "meta",
		Key: "meta-prop-" + property,
		Attrs: map[string]string{
			"property": property,
			"content":  content,
		},
		Priority: 50,
	})
	return h
}

// AddHeadLink adds a link tag.
func (h *HeadManager) AddHeadLink(rel, href string, extraAttrs ...map[string]string) *HeadManager {
	attrs := map[string]string{
		"rel":  rel,
		"href": href,
	}
	for _, extra := range extraAttrs {
		for k, v := range extra {
			attrs[k] = v
		}
	}
	h.elements = append(h.elements, HeadElement{
		Tag:      "link",
		Attrs:    attrs,
		Key:      "link-" + rel + "-" + href,
		Priority: 30,
	})
	return h
}

// AddHeadScript adds a script tag.
func (h *HeadManager) AddHeadScript(src string, async, defer_ bool) *HeadManager {
	attrs := map[string]string{
		"src": src,
	}
	if async {
		attrs["async"] = ""
	}
	if defer_ {
		attrs["defer"] = ""
	}
	h.elements = append(h.elements, HeadElement{
		Tag:      "script",
		Attrs:    attrs,
		Key:      "script-" + src,
		Priority: 10,
	})
	return h
}

// AddHeadInlineScript adds an inline script.
func (h *HeadManager) AddHeadInlineScript(content string) *HeadManager {
	h.elements = append(h.elements, HeadElement{
		Tag:      "script",
		Content:  content,
		Key:      "script-inline-" + content[:min(20, len(content))],
		Priority: 10,
	})
	return h
}

// AddHeadStyle adds a stylesheet link.
func (h *HeadManager) AddHeadStyle(href string) *HeadManager {
	h.elements = append(h.elements, HeadElement{
		Tag: "link",
		Attrs: map[string]string{
			"rel":  "stylesheet",
			"href": href,
		},
		Key:      "style-" + href,
		Priority: 40,
	})
	return h
}

// AddHeadInlineStyle adds inline CSS.
func (h *HeadManager) AddHeadInlineStyle(css string) *HeadManager {
	h.elements = append(h.elements, HeadElement{
		Tag:      "style",
		Content:  css,
		Key:      "style-inline-" + css[:min(20, len(css))],
		Priority: 40,
	})
	return h
}

// AddHeadElement adds a custom head element.
func (h *HeadManager) AddHeadElement(el HeadElement) *HeadManager {
	h.elements = append(h.elements, el)
	return h
}

// Render renders all head elements as a component.
func (h *HeadManager) Render() templ.Component {
	// Sort by priority (higher first)
	sorted := make([]HeadElement, len(h.elements))
	copy(sorted, h.elements)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Priority > sorted[i].Priority {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		for _, el := range sorted {
			if err := renderHeadElement(el, w); err != nil {
				return err
			}
		}
		return nil
	})
}

// renderHeadElement renders a single head element.
func renderHeadElement(el HeadElement, w io.Writer) error {
	// Add data-gospa-head attribute for client-side updates
	attrs := make(map[string]string)
	for k, v := range el.Attrs {
		attrs[k] = v
	}
	attrs["data-gospa-head"] = el.Key

	switch el.Tag {
	case "title":
		_, err := fmt.Fprintf(w, `<title data-gospa-head="title">%s</title>`, el.Content)
		return err
	case "meta", "link":
		// Self-closing tags
		_, err := fmt.Fprintf(w, `<%s`, el.Tag)
		if err != nil {
			return err
		}
		for k, v := range attrs {
			if v == "" {
				_, err = fmt.Fprintf(w, ` %s`, k)
			} else {
				_, err = fmt.Fprintf(w, ` %s="%s"`, k, v)
			}
			if err != nil {
				return err
			}
		}
		_, err = fmt.Fprintf(w, `>`)
		return err
	case "script", "style":
		// Tags with content
		_, err := fmt.Fprintf(w, `<%s`, el.Tag)
		if err != nil {
			return err
		}
		for k, v := range attrs {
			if k == "data-gospa-head" {
				continue // Don't add to script/style tags
			}
			if v == "" {
				_, err = fmt.Fprintf(w, ` %s`, k)
			} else {
				_, err = fmt.Fprintf(w, ` %s="%s"`, k, v)
			}
			if err != nil {
				return err
			}
		}
		_, err = fmt.Fprintf(w, `>%s</%s>`, el.Content, el.Tag)
		return err
	default:
		_, err := fmt.Fprintf(w, `<%s data-gospa-head="%s">%s</%s>`, el.Tag, el.Key, el.Content, el.Tag)
		return err
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// HeadTitle creates a title tag with data-gospa-head attribute.
func HeadTitle(title string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<title data-gospa-head="title">%s</title>`, title)
		return err
	})
}

// HeadMeta creates a meta tag with data-gospa-head attribute.
func HeadMeta(name, content string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<meta name="%s" content="%s" data-gospa-head="meta-%s">`, name, content, name)
		return err
	})
}

// HeadMetaProp creates a meta property tag with data-gospa-head attribute.
func HeadMetaProp(property, content string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<meta property="%s" content="%s" data-gospa-head="meta-prop-%s">`, property, content, property)
		return err
	})
}

// HeadLink creates a link tag with data-gospa-head attribute.
func HeadLink(rel, href string, extraAttrs ...map[string]string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		attrs := map[string]string{
			"rel":             rel,
			"href":            href,
			"data-gospa-head": "link-" + rel + "-" + href,
		}
		for _, extra := range extraAttrs {
			for k, v := range extra {
				attrs[k] = v
			}
		}
		_, err := fmt.Fprintf(w, `<link`)
		if err != nil {
			return err
		}
		for k, v := range attrs {
			if v == "" {
				_, err = fmt.Fprintf(w, ` %s`, k)
			} else {
				_, err = fmt.Fprintf(w, ` %s="%s"`, k, v)
			}
			if err != nil {
				return err
			}
		}
		_, err = fmt.Fprintf(w, `>`)
		return err
	})
}

// HeadScript creates a script tag with data-gospa-head attribute.
func HeadScript(src string, async, defer_ bool) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		attrs := map[string]string{
			"src": src,
		}
		if async {
			attrs["async"] = ""
		}
		if defer_ {
			attrs["defer"] = ""
		}
		_, err := fmt.Fprintf(w, `<script`)
		if err != nil {
			return err
		}
		for k, v := range attrs {
			if v == "" {
				_, err = fmt.Fprintf(w, ` %s`, k)
			} else {
				_, err = fmt.Fprintf(w, ` %s="%s"`, k, v)
			}
			if err != nil {
				return err
			}
		}
		_, err = fmt.Fprintf(w, `></script>`)
		return err
	})
}

// HeadStyle creates a stylesheet link with data-gospa-head attribute.
func HeadStyle(href string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<link rel="stylesheet" href="%s" data-gospa-head="style-%s">`, href, href)
		return err
	})
}
