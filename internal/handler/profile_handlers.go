package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/supabase-community/gotrue-go/types"

	"github.com/clownware/go-performance-starter/internal/auth"
	"github.com/clownware/go-performance-starter/internal/middleware"
	"github.com/clownware/go-performance-starter/internal/view"
	"github.com/clownware/go-performance-starter/internal/view/pages"
	"github.com/clownware/go-performance-starter/internal/view/partials"
	"github.com/clownware/go-performance-starter/internal/webutil"
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

// profileDisplayName prefers the persisted users row (the write target of
// ProfileUpdate, #70) and falls back to token metadata for identities whose
// row has no name yet.
func profileDisplayName(r *http.Request) string {
	if u := webutil.GetUserFromContext(r.Context()); u != nil && u.Name.Valid && u.Name.String != "" {
		return u.Name.String
	}
	return userNameFromContext(r)
}

// profilePageProps assembles the page props from the two identities the
// protected group puts in context: the GoTrue user (account email) and the
// users row (persisted name, is_anonymous). Guests have no email or password
// in GoTrue, so the page offers the upgrade flow instead (#97).
func profilePageProps(r *http.Request) pages.ProfilePageProps {
	userName := profileDisplayName(r)
	baseProps := view.NewBaseProps("Profile")
	baseProps.UserName = userName
	props := pages.ProfilePageProps{
		BaseProps: baseProps,
		Name:      userName,
	}
	if authUser, ok := middleware.GetUserFromContext(r.Context()); ok && authUser != nil {
		props.Email = authUser.Email
	}
	if dbUser := webutil.GetUserFromContext(r.Context()); dbUser != nil && dbUser.IsAnonymous {
		props.IsGuest = true
	}
	return props
}

// ProfileView renders the profile page (full page or fragment fallback).
func ProfileView(w http.ResponseWriter, r *http.Request) {
	if err := view.Render(w, r, http.StatusOK, pages.ProfilePage(profilePageProps(r))); err != nil {
		slog.Error("Failed to render profile page", "error", err)
	}
}

// ProfileUpdate processes the profile form submission with HTMX support.
func ProfileUpdate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		JSONError(w, http.StatusBadRequest, err)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	errors := make(map[string]string)
	if name == "" {
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
			pageProps := profilePageProps(r)
			pageProps.Name = name
			pageProps.Errors = errors
			if err := view.Render(w, r, http.StatusUnprocessableEntity, pages.ProfilePage(pageProps)); err != nil {
				slog.Error("Failed to render profile page", "error", err)
			}
		}
		return
	}

	// Persist to the users row — UserLoader put it (and UserRepoMiddleware
	// the repo) in context; without a row there is nothing to update.
	user := webutil.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/page", http.StatusSeeOther)
		return
	}
	repo := webutil.GetUserRepoFromContext(r.Context())
	if repo == nil {
		slog.Error("Profile update without a user repository in context")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if _, err := repo.UpdateName(r.Context(), user.ID, name); err != nil {
		slog.Error("Failed to persist profile name", "user_id", user.ID, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

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

// ProfilePasswordUpdate changes the signed-in user's password (#97). It is
// the authenticated twin of AuthResetPost: the session's access token is
// the credential and GoTrue's UpdateUser sets the password on it — no
// current-password check, matching GoTrue's own semantics (a deployment
// that wants re-authentication enables GoTrue's secure-password-change
// setting, which turns this into the nonce flow). Guests are refused: an
// anonymous identity has no password to change; the upgrade flow sets one.
func ProfilePasswordUpdate(authClient *auth.AuthClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := webutil.GetUserFromContext(r.Context())
		if user == nil {
			http.Redirect(w, r, "/auth/page", http.StatusSeeOther)
			return
		}
		cookie, err := r.Cookie("sb-access-token")
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/auth/page", http.StatusSeeOther)
			return
		}
		if user.IsAnonymous {
			renderPasswordForm(w, r, http.StatusForbidden, partials.PasswordFormProps{
				Errors: map[string]string{"password": "Guest accounts have no password — create an account first to set one."},
			})
			return
		}
		if err := r.ParseForm(); err != nil {
			JSONError(w, http.StatusBadRequest, err)
			return
		}
		password := r.FormValue("password")
		confirm := r.FormValue("password_confirm")
		errors := map[string]string{}
		if password == "" {
			errors["password"] = "Password cannot be empty."
		} else if password != confirm {
			errors["password_confirm"] = "Passwords do not match."
		}
		if len(errors) > 0 {
			renderPasswordForm(w, r, http.StatusUnprocessableEntity, partials.PasswordFormProps{Errors: errors})
			return
		}

		if _, err := authClient.Client.Auth.WithToken(cookie.Value).UpdateUser(types.UpdateUserRequest{Password: &password}); err != nil {
			// GoTrue's own policy (minimum length, reuse rules) is the
			// authority; surface a retryable error without echoing internals.
			slog.Warn("Supabase password update failed", "user_id", user.ID, "error", err)
			renderPasswordForm(w, r, http.StatusUnprocessableEntity, partials.PasswordFormProps{
				Errors: map[string]string{"password": "Could not update password. It may be too short — try a longer one."},
			})
			return
		}
		slog.Info("Password changed from profile", "user_id", user.ID)

		if view.IsHTMXRequest(r) {
			view.SetHXTrigger(w, "Password updated.")
			w.Header().Set("HX-Toast-Type", "success")
			renderPasswordForm(w, r, http.StatusOK, partials.PasswordFormProps{Success: true})
			return
		}
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
	}
}

// renderPasswordForm answers a change-password submission: the bare partial
// for HTMX (swapped into #password-form), the whole page otherwise, so a
// plain form post still shows its errors in place.
func renderPasswordForm(w http.ResponseWriter, r *http.Request, status int, props partials.PasswordFormProps) {
	if view.IsHTMXRequest(r) {
		if err := view.Render(w, r, status, partials.PasswordForm(props)); err != nil {
			slog.Error("Failed to render password form partial", "error", err)
		}
		return
	}
	pageProps := profilePageProps(r)
	pageProps.PasswordErrors = props.Errors
	pageProps.PasswordSuccess = props.Success
	if err := view.Render(w, r, status, pages.ProfilePage(pageProps)); err != nil {
		slog.Error("Failed to render profile page", "error", err)
	}
}
