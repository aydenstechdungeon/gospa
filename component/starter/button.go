// Package starter provides a library of reusable UI components for GoSPA applications
package starter

import (
	"github.com/a-h/templ"
)

// ButtonVariant defines the visual style of a button
type ButtonVariant string

const (
	// ButtonPrimary is a primary button
	ButtonPrimary ButtonVariant = "primary"
	// ButtonSecondary is a secondary button
	ButtonSecondary ButtonVariant = "secondary"
	// ButtonOutline is an outline button
	ButtonOutline ButtonVariant = "outline"
	// ButtonGhost is a ghost button
	ButtonGhost ButtonVariant = "ghost"
	// ButtonDanger is a danger button
	ButtonDanger ButtonVariant = "danger"
)

// ButtonSize defines the size of a button
type ButtonSize string

const (
	// ButtonSizeXS is an extra small button
	ButtonSizeXS ButtonSize = "xs"
	// ButtonSizeSM is a small button
	ButtonSizeSM ButtonSize = "sm"
	// ButtonSizeMD is a medium button
	ButtonSizeMD ButtonSize = "md"
	// ButtonSizeLG is a large button
	ButtonSizeLG ButtonSize = "lg"
	// ButtonSizeXL is an extra large button
	ButtonSizeXL ButtonSize = "xl"
)

// ButtonProps defines the properties for a Button component
type ButtonProps struct {
	// Variant defines the visual style (primary, secondary, outline, ghost, danger)
	Variant ButtonVariant
	// Size defines the button size (xs, sm, md, lg, xl)
	Size ButtonSize
	// Disabled disables the button
	Disabled bool
	// Loading shows a loading spinner
	Loading bool
	// FullWidth makes the button full width
	FullWidth bool
	// Type is the button type (button, submit, reset)
	Type string
	// Href converts the button to a link
	Href string
	// Target for links (_blank, etc.)
	Target string
	// ID is the element ID
	ID string
	// Class adds additional CSS classes
	Class string
	// Attributes adds additional HTML attributes
	Attributes templ.Attributes
	// Onclick is the click handler
	Onclick string
}

// ButtonClasses returns the CSS classes for a button based on props
func ButtonClasses(props ButtonProps) string {
	classes := "inline-flex items-center justify-center font-medium rounded-lg transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"

	// Variant classes
	switch props.Variant {
	case ButtonPrimary:
		classes += " bg-blue-600 text-white hover:bg-blue-700 focus:ring-blue-500"
	case ButtonSecondary:
		classes += " bg-gray-200 text-gray-900 hover:bg-gray-300 focus:ring-gray-500"
	case ButtonOutline:
		classes += " border-2 border-gray-300 text-gray-700 hover:bg-gray-50 focus:ring-gray-500"
	case ButtonGhost:
		classes += " text-gray-700 hover:bg-gray-100 focus:ring-gray-500"
	case ButtonDanger:
		classes += " bg-red-600 text-white hover:bg-red-700 focus:ring-red-500"
	default:
		classes += " bg-blue-600 text-white hover:bg-blue-700 focus:ring-blue-500"
	}

	// Size classes
	switch props.Size {
	case ButtonSizeXS:
		classes += " px-2.5 py-1.5 text-xs gap-1"
	case ButtonSizeSM:
		classes += " px-3 py-2 text-sm gap-1.5"
	case ButtonSizeMD:
		classes += " px-4 py-2 text-sm gap-2"
	case ButtonSizeLG:
		classes += " px-4 py-2.5 text-base gap-2"
	case ButtonSizeXL:
		classes += " px-6 py-3 text-lg gap-2"
	default:
		classes += " px-4 py-2 text-sm gap-2"
	}

	// Full width
	if props.FullWidth {
		classes += " w-full"
	}

	// Additional classes
	if props.Class != "" {
		classes += " " + props.Class
	}

	return classes
}

// DefaultButtonProps returns ButtonProps with default values
func DefaultButtonProps() ButtonProps {
	return ButtonProps{
		Variant: ButtonPrimary,
		Size:    ButtonSizeMD,
		Type:    "button",
	}
}

// MergeButtonProps merges provided props with defaults
func MergeButtonProps(props ButtonProps) ButtonProps {
	defaults := DefaultButtonProps()
	if props.Variant == "" {
		props.Variant = defaults.Variant
	}
	if props.Size == "" {
		props.Size = defaults.Size
	}
	if props.Type == "" {
		props.Type = defaults.Type
	}
	return props
}

// Button renders a button component
func Button(props ButtonProps) templ.Component {
	props = MergeButtonProps(props)
	if props.Onclick != "" {
		if props.Attributes == nil {
			props.Attributes = make(templ.Attributes)
		}
		props.Attributes["onclick"] = props.Onclick
	}
	return buttonTemplate(props)
}

// PrimaryButton renders a primary button
func PrimaryButton(props ButtonProps) templ.Component {
	props.Variant = ButtonPrimary
	return Button(props)
}

// SecondaryButton renders a secondary button
func SecondaryButton(props ButtonProps) templ.Component {
	props.Variant = ButtonSecondary
	return Button(props)
}

// OutlineButton renders an outline button
func OutlineButton(props ButtonProps) templ.Component {
	props.Variant = ButtonOutline
	return Button(props)
}

// GhostButton renders a ghost button
func GhostButton(props ButtonProps) templ.Component {
	props.Variant = ButtonGhost
	return Button(props)
}

// DangerButton renders a danger button
func DangerButton(props ButtonProps) templ.Component {
	props.Variant = ButtonDanger
	return Button(props)
}

// IconButton renders a button with only an icon
func IconButton(props ButtonProps) templ.Component {
	props = MergeButtonProps(props)
	props.Class = "p-2 " + props.Class
	return Button(props)
}
