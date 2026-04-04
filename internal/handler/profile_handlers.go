package handler

import (
	"net/http"
	"strings"

	"github.com/clownware/alpine-go-performance-starter/internal/view"
	"github.com/clownware/alpine-go-performance-starter/internal/view/pages"
	"github.com/clownware/alpine-go-performance-starter/internal/view/partials"
	"github.com/clownware/alpine-go-performance-starter/internal/webutil"
)

// ProfileView renders the profile page (full page or fragment fallback).
func ProfileView(w http.ResponseWriter, r *http.Request) {
	props := pages.ProfilePageProps{
		BaseProps: view.NewBaseProps("Profile"),
		Name:     "John Doe",
	}
	view.Render(w, r, http.StatusOK, pages.ProfilePage(props))
}

// ProfileUpdate processes the profile form submission with HTMX support.
func ProfileUpdate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		JSONError(w, http.StatusBadRequest, err)
		return
	}
	name := r.FormValue("name")
	errors := map[string]string{}
	if strings.TrimSpace(name) == "" {
		errors["name"] = "Name cannot be empty"
	}

	// Validation errors
	if len(errors) > 0 {
		formProps := partials.ProfileFormProps{Name: name, Errors: errors}
		if webutil.IsHTMXRequest(r) {
			view.Render(w, r, http.StatusUnprocessableEntity, partials.ProfileForm(formProps))
		} else {
			pageProps := pages.ProfilePageProps{
				BaseProps: view.NewBaseProps("Profile"),
				Name:     name,
				Errors:   errors,
			}
			view.Render(w, r, http.StatusUnprocessableEntity, pages.ProfilePage(pageProps))
		}
		return
	}

	// Stub update successful
	if webutil.IsHTMXRequest(r) {
		webutil.SetHXTrigger(w, "Profile updated successfully!")
		formProps := partials.ProfileFormProps{Name: name, Success: true}
		view.Render(w, r, http.StatusOK, partials.ProfileForm(formProps))
	} else {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
	}
}
