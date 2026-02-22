package component

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// Props is a map of component properties.
type Props map[string]interface{}

// Get returns a prop value by key.
func (p Props) Get(key string) interface{} {
	if p == nil {
		return nil
	}
	return p[key]
}

// Set sets a prop value.
func (p Props) Set(key string, value interface{}) {
	if p == nil {
		p = make(Props)
	}
	p[key] = value
}

// GetDefault returns a prop value with a default.
func (p Props) GetDefault(key string, defaultValue interface{}) interface{} {
	if p == nil {
		return defaultValue
	}
	if val, ok := p[key]; ok {
		return val
	}
	return defaultValue
}

// GetString returns a prop as a string.
func (p Props) GetString(key string) string {
	if p == nil {
		return ""
	}
	if val, ok := p[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", val)
	}
	return ""
}

// GetInt returns a prop as an int.
func (p Props) GetInt(key string) int {
	if p == nil {
		return 0
	}
	if val, ok := p[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		case float32:
			return int(v)
		}
	}
	return 0
}

// GetInt64 returns a prop as an int64.
func (p Props) GetInt64(key string) int64 {
	if p == nil {
		return 0
	}
	if val, ok := p[key]; ok {
		switch v := val.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case float64:
			return int64(v)
		case float32:
			return int64(v)
		}
	}
	return 0
}

// GetFloat64 returns a prop as a float64.
func (p Props) GetFloat64(key string) float64 {
	if p == nil {
		return 0
	}
	if val, ok := p[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		case int64:
			return float64(v)
		}
	}
	return 0
}

// GetBool returns a prop as a bool.
func (p Props) GetBool(key string) bool {
	if p == nil {
		return false
	}
	if val, ok := p[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// GetSlice returns a prop as a slice.
func (p Props) GetSlice(key string) []interface{} {
	if p == nil {
		return nil
	}
	if val, ok := p[key]; ok {
		if slice, ok := val.([]interface{}); ok {
			return slice
		}
	}
	return nil
}

// GetMap returns a prop as a map.
func (p Props) GetMap(key string) map[string]interface{} {
	if p == nil {
		return nil
	}
	if val, ok := p[key]; ok {
		if m, ok := val.(map[string]interface{}); ok {
			return m
		}
	}
	return nil
}

// Has checks if a prop exists.
func (p Props) Has(key string) bool {
	if p == nil {
		return false
	}
	_, ok := p[key]
	return ok
}

// Delete removes a prop.
func (p Props) Delete(key string) {
	if p != nil {
		delete(p, key)
	}
}

// Keys returns all prop keys.
func (p Props) Keys() []string {
	if p == nil {
		return nil
	}
	keys := make([]string, 0, len(p))
	for k := range p {
		keys = append(keys, k)
	}
	return keys
}

// Values returns all prop values.
func (p Props) Values() []interface{} {
	if p == nil {
		return nil
	}
	values := make([]interface{}, 0, len(p))
	for _, v := range p {
		values = append(values, v)
	}
	return values
}

// Clone creates a copy of the props.
func (p Props) Clone() Props {
	if p == nil {
		return nil
	}
	clone := make(Props, len(p))
	for k, v := range p {
		clone[k] = v
	}
	return clone
}

// Merge merges another Props into this one.
func (p Props) Merge(other Props) {
	if p == nil || other == nil {
		return
	}
	for k, v := range other {
		p[k] = v
	}
}

// ToJSON returns the props as JSON.
func (p Props) ToJSON() (string, error) {
	if p == nil {
		return "{}", nil
	}
	data, err := json.Marshal(p)
	return string(data), err
}

// FromJSON creates Props from JSON.
func PropsFromJSON(data string) (Props, error) {
	var p Props
	if err := json.Unmarshal([]byte(data), &p); err != nil {
		return nil, err
	}
	return p, nil
}

// Equals checks if two Props are equal.
func (p Props) Equals(other Props) bool {
	if p == nil && other == nil {
		return true
	}
	if p == nil || other == nil {
		return false
	}
	if len(p) != len(other) {
		return false
	}
	for k, v := range p {
		if ov, ok := other[k]; !ok || !reflect.DeepEqual(v, ov) {
			return false
		}
	}
	return true
}

// PropDefinition defines a prop's type and default value.
type PropDefinition struct {
	Name         string
	Type         reflect.Kind
	DefaultValue interface{}
	Required     bool
	Validator    func(interface{}) bool
}

// PropSchema defines the schema for component props.
type PropSchema struct {
	definitions map[string]PropDefinition
}

// NewPropSchema creates a new prop schema.
func NewPropSchema() *PropSchema {
	return &PropSchema{
		definitions: make(map[string]PropDefinition),
	}
}

// Define defines a prop.
func (s *PropSchema) Define(name string, kind reflect.Kind, defaultValue interface{}, required bool) *PropSchema {
	s.definitions[name] = PropDefinition{
		Name:         name,
		Type:         kind,
		DefaultValue: defaultValue,
		Required:     required,
	}
	return s
}

// DefineWithValidator defines a prop with a validator.
func (s *PropSchema) DefineWithValidator(name string, kind reflect.Kind, defaultValue interface{}, required bool, validator func(interface{}) bool) *PropSchema {
	s.definitions[name] = PropDefinition{
		Name:         name,
		Type:         kind,
		DefaultValue: defaultValue,
		Required:     required,
		Validator:    validator,
	}
	return s
}

// Validate validates props against the schema.
func (s *PropSchema) Validate(props Props) error {
	for name, def := range s.definitions {
		val, exists := props[name]

		// Check required
		if def.Required && !exists {
			return fmt.Errorf("required prop missing: %s", name)
		}

		// Use default if not provided
		if !exists {
			continue
		}

		// Check type
		if def.Type != reflect.Invalid {
			actualType := reflect.TypeOf(val)
			if actualType != nil && actualType.Kind() != def.Type {
				return fmt.Errorf("prop %s has wrong type: expected %s, got %s", name, def.Type, actualType.Kind())
			}
		}

		// Run validator
		if def.Validator != nil && !def.Validator(val) {
			return fmt.Errorf("prop %s failed validation", name)
		}
	}

	return nil
}

// ApplyDefaults applies default values to props.
func (s *PropSchema) ApplyDefaults(props Props) Props {
	if props == nil {
		props = make(Props)
	}

	for name, def := range s.definitions {
		if _, exists := props[name]; !exists && def.DefaultValue != nil {
			props[name] = def.DefaultValue
		}
	}

	return props
}

// ValidateAndApply validates props and applies defaults.
func (s *PropSchema) ValidateAndApply(props Props) (Props, error) {
	props = s.ApplyDefaults(props)
	if err := s.Validate(props); err != nil {
		return nil, err
	}
	return props, nil
}

// GetDefinition returns a prop definition.
func (s *PropSchema) GetDefinition(name string) (PropDefinition, bool) {
	def, ok := s.definitions[name]
	return def, ok
}

// Definitions returns all prop definitions.
func (s *PropSchema) Definitions() map[string]PropDefinition {
	return s.definitions
}

// BindableProp is a prop that can be bound two-way.
type BindableProp struct {
	name      string
	value     interface{}
	onChange  func(interface{})
	validator func(interface{}) bool
}

// NewBindableProp creates a new bindable prop.
func NewBindableProp(name string, initialValue interface{}) *BindableProp {
	return &BindableProp{
		name:  name,
		value: initialValue,
	}
}

// Get returns the current value.
func (p *BindableProp) Get() interface{} {
	return p.value
}

// Set sets a new value and triggers onChange.
func (p *BindableProp) Set(value interface{}) bool {
	if p.validator != nil && !p.validator(value) {
		return false
	}
	p.value = value
	if p.onChange != nil {
		p.onChange(value)
	}
	return true
}

// Name returns the prop name.
func (p *BindableProp) Name() string {
	return p.name
}

// OnChange sets the change callback.
func (p *BindableProp) OnChange(fn func(interface{})) {
	p.onChange = fn
}

// SetValidator sets the validator.
func (p *BindableProp) SetValidator(fn func(interface{}) bool) {
	p.validator = fn
}

// Bind creates a two-way binding.
func (p *BindableProp) Bind() (get func() interface{}, set func(interface{}) bool) {
	return p.Get, p.Set
}

// BindableProps is a collection of bindable props.
type BindableProps struct {
	props map[string]*BindableProp
}

// NewBindableProps creates a new bindable props collection.
func NewBindableProps() *BindableProps {
	return &BindableProps{
		props: make(map[string]*BindableProp),
	}
}

// Add adds a bindable prop.
func (bp *BindableProps) Add(name string, initialValue interface{}) *BindableProp {
	prop := NewBindableProp(name, initialValue)
	bp.props[name] = prop
	return prop
}

// Get returns a bindable prop by name.
func (bp *BindableProps) Get(name string) *BindableProp {
	return bp.props[name]
}

// Remove removes a bindable prop.
func (bp *BindableProps) Remove(name string) {
	delete(bp.props, name)
}

// Names returns all prop names.
func (bp *BindableProps) Names() []string {
	names := make([]string, 0, len(bp.props))
	for name := range bp.props {
		names = append(names, name)
	}
	return names
}

// ToProps converts to regular Props.
func (bp *BindableProps) ToProps() Props {
	p := make(Props)
	for name, prop := range bp.props {
		p[name] = prop.Get()
	}
	return p
}
