// Package fiber provides error overlay functionality for development.
package fiber

import (
	"fmt"
	"html/template"
	"net/http"
	"runtime"
	"strings"
)

// ErrorInfo contains information about an error for display.
type ErrorInfo struct {
	Message     string       `json:"message"`
	Type        string       `json:"type"`
	Stack       []StackFrame `json:"stack"`
	File        string       `json:"file"`
	Line        int          `json:"line"`
	Column      int          `json:"column"`
	CodeSnippet string       `json:"codeSnippet"`
	Timestamp   int64        `json:"timestamp"`
	Request     *RequestInfo `json:"request,omitempty"`
	Cause       *ErrorInfo   `json:"cause,omitempty"`
}

// StackFrame represents a single frame in the stack trace.
type StackFrame struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function"`
	Source   string `json:"source,omitempty"`
}

// RequestInfo contains information about the request that caused the error.
type RequestInfo struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Query   map[string]string `json:"query,omitempty"`
}

// ErrorOverlayConfig configures the error overlay behavior.
type ErrorOverlayConfig struct {
	Enabled     bool
	ShowStack   bool
	ShowRequest bool
	ShowCode    bool
	Theme       string // "dark" or "light"
	Editor      string // editor to open files in (e.g., "code", "idea")
	EditorPort  int
}

// DefaultErrorOverlayConfig returns the default configuration.
func DefaultErrorOverlayConfig() ErrorOverlayConfig {
	return ErrorOverlayConfig{
		Enabled:     true,
		ShowStack:   true,
		ShowRequest: true,
		ShowCode:    true,
		Theme:       "dark",
		Editor:      "code",
		EditorPort:  0,
	}
}

// ErrorOverlay handles error display in development.
type ErrorOverlay struct {
	config ErrorOverlayConfig
}

// NewErrorOverlay creates a new error overlay handler.
func NewErrorOverlay(config ErrorOverlayConfig) *ErrorOverlay {
	return &ErrorOverlay{config: config}
}

// RenderOverlay renders the error overlay HTML.
func (e *ErrorOverlay) RenderOverlay(err error, req *http.Request) string {
	info := e.parseError(err, req)
	return e.renderHTML(info)
}

// parseError extracts error information from an error.
func (e *ErrorOverlay) parseError(err error, req *http.Request) *ErrorInfo {
	info := &ErrorInfo{
		Message:   err.Error(),
		Type:      fmt.Sprintf("%T", err),
		Stack:     e.extractStack(err),
		Timestamp: getCurrentTimestamp(),
	}

	// Extract file and line from first stack frame
	if len(info.Stack) > 0 {
		info.File = info.Stack[0].File
		info.Line = info.Stack[0].Line
	}

	// Add request info if available
	if req != nil && e.config.ShowRequest {
		info.Request = &RequestInfo{
			Method:  req.Method,
			URL:     req.URL.String(),
			Headers: extractHeaders(req),
			Query:   extractQuery(req),
		}
	}

	// Check for cause (wrapped errors)
	if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
		if cause := unwrapper.Unwrap(); cause != nil {
			info.Cause = e.parseError(cause, nil)
		}
	}

	return info
}

// extractStack extracts stack frames from an error.
func (e *ErrorOverlay) extractStack(err error) []StackFrame {
	var frames []StackFrame

	// Try to get stack from error if it implements StackTracer
	if stackTracer, ok := err.(interface{ StackTrace() []StackFrame }); ok {
		return stackTracer.StackTrace()
	}

	// Get current stack
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	if n == 0 {
		return frames
	}

	frames = make([]StackFrame, 0, n)
	callers := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := callers.Next()

		// Skip runtime and standard library frames
		if strings.HasPrefix(frame.File, "runtime/") ||
			strings.Contains(frame.File, "go/src/") ||
			strings.Contains(frame.File, "go/pkg/") {
			if !more {
				break
			}
			continue
		}

		frames = append(frames, StackFrame{
			File:     frame.File,
			Line:     frame.Line,
			Function: frame.Function,
		})

		if !more {
			break
		}
	}

	return frames
}

// renderHTML generates the HTML for the error overlay.
func (e *ErrorOverlay) renderHTML(info *ErrorInfo) string {
	theme := e.config.Theme
	if theme == "" {
		theme = "dark"
	}

	// Build stack trace HTML
	stackHTML := e.buildStackHTML(info.Stack)

	// Build request info HTML
	requestHTML := ""
	if info.Request != nil {
		requestHTML = e.buildRequestHTML(info.Request)
	}

	// Build cause chain HTML
	causeHTML := ""
	if info.Cause != nil {
		causeHTML = e.buildCauseHTML(info.Cause)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Error: %s</title>
	<style>
		:root {
			--bg-primary: %s;
			--bg-secondary: %s;
			--bg-tertiary: %s;
			--text-primary: %s;
			--text-secondary: %s;
			--text-muted: %s;
			--accent: #ff4444;
			--accent-hover: #ff6666;
			--border: %s;
			--code-bg: %s;
		}
		* { box-sizing: border-box; margin: 0; padding: 0; }
		body {
			font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
			background: var(--bg-primary);
			color: var(--text-primary);
			line-height: 1.6;
			padding: 20px;
			min-height: 100vh;
		}
		.container {
			max-width: 1200px;
			margin: 0 auto;
		}
		.error-header {
			background: var(--bg-secondary);
			border: 1px solid var(--border);
			border-radius: 8px;
			padding: 24px;
			margin-bottom: 20px;
		}
		.error-type {
			font-size: 12px;
			color: var(--accent);
			text-transform: uppercase;
			letter-spacing: 1px;
			margin-bottom: 8px;
		}
		.error-message {
			font-size: 24px;
			font-weight: 600;
			margin-bottom: 16px;
			word-break: break-word;
		}
		.error-location {
			font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
			font-size: 14px;
			color: var(--text-secondary);
			background: var(--code-bg);
			padding: 8px 12px;
			border-radius: 4px;
			display: inline-flex;
			align-items: center;
			gap: 8px;
		}
		.error-location a {
			color: var(--text-secondary);
			text-decoration: none;
		}
		.error-location a:hover {
			color: var(--accent);
		}
		.section {
			background: var(--bg-secondary);
			border: 1px solid var(--border);
			border-radius: 8px;
			margin-bottom: 20px;
			overflow: hidden;
		}
		.section-header {
			background: var(--bg-tertiary);
			padding: 12px 16px;
			font-weight: 600;
			font-size: 14px;
			border-bottom: 1px solid var(--border);
			display: flex;
			align-items: center;
			gap: 8px;
		}
		.section-content {
			padding: 16px;
		}
		.stack-frame {
			padding: 12px;
			border-bottom: 1px solid var(--border);
			cursor: pointer;
			transition: background 0.15s;
		}
		.stack-frame:hover {
			background: var(--bg-tertiary);
		}
		.stack-frame:last-child {
			border-bottom: none;
		}
		.stack-frame-header {
			display: flex;
			justify-content: space-between;
			align-items: flex-start;
			margin-bottom: 4px;
		}
		.stack-function {
			font-family: 'SF Mono', Monaco, monospace;
			font-size: 13px;
			color: var(--text-primary);
		}
		.stack-file {
			font-family: 'SF Mono', Monaco, monospace;
			font-size: 12px;
			color: var(--text-secondary);
		}
		.stack-file a {
			color: var(--text-secondary);
			text-decoration: none;
		}
		.stack-file a:hover {
			color: var(--accent);
		}
		.code-block {
			background: var(--code-bg);
			border-radius: 4px;
			overflow: hidden;
			margin-top: 8px;
		}
		.code-line {
			display: flex;
			font-family: 'SF Mono', Monaco, monospace;
			font-size: 13px;
		}
		.code-line.highlight {
			background: rgba(255, 68, 68, 0.15);
		}
		.code-line-number {
			min-width: 50px;
			padding: 0 12px;
			color: var(--text-muted);
			text-align: right;
			user-select: none;
			background: var(--bg-tertiary);
		}
		.code-line-content {
			padding: 0 12px;
			white-space: pre;
		}
		.request-info {
			font-family: 'SF Mono', Monaco, monospace;
			font-size: 13px;
		}
		.request-row {
			display: flex;
			padding: 8px 0;
			border-bottom: 1px solid var(--border);
		}
		.request-row:last-child {
			border-bottom: none;
		}
		.request-key {
			min-width: 120px;
			color: var(--text-secondary);
		}
		.request-value {
			color: var(--text-primary);
			word-break: break-all;
		}
		.cause-chain {
			padding-left: 20px;
			border-left: 2px solid var(--border);
			margin-top: 12px;
		}
		.cause-item {
			padding: 12px;
			background: var(--bg-tertiary);
			border-radius: 4px;
			margin-bottom: 8px;
		}
		.cause-type {
			font-size: 11px;
			color: var(--accent);
			text-transform: uppercase;
			letter-spacing: 0.5px;
		}
		.cause-message {
			font-size: 14px;
			margin-top: 4px;
		}
		.actions {
			display: flex;
			gap: 12px;
			margin-top: 16px;
		}
		.btn {
			padding: 8px 16px;
			border-radius: 4px;
			font-size: 14px;
			cursor: pointer;
			border: none;
			transition: all 0.15s;
		}
		.btn-primary {
			background: var(--accent);
			color: white;
		}
		.btn-primary:hover {
			background: var(--accent-hover);
		}
		.btn-secondary {
			background: var(--bg-tertiary);
			color: var(--text-primary);
			border: 1px solid var(--border);
		}
		.btn-secondary:hover {
			background: var(--bg-secondary);
		}
		.icon {
			display: inline-block;
			width: 16px;
			height: 16px;
			margin-right: 4px;
		}
	</style>
</head>
<body>
	<div class="container">
		<div class="error-header">
			<div class="error-type">%s</div>
			<div class="error-message">%s</div>
			<div class="error-location">
				<span>📍</span>
				<a href="%s" title="Open in editor">%s:%d</a>
			</div>
			<div class="actions">
				<button class="btn btn-primary" onclick="copyError()">📋 Copy Error</button>
				<button class="btn btn-secondary" onclick="location.reload()">🔄 Reload</button>
			</div>
		</div>

		%s

		<div class="section">
			<div class="section-header">
				<span>📚</span> Stack Trace
			</div>
			<div class="section-content">
				%s
			</div>
		</div>

		%s
	</div>

	<script>
		function copyError() {
			const errorText = document.querySelector('.error-message').textContent;
			const location = document.querySelector('.error-location a').textContent;
			navigator.clipboard.writeText(errorText + '\\n  at ' + location);
		}
	</script>
</body>
</html>`,
		escapeHTML(info.Message),
		getThemeColor(theme, "bgPrimary"),
		getThemeColor(theme, "bgSecondary"),
		getThemeColor(theme, "bgTertiary"),
		getThemeColor(theme, "textPrimary"),
		getThemeColor(theme, "textSecondary"),
		getThemeColor(theme, "textMuted"),
		getThemeColor(theme, "border"),
		getThemeColor(theme, "codeBg"),
		escapeHTML(info.Type),
		escapeHTML(info.Message),
		e.buildEditorURL(info.File, info.Line),
		escapeHTML(info.File),
		info.Line,
		requestHTML,
		stackHTML,
		causeHTML,
	)
}

// buildStackHTML generates HTML for the stack trace.
func (e *ErrorOverlay) buildStackHTML(frames []StackFrame) string {
	if len(frames) == 0 {
		return "<div style='color: var(--text-muted);'>No stack trace available</div>"
	}

	var html strings.Builder
	for i, frame := range frames {
		editorURL := e.buildEditorURL(frame.File, frame.Line)
		html.WriteString(fmt.Sprintf(`
		<div class="stack-frame" onclick="toggleFrame(this)">
			<div class="stack-frame-header">
				<div class="stack-function">%s</div>
			</div>
			<div class="stack-file">
				<a href="%s" title="Open in editor">%s:%d</a>
			</div>
		</div>`,
			escapeHTML(frame.Function),
			editorURL,
			escapeHTML(frame.File),
			frame.Line,
		))

		// Only show first 10 frames by default
		if i >= 9 {
			break
		}
	}

	return html.String()
}

// buildRequestHTML generates HTML for request information.
func (e *ErrorOverlay) buildRequestHTML(req *RequestInfo) string {
	var rows strings.Builder

	rows.WriteString(fmt.Sprintf(`
		<div class="request-row">
			<div class="request-key">Method</div>
			<div class="request-value">%s</div>
		</div>
		<div class="request-row">
			<div class="request-key">URL</div>
			<div class="request-value">%s</div>
		</div>`,
		escapeHTML(req.Method),
		escapeHTML(req.URL),
	))

	// Add query params
	for key, value := range req.Query {
		rows.WriteString(fmt.Sprintf(`
		<div class="request-row">
			<div class="request-key">Query[%s]</div>
			<div class="request-value">%s</div>
		</div>`,
			escapeHTML(key),
			escapeHTML(value),
		))
	}

	return fmt.Sprintf(`
		<div class="section">
			<div class="section-header">
				<span>🌐</span> Request
			</div>
			<div class="section-content">
				<div class="request-info">
					%s
				</div>
			</div>
		</div>`,
		rows.String(),
	)
}

// buildCauseHTML generates HTML for the error cause chain.
func (e *ErrorOverlay) buildCauseHTML(cause *ErrorInfo) string {
	var html strings.Builder

	for current := cause; current != nil; current = current.Cause {
		html.WriteString(fmt.Sprintf(`
			<div class="cause-item">
				<div class="cause-type">%s</div>
				<div class="cause-message">%s</div>
			</div>`,
			escapeHTML(current.Type),
			escapeHTML(current.Message),
		))
	}

	return fmt.Sprintf(`
		<div class="section">
			<div class="section-header">
				<span>🔗</span> Caused By
			</div>
			<div class="section-content">
				<div class="cause-chain">
					%s
				</div>
			</div>
		</div>`,
		html.String(),
	)
}

// buildEditorURL generates a URL to open the file in an editor.
func (e *ErrorOverlay) buildEditorURL(file string, line int) string {
	editor := e.config.Editor
	switch editor {
	case "code":
		return fmt.Sprintf("vscode://file/%s:%d", file, line)
	case "idea":
		return fmt.Sprintf("idea://open?file=%s&line=%d", file, line)
	case "sublime":
		return fmt.Sprintf("subl://open?url=file://%s&line=%d", file, line)
	default:
		return fmt.Sprintf("vscode://file/%s:%d", file, line)
	}
}

// Helper functions

func escapeHTML(s string) string {
	return template.HTMLEscapeString(s)
}

func getCurrentTimestamp() int64 {
	return 0 // Would use time.Now().Unix() in real implementation
}

func extractHeaders(req *http.Request) map[string]string {
	headers := make(map[string]string)
	for key, values := range req.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	return headers
}

func extractQuery(req *http.Request) map[string]string {
	query := make(map[string]string)
	for key, values := range req.URL.Query() {
		if len(values) > 0 {
			query[key] = values[0]
		}
	}
	return query
}

func getThemeColor(theme, name string) string {
	colors := map[string]map[string]string{
		"dark": {
			"bgPrimary":     "#1a1a1a",
			"bgSecondary":   "#242424",
			"bgTertiary":    "#2a2a2a",
			"textPrimary":   "#ffffff",
			"textSecondary": "#a0a0a0",
			"textMuted":     "#666666",
			"border":        "#333333",
			"codeBg":        "#1e1e1e",
		},
		"light": {
			"bgPrimary":     "#ffffff",
			"bgSecondary":   "#f5f5f5",
			"bgTertiary":    "#ebebeb",
			"textPrimary":   "#1a1a1a",
			"textSecondary": "#666666",
			"textMuted":     "#999999",
			"border":        "#e0e0e0",
			"codeBg":        "#f8f8f8",
		},
	}

	if t, ok := colors[theme]; ok {
		if c, ok := t[name]; ok {
			return c
		}
	}
	return colors["dark"][name]
}
