package starter

// InputType defines the type of input field
type InputType string

const (
	InputText     InputType = "text"
	InputEmail    InputType = "email"
	InputPassword InputType = "password"
	InputNumber   InputType = "number"
	InputTel      InputType = "tel"
	InputURL      InputType = "url"
	InputSearch   InputType = "search"
	InputDate     InputType = "date"
	InputTime     InputType = "time"
	InputDateTime InputType = "datetime-local"
)

// InputSize defines the size of the input
type InputSize string

const (
	InputSmall  InputSize = "small"
	InputMedium InputSize = "medium"
	InputLarge  InputSize = "large"
)

// InputProps defines the properties for an input component
type InputProps struct {
	// ID is the unique identifier for the input
	ID string
	// Name is the name attribute for form submission
	Name string
	// Type is the input type (text, email, password, etc.)
	Type InputType
	// Placeholder is the placeholder text
	Placeholder string
	// Value is the current value
	Value string
	// DefaultValue is the default value for uncontrolled inputs
	DefaultValue string
	// Size is the input size
	Size InputSize
	// Disabled indicates if the input is disabled
	Disabled bool
	// ReadOnly indicates if the input is read-only
	ReadOnly bool
	// Required indicates if the input is required
	Required bool
	// AutoFocus indicates if the input should auto-focus
	AutoFocus bool
	// Error indicates if the input has an error state
	Error bool
	// ErrorMessage is the error message to display
	ErrorMessage string
	// Label is the label text for the input
	Label string
	// HelperText is additional helper text below the input
	HelperText string
	// Class is additional CSS classes
	Class string
	// Attributes are additional HTML attributes
	Attributes map[string]string
	// Min is the minimum value (for number/date inputs)
	Min string
	// Max is the maximum value (for number/date inputs)
	Max string
	// Step is the step value (for number inputs)
	Step string
	// Pattern is the regex pattern for validation
	Pattern string
	// MaxLength is the maximum character length
	MaxLength int
	// MinLength is the minimum character length
	MinLength int
	// AutoComplete is the autocomplete attribute
	AutoComplete string
}

// InputClasses returns the CSS classes for an input based on props
func InputClasses(props InputProps) string {
	base := "block w-full rounded-md border shadow-sm transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-offset-0"

	// Size classes
	sizeClasses := map[InputSize]string{
		InputSmall:  "px-3 py-1.5 text-sm",
		InputMedium: "px-4 py-2 text-base",
		InputLarge:  "px-4 py-3 text-lg",
	}

	sizeClass := sizeClasses[props.Size]
	if sizeClass == "" {
		sizeClass = sizeClasses[InputMedium]
	}

	// State classes
	var stateClass string
	if props.Disabled {
		stateClass = "bg-gray-100 border-gray-300 text-gray-500 cursor-not-allowed"
	} else if props.Error {
		stateClass = "bg-white border-red-500 text-gray-900 focus:ring-red-500 focus:border-red-500"
	} else {
		stateClass = "bg-white border-gray-300 text-gray-900 focus:ring-blue-500 focus:border-blue-500 hover:border-gray-400"
	}

	classes := base + " " + sizeClass + " " + stateClass

	if props.Class != "" {
		classes += " " + props.Class
	}

	return classes
}

// DefaultInputProps returns input props with default values
func DefaultInputProps() InputProps {
	return InputProps{
		Type:  InputText,
		Size:  InputMedium,
		Class: "",
	}
}

// MergeInputProps merges provided props with defaults
func MergeInputProps(props InputProps) InputProps {
	defaults := DefaultInputProps()
	if props.Type == "" {
		props.Type = defaults.Type
	}
	if props.Size == "" {
		props.Size = defaults.Size
	}
	return props
}

// LabelClasses returns CSS classes for input labels
func LabelClasses(props InputProps) string {
	base := "block text-sm font-medium mb-1"
	if props.Disabled {
		base += " text-gray-400"
	} else if props.Error {
		base += " text-red-700"
	} else {
		base += " text-gray-700"
	}
	return base
}

// HelperTextClasses returns CSS classes for helper text
func HelperTextClasses(props InputProps) string {
	if props.Error {
		return "mt-1 text-sm text-red-600"
	}
	return "mt-1 text-sm text-gray-500"
}
