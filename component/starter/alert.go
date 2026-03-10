package starter

// AlertVariant defines the style variant for an alert
type AlertVariant string

const (
	// AlertVariantInfo is an info alert
	AlertVariantInfo AlertVariant = "info"
	// AlertVariantSuccess is a success alert
	AlertVariantSuccess AlertVariant = "success"
	// AlertVariantWarning is a warning alert
	AlertVariantWarning AlertVariant = "warning"
	// AlertVariantError is an error alert
	AlertVariantError AlertVariant = "error"
)

// AlertProps defines the properties for an alert component
type AlertProps struct {
	// ID is the unique identifier
	ID string
	// Title is the alert title
	Title string
	// Message is the alert message
	Message string
	// Variant controls the alert style
	Variant AlertVariant
	// Dismissible allows the alert to be closed
	Dismissible bool
	// Class is additional CSS classes
	Class string
}

// AlertVariantClasses returns CSS classes for alert variant
func AlertVariantClasses(variant AlertVariant) string {
	switch variant {
	case AlertVariantInfo:
		return "bg-blue-50 border-blue-200 text-blue-800"
	case AlertVariantSuccess:
		return "bg-green-50 border-green-200 text-green-800"
	case AlertVariantWarning:
		return "bg-yellow-50 border-yellow-200 text-yellow-800"
	case AlertVariantError:
		return "bg-red-50 border-red-200 text-red-800"
	default:
		return "bg-gray-50 border-gray-200 text-gray-800"
	}
}

// AlertIconClasses returns icon color classes for alert variant
func AlertIconClasses(variant AlertVariant) string {
	switch variant {
	case AlertVariantInfo:
		return "text-blue-400"
	case AlertVariantSuccess:
		return "text-green-400"
	case AlertVariantWarning:
		return "text-yellow-400"
	case AlertVariantError:
		return "text-red-400"
	default:
		return "text-gray-400"
	}
}

// DefaultAlertProps returns alert props with default values
func DefaultAlertProps() AlertProps {
	return AlertProps{
		Variant:     AlertVariantInfo,
		Dismissible: false,
		Class:       "",
	}
}

// BadgeVariant defines the style variant for a badge
type BadgeVariant string

const (
	// BadgeVariantDefault is a default badge
	BadgeVariantDefault BadgeVariant = "default"
	// BadgeVariantPrimary is a primary badge
	BadgeVariantPrimary BadgeVariant = "primary"
	// BadgeVariantSuccess is a success badge
	BadgeVariantSuccess BadgeVariant = "success"
	// BadgeVariantWarning is a warning badge
	BadgeVariantWarning BadgeVariant = "warning"
	// BadgeVariantError is an error badge
	BadgeVariantError BadgeVariant = "error"
)

// BadgeSize defines the size of a badge
type BadgeSize string

const (
	// BadgeSizeSmall is a small badge
	BadgeSizeSmall BadgeSize = "small"
	// BadgeSizeMedium is a medium badge
	BadgeSizeMedium BadgeSize = "medium"
	// BadgeSizeLarge is a large badge
	BadgeSizeLarge BadgeSize = "large"
)

// BadgeProps defines the properties for a badge component
type BadgeProps struct {
	// ID is the unique identifier
	ID string
	// Text is the badge text
	Text string
	// Variant controls the badge style
	Variant BadgeVariant
	// Size controls the badge size
	Size BadgeSize
	// Class is additional CSS classes
	Class string
}

// BadgeVariantClasses returns CSS classes for badge variant
func BadgeVariantClasses(variant BadgeVariant) string {
	switch variant {
	case BadgeVariantPrimary:
		return "bg-blue-100 text-blue-800"
	case BadgeVariantSuccess:
		return "bg-green-100 text-green-800"
	case BadgeVariantWarning:
		return "bg-yellow-100 text-yellow-800"
	case BadgeVariantError:
		return "bg-red-100 text-red-800"
	default:
		return "bg-gray-100 text-gray-800"
	}
}

// BadgeSizeClasses returns CSS classes for badge size
func BadgeSizeClasses(size BadgeSize) string {
	switch size {
	case BadgeSizeSmall:
		return "text-xs px-2 py-0.5"
	case BadgeSizeLarge:
		return "text-sm px-3 py-1"
	default:
		return "text-xs px-2.5 py-0.5"
	}
}

// DefaultBadgeProps returns badge props with default values
func DefaultBadgeProps() BadgeProps {
	return BadgeProps{
		Variant: BadgeVariantDefault,
		Size:    BadgeSizeMedium,
		Class:   "",
	}
}
