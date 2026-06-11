package handler

import (
	"log/slog"
	"net/http"

	"github.com/clownware/alpine-go-performance-starter/internal/view"
	"github.com/clownware/alpine-go-performance-starter/internal/view/partials"
	"github.com/clownware/alpine-go-performance-starter/internal/webutil"
	"github.com/go-chi/chi/v5"
)

// FirstRunHandlers registers first-run onboarding routes.
func FirstRunHandlers(r chi.Router) {
	r.Get("/first-run", ShowFirstRunWelcome)
	r.Get("/first-run/profile", ShowFirstRunProfile)
	r.Get("/first-run/ctas", ShowFirstRunCTAs)
}

// ShowFirstRunWelcome serves the welcome banner (step 1).
func ShowFirstRunWelcome(w http.ResponseWriter, r *http.Request) {
	user := webutil.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/page", http.StatusSeeOther)
		return
	}
	props := partials.FirstRunProps{}
	if err := view.Render(w, r, http.StatusOK, partials.FirstRunExperience(props)); err != nil {
		slog.Error("Failed to render first-run welcome", "error", err)
	}
}

// ShowFirstRunProfile serves the profile setup prompt (step 2).
func ShowFirstRunProfile(w http.ResponseWriter, r *http.Request) {
	user := webutil.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/page", http.StatusSeeOther)
		return
	}
	props := partials.FirstRunProps{
		ShowProfileSetup: true,
	}
	if err := view.Render(w, r, http.StatusOK, partials.FirstRunExperience(props)); err != nil {
		slog.Error("Failed to render first-run profile", "error", err)
	}
}

// ShowFirstRunCTAs serves the final CTAs (step 3).
func ShowFirstRunCTAs(w http.ResponseWriter, r *http.Request) {
	user := webutil.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/page", http.StatusSeeOther)
		return
	}
	// Mark onboarding as complete
	repo := webutil.GetUserRepoFromContext(r.Context())
	err := repo.UpdateFirstRunComplete(r.Context(), user.ID, true)
	if err != nil {
		props := partials.FirstRunProps{
			ShowCTAs: true,
			Error:    "Could not complete onboarding. Please try again.",
		}
		if renderErr := view.Render(w, r, http.StatusInternalServerError, partials.FirstRunExperience(props)); renderErr != nil {
			slog.Error("Failed to render first-run CTAs", "error", renderErr)
		}
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
