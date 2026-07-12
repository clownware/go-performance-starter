package view

// PatternSection describes one pattern in the /patterns showcase (ADR-024):
// a live demo panel plus the templ and handler source that produce it.
type PatternSection struct {
	Slug           string
	Title          string
	Summary        string
	Category       string // slug of the PatternCategory this belongs to
	HTMXFeatures   []string
	AlpineFeatures []string
	TemplSource    string
	HandlerSource  string
}

// PatternCategory is one teaching group on the showcase.
type PatternCategory struct {
	Slug  string
	Title string
	Blurb string
}

// PatternGroup pairs a category with its patterns, in display order.
type PatternGroup struct {
	Category PatternCategory
	Sections []PatternSection
}

// QuizScore is the running quiz progress shown on the quiz surfaces (ADR-024).
type QuizScore struct {
	Correct int
	Total   int
}
