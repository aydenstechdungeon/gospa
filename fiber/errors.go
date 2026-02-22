package fiber

import (
	"encoding/json"
	"fmt"
	"runtime/debug"

	"github.com/aydenstechdungeon/gospa/state"
	"github.com/gofiber/fiber/v2"
)

// ErrorCode represents an error code.
type ErrorCode string

const (
	ErrorCodeInternal     ErrorCode = "INTERNAL_ERROR"
	ErrorCodeNotFound     ErrorCode = "NOT_FOUND"
	ErrorCodeBadRequest   ErrorCode = "BAD_REQUEST"
	ErrorCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrorCodeForbidden    ErrorCode = "FORBIDDEN"
	ErrorCodeConflict     ErrorCode = "CONFLICT"
	ErrorCodeValidation   ErrorCode = "VALIDATION_ERROR"
	ErrorCodeTimeout      ErrorCode = "TIMEOUT"
	ErrorCodeUnavailable  ErrorCode = "SERVICE_UNAVAILABLE"
)

// AppError represents an application error.
type AppError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Stack      string                 `json:"stack,omitempty"`
	StatusCode int                    `json:"-"`
	Recover    bool                   `json:"-"` // Whether state can be recovered
}

// Error implements the error interface.
func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewAppError creates a new application error.
func NewAppError(code ErrorCode, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Details:    make(map[string]interface{}),
		Recover:    true,
	}
}

// WithDetails adds details to the error.
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	e.Details = details
	return e
}

// WithStack adds a stack trace to the error.
func (e *AppError) WithStack(stack string) *AppError {
	e.Stack = stack
	return e
}

// WithRecover sets whether state can be recovered.
func (e *AppError) WithRecover(recover bool) *AppError {
	e.Recover = recover
	return e
}

// Common errors.
var (
	ErrInternal     = NewAppError(ErrorCodeInternal, "Internal server error", fiber.StatusInternalServerError)
	ErrNotFound     = NewAppError(ErrorCodeNotFound, "Resource not found", fiber.StatusNotFound)
	ErrBadRequest   = NewAppError(ErrorCodeBadRequest, "Bad request", fiber.StatusBadRequest)
	ErrUnauthorized = NewAppError(ErrorCodeUnauthorized, "Unauthorized", fiber.StatusUnauthorized)
	ErrForbidden    = NewAppError(ErrorCodeForbidden, "Forbidden", fiber.StatusForbidden)
	ErrConflict     = NewAppError(ErrorCodeConflict, "Conflict", fiber.StatusConflict)
	ErrValidation   = NewAppError(ErrorCodeValidation, "Validation error", fiber.StatusBadRequest)
	ErrTimeout      = NewAppError(ErrorCodeTimeout, "Request timeout", fiber.StatusRequestTimeout)
	ErrUnavailable  = NewAppError(ErrorCodeUnavailable, "Service unavailable", fiber.StatusServiceUnavailable)
)

// ErrorHandlerConfig holds error handler configuration.
type ErrorHandlerConfig struct {
	// DevMode enables development features like stack traces
	DevMode bool
	// StateKey is the context key for state
	StateKey string
	// CustomErrorPages maps error codes to custom handlers
	CustomErrorPages map[ErrorCode]func(*fiber.Ctx, *AppError) error
	// OnError is called when an error occurs
	OnError func(*fiber.Ctx, *AppError)
	// RecoverState attempts to recover state on error
	RecoverState bool
}

// DefaultErrorHandlerConfig returns default error handler configuration.
func DefaultErrorHandlerConfig() ErrorHandlerConfig {
	return ErrorHandlerConfig{
		DevMode:          false,
		StateKey:         "gospa.state",
		RecoverState:     true,
		CustomErrorPages: make(map[ErrorCode]func(*fiber.Ctx, *AppError) error),
	}
}

// ErrorHandler creates a Fiber error handler.
func ErrorHandler(config ErrorHandlerConfig) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		// Convert to AppError
		var appErr *AppError
		switch e := err.(type) {
		case *AppError:
			appErr = e
		case *fiber.Error:
			appErr = NewAppError(ErrorCodeInternal, e.Message, e.Code)
		default:
			appErr = NewAppError(ErrorCodeInternal, err.Error(), fiber.StatusInternalServerError)
			if config.DevMode {
				appErr = appErr.WithStack(string(debug.Stack()))
			}
		}

		// Call error hook
		if config.OnError != nil {
			config.OnError(c, appErr)
		}

		// Check for custom error page
		if handler, ok := config.CustomErrorPages[appErr.Code]; ok {
			return handler(c, appErr)
		}

		// Recover state if possible
		var stateData map[string]interface{}
		if config.RecoverState {
			if stateMap, ok := c.Locals(config.StateKey).(*state.StateMap); ok && stateMap != nil {
				if jsonData, err := stateMap.ToJSON(); err == nil {
					_ = json.Unmarshal([]byte(jsonData), &stateData)
				}
			}
		}

		// Determine response type
		accept := string(c.Request().Header.Peek("Accept"))
		if accept != "" && len(accept) >= 4 && accept[:4] == "appl" && len(accept) >= 16 && accept[:16] == "application/json" {
			// JSON response
			return c.Status(appErr.StatusCode).JSON(fiber.Map{
				"error":   appErr.Code,
				"message": appErr.Message,
				"details": appErr.Details,
				"recover": appErr.Recover,
				"state":   stateData,
			})
		}

		// HTML response
		return renderErrorPage(c, appErr, stateData, config.DevMode)
	}
}

// renderErrorPage renders an error page.
func renderErrorPage(c *fiber.Ctx, appErr *AppError, stateData map[string]interface{}, devMode bool) error {
	c.Set("Content-Type", "text/html; charset=utf-8")

	// Build error page HTML
	html := `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Error - ` + string(appErr.Code) + `</title>
	<style>
		* { margin: 0; padding: 0; box-sizing: border-box; }
		body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #1a1a2e; color: #eee; min-height: 100vh; display: flex; align-items: center; justify-content: center; }
		.container { max-width: 600px; padding: 2rem; text-align: center; }
		.error-code { font-size: 4rem; font-weight: bold; color: #e94560; margin-bottom: 1rem; }
		.error-message { font-size: 1.5rem; margin-bottom: 2rem; color: #ccc; }
		.error-details { background: #16213e; padding: 1rem; border-radius: 8px; margin-bottom: 2rem; text-align: left; font-family: monospace; font-size: 0.9rem; overflow-x: auto; }
		.error-details pre { white-space: pre-wrap; word-wrap: break-word; }
		.actions { display: flex; gap: 1rem; justify-content: center; }
		.btn { padding: 0.75rem 1.5rem; border-radius: 8px; text-decoration: none; font-weight: 500; transition: all 0.2s; }
		.btn-primary { background: #e94560; color: white; }
		.btn-primary:hover { background: #ff6b6b; }
		.btn-secondary { background: #16213e; color: #ccc; border: 1px solid #333; }
		.btn-secondary:hover { background: #1a1a2e; }
		.stack { margin-top: 2rem; background: #0f0f23; padding: 1rem; border-radius: 8px; text-align: left; font-family: monospace; font-size: 0.8rem; overflow-x: auto; max-height: 300px; overflow-y: auto; }
		.stack pre { white-space: pre-wrap; word-wrap: break-word; }
	</style>
</head>
<body>
	<div class="container">
		<div class="error-code">` + string(appErr.Code) + `</div>
		<div class="error-message">` + appErr.Message + `</div>`

	// Add details if present
	if len(appErr.Details) > 0 {
		detailsJSON, _ := json.MarshalIndent(appErr.Details, "", "  ")
		html += `
		<div class="error-details">
			<pre>` + string(detailsJSON) + `</pre>
		</div>`
	}

	// Add actions
	html += `
		<div class="actions">
			<a href="/" class="btn btn-primary">Go Home</a>
			<a href="javascript:history.back()" class="btn btn-secondary">Go Back</a>
		</div>`

	// Add stack trace in dev mode
	if devMode && appErr.Stack != "" {
		html += `
		<div class="stack">
			<pre>` + appErr.Stack + `</pre>
		</div>`
	}

	// Add state recovery script
	if stateData != nil {
		stateJSON, _ := json.Marshal(stateData)
		html += `
		<script>
			window.__GOSPA_STATE__ = ` + string(stateJSON) + `;
			window.__GOSPA_ERROR__ = {
				code: "` + string(appErr.Code) + `",
				message: "` + appErr.Message + `",
				recover: ` + fmt.Sprintf("%v", appErr.Recover) + `
			};
		</script>`
	}

	html += `
	</div>
</body>
</html>`

	return c.Status(appErr.StatusCode).SendString(html)
}

// NotFoundHandler creates a 404 handler.
func NotFoundHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return NewAppError(ErrorCodeNotFound, "Page not found: "+c.Path(), fiber.StatusNotFound).
			WithRecover(true)
	}
}

// ValidationError creates a validation error.
func ValidationError(field, message string) *AppError {
	return NewAppError(ErrorCodeValidation, "Validation failed", fiber.StatusBadRequest).
		WithDetails(map[string]interface{}{
			"field":   field,
			"message": message,
		})
}

// ValidationErrors creates a validation error with multiple fields.
func ValidationErrors(errors map[string]string) *AppError {
	details := make(map[string]interface{})
	for field, msg := range errors {
		details[field] = msg
	}
	return NewAppError(ErrorCodeValidation, "Validation failed", fiber.StatusBadRequest).
		WithDetails(details)
}

// PanicHandler creates a panic recovery handler.
func PanicHandler(config ErrorHandlerConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				var err error
				switch v := r.(type) {
				case error:
					err = v
				case string:
					err = fmt.Errorf("%s", v)
				default:
					err = fmt.Errorf("%v", v)
				}

				appErr := NewAppError(ErrorCodeInternal, err.Error(), fiber.StatusInternalServerError).
					WithRecover(true)

				if config.DevMode {
					appErr = appErr.WithStack(string(debug.Stack()))
				}

				// Use error handler
				_ = ErrorHandler(config)(c, appErr)
			}
		}()
		return c.Next()
	}
}

// IsAppError checks if an error is an AppError.
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// AsAppError converts an error to AppError.
func AsAppError(err error) (*AppError, bool) {
	if appErr, ok := err.(*AppError); ok {
		return appErr, true
	}
	return nil, false
}

// WrapError wraps a generic error into an AppError.
func WrapError(err error, code ErrorCode, statusCode int) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return NewAppError(code, err.Error(), statusCode)
}
