// Package state provides serialization utilities for reactive state.
// These helpers convert state to JSON for client transmission.
package state

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"
)

// Serializable represents a value that can be serialized to JSON
type Serializable interface {
	Serialize() ([]byte, error)
}

// StateMap is a collection of generic observables for component state
type StateMap struct {
	mu           sync.RWMutex
	observables  map[string]Observable
	unsubscribes map[string]Unsubscribe
	// OnChange is invoked when any state variable changes.
	// DEADLOCK WARNING: OnChange must NOT call back into StateMap.Add, StateMap.Remove,
	// or any method that acquires sm.mu. It is invoked inside a goroutine spawned by
	// SubscribeAny, which runs after the mutex is released — but if your handler triggers
	// a synchronous chain that calls back into Add/Remove on the SAME StateMap, you will
	// deadlock. Safe operations inside OnChange: read sm.Get(), send on channels, call
	// external callbacks. Unsafe: sm.Add(), sm.Remove(), sm.AddAny().
	OnChange func(key string, value any)
}

// NewStateMap creates a new state collection
func NewStateMap() *StateMap {
	return &StateMap{
		observables:  make(map[string]Observable),
		unsubscribes: make(map[string]Unsubscribe),
	}
}

// Add adds an observable to the state collection
func (sm *StateMap) Add(name string, obs Observable) *StateMap {
	// Capture all data needed for value transfer before acquiring lock
	// This ensures we have everything we need before entering critical section
	var existingValue any
	var hasExisting bool
	var settable Settable
	var isSettable bool

	// Check if the observable is settable upfront (doesn't need lock)
	settable, isSettable = obs.(Settable)

	sm.mu.Lock()

	// Clear out old subscription if one exists
	if unsub, ok := sm.unsubscribes[name]; ok {
		unsub()
	}

	// Capture existing value while holding the lock
	if existing, ok := sm.observables[name]; ok {
		existingValue = existing.GetAny()
		hasExisting = true
	}

	sm.observables[name] = obs

	// Subscribe outside the lock to prevent immediate callback deadlock
	sm.mu.Unlock()

	// Subscribe to changes to trigger differential sync pushes
	unsub := obs.SubscribeAny(func(v any) {
		sm.mu.RLock()
		handler := sm.OnChange
		sm.mu.RUnlock()
		if handler != nil {
			// PERFORMANCE: Use goroutine to prevent blocking state updates
			// The handler (e.g., WebSocket broadcast) may perform I/O operations
			// that shouldn't delay the state notification chain
			go func(h func(string, any), key string, value any) {
				defer func() {
					// Recover from panics in handler to prevent crashing the application
					_ = recover()
				}()
				h(key, value)
			}(handler, name, v)
		}
	})

	sm.mu.Lock()
	sm.unsubscribes[name] = unsub
	sm.mu.Unlock()

	// Transfer value from existing observable if the new one is Settable
	// This happens entirely outside the lock to avoid deadlocks if SetAny triggers callbacks
	if hasExisting && isSettable {
		// Use a non-blocking set to avoid deadlocks
		func() {
			defer func() {
				// Recover from any panics during SetAny to prevent crashes
				_ = recover()
			}()
			_ = settable.SetAny(existingValue)
		}()
	}

	return sm
}

// AddAny adds any primitive value as a rune to the state collection
func (sm *StateMap) AddAny(name string, value interface{}) *StateMap {
	return sm.Add(name, NewRune[any](value))
}

// Get retrieves an observable by name
func (sm *StateMap) Get(name string) (Observable, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	r, ok := sm.observables[name]
	return r, ok
}

// Remove removes an observable from the state collection.
// Returns the StateMap for method chaining.
func (sm *StateMap) Remove(name string) *StateMap {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Unsubscribe if there's an active subscription
	if unsub, ok := sm.unsubscribes[name]; ok {
		unsub()
		delete(sm.unsubscribes, name)
	}

	delete(sm.observables, name)
	return sm
}

// StateMapComparison represents a diff between two StateMaps
// with added, removed, and changed keys.
type StateMapComparison struct {
	Added   map[string]interface{} `json:"added"`
	Removed map[string]interface{} `json:"removed"`
	Changed map[string]interface{} `json:"changed"`
}

// Diff computes the difference between this StateMap and another.
// Returns a StateMapComparison containing added, removed, and changed keys.
func (sm *StateMap) Diff(other *StateMap) *StateMapComparison {
	if other == nil {
		return &StateMapComparison{
			Added:   sm.ToMap(),
			Removed: make(map[string]interface{}),
			Changed: make(map[string]interface{}),
		}
	}

	sm.mu.RLock()
	other.mu.RLock()
	defer sm.mu.RUnlock()
	defer other.mu.RUnlock()

	added := make(map[string]interface{})
	removed := make(map[string]interface{})
	changed := make(map[string]interface{})

	// Find added and changed keys
	for name, obs := range sm.observables {
		value := obs.GetAny()
		if otherObs, ok := other.observables[name]; ok {
			// Key exists in both, check if changed
			otherValue := otherObs.GetAny()
			if !deepEqualValues(value, otherValue) {
				changed[name] = value
			}
		} else {
			// Key only in sm (added)
			added[name] = value
		}
	}

	// Find removed keys
	for name, obs := range other.observables {
		if _, ok := sm.observables[name]; !ok {
			removed[name] = obs.GetAny()
		}
	}

	return &StateMapComparison{
		Added:   added,
		Removed: removed,
		Changed: changed,
	}
}

// deepEqualValues compares two values for equality.
// Helper for Diff method.
func deepEqualValues(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Use JSON marshaling for reliable comparison
	aJSON, err1 := json.Marshal(a)
	bJSON, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		// Fallback to string comparison
		return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
	}
	return string(aJSON) == string(bJSON)
}

// ForEach iterates over all observables in the state map
func (sm *StateMap) ForEach(fn func(key string, value any)) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	for key, obs := range sm.observables {
		fn(key, obs.GetAny())
	}
}

// ToMap returns all state values as a plain map
func (sm *StateMap) ToMap() map[string]any {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	result := make(map[string]any)
	for key, obs := range sm.observables {
		result[key] = obs.GetAny()
	}
	return result
}

// MarshalJSON serializes the state map to JSON
func (sm *StateMap) MarshalJSON() ([]byte, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	data := make(map[string]interface{})
	for name, obs := range sm.observables {
		data[name] = obs.GetAny()
	}
	return json.Marshal(data)
}

// ToJSON returns the state as a JSON string
func (sm *StateMap) ToJSON() (string, error) {
	data, err := sm.MarshalJSON()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SerializeState serializes multiple runes into a JSON object
func SerializeState(runes map[string]interface{}) ([]byte, error) {
	data := make(map[string]interface{})
	for name, r := range runes {
		switch v := r.(type) {
		case *Rune[any]:
			data[name] = v.Get()
		case *Derived[any]:
			data[name] = v.Get()
		case Serializable:
			serialized, err := v.Serialize()
			if err != nil {
				return nil, err
			}
			var value interface{}
			if err := json.Unmarshal(serialized, &value); err != nil {
				return nil, err
			}
			data[name] = value
		default:
			// Try to get the value using reflection
			data[name] = extractValue(r)
		}
	}
	return json.Marshal(data)
}

// extractValue extracts the underlying value from a rune-like type
func extractValue(r interface{}) interface{} {
	val := reflect.ValueOf(r)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Try to find a Get method
	getMethod := val.MethodByName("Get")
	if getMethod.IsValid() {
		results := getMethod.Call(nil)
		if len(results) > 0 {
			return results[0].Interface()
		}
	}

	// Return the value itself if no Get method
	return r
}

// StateSnapshot represents a snapshot of component state at a point in time
type StateSnapshot struct {
	ComponentID string                 `json:"componentId"`
	State       map[string]interface{} `json:"state"`
	Timestamp   int64                  `json:"timestamp"`
}

// NewSnapshot creates a new state snapshot
func NewSnapshot(componentID string, state map[string]interface{}) *StateSnapshot {
	return &StateSnapshot{
		ComponentID: componentID,
		State:       state,
		Timestamp:   currentTimeMillis(),
	}
}

// MarshalJSON serializes the snapshot to JSON
func (s *StateSnapshot) MarshalJSON() ([]byte, error) {
	type Alias StateSnapshot
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	})
}

// StateDiff represents a change in state
type StateDiff struct {
	ComponentID string      `json:"componentId"`
	Key         string      `json:"key"`
	OldValue    interface{} `json:"oldValue,omitempty"`
	NewValue    interface{} `json:"newValue"`
	Timestamp   int64       `json:"timestamp"`
}

// NewStateDiff creates a new state diff
func NewStateDiff(componentID, key string, oldValue, newValue interface{}) *StateDiff {
	return &StateDiff{
		ComponentID: componentID,
		Key:         key,
		OldValue:    oldValue,
		NewValue:    newValue,
		Timestamp:   currentTimeMillis(),
	}
}

// MarshalJSON serializes the diff to JSON
func (d *StateDiff) MarshalJSON() ([]byte, error) {
	type Alias StateDiff
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(d),
	})
}

// StateMessage represents a message sent between server and client
type StateMessage struct {
	Type        string      `json:"type"` // "init", "update", "sync", "error"
	ComponentID string      `json:"componentId,omitempty"`
	Key         string      `json:"key,omitempty"`
	Value       interface{} `json:"value,omitempty"`
	State       interface{} `json:"state,omitempty"`
	Error       string      `json:"error,omitempty"`
	Timestamp   int64       `json:"timestamp"`
}

// NewInitMessage creates an initialization message
func NewInitMessage(componentID string, state interface{}) *StateMessage {
	return &StateMessage{
		Type:        "init",
		ComponentID: componentID,
		State:       state,
		Timestamp:   currentTimeMillis(),
	}
}

// NewUpdateMessage creates an update message
func NewUpdateMessage(componentID, key string, value interface{}) *StateMessage {
	return &StateMessage{
		Type:        "update",
		ComponentID: componentID,
		Key:         key,
		Value:       value,
		Timestamp:   currentTimeMillis(),
	}
}

// NewSyncMessage creates a sync message
func NewSyncMessage(componentID string, state interface{}) *StateMessage {
	return &StateMessage{
		Type:        "sync",
		ComponentID: componentID,
		State:       state,
		Timestamp:   currentTimeMillis(),
	}
}

// NewErrorMessage creates an error message
func NewErrorMessage(componentID, errMsg string) *StateMessage {
	return &StateMessage{
		Type:        "error",
		ComponentID: componentID,
		Error:       errMsg,
		Timestamp:   currentTimeMillis(),
	}
}

// MarshalJSON serializes the message to JSON
func (m *StateMessage) MarshalJSON() ([]byte, error) {
	type Alias StateMessage
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}

// ParseMessage parses a JSON message
func ParseMessage(data []byte) (*StateMessage, error) {
	var msg StateMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// currentTimeMillis returns current time in milliseconds
func currentTimeMillis() int64 {
	return time.Now().UnixMilli()
}

// ValidateState validates state against a schema
type Validator func(interface{}) error

// StateValidator validates state values
type StateValidator struct {
	validators map[string]Validator
}

// NewStateValidator creates a new state validator
func NewStateValidator() *StateValidator {
	return &StateValidator{
		validators: make(map[string]Validator),
	}
}

// AddValidator adds a validator for a key
func (sv *StateValidator) AddValidator(key string, v Validator) {
	sv.validators[key] = v
}

// Validate validates a value for a key
func (sv *StateValidator) Validate(key string, value interface{}) error {
	if v, ok := sv.validators[key]; ok {
		return v(value)
	}
	return nil
}

// ValidateAll validates all values in a map
func (sv *StateValidator) ValidateAll(values map[string]interface{}) error {
	for key, value := range values {
		if err := sv.Validate(key, value); err != nil {
			return err
		}
	}
	return nil
}
