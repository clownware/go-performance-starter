package view

// PatternSection describes one pattern in the /patterns showcase (ADR-024):
// a live demo panel plus the templ and handler source that produce it.
type PatternSection struct {
	Slug           string
	Title          string
	Summary        string
	HTMXFeatures   []string
	AlpineFeatures []string
	TemplSource    string
	HandlerSource  string
}

// QuizScore is the running quiz progress shown on the quiz surfaces (ADR-024).
type QuizScore struct {
	Correct int
	Total   int
}
