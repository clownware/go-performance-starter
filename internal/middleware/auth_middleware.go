package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/supabase-community/gotrue-go/types"

	"github.com/clownware/alpine-go-performance-starter/internal/auth"
	"github.com/clownware/alpine-go-performance-starter/internal/view"
	"github.com/clownware/alpine-go-performance-starter/internal/webutil"
)

// UserContextKey is the key used to store user information in the request context.
type UserContextKey string

const ( // Renamed key to avoid potential conflict with other string values
	ContextUserKey UserContextKey = "user"
)

// AuthMiddleware validates the JWT token from Supabase.
func AuthMiddleware(authClient *auth.AuthClient) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Get token from cookie
			cookie, err := r.Cookie("sb-access-token") // Default Supabase cookie name
			if err != nil {
				slog.Debug("Unauthorized: no access token cookie", "error", err)
				// If HTMX request, maybe return a trigger to redirect, otherwise 401
				if view.IsHTMXRequest(r) {
					view.SetHXRedirect(w, "/auth/page") // Redirect to login
					w.WriteHeader(http.StatusUnauthorized)
				} else {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
				}
				return
			}

			accessToken := cookie.Value
			if accessToken == "" {
				slog.Debug("Unauthorized: access token cookie is empty")
				if view.IsHTMXRequest(r) {
					view.SetHXRedirect(w, "/auth/page")
					w.WriteHeader(http.StatusUnauthorized)
				} else {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
				}
				return
			}

			// 2. Validate token by getting user info
			// The GetUser method implicitly validates the token.
			// We use the underlying gotrue client here.
			user, err := authClient.Client.Auth.WithToken(accessToken).GetUser()
			if err != nil {
				slog.Warn("Unauthorized: token validation failed", "error", err)
				// Clear potentially invalid cookies and redirect
				http.SetCookie(w, &http.Cookie{Name: "sb-access-token", Value: "", Path: "/", MaxAge: -1})
				http.SetCookie(w, &http.Cookie{Name: "sb-refresh-token", Value: "", Path: "/", MaxAge: -1})
				if view.IsHTMXRequest(r) {
					view.SetHXRedirect(w, "/auth/page")
					w.WriteHeader(http.StatusUnauthorized)
				} else {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
				}
				return
			}

			// 3. Add user info to context
			ctx := context.WithValue(r.Context(), ContextUserKey, user)

			// 4. Attach the validated identity as claims so the repository
			// layer applies it to database connections and RLS evaluates
			// against this user (ADR-004; see repository/postgres/scope.go).
			ctx = webutil.WithAuthClaims(ctx, webutil.AuthClaims{
				Sub:  user.ID.String(),
				Role: webutil.RoleAuthenticated,
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
