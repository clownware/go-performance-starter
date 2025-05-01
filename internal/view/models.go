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
