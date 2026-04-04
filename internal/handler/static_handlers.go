package handler

import (
	"net/http"
	"github.com/clownware/alpine-go-performance-starter/internal/webutil"
)

// DashboardPage renders the dashboard page.
func DashboardPage(w http.ResponseWriter, r *http.Request) {
	webutil.RenderTemplate(w, r, http.StatusOK, "pages/dashboard.html", nil)
}

// TermsPage renders the terms of service page.
func TermsPage(w http.ResponseWriter, r *http.Request) {
	webutil.RenderTemplate(w, r, http.StatusOK, "pages/terms.html", nil)
}

// PrivacyPage renders the privacy policy page.
func PrivacyPage(w http.ResponseWriter, r *http.Request) {
	webutil.RenderTemplate(w, r, http.StatusOK, "pages/privacy.html", nil)
}

// LogoutPage handles GET /auth/logout by POSTing to logout and redirecting to home.
func LogoutPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Render a form that POSTs to /auth/logout (HTMX or standard)
		webutil.RenderTemplate(w, r, http.StatusOK, "pages/logout.html", nil)
		return
	}
	// Fallback: redirect to home
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
