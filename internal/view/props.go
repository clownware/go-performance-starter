package view

// BaseProps holds data needed by the base layout component.
type BaseProps struct {
	Title       string
	CurrentYear int
	UserName    string // display name shown in the authenticated user menu; empty for guest pages
}

// NewBaseProps constructs a BaseProps with CurrentYear computed automatically.
func NewBaseProps(title string) BaseProps {
	return BaseProps{
		Title:       title,
		CurrentYear: CurrentYear(),
	}
}
