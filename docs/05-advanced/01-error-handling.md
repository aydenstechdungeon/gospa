# GoSPA Error Handling

GoSPA provides a comprehensive error handling system with typed error codes, structured error responses, and middleware for consistent error management.

## Overview

The error handling system in `fiber/errors.go` provides:

- **AppError**: Structured application errors with codes
- **ErrorCode**: Typed error codes for categorization
- **ErrorHandler**: Middleware for consistent error responses
- **Error Pages**: Custom error page support

---

## ErrorCode

Error codes categorize errors for consistent handling.

### Available Error Codes

```go
const (
    ErrorCodeInternal    ErrorCode = "internal_error"    // 500
    ErrorCodeNotFound    ErrorCode = "not_found"         // 404
    ErrorCodeBadRequest  ErrorCode = "bad_request"       // 400
    ErrorCodeUnauthorized ErrorCode = "unauthorized"     // 401
    ErrorCodeForbidden   ErrorCode = "forbidden"         // 403
    ErrorCodeConflict    ErrorCode = "conflict"          // 409
    ErrorCodeValidation  ErrorCode = "validation_error"  // 422
    ErrorCodeTimeout     ErrorCode = "timeout"           // 408
    ErrorCodeUnavailable ErrorCode = "service_unavailable" // 503
)
```

### HTTP Status Mapping

| ErrorCode | HTTP Status |
|-----------|-------------|
| `ErrorCodeInternal` | 500 |
| `ErrorCodeNotFound` | 404 |
| `ErrorCodeBadRequest` | 400 |
| `ErrorCodeUnauthorized` | 401 |
| `ErrorCodeForbidden` | 403 |
| `ErrorCodeConflict` | 409 |
| `ErrorCodeValidation` | 422 |
| `ErrorCodeTimeout` | 408 |
| `ErrorCodeUnavailable` | 503 |

---

## AppError

The main error type for application errors.

### Creating Errors

```go
import "github.com/gospa/gospa/fiber"

// Basic error
err := fiber.NewAppError(fiber.ErrorCodeNotFound, "User not found")

// With details
err := fiber.NewAppError(fiber.ErrorCodeValidation, "Invalid input").
    WithDetails(map[string]any{
        "field": "email",
        "value": "invalid-email",
    })

// With stack trace (dev mode)
err := fiber.NewAppError(fiber.ErrorCodeInternal, "Database error").
    WithStack(debug.Stack())

// With recovery info
err := fiber.NewAppError(fiber.ErrorCodeInternal, "Panic recovered").
    WithRecover(recoveredValue)
```

### AppError Structure

```go
type AppError struct {
    Code      ErrorCode  `json:"code"`
    Message   string     `json:"message"`
    Details   any        `json:"details,omitempty"`
    Stack     []byte     `json:"-"`              // Not serialized
    Recover   any        `json:"-"`              // Panic recovery info
    Timestamp time.Time  `json:"timestamp"`
    RequestID string     `json:"requestId,omitempty"`
}
```

### Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `Error()` | `string` | Implements error interface |
| `WithDetails()` | `WithDetails(details any) *AppError` | Add error details |
| `WithStack()` | `WithStack(stack []byte) *AppError` | Add stack trace |
| `WithRecover()` | `WithRecover(r any) *AppError` | Add recovery info |
| `StatusCode()` | `int` | Get HTTP status code |
| `ToJSON()` | `([]byte, error)` | Serialize to JSON |

---

## Pre-defined Errors

```go
var (
    ErrInternal    = NewAppError(ErrorCodeInternal, "Internal server error")
    ErrNotFound    = NewAppError(ErrorCodeNotFound, "Resource not found")
    ErrBadRequest  = NewAppError(ErrorCodeBadRequest, "Bad request")
    ErrUnauthorized = NewAppError(ErrorCodeUnauthorized, "Unauthorized")
    ErrForbidden   = NewAppError(ErrorCodeForbidden, "Forbidden")
    ErrConflict    = NewAppError(ErrorCodeConflict, "Conflict")
    ErrValidation  = NewAppError(ErrorCodeValidation, "Validation error")
    ErrTimeout     = NewAppError(ErrorCodeTimeout, "Request timeout")
    ErrUnavailable = NewAppError(ErrorCodeUnavailable, "Service unavailable")
)
```

### Usage

```go
// Return pre-defined error
return fiber.ErrNotFound

// Clone with custom message
return fiber.ErrNotFound.WithDetails(map[string]any{
    "resource": "user",
    "id": userId,
})
```

---

## Validation Errors

### Single Field Error

```go
err := fiber.ValidationError("email", "Invalid email format")
// Returns AppError with code "validation_error"
```

### Multiple Field Errors

```go
errors := map[string]string{
    "email": "Invalid email format",
    "password": "Password must be at least 8 characters",
    "name": "Name is required",
}
err := fiber.ValidationErrors(errors)
// Returns AppError with all errors in details
```

### Example Response

```json
{
    "code": "validation_error",
    "message": "Validation error",
    "details": {
        "email": "Invalid email format",
        "password": "Password must be at least 8 characters",
        "name": "Name is required"
    },
    "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Error Handler Middleware

### Configuration

```go
type ErrorHandlerConfig struct {
    DevMode         bool                     // Include stack traces
    StateKey        string                   // State key for error pages
    CustomErrorPages map[int]templ.Component // Custom error pages
    OnError         func(*AppError)          // Error callback
    RecoverState    bool                     // Recover state on error
}
```

### Setup

```go
// Basic setup
app.Use(fiber.ErrorHandler(fiber.ErrorHandlerConfig{
    DevMode: config.DevMode,
}))

// With custom error pages
app.Use(fiber.ErrorHandler(fiber.ErrorHandlerConfig{
    DevMode:  config.DevMode,
    StateKey: "error",
    CustomErrorPages: map[int]templ.Component{
        404: NotFoundPage(),
        500: ServerErrorPage(),
    },
    OnError: func(err *fiber.AppError) {
        log.Printf("Error: %s - %s", err.Code, err.Message)
    },
}))
```

### Error Response Format

```json
{
    "code": "not_found",
    "message": "User not found",
    "details": {
        "userId": "123"
    },
    "timestamp": "2024-01-15T10:30:00Z",
    "requestId": "req-abc123"
}
```

---

## Not Found Handler

```go
// 404 handler
app.Use(fiber.NotFoundHandler())
```

Returns 404 for unmatched routes with proper error format.

---

## Panic Handler

Recover from panics and return structured errors.

```go
// Panic recovery middleware
app.Use(fiber.PanicHandler())
```

### Example

```go
app.Get("/panic", func(c *fiber.Ctx) error {
    panic("something went wrong")
    // PanicHandler catches this and returns:
    // {
    //   "code": "internal_error",
    //   "message": "Internal server error",
    //   "timestamp": "..."
    // }
})
```

---

## Helper Functions

### IsAppError

Check if an error is an AppError.

```go
if fiber.IsAppError(err) {
    // Handle as AppError
}
```

### AsAppError

Convert error to AppError.

```go
appErr, ok := fiber.AsAppError(err)
if ok {
    fmt.Println("Code:", appErr.Code)
    fmt.Println("Message:", appErr.Message)
}
```

### WrapError

Wrap any error as an AppError.

```go
err := someFunction()
if err != nil {
    return fiber.WrapError(err, fiber.ErrorCodeInternal)
}
```

---

## Custom Error Pages

### Creating Custom Pages

```go
// Define custom error page
func NotFoundPage() templ.Component {
    return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
        _, err := fmt.Fprintf(w, `
            <html>
                <body>
                    <h1>404 - Page Not Found</h1>
                    <p>The page you're looking for doesn't exist.</p>
                    <a href="/">Go Home</a>
                </body>
            </html>
        `)
        return err
    })
}

// Register with error handler
app.Use(fiber.ErrorHandler(fiber.ErrorHandlerConfig{
    CustomErrorPages: map[int]templ.Component{
        404: NotFoundPage(),
        500: ServerErrorPage(),
        401: UnauthorizedPage(),
        403: ForbiddenPage(),
    },
}))
```

### Using State in Error Pages

```go
func ErrorPage() templ.Component {
    return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
        // Get error from state
        state := ctx.Value("state").(*state.StateMap)
        errData := state.Get("error")
        
        // Render error page with data
        // ...
    })
}
```

---

## Error Handling Patterns

### In Route Handlers

```go
func GetUser(c *fiber.Ctx) error {
    id := c.Params("id")
    
    user, err := db.FindUser(id)
    if err != nil {
        if errors.Is(err, ErrUserNotFound) {
            return fiber.ErrNotFound.WithDetails(map[string]any{
                "userId": id,
            })
        }
        return fiber.WrapError(err, fiber.ErrorCodeInternal)
    }
    
    return c.JSON(user)
}
```

### With Validation

```go
func CreateUser(c *fiber.Ctx) error {
    var input CreateUserInput
    if err := c.BodyParser(&input); err != nil {
        return fiber.ErrBadRequest.WithDetails(map[string]any{
            "error": "Invalid JSON body",
        })
    }
    
    // Validate input
    errors := validateInput(input)
    if len(errors) > 0 {
        return fiber.ValidationErrors(errors)
    }
    
    // Create user...
}
```

### With Context

```go
func ProtectedRoute(c *fiber.Ctx) error {
    user := c.Locals("user")
    if user == nil {
        return fiber.ErrUnauthorized.WithDetails(map[string]any{
            "reason": "Authentication required",
        })
    }
    
    // Handle request...
}
```

---

## Error Logging

```go
app.Use(fiber.ErrorHandler(fiber.ErrorHandlerConfig{
    OnError: func(err *fiber.AppError) {
        // Log to external service
        logger.Error("Application error",
            "code", err.Code,
            "message", err.Message,
            "requestId", err.RequestID,
            "details", err.Details,
        )
        
        // Send to error tracking
        sentry.CaptureException(err)
    },
}))
```

---

## Best Practices

1. **Use appropriate error codes**: Match error codes to HTTP semantics
2. **Include helpful details**: Add context for debugging
3. **Don't expose internals**: Sanitize error messages in production
4. **Log errors**: Always log errors for debugging
5. **Use custom pages**: Provide user-friendly error pages
6. **Handle panics**: Always use PanicHandler in production
7. **Validate early**: Return validation errors before processing
8. **Use typed errors**: Create domain-specific error types
