// Package templ provides PPR (Partial Prerendering) helpers for GoSPA.
//
// PPR splits a page into a static shell (cached at first render) and named
// dynamic slots that are re-rendered fresh on every request. The shell is
// streamed first with slot placeholders (<!--gospa-slot:name-->), then each
// dynamic slot fragment is appended inline.
package templ

import (
	"context"
	"fmt"
	"io"

	"github.com/a-h/templ"
)

// pprShellKey is the context key used to signal that we are building the PPR
// static shell. SlotFunc implementations should check IsPPRShellBuild and emit
// only a placeholder if true.
type pprShellKey struct{}

// WithPPRShellBuild attaches a marker to ctx indicating that the current render
// pass is building the PPR shell (not a live request render).
func WithPPRShellBuild(ctx context.Context) context.Context {
	return context.WithValue(ctx, pprShellKey{}, true)
}

// IsPPRShellBuild reports whether ctx is in a PPR shell-build pass.
func IsPPRShellBuild(ctx context.Context) bool {
	v, _ := ctx.Value(pprShellKey{}).(bool)
	return v
}

// DynamicSlot creates a templ.Component that behaves differently depending on
// the rendering pass:
//
//   - Shell build (IsPPRShellBuild == true): emits <!--gospa-slot:name--> placeholder.
//   - Normal request: wraps content in <div data-gospa-slot="name">…</div>.
//
// Use this inside page templ components registered with StrategyPPR. The slot
// name must match a key in RouteOptions.DynamicSlots and a SlotFunc registered
// with routing.RegisterSlot for the same page path.
func DynamicSlot(name string, content templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if IsPPRShellBuild(ctx) {
			// Emit a comment placeholder so the framework can locate and replace it.
			_, err := fmt.Fprintf(w, "<!--gospa-slot:%s-->", name)
			return err
		}
		// Live render: wrap in a slot marker div.
		if _, err := fmt.Fprintf(w, `<div data-gospa-slot="%s">`, name); err != nil {
			return err
		}
		if err := content.Render(ctx, w); err != nil {
			return err
		}
		_, err := fmt.Fprint(w, `</div>`)
		return err
	})
}

// SlotPlaceholder returns the raw placeholder comment string for a given slot
// name. This is used internally by the PPR streamer to locate replacement
// positions in the cached shell HTML.
func SlotPlaceholder(name string) string {
	return fmt.Sprintf("<!--gospa-slot:%s-->", name)
}
