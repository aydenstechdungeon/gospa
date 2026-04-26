// Package kit provides control-flow helpers for route load/action handlers.
package kit

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aydenstechdungeon/gospa/routing"
)

// RedirectError represents a control-flow redirect for load/actions.
type RedirectError struct {
	Status   int
	Location string
}

func (e *RedirectError) Error() string {
	return fmt.Sprintf("redirect(%d,%q)", e.Status, e.Location)
}

// FailError represents a structured non-500 failure for load/actions.
type FailError struct {
	Status int
	Data   interface{}
}

func (e *FailError) Error() string {
	return fmt.Sprintf("fail(%d)", e.Status)
}

// HTTPError represents typed HTTP control-flow errors in load/actions.
type HTTPError struct {
	Status int
	Body   interface{}
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("error(%d)", e.Status)
}

// Redirect creates a redirect control-flow error.
func Redirect(status int, to string) error {
	if status == 0 {
		status = 303
	}
	return &RedirectError{
		Status:   status,
		Location: to,
	}
}

// Fail creates a structured failure control-flow error.
func Fail(status int, data interface{}) error {
	if status == 0 {
		status = 400
	}
	return &FailError{
		Status: status,
		Data:   data,
	}
}

// Error creates a typed HTTP control-flow error.
func Error(status int, body interface{}) error {
	if status == 0 {
		status = 500
	}
	return &HTTPError{
		Status: status,
		Body:   body,
	}
}

// AsRedirect extracts RedirectError when present.
func AsRedirect(err error) (*RedirectError, bool) {
	if err == nil {
		return nil, false
	}
	var target *RedirectError
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}

// AsFail extracts FailError when present.
func AsFail(err error) (*FailError, bool) {
	if err == nil {
		return nil, false
	}
	var target *FailError
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}

// AsError extracts HTTPError when present.
func AsError(err error) (*HTTPError, bool) {
	if err == nil {
		return nil, false
	}
	var target *HTTPError
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}

// Parent returns the nearest parent layout data for the current load/action execution scope.
func Parent[T any](c routing.LoadContext) (T, error) {
	var zero T
	parent, ok := parentDataFromContext(c)
	if !ok || parent == nil {
		return zero, errors.New("kit.Parent: no parent data available")
	}

	raw, err := json.Marshal(parent)
	if err != nil {
		return zero, fmt.Errorf("kit.Parent: marshal parent data: %w", err)
	}

	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		return zero, fmt.Errorf("kit.Parent: decode parent data: %w", err)
	}
	return out, nil
}
