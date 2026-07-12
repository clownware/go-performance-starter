package middleware

import (
	"net/http"

	"github.com/clownware/go-performance-starter/internal/repository"
	"github.com/clownware/go-performance-starter/internal/webutil"
)

// UserRepoMiddleware injects the user repository into the request context for downstream handlers.
func UserRepoMiddleware(repo repository.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := webutil.WithUserRepo(r.Context(), repo)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
