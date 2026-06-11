package components

import "github.com/a-h/templ"

// ---------------------------------------------------------------------------
// Button
// ---------------------------------------------------------------------------

// ButtonVariant controls the visual style of a Button.
type ButtonVariant string

const (
	ButtonPrimary   ButtonVariant = "primary"
	ButtonSecondary ButtonVariant = "secondary"
	ButtonDanger    ButtonVariant = "danger"
)

// ButtonProps configures a Button component.
type ButtonProps struct {
	Text     string
	Type     string        // "button" | "submit" | "reset"; defaults to "button"
	Variant  ButtonVariant // defaults to ButtonPrimary
	ID       string
	Class    string // overrides variant-based class when set
	Disabled bool
	Attrs    templ.Attributes // HTMX, ARIA, or any extra HTML attributes
}

// ---------------------------------------------------------------------------
// Input
// ---------------------------------------------------------------------------

// InputProps configures an Input component.
type InputProps struct {
	ID           string
	Name         string
	Type         string // defaults to "text"
	Value        string
	Placeholder  string
	Label        string
	Required     bool
	Disabled     bool
	Readonly     bool
	HasError     bool
	ErrorMessage string
	HelperText   string
	Class        string           // overrides default input class
	Attrs        templ.Attributes // HTMX, ARIA, or any extra HTML attributes
}

// ---------------------------------------------------------------------------
// Card
// ---------------------------------------------------------------------------

// CardProps configures a Card component. Body content is passed via {children...}.
type CardProps struct {
	Title    string
	Subtitle string
	ID       string
	Class    string
	Footer   templ.Component  // optional footer slot
	Attrs    templ.Attributes // extra HTML attributes
}

// ---------------------------------------------------------------------------
// Alert
// ---------------------------------------------------------------------------

// AlertType controls the color scheme and icon of an Alert.
type AlertType string

const (
	AlertSuccess AlertType = "success"
	AlertWarning AlertType = "warning"
	AlertError   AlertType = "error"
	AlertInfo    AlertType = "info"
)

// AlertProps configures an Alert component.
type AlertProps struct {
	Type        AlertType // defaults to AlertInfo
	Title       string
	Message     string
	Dismissible bool
	ID          string
	Class       string
	Attrs       templ.Attributes
}

// ---------------------------------------------------------------------------
// Form
// ---------------------------------------------------------------------------

// FormProps configures a Form wrapper component. Body content is passed via {children...}.
type FormProps struct {
	ID     string
	Action string
	Method string           // defaults to "post"
	Class  string           // defaults to "space-y-6"
	Attrs  templ.Attributes // HTMX attributes, novalidate, etc.
}

// ---------------------------------------------------------------------------
// FormValidation
// ---------------------------------------------------------------------------

// FormValidationProps configures a FormValidation input with Alpine.js
// client-side + server-side error display.
type FormValidationProps struct {
	ID              string
	Name            string
	Type            string // defaults to "text"
	Label           string
	Placeholder     string
	Value           string
	Required        bool
	Pattern         string
	MinLength       int
	MaxLength       int
	ServerError     string
	RequiredMessage string // defaults to "This field is required"
	PatternMessage  string // defaults to "Please enter a valid value"
	HelperText      string
	Class           string // wrapper class; defaults to "mb-4"
	Attrs           templ.Attributes
}

// ---------------------------------------------------------------------------
// Accessibility
// ---------------------------------------------------------------------------

// SkipLinkProps configures a skip-to-content accessibility link.
type SkipLinkProps struct {
	Target string // href target, e.g. "#main-content"
	Text   string // visible label, e.g. "Skip to content"
}

// AriaLiveRegionProps configures an ARIA live region for dynamic announcements.
type AriaLiveRegionProps struct {
	ID         string
	AriaLive   string // defaults to "polite"
	AriaAtomic string // defaults to "true"
	Class      string // defaults to "sr-only"
}

// LoadingIndicatorProps configures an HTMX loading spinner.
type LoadingIndicatorProps struct {
	ID    string
	Class string
}

// FocusableElementProps configures a keyboard-accessible container.
// Body content is passed via {children...}.
type FocusableElementProps struct {
	Role         string // defaults to "button"
	AriaLabel    string
	AriaExpanded string
	AriaControls string
	ID           string
	Class        string
	Attrs        templ.Attributes // @click, @keydown, etc.
}
