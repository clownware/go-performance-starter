package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/clownware/alpine-go-performance-starter/internal/middleware"
)

// ProfilePage displays the user's profile information.
func ProfilePage(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		// This shouldn't happen if middleware is applied correctly
		log.Println("[ERROR] User not found in context on protected route")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// TODO: Fetch user profile data from database if needed

	// For now, just display the email from the JWT
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "<h1>User Profile</h1><p>Welcome, %s!</p><p>User ID: %s</p>", user.Email, user.ID)
	// Add a logout button
	_, _ = fmt.Fprint(w, `<form hx-post="/auth/logout" hx-target="body"><button type="submit">Logout</button></form>`)

	log.Printf("[INFO] Displayed profile page for user: %s", user.Email)
}
