package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/supabase-community/gotrue-go/types"

	"github.com/clownware/go-performance-starter/internal/auth"
	"github.com/clownware/go-performance-starter/internal/view"
	"github.com/clownware/go-performance-starter/internal/view/pages"
	"github.com/clownware/go-performance-starter/internal/view/partials"
)

// AuthPage renders the tabbed login/signup card. ?mode=signup activates the
// signup tab server-side, so tab switching works without JS; Alpine enhances
// it into an instant toggle.
func AuthPage(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("mode")
	if mode != "signup" {
		mode = "login"
	}
	props := pages.AuthPageProps{
		BaseProps: view.NewBaseProps("Login or Sign Up"),
		Mode:      mode,
	}
	if err := view.Render(w, r, http.StatusOK, pages.AuthPage(props)); err != nil {
		slog.Error("Failed to render auth page", "error", err)
	}
}

// authFeedback answers an auth form submit with the full feedback contract:
// a plain-message toast (HX-Trigger + HX-Toast-Type — the layout listener
// reads exactly these; JSON envelopes render as raw text), and an inline
// AuthMessage body that HTMX swaps into #auth-messages so the outcome stays
// visible after the toast fades.
func authFeedback(w http.ResponseWriter, r *http.Request, status int, kind, message string) {
	view.SetHXTrigger(w, message)
	w.Header().Set("HX-Toast-Type", kind)
	if err := view.Render(w, r, status, partials.AuthMessage(kind, message)); err != nil {
		slog.Error("Failed to render auth feedback", "error", err)
	}
}

// AuthLoginPost handles the login form submission. secureCookie marks the
// issued session cookies Secure (true in production, where TLS terminates at
// the edge and r.TLS is nil — ADR-025), matching the guest-session, upgrade,
// and logout flows so every cookie mutation shares one security posture.
func AuthLoginPost(authClient *auth.AuthClient, secureCookie bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			slog.Error("Failed to parse login form", "error", err)
			authFeedback(w, r, http.StatusBadRequest, "error", "Failed to process form.")
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")

		if email == "" || password == "" {
			authFeedback(w, r, http.StatusBadRequest, "error", "Email and password cannot be empty.")
			return
		}

		// gotrue-go is a server-to-server HTTP client: it authenticates against
		// GoTrue and returns the session, but it never touches this browser's
		// response — so the handler must persist the tokens as cookies itself.
		// AuthMiddleware reads sb-access-token on the next request; without this
		// the browser lands on /profile with no session and bounces to login.
		session, err := authClient.Client.SignInWithEmailPassword(email, password)
		if err != nil {
			// Email is intentionally not logged (ADR-014 §7: no PII in logs)
			slog.Warn("Supabase login failed", "error", err)
			// Provide a generic error for security
			authFeedback(w, r, http.StatusUnauthorized, "error", "Invalid login credentials.")
			return
		}

		// Same attributes as the guest-session/upgrade flows: the access token
		// lives as long as GoTrue says (ExpiresIn), the refresh token for 30
		// days so a returning user can mint a fresh access token.
		http.SetCookie(w, &http.Cookie{
			Name:     "sb-access-token",
			Value:    session.AccessToken,
			Path:     "/",
			MaxAge:   session.ExpiresIn,
			HttpOnly: true,
			Secure:   secureCookie,
			SameSite: http.SameSiteLaxMode,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "sb-refresh-token",
			Value:    session.RefreshToken,
			Path:     "/",
			MaxAge:   int((30 * 24 * time.Hour).Seconds()),
			HttpOnly: true,
			Secure:   secureCookie,
			SameSite: http.SameSiteLaxMode,
		})

		slog.Info("User login successful")
		// HTMX performs a client-side navigation on HX-Redirect; the session
		// cookies set above ride along on that request to /profile.
		view.SetHXRedirect(w, "/profile")
		w.WriteHeader(http.StatusOK)
	}
}

// AuthSignupPost handles the signup form submission.
func AuthSignupPost(authClient *auth.AuthClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			slog.Error("Failed to parse signup form", "error", err)
			authFeedback(w, r, http.StatusBadRequest, "error", "Failed to process form.")
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")

		if email == "" || password == "" {
			authFeedback(w, r, http.StatusBadRequest, "error", "Email and password cannot be empty.")
			return
		}

		// Use the Signup method from the underlying gotrue client via authClient.Client.Auth
		_, err := authClient.Client.Auth.Signup(types.SignupRequest{
			Email:    email,
			Password: password,
			// Data: map[string]interface{}{"custom_field": "value"}, // Optional: Add custom user data
		})

		if err != nil {
			slog.Warn("Supabase signup failed", "error", err)
			// Provide a more user-friendly error based on the type of Supabase error if possible
			authFeedback(w, r, http.StatusConflict, "error", "Signup failed. User might already exist or password is too weak.")
			return
		}

		slog.Info("User signup initiated")
		authFeedback(w, r, http.StatusOK, "success", "Signup successful! Please check your email to confirm your account, then sign in.")
	}
}

// AuthLogoutPost handles the logout request. secureCookie marks the cleared
// cookies Secure (true in production, where TLS terminates at the edge and
// r.TLS is nil — ADR-025), matching how the CSRF and guest-session cookies are
// issued so every cookie mutation shares one security posture.
func AuthLogoutPost(authClient *auth.AuthClient, secureCookie bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Call Logout on the underlying gotrue client.
		// It implicitly uses the auth token from the request cookies/headers.
		err := authClient.Client.Auth.Logout()
		if err != nil {
			// Log the error, but proceed with logout flow client-side anyway
			slog.Warn("Supabase signout failed", "error", err)
		}

		// Clear Supabase cookies explicitly (best practice)
		// Supabase might set these headers itself, but being explicit doesn't hurt.
		http.SetCookie(w, &http.Cookie{
			Name:     "sb-access-token",
			Value:    "",
			Path:     "/",
			MaxAge:   -1, // Expire immediately
			HttpOnly: true,
			Secure:   secureCookie,
			SameSite: http.SameSiteLaxMode,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "sb-refresh-token",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   secureCookie,
			SameSite: http.SameSiteLaxMode,
		})

		slog.Info("User logout processed")
		view.SetHXTrigger(w, `{"showToast":{"level":"success","message":"You have been logged out."}}`)
		view.SetHXRedirect(w, "/auth/page") // Redirect to login page
		w.WriteHeader(http.StatusOK)
	}
}
