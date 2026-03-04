# GoSPA Component System

The component system provides a structured way to build reusable, composable UI components with lifecycle management, props validation, and state handling.

## Overview

GoSPA components are built using the `component` package which provides:

- **BaseComponent**: Foundation for all components
- **ComponentTree**: Hierarchical component management
- **Lifecycle**: Component lifecycle hooks
- **Props**: Type-safe property handling with validation

---

## BaseComponent

The `BaseComponent` is the foundation for all GoSPA components.

### Creating a Component

```go
import "github.com/gospa/gospa/component"

// Create a basic component
comp := component.NewBaseComponent("my-component")

// Create with options
comp := component.NewBaseComponent("my-component",
    component.WithProps(component.Props{
        "title": "Hello",
        "count": 0,
    }),
    component.WithState(stateMap),
    component.WithChildren(childComponent),
)
```

### Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `ID()` | `string` | Returns unique component identifier |
| `Name()` | `string` | Returns component name |
| `State()` | `*state.StateMap` | Returns component state |
| `Props()` | `Props` | Returns component props |
| `Children()` | `[]Component` | Returns child components |
| `Parent()` | `Component` | Returns parent component |
| `AddChild()` | `AddChild(child Component)` | Adds a child component |
| `RemoveChild()` | `RemoveChild(child Component)` | Removes a child component |
| `GetSlot()` | `GetSlot(name string) templ.Component` | Gets a named slot |
| `SetSlot()` | `SetSlot(name string, comp templ.Component)` | Sets a named slot |
| `Context()` | `context.Context` | Returns component context |
| `SetContext()` | `SetContext(ctx context.Context)` | Sets component context |
| `ToJSON()` | `([]byte, error)` | Serializes component to JSON |
| `Clone()` | `Component` | Creates a deep copy |

### Component Options

```go
// Props option
component.WithProps(component.Props{
    "title": "My Title",
    "visible": true,
})

// State option
component.WithState(stateMap)

// Children option
component.WithChildren(child1, child2, child3)

// Parent option
component.WithParent(parentComponent)

// Context option
component.WithContext(context.Background())

// Slots option
component.WithSlots(map[string]templ.Component{
    "header": headerComponent,
    "footer": footerComponent,
})
```

---

## Component Interface

```go
type Component interface {
    ID() string
    Name() string
    State() *state.StateMap
    Props() Props
    Children() []Component
    Parent() Component
    AddChild(child Component)
    RemoveChild(child Component)
    GetSlot(name string) templ.Component
    SetSlot(name string, comp templ.Component)
    Context() context.Context
    SetContext(ctx context.Context)
    ToJSON() ([]byte, error)
    Clone() Component
}
```

---

## ComponentTree

Manages hierarchical component relationships.

### Creating a Tree

```go
// Create with root component
tree := component.NewComponentTree(rootComponent)

// Get root
root := tree.Root()

// Get component by ID
comp := tree.Get("component-id")

// Add component to tree
tree.Add(parent, child)

// Remove component from tree
tree.Remove(component)
```

### Tree Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `Root()` | `Component` | Returns root component |
| `Get()` | `Get(id string) Component` | Get component by ID |
| `Add()` | `Add(parent, child Component)` | Add child to parent |
| `Remove()` | `Remove(component Component)` | Remove from tree |
| `OnMount()` | `OnMount(fn func(Component))` | Register mount callback |
| `OnUpdate()` | `OnUpdate(fn func(Component))` | Register update callback |
| `OnDestroy()` | `OnDestroy(fn func(Component))` | Register destroy callback |
| `Mount()` | `Mount()` | Trigger mount lifecycle |
| `Update()` | `Update()` | Trigger update lifecycle |
| `Walk()` | `Walk(fn func(Component) bool)` | Walk tree depth-first |
| `Find()` | `Find(fn func(Component) bool) Component` | Find first matching |
| `FindAll()` | `FindAll(fn func(Component) bool) []Component` | Find all matching |
| `FindByName()` | `FindByName(name string) Component` | Find by component name |
| `FindByProp()` | `FindByProp(key string, value any) Component` | Find by prop value |

### Walking the Tree

```go
// Walk all components
tree.Walk(func(comp Component) bool {
    fmt.Println("Component:", comp.Name())
    return true // continue walking
})

// Find specific component
found := tree.Find(func(comp Component) bool {
    return comp.Name() == "target-component"
})

// Find all matching
all := tree.FindAll(func(comp Component) bool {
    props := comp.Props()
    return props.GetBool("active")
})
```

---

## Lifecycle

The `Lifecycle` type manages component lifecycle phases and hooks.

### Lifecycle Phases

```go
const (
    PhaseCreated   LifecyclePhase = iota // Component created
    PhaseMounting                        // Component mounting
    PhaseMounted                         // Component mounted
    PhaseUpdating                        // Component updating
    PhaseUpdated                         // Component updated
    PhaseDestroying                      // Component destroying
    PhaseDestroyed                       // Component destroyed
)
```

### Creating a Lifecycle

```go
lc := component.NewLifecycle()

// Check current phase
phase := lc.Phase()

// Check if mounted
if lc.IsMounted() {
    // Component is mounted
}
```

### Registering Hooks

```go
// Before mount
lc.OnBeforeMount(func() {
    fmt.Println("About to mount")
})

// On mount
lc.OnMount(func() {
    fmt.Println("Mounted")
})

// Before update
lc.OnBeforeUpdate(func() {
    fmt.Println("About to update")
})

// On update
lc.OnUpdate(func() {
    fmt.Println("Updated")
})

// Before destroy
lc.OnBeforeDestroy(func() {
    fmt.Println("About to destroy")
})

// On destroy
lc.OnDestroy(func() {
    fmt.Println("Destroyed")
})

// Cleanup (runs after destroy)
lc.OnCleanup(func() {
    fmt.Println("Cleanup")
})
```

### Triggering Lifecycle Events

```go
// Trigger mount
lc.Mount()

// Trigger update
lc.Update()

// Trigger destroy
lc.Destroy()

// Clear all hooks
lc.ClearHooks()
```

### Lifecycle-Aware Components

```go
type LifecycleAware interface {
    OnBeforeMount()
    OnMount()
    OnBeforeUpdate()
    OnUpdate()
    OnBeforeDestroy()
    OnDestroy()
}
```

### Helper Functions

```go
// Mount a component with lifecycle
err := component.MountComponent(comp)

// Update a component with lifecycle
err := component.UpdateComponent(comp)

// Destroy a component with lifecycle
err := component.DestroyComponent(comp)
```

---

## Props

Type-safe property handling with validation.

### Basic Usage

```go
// Create props
props := component.Props{
    "title": "Hello",
    "count": 42,
    "active": true,
}

// Get values
title := props.Get("title")           // any
titleStr := props.GetString("title")  // string
count := props.GetInt("count")        // int
count64 := props.GetInt64("count")    // int64
price := props.GetFloat64("price")    // float64
active := props.GetBool("active")     // bool
items := props.GetSlice("items")      // []any
config := props.GetMap("config")      // map[string]any

// Get with default
title := props.GetDefault("title", "Default Title")

// Set value
props.Set("newKey", "newValue")

// Check existence
if props.Has("title") {
    // Key exists
}

// Delete key
props.Delete("oldKey")

// Get all keys
keys := props.Keys()

// Get all values
values := props.Values()

// Clone props
cloned := props.Clone()

// Merge props
props.Merge(otherProps)

// JSON serialization
json, err := props.ToJSON()

// Compare props
if props.Equals(otherProps) {
    // Props are equal
}
```

### Props Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `Get()` | `Get(key string) any` | Get prop value |
| `Set()` | `Set(key string, value any)` | Set prop value |
| `GetDefault()` | `GetDefault(key string, def any) any` | Get with default |
| `GetString()` | `GetString(key string) string` | Get as string |
| `GetInt()` | `GetInt(key string) int` | Get as int |
| `GetInt64()` | `GetInt(key string) int64` | Get as int64 |
| `GetFloat64()` | `GetFloat64(key string) float64` | Get as float64 |
| `GetBool()` | `GetBool(key string) bool` | Get as bool |
| `GetSlice()` | `GetSlice(key string) []any` | Get as slice |
| `GetMap()` | `GetMap(key string) map[string]any` | Get as map |
| `Has()` | `Has(key string) bool` | Check if exists |
| `Delete()` | `Delete(key string)` | Delete prop |
| `Keys()` | `Keys() []string` | Get all keys |
| `Values()` | `Values() []any` | Get all values |
| `Clone()` | `Clone() Props` | Clone props |
| `Merge()` | `Merge(other Props)` | Merge props |
| `ToJSON()` | `ToJSON() ([]byte, error)` | JSON serialization |
| `Equals()` | `Equals(other Props) bool` | Compare props |

---

## PropSchema

Define and validate prop schemas.

### Creating a Schema

```go
schema := component.NewPropSchema()

// Define props
schema.Define("title", reflect.String).
       Define("count", reflect.Int).
       Define("active", reflect.Bool)

// Define with validator
schema.DefineWithValidator("email", func(value any) error {
    str, ok := value.(string)
    if !ok {
        return errors.New("email must be string")
    }
    if !strings.Contains(str, "@") {
        return errors.New("invalid email format")
    }
    return nil
})
```

### PropDefinition

```go
type PropDefinition struct {
    Name         string            // Prop name
    Type         reflect.Kind      // Expected type
    DefaultValue any               // Default value
    Required     bool              // Is required
    Validator    func(any) error   // Custom validator
}
```

### Validation

```go
// Validate props
err := schema.Validate(props)

// Apply defaults
propsWithDefaults := schema.ApplyDefaults(props)

// Validate and apply
validated, err := schema.ValidateAndApply(props)

// Get definition
def := schema.GetDefinition("title")

// Get all definitions
defs := schema.Definitions()
```

---

## BindableProp

Two-way bindable properties.

### Creating Bindable Props

```go
// Create bindable prop
bp := component.NewBindableProp("count", 0)

// Get value
value := bp.Get()

// Set value
bp.Set(42)

// Get name
name := bp.Name()

// On change callback
bp.OnChange(func(newValue any) {
    fmt.Println("Value changed to:", newValue)
})

// Set validator
bp.SetValidator(func(value any) error {
    if value.(int) < 0 {
        return errors.New("count cannot be negative")
    }
    return nil
})

// Two-way bind to another prop
bp.Bind(otherBindableProp)
```

### BindableProps Collection

```go
// Create collection
bps := component.NewBindableProps()

// Add bindable
bps.Add(component.NewBindableProp("count", 0))

// Get bindable
bp := bps.Get("count")

// Remove bindable
bps.Remove("count")

// Get all names
names := bps.Names()

// Convert to Props
props := bps.ToProps()
```

---

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/gospa/gospa/component"
    "github.com/gospa/gospa/state"
)

// Custom component
type ButtonComponent struct {
    *component.BaseComponent
    lifecycle *component.Lifecycle
}

func NewButtonComponent(text string, onClick func()) *ButtonComponent {
    btn := &ButtonComponent{
        BaseComponent: component.NewBaseComponent("button",
            component.WithProps(component.Props{
                "text":    text,
                "onClick": onClick,
            }),
        ),
        lifecycle: component.NewLifecycle(),
    }
    
    // Setup lifecycle hooks
    btn.lifecycle.OnMount(func() {
        fmt.Println("Button mounted")
    })
    
    btn.lifecycle.OnDestroy(func() {
        fmt.Println("Button destroyed")
    })
    
    return btn
}

func main() {
    // Create state
    sm := state.NewStateMap()
    sm.Add("counter", state.NewRune(0))
    
    // Create component tree
    root := component.NewBaseComponent("app",
        component.WithState(sm),
    )
    
    tree := component.NewComponentTree(root)
    
    // Add child components
    button := NewButtonComponent("Click Me", func() {
        counter := sm.Get("counter").(*state.Rune[int])
        counter.Set(counter.Get() + 1)
    })
    
    tree.Add(root, button)
    
    // Mount tree
    tree.Mount()
    
    // Walk tree
    tree.Walk(func(comp component.Component) bool {
        fmt.Printf("Component: %s (ID: %s)\n", comp.Name(), comp.ID())
        return true
    })
    
    // Cleanup
    tree.Destroy()
}
```
