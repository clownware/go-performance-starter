package handler

import (
	"log/slog"
	"net/http"

	"github.com/clownware/alpine-go-performance-starter/internal/view"
	"github.com/clownware/alpine-go-performance-starter/internal/view/pages"
)

// DashboardPage renders the dashboard page.
func DashboardPage(w http.ResponseWriter, r *http.Request) {
	props := pages.DashboardPageProps{
		BaseProps: view.NewBaseProps("Dashboard"),
	}
	if err := view.Render(w, r, http.StatusOK, pages.DashboardPage(props)); err != nil {
		slog.Error("Failed to render dashboard page", "error", err)
	}
}

// TermsPage renders the terms of service page.
func TermsPage(w http.ResponseWriter, r *http.Request) {
	props := view.NewBaseProps("Terms of Service")
	if err := view.Render(w, r, http.StatusOK, pages.TermsPage(props)); err != nil {
		slog.Error("Failed to render terms page", "error", err)
	}
}

// PrivacyPage renders the privacy policy page.
func PrivacyPage(w http.ResponseWriter, r *http.Request) {
	props := view.NewBaseProps("Privacy Policy")
	if err := view.Render(w, r, http.StatusOK, pages.PrivacyPage(props)); err != nil {
		slog.Error("Failed to render privacy page", "error", err)
	}
}

// LogoutPage handles GET /auth/logout by rendering a confirmation form.
func LogoutPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		props := view.NewBaseProps("Sign Out")
		if err := view.Render(w, r, http.StatusOK, pages.LogoutPage(props)); err != nil {
			slog.Error("Failed to render logout page", "error", err)
		}
		return
	}
	// Fallback: redirect to home
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
