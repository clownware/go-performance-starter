package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/supabase-community/gotrue-go/types"

	"github.com/clownware/go-performance-starter/internal/auth"
	"github.com/clownware/go-performance-starter/internal/view"
	"github.com/clownware/go-performance-starter/internal/webutil"
)

// UserContextKey is the key used to store user information in the request context.
type UserContextKey string

const ( // Renamed key to avoid potential conflict with other string values
	ContextUserKey UserContextKey = "user"
)

// toLogin sends an unauthenticated requester to the login page. Browsers get
// a real redirect (a plain-text 401 is a dead end for a person following a
// link); HTMX requests get the 401 + HX-Redirect contract instead, because
// HTMX treats a 3xx as a fragment to swap, not a navigation (ADR-028 review).
func toLogin(w http.ResponseWriter, r *http.Request) {
	if view.IsHTMXRequest(r) {
		view.SetHXRedirect(w, "/auth/page")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, "/auth/page", http.StatusSeeOther)
}

// AuthMiddleware validates the JWT token from Supabase. secureCookie marks the
// cookies it clears on a failed validation Secure, matching how the login and
// CSRF cookies are issued (ADR-025: TLS terminates at the edge, so r.TLS is nil
// in production and cannot be used to decide the flag).
func AuthMiddleware(authClient *auth.AuthClient, secureCookie bool) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Get token from cookie
			cookie, err := r.Cookie("sb-access-token") // Default Supabase cookie name
			if err != nil || cookie.Value == "" {
				slog.Debug("Unauthenticated: no access token cookie")
				toLogin(w, r)
				return
			}
			accessToken := cookie.Value

			// 2. Validate token by getting user info
			// The GetUser method implicitly validates the token.
			// We use the underlying gotrue client here.
			user, err := authClient.Client.Auth.WithToken(accessToken).GetUser()
			if err != nil {
				slog.Warn("Unauthenticated: token validation failed", "error", err)
				// Clear potentially invalid cookies and redirect. Mirror the
				// full attribute set used when issuing them so the browser
				// reliably matches and drops the cookie.
				http.SetCookie(w, &http.Cookie{Name: "sb-access-token", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: secureCookie, SameSite: http.SameSiteLaxMode})
				http.SetCookie(w, &http.Cookie{Name: "sb-refresh-token", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: secureCookie, SameSite: http.SameSiteLaxMode})
				toLogin(w, r)
				return
			}

			// 3. Add user info to context
			ctx := context.WithValue(r.Context(), ContextUserKey, user)

			// 4. Attach the validated identity as claims so the repository
			// layer applies it to database connections and RLS evaluates
			// against this user (ADR-004; see repository/postgres/scope.go).
			// is_anonymous comes from the token payload — safe to decode
			// unverified here because GetUser above validated the token.
			_, isAnonymous, err := auth.TokenClaims(accessToken)
			if err != nil {
				slog.Warn("Failed to parse token claims; treating session as non-anonymous", "error", err)
			}
			ctx = webutil.WithAuthClaims(ctx, webutil.AuthClaims{
				Sub:         user.ID.String(),
				Role:        webutil.RoleAuthenticated,
				IsAnonymous: isAnonymous,
			})

			// 5. Call next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserFromContext retrieves the user information from the request context.
func GetUserFromContext(ctx context.Context) (*types.UserResponse, bool) {
	user, ok := ctx.Value(ContextUserKey).(*types.UserResponse)
	return user, ok
}
