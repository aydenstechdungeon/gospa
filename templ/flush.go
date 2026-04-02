package templ

import (
	"context"
	"io"

	"github.com/a-h/templ"
)

// Flusher defines the interface for flushing a writer.
type Flusher interface {
	Flush() error
}

// Flush returns a Templ component that attempts to flush the current writer.
// This is useful for sending the <head> section to the browser early,
// allowing it to start downloading CSS and JS while the server is still
// rendering the rest of the page.
func Flush() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		// Check if the writer itself is a flusher
		if f, ok := w.(Flusher); ok {
			return f.Flush()
		}

		// Note: Templ sometimes wraps writers in its own internal buffers.
		// If using templ.ComponentFunc directly, 'w' might be the underlying writer.
		return nil
	})
}
