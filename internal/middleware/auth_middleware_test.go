package middleware

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/clownware/go-performance-starter/internal/auth"
	"github.com/clownware/go-performance-starter/internal/webutil"
)

// sessionJWT builds an unsigned JWT carrying the claims validateSession reads.
// The signature is never checked here — GetUser against the fake GoTrue is
// what validates the token (ADR-024 gating).
func sessionJWT(t *testing.T, sub string, isAnonymous bool) string {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"sub": sub, "is_anonymous": isAnonymous})
	if err != nil {
		t.Fatal(err)
	}
	head := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
	return head + "." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}

// fakeGoTrue stands in for Supabase's GET /auth/v1/user token validation.
// Any token in validTokens gets the user JSON back; everything else gets 401.
func fakeGoTrue(t *testing.T, userID uuid.UUID, validTokens map[string]bool) *auth.AuthClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/v1/user" {
			t.Errorf("unexpected GoTrue call: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		token, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !validTokens[token] {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"msg":"invalid token"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"id":%q,"aud":"authenticated","email":"guest@example.com"}`, userID)
	}))
	t.Cleanup(srv.Close)

	client, err := auth.NewAuthClient(srv.URL, "anon-key")
	if err != nil {
		t.Fatalf("NewAuthClient: %v", err)
	}
	return client
}

// TestAuthMiddleware_ValidSession pins the success path: a token GoTrue
// accepts lets the request through with the gotrue user and the RLS claims
// (ADR-004) in context, including the is_anonymous claim ADR-024 gates on.
func TestAuthMiddleware_ValidSession(t *testing.T) {
	tests := []struct {
		name     string
		token    func(t *testing.T) string
		wantAnon bool
	}{
		{
			name:     "registered user session",
			token:    func(t *testing.T) string { return sessionJWT(t, "sub-1", false) },
			wantAnon: false,
		},
		{
			name:     "anonymous guest session carries is_anonymous",
			token:    func(t *testing.T) string { return sessionJWT(t, "sub-1", true) },
			wantAnon: true,
		},
		{
			name: "unparseable claims degrade to non-anonymous, not a rejection",
			token: func(t *testing.T) string {
				return "opaque-but-gotrue-accepts-it"
			},
			wantAnon: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := uuid.New()
			token := tt.token(t)
			authClient := fakeGoTrue(t, userID, map[string]bool{token: true})

			nextRan := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextRan = true
				user, ok := GetUserFromContext(r.Context())
				if !ok || user == nil {
					t.Fatal("gotrue user missing from context on a valid session")
				}
				if user.ID != userID {
					t.Errorf("context user id = %v, want %v", user.ID, userID)
				}
				claims, ok := webutil.AuthClaimsFromContext(r.Context())
				if !ok {
					t.Fatal("auth claims missing from context — RLS would evaluate as anon (ADR-004)")
				}
				if claims.Sub != userID.String() {
					t.Errorf("claims.Sub = %q, want %q (must come from the validated GetUser response)", claims.Sub, userID)
				}
				if claims.Role != webutil.RoleAuthenticated {
					t.Errorf("claims.Role = %q, want %q", claims.Role, webutil.RoleAuthenticated)
				}
				if claims.IsAnonymous != tt.wantAnon {
					t.Errorf("claims.IsAnonymous = %v, want %v", claims.IsAnonymous, tt.wantAnon)
				}
			})

			req := httptest.NewRequest(http.MethodGet, "/learn/quiz", nil)
			req.AddCookie(&http.Cookie{Name: "sb-access-token", Value: token})
			rec := httptest.NewRecorder()

			AuthMiddleware(authClient, false)(next).ServeHTTP(rec, req)

			if !nextRan {
				t.Fatalf("next handler did not run; status = %d", rec.Code)
			}
			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want 200", rec.Code)
			}
		})
	}
}

// TestAuthMiddleware_RejectedToken pins the invalid-cookie path: GoTrue says
// no, so both session cookies are cleared (with the same attribute set they
// were issued with, per ADR-025) and the browser goes back to login.
func TestAuthMiddleware_RejectedToken(t *testing.T) {
	authClient := fakeGoTrue(t, uuid.New(), map[string]bool{}) // accepts nothing

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler must not run with a rejected token")
	})

	req := httptest.NewRequest(http.MethodGet, "/learn/quiz", nil)
	req.AddCookie(&http.Cookie{Name: "sb-access-token", Value: "stale-token"})
	rec := httptest.NewRecorder()

	AuthMiddleware(authClient, true)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want 303 redirect to login", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/auth/page" {
		t.Errorf("Location = %q, want /auth/page", got)
	}

	cleared := map[string]bool{}
	for _, c := range rec.Result().Cookies() {
		if c.MaxAge >= 0 {
			t.Errorf("cookie %s not cleared (MaxAge = %d, want -1)", c.Name, c.MaxAge)
		}
		if !c.HttpOnly || !c.Secure {
			t.Errorf("cleared cookie %s must keep HttpOnly+Secure so the browser drops the original", c.Name)
		}
		cleared[c.Name] = true
	}
	for _, name := range []string{"sb-access-token", "sb-refresh-token"} {
		if !cleared[name] {
			t.Errorf("cookie %s was not cleared", name)
		}
	}
}

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

// TestAuthMiddleware_ClaimParseLogging pins the observability contract around
// TokenClaims: a token GoTrue accepts but whose claims don't parse must warn
// (it's the only signal RLS is running without is_anonymous), and a clean
// parse must not cry wolf.
func TestAuthMiddleware_ClaimParseLogging(t *testing.T) {
	var mu sync.Mutex
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(writerFunc(func(p []byte) (int, error) {
		mu.Lock()
		defer mu.Unlock()
		return buf.Write(p)
	}), nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	logged := func() string { mu.Lock(); defer mu.Unlock(); return buf.String() }

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	serve := func(token string) {
		authClient := fakeGoTrue(t, uuid.New(), map[string]bool{token: true})
		req := httptest.NewRequest(http.MethodGet, "/learn/quiz", nil)
		req.AddCookie(&http.Cookie{Name: "sb-access-token", Value: token})
		AuthMiddleware(authClient, false)(next).ServeHTTP(httptest.NewRecorder(), req)
	}

	serve(sessionJWT(t, "sub-1", false))
	if strings.Contains(logged(), "Failed to parse token claims") {
		t.Error("clean claims must not log a parse warning")
	}

	serve("opaque-token")
	if !strings.Contains(logged(), "Failed to parse token claims") {
		t.Error("unparseable claims must be logged")
	}
}

type writerFunc func(p []byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }
