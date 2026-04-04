package handler

import (
	"net/http"

	"github.com/clownware/alpine-go-performance-starter/internal/view"
	"github.com/clownware/alpine-go-performance-starter/internal/view/pages"
)

// DashboardPage renders the dashboard page.
func DashboardPage(w http.ResponseWriter, r *http.Request) {
	props := pages.DashboardPageProps{BaseProps: view.NewBaseProps("Dashboard")}
	view.Render(w, r, http.StatusOK, pages.DashboardPage(props))
}

// TermsPage renders the terms of service page.
func TermsPage(w http.ResponseWriter, r *http.Request) {
	props := pages.TermsPageProps{BaseProps: view.NewBaseProps("Terms of Service")}
	view.Render(w, r, http.StatusOK, pages.TermsPage(props))
}

// PrivacyPage renders the privacy policy page.
func PrivacyPage(w http.ResponseWriter, r *http.Request) {
	props := pages.PrivacyPageProps{BaseProps: view.NewBaseProps("Privacy Policy")}
	view.Render(w, r, http.StatusOK, pages.PrivacyPage(props))
}

// LogoutPage handles GET /auth/logout by POSTing to logout and redirecting to home.
func LogoutPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		props := pages.LogoutPageProps{BaseProps: view.NewBaseProps("Sign Out")}
		view.Render(w, r, http.StatusOK, pages.LogoutPage(props))
		return
	}
	// Fallback: redirect to home
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
