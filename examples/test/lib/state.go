package lib

// GlobalCounter is a global state for the counter
var GlobalCounter = struct {
	Count int
}{
	Count: 0,
}

// AppState holds application-wide state.
type AppState struct {
	// Add your application state here
}

// NewAppState creates a new application state.
func NewAppState() *AppState {
	return &AppState{}
}
