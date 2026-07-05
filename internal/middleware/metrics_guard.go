package middleware

import (
	"crypto/subtle"
	"net/http"
)

// MetricsGuard protects the Prometheus /metrics endpoint (2026-07-05 audit:
// it was publicly readable, leaking route names and runtime metadata).
//
// Policy: if a token is configured, require `Authorization: Bearer <token>`
// (any environment). With no token, the endpoint stays open in development
// but returns 404 in production — scraping in production requires
// METRICS_TOKEN to be set.
func MetricsGuard(token string, isProd bool) func(http.Handler) http.Handler {
	expected := []byte("Bearer " + token)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case token != "":
				got := []byte(r.Header.Get("Authorization"))
				if subtle.ConstantTimeCompare(expected, got) != 1 {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			case isProd:
				http.NotFound(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
