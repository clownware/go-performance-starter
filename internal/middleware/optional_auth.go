package middleware

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/clownware/go-performance-starter/internal/auth"
	"github.com/clownware/go-performance-starter/internal/repository"
	"github.com/clownware/go-performance-starter/internal/webutil"
)

// OptionalAuth is the browse-first counterpart of AuthMiddleware: a valid
// session enriches the context exactly the same way, but an anonymous
// request continues to the handler with no identity instead of being sent
// to the login page. The /learn surfaces use it so signed-out visitors see
// a teaser that sells the sign-in rather than a blind redirect; handlers
// remain responsible for guarding mutations (nil user → redirect).
func OptionalAuth(authClient *auth.AuthClient, secureCookie bool) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// The no-cookie path never touches GoTrue, so a nil client only
			// needs guarding when a cookie is actually present.
			if c, err := r.Cookie("sb-access-token"); err != nil || c.Value == "" || authClient == nil {
				next.ServeHTTP(w, r)
				return
			}
			ctx, ok := validateSession(authClient, w, r, secureCookie)
			if !ok {
				next.ServeHTTP(w, r) // cookies cleared; continue anonymously
				return
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalUserLoader resolves the authenticated identity to its users row
// (JIT-provisioning like UserLoader) when claims are present, and passes
// anonymous requests through untouched instead of 401ing.
func OptionalUserLoader(repo repository.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := webutil.AuthClaimsFromContext(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			user, err := repo.GetByAuthID(r.Context(), claims.Sub)
			if errors.Is(err, repository.ErrNotFound) {
				user, err = provisionUser(r, repo, claims)
			}
			if err != nil {
				slog.Error("Failed to load user row", "sub", claims.Sub, "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			next.ServeHTTP(w, r.WithContext(webutil.WithUser(r.Context(), user)))
		})
	}
}
