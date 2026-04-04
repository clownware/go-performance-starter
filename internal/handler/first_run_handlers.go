package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/clownware/alpine-go-performance-starter/internal/view"
	"github.com/clownware/alpine-go-performance-starter/internal/view/partials"
	"github.com/clownware/alpine-go-performance-starter/internal/webutil"
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
	// TODO: Re-enable after adding first_run_complete column to users table
	// if user.FirstRunComplete {
	// 	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	// 	return
	// }
	view.Render(w, r, http.StatusOK, partials.FirstRun(partials.FirstRunProps{Step: 1}))
}

// ShowFirstRunProfile serves the profile setup prompt (step 2).
func ShowFirstRunProfile(w http.ResponseWriter, r *http.Request) {
	user := webutil.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/page", http.StatusSeeOther)
		return
	}
	// TODO: Re-enable after adding first_run_complete column to users table
	// if user.FirstRunComplete {
	// 	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	// 	return
	// }
	userName := ""
	if user.Name.Valid {
		userName = user.Name.String
	}
	view.Render(w, r, http.StatusOK, partials.FirstRun(partials.FirstRunProps{Step: 2, UserName: userName}))
}

// ShowFirstRunCTAs serves the final CTAs (step 3).
func ShowFirstRunCTAs(w http.ResponseWriter, r *http.Request) {
	user := webutil.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/page", http.StatusSeeOther)
		return
	}
	// TODO: Re-enable after adding first_run_complete column to users table
	// if user.FirstRunComplete {
	// 	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	// 	return
	// }
	// Mark onboarding as complete
	repo := webutil.GetUserRepoFromContext(r.Context())
	err := repo.UpdateFirstRunComplete(r.Context(), user.ID, true)
	if err != nil {
		view.Render(w, r, http.StatusInternalServerError, partials.FirstRun(partials.FirstRunProps{
			Step:  3,
			Error: "Could not complete onboarding. Please try again.",
		}))
		return
	}
	// Optionally update user in session/context
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

