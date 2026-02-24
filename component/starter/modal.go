package starter

// ModalSize defines the size of a modal
type ModalSize string

const (
	ModalSizeSmall  ModalSize = "small"
	ModalSizeMedium ModalSize = "medium"
	ModalSizeLarge  ModalSize = "large"
	ModalSizeFull   ModalSize = "full"
)

// ModalProps defines the properties for a modal component
type ModalProps struct {
	// ID is the unique identifier
	ID string
	// Title is the modal title
	Title string
	// Size controls the modal size
	Size ModalSize
	// Open controls whether the modal is visible
	Open bool
	// CloseOnOverlay enables closing when clicking outside
	CloseOnOverlay bool
	// CloseOnEscape enables closing with Escape key
	CloseOnEscape bool
	// ShowCloseButton shows the close button
	ShowCloseButton bool
	// Class is additional CSS classes
	Class string
}

// ModalSizeClasses returns CSS classes for modal size
func ModalSizeClasses(size ModalSize) string {
	switch size {
	case ModalSizeSmall:
		return "max-w-sm"
	case ModalSizeMedium:
		return "max-w-lg"
	case ModalSizeLarge:
		return "max-w-2xl"
	case ModalSizeFull:
		return "max-w-full mx-4"
	default:
		return "max-w-lg"
	}
}

// DefaultModalProps returns modal props with default values
func DefaultModalProps() ModalProps {
	return ModalProps{
		Size:            ModalSizeMedium,
		Open:            false,
		CloseOnOverlay:  true,
		CloseOnEscape:   true,
		ShowCloseButton: true,
		Class:           "",
	}
}

// ModalHeaderProps defines properties for a modal header
type ModalHeaderProps struct {
	// Class is additional CSS classes
	Class string
}

// ModalBodyProps defines properties for a modal body
type ModalBodyProps struct {
	// Class is additional CSS classes
	Class string
}

// ModalFooterProps defines properties for a modal footer
type ModalFooterProps struct {
	// Class is additional CSS classes
	Class string
}

// ModalHeaderClasses returns CSS classes for modal header
func ModalHeaderClasses(props ModalHeaderProps) string {
	base := "px-6 py-4 border-b border-gray-200"
	if props.Class != "" {
		base += " " + props.Class
	}
	return base
}

// ModalBodyClasses returns CSS classes for modal body
func ModalBodyClasses(props ModalBodyProps) string {
	base := "p-6"
	if props.Class != "" {
		base += " " + props.Class
	}
	return base
}

// ModalFooterClasses returns CSS classes for modal footer
func ModalFooterClasses(props ModalFooterProps) string {
	base := "px-6 py-4 border-t border-gray-200 bg-gray-50 flex justify-end gap-3"
	if props.Class != "" {
		base += " " + props.Class
	}
	return base
}
