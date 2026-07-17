package handler

import (
	"log/slog"
	"net/http"

	"github.com/supabase-community/gotrue-go/types"

	"github.com/clownware/go-performance-starter/internal/auth"
	"github.com/clownware/go-performance-starter/internal/view"
	"github.com/clownware/go-performance-starter/internal/view/pages"
)

// recoverGenericMessage is returned for EVERY recover outcome that passes
// client-side validation — success, unknown email, and GoTrue rate limits
// alike — so the endpoint cannot be used to enumerate registered emails.
const recoverGenericMessage = "If that email has an account, a reset link is on its way."

// RecoverPage renders the request-a-reset-link form.
func RecoverPage(w http.ResponseWriter, r *http.Request) {
	props := pages.RecoverPageProps{
		BaseProps: view.NewBaseProps("Reset your password"),
	}
	if err := view.Render(w, r, http.StatusOK, pages.RecoverPage(props)); err != nil {
		slog.Error("Failed to render recover page", "error", err)
	}
}

// AuthRecoverPost asks GoTrue to send the password-reset email.
func AuthRecoverPost(authClient *auth.AuthClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			slog.Error("Failed to parse recover form", "error", err)
			authFeedback(w, r, http.StatusBadRequest, "error", "Failed to process form.")
			return
		}
		email := r.FormValue("email")
		if email == "" {
			authFeedback(w, r, http.StatusBadRequest, "error", "Email cannot be empty.")
			return
		}

		if err := authClient.Client.Auth.Recover(types.RecoverRequest{Email: email}); err != nil {
			// Email is intentionally not logged (ADR-014 §7: no PII in logs).
			// The client still gets the generic success below — a distinct
			// error here would leak which addresses have accounts.
			slog.Warn("Supabase recover failed", "error", err)
		}
		authFeedback(w, r, http.StatusOK, "success", recoverGenericMessage)
	}
}

// AuthResetPage exchanges the email link's token_hash for a recovery session
// and shows the update-password form. The token_hash travels in the query
// string (the email template links to /auth/reset?token_hash={{ .TokenHash }}
// &type=recovery), so the whole exchange happens server-side — no JS reading
// URL fragments. An invalid or expired link renders a state that points back
// to /auth/recover and sets no cookies.
func AuthResetPage(authClient *auth.AuthClient, secureCookie bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		props := pages.ResetPageProps{
			BaseProps: view.NewBaseProps("Choose a new password"),
		}

		tokenHash := r.URL.Query().Get("token_hash")
		if tokenHash != "" {
			session, err := authClient.VerifyRecovery(r.Context(), tokenHash)
			if err != nil {
				slog.Warn("Recovery token verification failed", "error", err)
			} else {
				// The recovery session is a real session: issue the same
				// cookie pair as login so the reset POST (and the user's
				// onward navigation) is authenticated.
				setSessionCookies(w, session, secureCookie)
				props.Valid = true
			}
		}

		if err := view.Render(w, r, http.StatusOK, pages.ResetPage(props)); err != nil {
			slog.Error("Failed to render reset page", "error", err)
		}
	}
}

// AuthResetPost sets the new password using the recovery session issued by
// AuthResetPage.
func AuthResetPost(authClient *auth.AuthClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("sb-access-token")
		if err != nil || cookie.Value == "" {
			authFeedback(w, r, http.StatusUnauthorized, "error", "Your reset link has expired. Request a new one.")
			return
		}

		if err := r.ParseForm(); err != nil {
			slog.Error("Failed to parse reset form", "error", err)
			authFeedback(w, r, http.StatusBadRequest, "error", "Failed to process form.")
			return
		}
		password := r.FormValue("password")
		confirm := r.FormValue("password_confirm")
		if password == "" {
			authFeedback(w, r, http.StatusBadRequest, "error", "Password cannot be empty.")
			return
		}
		if password != confirm {
			authFeedback(w, r, http.StatusBadRequest, "error", "Passwords do not match.")
			return
		}

		_, err = authClient.Client.Auth.WithToken(cookie.Value).UpdateUser(types.UpdateUserRequest{
			Password: &password,
		})
		if err != nil {
			slog.Warn("Supabase password update failed", "error", err)
			// GoTrue's own policy (e.g. minimum length) is the authority;
			// surface a retryable error without echoing its internals.
			authFeedback(w, r, http.StatusUnprocessableEntity, "error", "Could not update password. It may be too short — try a longer one.")
			return
		}

		slog.Info("Password reset completed")
		view.SetHXTrigger(w, "Your password has been updated.")
		w.Header().Set("HX-Toast-Type", "success")
		view.SetHXRedirect(w, "/profile")
		w.WriteHeader(http.StatusOK)
	}
}
