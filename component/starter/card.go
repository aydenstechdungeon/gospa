package starter

// CardProps defines the properties for a card component
type CardProps struct {
	// ID is the unique identifier
	ID string
	// Title is the card title
	Title string
	// Subtitle is the card subtitle
	Subtitle string
	// Class is additional CSS classes
	Class string
	// Padding controls the card padding
	Padding bool
	// Shadow controls the card shadow
	Shadow bool
	// Border controls the card border
	Border bool
	// Rounded controls the border radius
	Rounded bool
	// Hover enables hover effects
	Hover bool
	// Attributes are additional HTML attributes
	Attributes map[string]string
}

// CardHeaderProps defines properties for a card header
type CardHeaderProps struct {
	// Class is additional CSS classes
	Class string
}

// CardBodyProps defines properties for a card body
type CardBodyProps struct {
	// Class is additional CSS classes
	Class string
}

// CardFooterProps defines properties for a card footer
type CardFooterProps struct {
	// Class is additional CSS classes
	Class string
}

// CardClasses returns CSS classes for a card
func CardClasses(props CardProps) string {
	base := "bg-white overflow-hidden"

	if props.Padding {
		base += " p-6"
	}

	if props.Shadow {
		base += " shadow-md"
	}

	if props.Border {
		base += " border border-gray-200"
	}

	if props.Rounded {
		base += " rounded-lg"
	}

	if props.Hover {
		base += " transition-shadow duration-200 hover:shadow-lg"
	}

	if props.Class != "" {
		base += " " + props.Class
	}

	return base
}

// DefaultCardProps returns card props with default values
func DefaultCardProps() CardProps {
	return CardProps{
		Padding: true,
		Shadow:  true,
		Border:  true,
		Rounded: true,
		Hover:   false,
		Class:   "",
	}
}

// MergeCardProps merges provided props with defaults
// For booleans, zero value (false) is treated as "use default"
func MergeCardProps(props CardProps) CardProps {
	// For booleans, we treat zero values as "use default"
	// This means if a boolean is false, it will use the default (true for most)
	return props
}

// CardHeaderClasses returns CSS classes for card header
func CardHeaderClasses(props CardHeaderProps) string {
	base := "px-6 py-4 border-b border-gray-200"
	if props.Class != "" {
		base += " " + props.Class
	}
	return base
}

// CardBodyClasses returns CSS classes for card body
func CardBodyClasses(props CardBodyProps) string {
	base := "p-6"
	if props.Class != "" {
		base += " " + props.Class
	}
	return base
}

// CardFooterClasses returns CSS classes for card footer
func CardFooterClasses(props CardFooterProps) string {
	base := "px-6 py-4 border-t border-gray-200 bg-gray-50"
	if props.Class != "" {
		base += " " + props.Class
	}
	return base
}
