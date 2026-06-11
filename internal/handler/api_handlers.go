package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/clownware/alpine-go-performance-starter/internal/view"
)

// APIPlaceholder returns a simple JSON message
func APIPlaceholder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "API is running"}); err != nil {
		slog.Error("failed to encode response", "handler", "APIPlaceholder", "error", err)
	}
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
	if err := json.NewEncoder(w).Encode(user); err != nil {
		slog.Error("failed to encode response", "handler", "GetUserProfile", "error", err)
	}
}

// ListOrganizations returns a stub list of organizations
func ListOrganizations(w http.ResponseWriter, r *http.Request) {
	orgs := []view.Organization{
		{ID: "org1", Name: "Organization One"},
		{ID: "org2", Name: "Organization Two"},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(orgs); err != nil {
		slog.Error("failed to encode response", "handler", "ListOrganizations", "error", err)
	}
}
