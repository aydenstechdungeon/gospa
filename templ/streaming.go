// Package templ provides streaming SSR support for GoSPA.
package templ

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/a-h/templ"
	"github.com/aydenstechdungeon/gospa/component"
)

// StreamChunk represents a chunk of streamed content.
type StreamChunk struct {
	Type    string         `json:"type"`    // "html", "island", "script", "state", "error"
	ID      string         `json:"id"`      // Island ID or chunk identifier
	Content string         `json:"content"` // HTML content
	Data    map[string]any `json:"data"`    // Additional data (props, state, etc.)
}

// StreamRenderer handles streaming SSR operations.
type StreamRenderer struct {
	islandRegistry *component.IslandRegistry
	flushInterval  time.Duration
	bufferSize     int
}

// StreamRendererConfig configures the stream renderer.
type StreamRendererConfig struct {
	FlushInterval  time.Duration
	BufferSize     int
	IslandRegistry *component.IslandRegistry
}

// NewStreamRenderer creates a new streaming renderer.
func NewStreamRenderer(config StreamRendererConfig) *StreamRenderer {
	if config.FlushInterval == 0 {
		config.FlushInterval = 50 * time.Millisecond
	}
	if config.BufferSize == 0 {
		config.BufferSize = 4096
	}
	return &StreamRenderer{
		islandRegistry: config.IslandRegistry,
		flushInterval:  config.FlushInterval,
		bufferSize:     config.BufferSize,
	}
}

// StreamWriter handles writing streamed content.
type StreamWriter struct {
	w            io.Writer
	flusher      interface{ Flush() error }
	chunks       chan StreamChunk
	done         chan struct{}
	err          error
	mu           sync.Mutex
	wroteHeader  bool
	pendingChunk strings.Builder
}

// NewStreamWriter creates a new stream writer.
func NewStreamWriter(w io.Writer, flusher interface{ Flush() error }, bufferSize int) *StreamWriter {
	sw := &StreamWriter{
		w:       w,
		flusher: flusher,
		chunks:  make(chan StreamChunk, 100),
		done:    make(chan struct{}),
	}
	go sw.processChunks()
	return sw
}

// Write writes content to the stream.
func (sw *StreamWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.w.Write(p)
}

// WriteChunk queues a chunk for streaming.
func (sw *StreamWriter) WriteChunk(chunk StreamChunk) {
	select {
	case sw.chunks <- chunk:
	case <-sw.done:
	}
}

// Flush flushes pending content.
func (sw *StreamWriter) Flush() error {
	if sw.flusher != nil {
		return sw.flusher.Flush()
	}
	return nil
}

// Close closes the stream writer.
func (sw *StreamWriter) Close() error {
	close(sw.done)
	close(sw.chunks)
	return sw.Flush()
}

// processChunks processes chunks from the channel.
func (sw *StreamWriter) processChunks() {
	for chunk := range sw.chunks {
		sw.mu.Lock()
		data, err := json.Marshal(chunk)
		if err != nil {
			sw.err = err
			sw.mu.Unlock()
			continue
		}
		fmt.Fprintf(sw.w, "<script>__GOSPA_STREAM__(%s)</script>\n", string(data))
		sw.Flush()
		sw.mu.Unlock()
	}
}

// StreamOptions configures streaming behavior.
type StreamOptions struct {
	// EnableProgressiveHydration enables progressive island hydration.
	EnableProgressiveHydration bool
	// CriticalIslands are islands to render immediately.
	CriticalIslands []string
	// DeferBelowFold defers islands below the fold.
	DeferBelowFold bool
	// MaxBufferSize is the max buffer before flushing.
	MaxBufferSize int
}

// StreamComponent streams a component with progressive enhancement.
func (sr *StreamRenderer) StreamComponent(
	ctx context.Context,
	component templ.Component,
	w io.Writer,
	flusher interface{ Flush() error },
	opts StreamOptions,
) error {
	sw := NewStreamWriter(w, flusher, sr.bufferSize)
	defer sw.Close()

	// Write streaming preamble
	if err := sr.writePreamble(sw); err != nil {
		return fmt.Errorf("failed to write preamble: %w", err)
	}

	// Render main content
	var buf strings.Builder
	if err := component.Render(ctx, &buf); err != nil {
		return fmt.Errorf("failed to render component: %w", err)
	}

	// Write main content
	if _, err := sw.Write([]byte(buf.String())); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	// Stream islands progressively
	if opts.EnableProgressiveHydration {
		if err := sr.streamIslands(ctx, sw, opts); err != nil {
			return fmt.Errorf("failed to stream islands: %w", err)
		}
	}

	// Write streaming epilogue
	if err := sr.writeEpilogue(sw); err != nil {
		return fmt.Errorf("failed to write epilogue: %w", err)
	}

	return sw.Flush()
}

// writePreamble writes the streaming initialization script.
func (sr *StreamRenderer) writePreamble(sw *StreamWriter) error {
	preamble := `<script>
window.__GOSPA_STREAM__ = window.__GOSPA_STREAM__ || function(chunk) {
	switch(chunk.type) {
		case 'html':
			var el = document.getElementById(chunk.id);
			if (el) el.innerHTML = chunk.content;
			break;
		case 'island':
			window.__GOSPA_ISLANDS__ = window.__GOSPA_ISLANDS__ || [];
			window.__GOSPA_ISLANDS__.push(chunk.data);
			break;
		case 'state':
			window.__GOSPA_STATE__ = window.__GOSPA_STATE__ || {};
			window.__GOSPA_STATE__[chunk.id] = chunk.data;
			break;
		case 'script':
			var script = document.createElement('script');
			script.textContent = chunk.content;
			document.head.appendChild(script);
			break;
		case 'error':
			console.error('GoSPA Stream Error:', chunk.content);
			break;
	}
};
</script>`
	_, err := sw.Write([]byte(preamble))
	return err
}

// writeEpilogue writes the streaming finalization script.
func (sr *StreamRenderer) writeEpilogue(sw *StreamWriter) error {
	epilogue := `<script>
if (window.__GOSPA_ISLANDS__ && window.__GOSPA_ISLAND_MANAGER__) {
	window.__GOSPA_ISLAND_MANAGER__.init();
}
</script>`
	_, err := sw.Write([]byte(epilogue))
	return err
}

// streamIslands streams island data progressively.
func (sr *StreamRenderer) streamIslands(ctx context.Context, sw *StreamWriter, opts StreamOptions) error {
	islands := component.GetAllIslands()

	// Sort islands by priority
	highPriority := make([]*component.Island, 0)
	normalPriority := make([]*component.Island, 0)
	lowPriority := make([]*component.Island, 0)

	for _, island := range islands {
		// Check if critical
		isCritical := false
		for _, critical := range opts.CriticalIslands {
			if island.Config.Name == critical {
				isCritical = true
				break
			}
		}

		if isCritical || island.Config.Priority == component.PriorityHigh {
			highPriority = append(highPriority, island)
		} else if island.Config.Priority == component.PriorityLow {
			lowPriority = append(lowPriority, island)
		} else {
			normalPriority = append(normalPriority, island)
		}
	}

	// Stream high priority islands immediately
	for _, island := range highPriority {
		if err := sr.streamIsland(sw, island); err != nil {
			return err
		}
	}

	// Stream normal priority islands
	for _, island := range normalPriority {
		if err := sr.streamIsland(sw, island); err != nil {
			return err
		}
	}

	// Stream low priority islands last
	for _, island := range lowPriority {
		if err := sr.streamIsland(sw, island); err != nil {
			return err
		}
	}

	return nil
}

// streamIsland streams a single island.
func (sr *StreamRenderer) streamIsland(sw *StreamWriter, island *component.Island) error {
	data := island.ToData()
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal island data: %w", err)
	}

	sw.WriteChunk(StreamChunk{
		Type: "island",
		ID:   island.ID,
		Data: map[string]any{
			"id":       island.ID,
			"name":     island.Config.Name,
			"mode":     string(island.Config.HydrationMode),
			"priority": string(island.Config.Priority),
			"props":    island.Props,
			"state":    island.State,
		},
		Content: string(jsonData),
	})

	return nil
}

// StreamState streams state updates to the client.
func (sr *StreamRenderer) StreamState(sw *StreamWriter, id string, state map[string]any) error {
	sw.WriteChunk(StreamChunk{
		Type: "state",
		ID:   id,
		Data: state,
	})
	return nil
}

// StreamScript streams a script to the client.
func (sr *StreamRenderer) StreamScript(sw *StreamWriter, script string) error {
	sw.WriteChunk(StreamChunk{
		Type:    "script",
		Content: script,
	})
	return nil
}

// StreamError streams an error to the client.
func (sr *StreamRenderer) StreamError(sw *StreamWriter, err error) error {
	sw.WriteChunk(StreamChunk{
		Type:    "error",
		Content: err.Error(),
	})
	return nil
}

// DeferredIsland represents an island to be loaded later.
type DeferredIsland struct {
	Name        string
	Placeholder templ.Component
	Loader      func() (templ.Component, error)
}

// Deferred renders a deferred island with a placeholder.
func Deferred(name string, placeholder templ.Component, loader func() (templ.Component, error)) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		// Render placeholder
		if placeholder != nil {
			if err := placeholder.Render(ctx, w); err != nil {
				return err
			}
		}

		// Add deferred loading marker
		fmt.Fprintf(w, `<script data-gospa-deferred="%s"></script>`, name)

		return nil
	})
}

// Suspense renders content with a fallback while loading.
func Suspense(loader func() (templ.Component, error), fallback templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		// Generate unique ID for this suspense boundary
		id := fmt.Sprintf("suspense-%d", time.Now().UnixNano())

		// Render fallback with wrapper
		fmt.Fprintf(w, `<div id="%s" data-gospa-suspense="loading">`, id)
		if fallback != nil {
			if err := fallback.Render(ctx, w); err != nil {
				return err
			}
		}
		w.Write([]byte(`</div>`))

		// Start async loading
		go func() {
			content, err := loader()
			if err != nil {
				// Stream error
				fmt.Fprintf(w, `<script>__GOSPA_STREAM__({type:'error',id:'%s',content:'%s'})</script>`, id, err.Error())
				return
			}

			var buf strings.Builder
			if err := content.Render(ctx, &buf); err != nil {
				fmt.Fprintf(w, `<script>__GOSPA_STREAM__({type:'error',id:'%s',content:'%s'})</script>`, id, err.Error())
				return
			}

			// Stream loaded content
			fmt.Fprintf(w, `<script>__GOSPA_STREAM__({type:'html',id:'%s',content:'%s'})</script>`, id, buf.String())
		}()

		return nil
	})
}
