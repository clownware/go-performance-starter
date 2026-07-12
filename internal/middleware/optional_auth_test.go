package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clownware/go-performance-starter/internal/webutil"
)

// TestOptionalAuth_AnonymousPassthrough pins the browse-first contract for
// the /learn surfaces: without a session the request continues to the
// handler with no user identity in context (the handler renders a teaser),
// instead of being redirected the way AuthMiddleware does. The no-cookie
// path never touches GoTrue, so a nil client is safe.
func TestOptionalAuth_AnonymousPassthrough(t *testing.T) {
	tests := []struct {
		name        string
		emptyCookie bool
		htmx        bool
	}{
		{name: "no cookie continues anonymously"},
		{name: "empty cookie value continues anonymously", emptyCookie: true},
		{name: "HTMX request continues anonymously too", htmx: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextRan := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextRan = true
				if _, ok := webutil.AuthClaimsFromContext(r.Context()); ok {
					t.Error("anonymous request must not carry auth claims")
				}
				if _, ok := GetUserFromContext(r.Context()); ok {
					t.Error("anonymous request must not carry a gotrue user")
				}
				w.WriteHeader(http.StatusOK)
			})
			h := OptionalAuth(nil, false)(next)

			req := httptest.NewRequest(http.MethodGet, "/learn/quiz", nil)
			if tt.emptyCookie {
				req.AddCookie(&http.Cookie{Name: "sb-access-token", Value: ""})
			}
			if tt.htmx {
				req.Header.Set("HX-Request", "true")
			}
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			if !nextRan {
				t.Fatal("next handler did not run — OptionalAuth must pass anonymous requests through")
			}
			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want 200 (no redirect, no 401)", rec.Code)
			}
		})
	}
}

// TestOptionalUserLoader_AnonymousPassthrough: without claims the loader
// passes through rather than 401ing the way UserLoader does.
func TestOptionalUserLoader_AnonymousPassthrough(t *testing.T) {
	nextRan := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextRan = true
		if webutil.GetUserFromContext(r.Context()) != nil {
			t.Error("anonymous request must not carry a users row")
		}
		w.WriteHeader(http.StatusOK)
	})
	h := OptionalUserLoader(nil)(next)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/learn/quiz", nil))

	if !nextRan {
		t.Fatal("next handler did not run — OptionalUserLoader must pass anonymous requests through")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}
