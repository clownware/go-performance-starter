package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAuthMiddleware_UnauthenticatedNavigation pins the ADR-028/Phase-A UX
// fix: a browser user navigating to a protected page without a session gets
// redirected to the login page, not a plain-text 401. The HTMX path keeps
// its 401 + HX-Redirect contract (HTMX won't follow a 3xx to swap a page).
// All rows exercise the no-cookie path, which never touches GoTrue, so a nil
// auth client is safe.
func TestAuthMiddleware_UnauthenticatedNavigation(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		htmx           bool
		emptyCookie    bool // cookie present but empty value
		wantStatus     int
		wantLocation   string
		wantHXRedirect string
	}{
		{
			name:         "browser GET redirects to login",
			method:       http.MethodGet,
			wantStatus:   http.StatusSeeOther,
			wantLocation: "/auth/page",
		},
		{
			name:         "browser POST redirects to login",
			method:       http.MethodPost,
			wantStatus:   http.StatusSeeOther,
			wantLocation: "/auth/page",
		},
		{
			name:         "empty cookie value behaves like no cookie",
			method:       http.MethodGet,
			emptyCookie:  true,
			wantStatus:   http.StatusSeeOther,
			wantLocation: "/auth/page",
		},
		{
			name:           "HTMX request keeps 401 with HX-Redirect",
			method:         http.MethodGet,
			htmx:           true,
			wantStatus:     http.StatusUnauthorized,
			wantHXRedirect: "/auth/page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("next handler must not run without a session")
			})
			h := AuthMiddleware(nil, false)(next)

			req := httptest.NewRequest(tt.method, "/learn/quiz", nil)
			if tt.emptyCookie {
				req.AddCookie(&http.Cookie{Name: "sb-access-token", Value: ""})
			}
			if tt.htmx {
				req.Header.Set("HX-Request", "true")
			}
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("%s status = %d, want %d", tt.method, rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get("Location"); got != tt.wantLocation {
				t.Errorf("Location = %q, want %q", got, tt.wantLocation)
			}
			if got := rec.Header().Get("HX-Redirect"); got != tt.wantHXRedirect {
				t.Errorf("HX-Redirect = %q, want %q", got, tt.wantHXRedirect)
			}
		})
	}
}
