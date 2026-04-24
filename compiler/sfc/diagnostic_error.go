package sfc

import "fmt"

// DiagnosticError is a structured compile diagnostic with exact source location
// and optional remediation guidance.
type DiagnosticError struct {
	Line       int
	Column     int
	Message    string
	Suggestion string
	Snippet    string
}

// Error implements error.
func (e *DiagnosticError) Error() string {
	if e == nil {
		return "<nil>"
	}
	msg := fmt.Sprintf("at %d:%d: %s", e.Line, e.Column, e.Message)
	if e.Suggestion != "" {
		msg += "\nSuggestion: " + e.Suggestion
	}
	if e.Snippet != "" {
		msg += "\nTry:\n" + e.Snippet
	}
	return msg
}

func newDiagnosticError(line, col int, message, suggestion, snippet string) error {
	return &DiagnosticError{
		Line:       line,
		Column:     col,
		Message:    message,
		Suggestion: suggestion,
		Snippet:    snippet,
	}
}
