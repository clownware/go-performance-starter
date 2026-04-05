package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/clownware/alpine-go-performance-starter/internal/middleware"
	"github.com/clownware/alpine-go-performance-starter/internal/view"
	"github.com/clownware/alpine-go-performance-starter/internal/view/pages"
	"github.com/clownware/alpine-go-performance-starter/internal/view/partials"
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

// ProfileView renders the profile page (full page or fragment fallback).
func ProfileView(w http.ResponseWriter, r *http.Request) {
	userName := userNameFromContext(r)
	baseProps := view.NewBaseProps("Profile")
	baseProps.UserName = userName
	props := pages.ProfilePageProps{
		BaseProps: baseProps,
		Name:     userName,
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
	name := r.FormValue("name")
	errors := make(map[string]string)
	if strings.TrimSpace(name) == "" {
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
				Name:     name,
				Errors:   errors,
			}
			if err := view.Render(w, r, http.StatusUnprocessableEntity, pages.ProfilePage(pageProps)); err != nil {
				slog.Error("Failed to render profile page", "error", err)
			}
		}
		return
	}

	// Stub update successful
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
