// Package templ provides island rendering helpers for GoSPA templates.
package templ

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/a-h/templ"
	"github.com/aydenstechdungeon/gospa/component"
)

// IslandOptions configures island rendering behavior.
type IslandOptions struct {
	// HydrationMode determines when the island hydrates.
	HydrationMode component.IslandHydrationMode
	// Priority affects loading order.
	Priority component.IslandPriority
	// ClientOnly skips SSR entirely.
	ClientOnly bool
	// ServerOnly renders HTML without client JS.
	ServerOnly bool
	// LazyThreshold for visible mode - margin in pixels.
	LazyThreshold int
	// DeferDelay for idle mode - max delay in ms.
	DeferDelay int
	// Class adds custom CSS classes.
	Class string
	// Tag specifies the wrapper element tag.
	Tag string
}

// IslandRenderer handles island rendering operations.
type IslandRenderer struct {
	registry *component.IslandRegistry
}

// NewIslandRenderer creates a new island renderer.
func NewIslandRenderer(registry *component.IslandRegistry) *IslandRenderer {
	return &IslandRenderer{registry: registry}
}

// Island creates an island component with the given name and content.
func Island(name string, content templ.Component, opts ...IslandOptions) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		// Merge options
		opt := IslandOptions{
			HydrationMode: component.HydrationImmediate,
			Priority:      component.PriorityNormal,
			Tag:           "div",
		}
		if len(opts) > 0 {
			opt = opts[0]
		}

		// Create island instance
		island, err := component.CreateIsland(name, nil)
		if err != nil {
			return fmt.Errorf("failed to create island: %w", err)
		}

		// Apply options
		island.Config.HydrationMode = opt.HydrationMode
		island.Config.Priority = opt.Priority
		island.Config.ClientOnly = opt.ClientOnly
		island.Config.ServerOnly = opt.ServerOnly
		island.Config.LazyThreshold = opt.LazyThreshold
		island.Config.DeferDelay = opt.DeferDelay

		// Render content
		var buf strings.Builder
		if err := content.Render(ctx, &buf); err != nil {
			return fmt.Errorf("failed to render island content: %w", err)
		}
		island.Children = buf.String()

		// Build attributes
		attrs := buildIslandAttributes(island, opt)

		// Render wrapper
		return renderIslandWrapper(island, attrs, opt, w)
	})
}

// IslandWithProps creates an island with props.
func IslandWithProps(name string, props map[string]any, content templ.Component, opts ...IslandOptions) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		opt := IslandOptions{
			HydrationMode: component.HydrationImmediate,
			Priority:      component.PriorityNormal,
			Tag:           "div",
		}
		if len(opts) > 0 {
			opt = opts[0]
		}

		island, err := component.CreateIsland(name, props)
		if err != nil {
			return fmt.Errorf("failed to create island: %w", err)
		}

		island.Config.HydrationMode = opt.HydrationMode
		island.Config.Priority = opt.Priority
		island.Config.ClientOnly = opt.ClientOnly
		island.Config.ServerOnly = opt.ServerOnly

		var buf strings.Builder
		if err := content.Render(ctx, &buf); err != nil {
			return fmt.Errorf("failed to render island content: %w", err)
		}
		island.Children = buf.String()

		attrs := buildIslandAttributes(island, opt)
		return renderIslandWrapper(island, attrs, opt, w)
	})
}

// ClientOnly renders content only on the client.
func ClientOnly(name string, placeholder ...templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		// Render placeholder if provided
		if len(placeholder) > 0 && placeholder[0] != nil {
			return placeholder[0].Render(ctx, w)
		}
		// Otherwise render empty placeholder with island marker
		_, err := fmt.Fprintf(w, `<div data-gospa-island="%s" data-gospa-client-only="true"></div>`, name)
		return err
	})
}

// ServerOnly renders content only on the server (no hydration).
func ServerOnly(content templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		// Render without any island markers
		return content.Render(ctx, w)
	})
}

// LazyIsland creates a lazily hydrated island.
func LazyIsland(name string, content templ.Component, threshold ...int) templ.Component {
	opts := IslandOptions{
		HydrationMode: component.HydrationVisible,
		Tag:           "div",
	}
	if len(threshold) > 0 {
		opts.LazyThreshold = threshold[0]
	}
	return Island(name, content, opts)
}

// IdleIsland creates an island that hydrates during idle time.
func IdleIsland(name string, content templ.Component, maxDelay ...int) templ.Component {
	opts := IslandOptions{
		HydrationMode: component.HydrationIdle,
		Tag:           "div",
	}
	if len(maxDelay) > 0 {
		opts.DeferDelay = maxDelay[0]
	}
	return Island(name, content, opts)
}

// InteractionIsland creates an island that hydrates on first interaction.
func InteractionIsland(name string, content templ.Component) templ.Component {
	return Island(name, content, IslandOptions{
		HydrationMode: component.HydrationInteraction,
		Tag:           "div",
	})
}

// HighPriorityIsland creates a high-priority island.
func HighPriorityIsland(name string, content templ.Component) templ.Component {
	return Island(name, content, IslandOptions{
		Priority:      component.PriorityHigh,
		HydrationMode: component.HydrationImmediate,
		Tag:           "div",
	})
}

// LowPriorityIsland creates a low-priority island.
func LowPriorityIsland(name string, content templ.Component) templ.Component {
	return Island(name, content, IslandOptions{
		Priority:      component.PriorityLow,
		HydrationMode: component.HydrationIdle,
		Tag:           "div",
	})
}

// buildIslandAttributes builds the HTML attributes for an island.
func buildIslandAttributes(island *component.Island, opts IslandOptions) map[string]string {
	attrs := map[string]string{
		"id":                  island.ID,
		"data-gospa-island":   island.Config.Name,
		"data-gospa-mode":     string(island.Config.HydrationMode),
		"data-gospa-priority": string(island.Config.Priority),
	}

	// Add props as JSON
	if len(island.Props) > 0 {
		if propsJSON, err := json.Marshal(island.Props); err == nil {
			attrs["data-gospa-props"] = string(propsJSON)
		}
	}

	// Add state as JSON
	if len(island.State) > 0 {
		if stateJSON, err := json.Marshal(island.State); err == nil {
			attrs["data-gospa-state"] = string(stateJSON)
		}
	}

	// Add lazy threshold
	if opts.LazyThreshold > 0 {
		attrs["data-gospa-threshold"] = fmt.Sprintf("%d", opts.LazyThreshold)
	}

	// Add defer delay
	if opts.DeferDelay > 0 {
		attrs["data-gospa-defer"] = fmt.Sprintf("%d", opts.DeferDelay)
	}

	// Add client-only flag
	if opts.ClientOnly {
		attrs["data-gospa-client-only"] = "true"
	}

	// Add server-only flag
	if opts.ServerOnly {
		attrs["data-gospa-server-only"] = "true"
	}

	// Add custom class
	if opts.Class != "" {
		attrs["class"] = opts.Class
	}

	return attrs
}

// renderIslandWrapper renders the island wrapper element.
func renderIslandWrapper(island *component.Island, attrs map[string]string, opts IslandOptions, w io.Writer) error {
	tag := opts.Tag
	if tag == "" {
		tag = "div"
	}

	// Build opening tag
	var sb strings.Builder
	sb.WriteString("<")
	sb.WriteString(tag)
	for name, value := range attrs {
		sb.WriteString(fmt.Sprintf(` %s="%s"`, name, templ.EscapeString(value)))
	}
	sb.WriteString(">")

	// Write content
	sb.WriteString(island.Children)

	// Closing tag
	sb.WriteString("</")
	sb.WriteString(tag)
	sb.WriteString(">")

	_, err := w.Write([]byte(sb.String()))
	return err
}

// IslandScript generates the script tag for island hydration.
func IslandScript() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		script := `<script data-gospa-islands="true">
window.__GOSPA_ISLANDS__ = window.__GOSPA_ISLANDS__ || [];
document.querySelectorAll('[data-gospa-island]').forEach(function(el) {
	var data = {
		id: el.id,
		name: el.getAttribute('data-gospa-island'),
		mode: el.getAttribute('data-gospa-mode'),
		priority: el.getAttribute('data-gospa-priority'),
		props: el.getAttribute('data-gospa-props'),
		state: el.getAttribute('data-gospa-state'),
		threshold: el.getAttribute('data-gospa-threshold'),
		defer: el.getAttribute('data-gospa-defer'),
		clientOnly: el.getAttribute('data-gospa-client-only') === 'true',
		serverOnly: el.getAttribute('data-gospa-server-only') === 'true'
	};
	window.__GOSPA_ISLANDS__.push(data);
});
</script>`
		_, err := w.Write([]byte(script))
		return err
	})
}

// SerializeIslands serializes all islands for client transfer.
func SerializeIslands() (string, error) {
	islands := component.GetAllIslands()
	data := make([]component.IslandData, len(islands))
	for i, island := range islands {
		data[i] = island.ToData()
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to serialize islands: %w", err)
	}

	return string(jsonData), nil
}

// IslandDataScript generates a script with serialized island data.
func IslandDataScript() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		data, err := SerializeIslands()
		if err != nil {
			return err
		}

		script := fmt.Sprintf(`<script data-gospa-island-data="true">window.__GOSPA_ISLAND_DATA__ = %s;</script>`, data)
		_, err = w.Write([]byte(script))
		return err
	})
}
