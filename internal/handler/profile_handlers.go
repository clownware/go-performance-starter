package handler

import (
	"net/http"
	"strings"

	"github.com/clownware/alpine-go-performance-starter/internal/webutil"
)

// ProfileView renders the profile page (full page or fragment fallback).
func ProfileView(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Name": "John Doe",
	}
	webutil.RenderTemplate(w, r, http.StatusOK, "pages/profile.html", data)
}

// ProfileUpdate processes the profile form submission with HTMX support.
func ProfileUpdate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		JSONError(w, http.StatusBadRequest, err)
		return
	}
	name := r.FormValue("name")
	errors := webutil.FormErrors{}
	if strings.TrimSpace(name) == "" {
		errors["name"] = "Name cannot be empty"
	}
	data := map[string]interface{}{
		"Name": name,
	}
	// Validation errors
	if len(errors) > 0 {
		if webutil.IsHTMXRequest(r) {
			webutil.RenderTemplate(w, r, http.StatusUnprocessableEntity, "partials/profile_form.html", map[string]interface{}{ // fragment
				"Name":   name,
				"Errors": errors,
			})
		} else {
			webutil.RenderTemplateWithErrors(w, r, http.StatusUnprocessableEntity, "pages/profile.html", data, errors)
		}
		return
	}
	// Stub update successful
	data["Success"] = true
	if webutil.IsHTMXRequest(r) {
		// Trigger global toast event
		webutil.SetHXTrigger(w, "Profile updated successfully!")
	}
	if webutil.IsHTMXRequest(r) {
		// Return updated form fragment with success message
		webutil.RenderTemplate(w, r, http.StatusOK, "partials/profile_form.html", data)
	} else {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
	}
}
