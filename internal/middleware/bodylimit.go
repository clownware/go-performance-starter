package middleware

import "net/http"

// MaxBodyBytes caps the request body at n bytes using http.MaxBytesReader, so
// oversized form or upload requests are rejected before a handler buffers them
// into memory (2026-07-06 audit). A read past the limit fails and the server
// responds 413 Request Entity Too Large.
func MaxBodyBytes(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, n)
			}
			next.ServeHTTP(w, r)
		})
	}
}
