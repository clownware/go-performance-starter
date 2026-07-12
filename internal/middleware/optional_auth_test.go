package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

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

// TestOptionalAuth_ValidSession pins the half of OptionalAuth the anonymous
// tests can't: a valid cookie must actually be validated and enrich the
// context — a regression that silently skips validation would demote every
// signed-in /learn visitor to the teaser view.
func TestOptionalAuth_ValidSession(t *testing.T) {
	userID := uuid.New()
	token := sessionJWT(t, "sub-1", true)
	authClient := fakeGoTrue(t, userID, map[string]bool{token: true})

	nextRan := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextRan = true
		user, ok := GetUserFromContext(r.Context())
		if !ok || user == nil || user.ID != userID {
			t.Error("valid session must put the gotrue user in context")
		}
		if _, ok := webutil.AuthClaimsFromContext(r.Context()); !ok {
			t.Error("valid session must carry RLS claims (ADR-004)")
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/learn", nil)
	req.AddCookie(&http.Cookie{Name: "sb-access-token", Value: token})
	rec := httptest.NewRecorder()

	OptionalAuth(authClient, false)(next).ServeHTTP(rec, req)

	if !nextRan {
		t.Fatalf("next handler did not run; status = %d", rec.Code)
	}
}

// TestOptionalAuth_RejectedTokenContinuesAnonymously pins the degrade path:
// a stale cookie is cleared but the page still renders, without identity.
func TestOptionalAuth_RejectedTokenContinuesAnonymously(t *testing.T) {
	authClient := fakeGoTrue(t, uuid.New(), map[string]bool{}) // accepts nothing

	nextRan := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextRan = true
		if _, ok := GetUserFromContext(r.Context()); ok {
			t.Error("rejected token must not leave a user in context")
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/learn", nil)
	req.AddCookie(&http.Cookie{Name: "sb-access-token", Value: "stale"})
	rec := httptest.NewRecorder()

	OptionalAuth(authClient, false)(next).ServeHTTP(rec, req)

	if !nextRan {
		t.Fatal("rejected token must degrade to anonymous, not block the page")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}
