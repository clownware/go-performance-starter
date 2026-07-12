package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/clownware/go-performance-starter/internal/middleware"
	"github.com/clownware/go-performance-starter/internal/view"
	"github.com/clownware/go-performance-starter/internal/view/pages"
	"github.com/clownware/go-performance-starter/internal/view/partials"
	"github.com/clownware/go-performance-starter/internal/webutil"
)

// userNameFromContext extracts the display name from the authenticated user context.
// Falls back to email, then "User" if no auth data is available.
func userNameFromContext(r *http.Request) string {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		return "User"
	}
	if name, _ := user.UserMetadata["full_name"].(string); name != "" {
		return name
	}
	if name, _ := user.UserMetadata["name"].(string); name != "" {
		return name
	}
	if user.Email != "" {
		return user.Email
	}
	return "User"
}

// profileDisplayName prefers the persisted users row (the write target of
// ProfileUpdate, #70) and falls back to token metadata for identities whose
// row has no name yet.
func profileDisplayName(r *http.Request) string {
	if u := webutil.GetUserFromContext(r.Context()); u != nil && u.Name.Valid && u.Name.String != "" {
		return u.Name.String
	}
	return userNameFromContext(r)
}

// ProfileView renders the profile page (full page or fragment fallback).
func ProfileView(w http.ResponseWriter, r *http.Request) {
	userName := profileDisplayName(r)
	baseProps := view.NewBaseProps("Profile")
	baseProps.UserName = userName
	props := pages.ProfilePageProps{
		BaseProps: baseProps,
		Name:      userName,
	}
	if err := view.Render(w, r, http.StatusOK, pages.ProfilePage(props)); err != nil {
		slog.Error("Failed to render profile page", "error", err)
	}
}

// ProfileUpdate processes the profile form submission with HTMX support.
func ProfileUpdate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		JSONError(w, http.StatusBadRequest, err)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	errors := make(map[string]string)
	if name == "" {
		errors["name"] = "Name cannot be empty"
	}

	// Validation errors
	if len(errors) > 0 {
		formProps := partials.ProfileFormProps{
			Name:   name,
			Errors: errors,
		}
		if view.IsHTMXRequest(r) {
			if err := view.Render(w, r, http.StatusUnprocessableEntity, partials.ProfileForm(formProps)); err != nil {
				slog.Error("Failed to render profile form partial", "error", err)
			}
		} else {
			baseProps := view.NewBaseProps("Profile")
			baseProps.UserName = userNameFromContext(r)
			pageProps := pages.ProfilePageProps{
				BaseProps: baseProps,
				Name:      name,
				Errors:    errors,
			}
			if err := view.Render(w, r, http.StatusUnprocessableEntity, pages.ProfilePage(pageProps)); err != nil {
				slog.Error("Failed to render profile page", "error", err)
			}
		}
		return
	}

	// Persist to the users row — UserLoader put it (and UserRepoMiddleware
	// the repo) in context; without a row there is nothing to update.
	user := webutil.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/page", http.StatusSeeOther)
		return
	}
	repo := webutil.GetUserRepoFromContext(r.Context())
	if repo == nil {
		slog.Error("Profile update without a user repository in context")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if _, err := repo.UpdateName(r.Context(), user.ID, name); err != nil {
		slog.Error("Failed to persist profile name", "user_id", user.ID, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if view.IsHTMXRequest(r) {
		view.SetHXTrigger(w, "Profile updated successfully!")
		formProps := partials.ProfileFormProps{
			Name:    name,
			Success: true,
		}
		if err := view.Render(w, r, http.StatusOK, partials.ProfileForm(formProps)); err != nil {
			slog.Error("Failed to render profile form partial", "error", err)
		}
	} else {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
	}
}
