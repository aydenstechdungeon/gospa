// Package state provides serialization utilities for reactive state.
// These helpers convert state to JSON for client transmission.
package state

import (
	"encoding/json"
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
	OnChange     func(key string, value any) // Callback invoked when a state variable changes
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
	sm.mu.Lock()

	// Clear out old subscription if one exists
	if unsub, ok := sm.unsubscribes[name]; ok {
		unsub()
	}

	var existingValue any
	var hasExisting bool
	if existing, ok := sm.observables[name]; ok {
		existingValue = existing.GetAny()
		hasExisting = true
	}

	sm.observables[name] = obs

	// Subscribe to changes to trigger differential sync pushes
	sm.unsubscribes[name] = obs.SubscribeAny(func(v any) {
		sm.mu.RLock()
		handler := sm.OnChange
		sm.mu.RUnlock()
		if handler != nil {
			handler(name, v)
		}
	})

	sm.mu.Unlock()

	// Transfer value from existing observable if the new one is Settable
	if hasExisting {
		if settable, isSettable := obs.(Settable); isSettable {
			_ = settable.SetAny(existingValue)
		}
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
