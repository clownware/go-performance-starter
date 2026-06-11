package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/clownware/alpine-go-performance-starter/internal/view"
	"github.com/go-chi/chi/v5"
)

// APIPlaceholder returns a simple JSON message
func APIPlaceholder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "API is running"})
}

// GetUserProfile returns a stub user profile
func GetUserProfile(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		JSONError(w, http.StatusBadRequest, errors.New("missing userID"))
		return
	}
	// TODO: fetch real user from database by ID
	user := view.UserProfile{
		ID:    userID,
		Name:  "Stub User",
		Email: "stub@example.com",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

// ListOrganizations returns a stub list of organizations
func ListOrganizations(w http.ResponseWriter, r *http.Request) {
	orgs := []view.Organization{
		{ID: "org1", Name: "Organization One"},
		{ID: "org2", Name: "Organization Two"},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(orgs)
}
