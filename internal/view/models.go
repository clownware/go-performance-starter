package view

// UserProfile represents the data sent for a user profile view.
type UserProfile struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Organization represents the data sent for an organization view.
type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Item represents a generic item for list examples.
type Item struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	IsFavorite bool   `json:"isFavorite"`
}

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
