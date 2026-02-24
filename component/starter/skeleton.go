package starter

// SkeletonProps defines the properties for a skeleton loading component
type SkeletonProps struct {
	// ID is the unique identifier
	ID string
	// Width is the skeleton width (CSS value)
	Width string
	// Height is the skeleton height (CSS value)
	Height string
	// Variant controls the skeleton shape
	Variant SkeletonVariant
	// Class is additional CSS classes
	Class string
	// Animated controls whether the skeleton has pulse animation
	Animated bool
}

// SkeletonVariant defines the shape variant for skeleton
type SkeletonVariant string

const (
	SkeletonVariantText    SkeletonVariant = "text"
	SkeletonVariantCircle  SkeletonVariant = "circle"
	SkeletonVariantRect    SkeletonVariant = "rect"
	SkeletonVariantRounded SkeletonVariant = "rounded"
)

// SkeletonVariantClasses returns CSS classes for skeleton variant
func SkeletonVariantClasses(variant SkeletonVariant) string {
	switch variant {
	case SkeletonVariantCircle:
		return "rounded-full"
	case SkeletonVariantRect:
		return "rounded-none"
	case SkeletonVariantRounded:
		return "rounded-lg"
	default:
		return "rounded"
	}
}

// DefaultSkeletonProps returns skeleton props with default values
func DefaultSkeletonProps() SkeletonProps {
	return SkeletonProps{
		Variant:  SkeletonVariantText,
		Animated: true,
		Class:    "",
	}
}

// SkeletonTextProps defines properties for skeleton text
type SkeletonTextProps struct {
	// ID is the unique identifier
	ID string
	// Lines is the number of text lines
	Lines int
	// LineHeight is the height of each line
	LineHeight string
	// LastLineWidth is the width of the last line (percentage or CSS value)
	LastLineWidth string
	// Class is additional CSS classes
	Class string
}

// DefaultSkeletonTextProps returns skeleton text props with defaults
func DefaultSkeletonTextProps() SkeletonTextProps {
	return SkeletonTextProps{
		Lines:         3,
		LineHeight:    "1rem",
		LastLineWidth: "60%",
		Class:         "",
	}
}

// SkeletonCardProps defines properties for a skeleton card
type SkeletonCardProps struct {
	// ID is the unique identifier
	ID string
	// ShowImage controls whether to show image placeholder
	ShowImage bool
	// ImageHeight is the height of the image placeholder
	ImageHeight string
	// ShowAvatar controls whether to show avatar placeholder
	ShowAvatar bool
	// ShowTitle controls whether to show title placeholder
	ShowTitle bool
	// ShowDescription controls whether to show description placeholder
	ShowDescription bool
	// DescriptionLines is the number of description lines
	DescriptionLines int
	// Class is additional CSS classes
	Class string
}

// DefaultSkeletonCardProps returns skeleton card props with defaults
func DefaultSkeletonCardProps() SkeletonCardProps {
	return SkeletonCardProps{
		ShowImage:        true,
		ImageHeight:      "12rem",
		ShowAvatar:       false,
		ShowTitle:        true,
		ShowDescription:  true,
		DescriptionLines: 3,
		Class:            "",
	}
}

// SkeletonTableProps defines properties for a skeleton table
type SkeletonTableProps struct {
	// ID is the unique identifier
	ID string
	// Rows is the number of rows
	Rows int
	// Columns is the number of columns
	Columns int
	// ShowHeader controls whether to show header row
	ShowHeader bool
	// Class is additional CSS classes
	Class string
}

// DefaultSkeletonTableProps returns skeleton table props with defaults
func DefaultSkeletonTableProps() SkeletonTableProps {
	return SkeletonTableProps{
		Rows:       5,
		Columns:    4,
		ShowHeader: true,
		Class:      "",
	}
}
