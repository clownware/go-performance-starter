package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clownware/go-performance-starter/internal/config"
	mw "github.com/clownware/go-performance-starter/internal/middleware"
)

func newTestServer(t *testing.T, env string) *Server {
	t.Helper()
	cfg := &config.Config{
		Env:         env,
		HTTPPort:    "4000",
		DatabaseURL: "postgres://localhost:5432/test",
		DBMaxConns:  25,
	}
	srv, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return srv
}

// TestServer_CSRFWiring proves the CSRF middleware is wired into the router,
// not just unit-correct: unsafe requests without a token are rejected and
// pages issue the cookie.
func TestServer_CSRFWiring(t *testing.T) {
	srv := newTestServer(t, "development")

	// GET issues the CSRF cookie.
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", rec.Code)
	}
	var token string
	for _, c := range rec.Result().Cookies() {
		if c.Name == mw.CSRFCookieName {
			token = c.Value
		}
	}
	if token == "" {
		t.Fatal("GET / did not set the CSRF cookie")
	}

	// POST without a token is rejected.
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/patterns/api/toast", nil))
	if rec.Code != http.StatusForbidden {
		t.Errorf("POST without CSRF token: status = %d, want 403", rec.Code)
	}

	// POST with cookie + matching header passes CSRF.
	req := httptest.NewRequest(http.MethodPost, "/patterns/api/toast", nil)
	req.AddCookie(&http.Cookie{Name: mw.CSRFCookieName, Value: token})
	req.Header.Set(mw.CSRFHeaderName, token)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code == http.StatusForbidden {
		t.Errorf("POST with valid CSRF token: status = %d, want non-403", rec.Code)
	}
}

// TestServer_StubDemosRetired proves the pre-ADR-024 stub surfaces are gone
// and the replacement demo is routed: the in-memory Items demo and the
// hardcoded JSON API served fake data ("Stub User", org1) that under-sold the
// stack, and ADR-024 Slice C retires them.
func TestServer_StubDemosRetired(t *testing.T) {
	srv := newTestServer(t, "development")

	tests := []struct {
		name       string
		method     string
		target     string
		wantStatus int
	}{
		{"items page retired", http.MethodGet, "/items", http.StatusNotFound},
		{"items list retired", http.MethodGet, "/items/list", http.StatusNotFound},
		{"stub user API retired", http.MethodGet, "/api/users/123", http.StatusNotFound},
		{"stub organizations API retired", http.MethodGet, "/api/organizations", http.StatusNotFound},
		{"API placeholder retired", http.MethodGet, "/api", http.StatusNotFound},
		{"patterns showcase replaces them", http.MethodGet, "/patterns", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.target, nil))
			if rec.Code != tt.wantStatus {
				t.Errorf("%s %s status = %d, want %d", tt.method, tt.target, rec.Code, tt.wantStatus)
			}
		})
	}
}

// TestServer_MetricsGating proves /metrics visibility follows environment.
func TestServer_MetricsGating(t *testing.T) {
	tests := []struct {
		name       string
		env        string
		wantStatus int
	}{
		{"open in development", "development", http.StatusOK},
		{"hidden in production without token", "production", http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestServer(t, tt.env)
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
			if rec.Code != tt.wantStatus {
				t.Errorf("GET /metrics status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

// TestServer_HSTSGating proves the HSTS header follows environment (ADR-025 §2).
func TestServer_HSTSGating(t *testing.T) {
	tests := []struct {
		env      string
		wantHSTS bool
	}{
		{"development", false},
		{"production", true},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			srv := newTestServer(t, tt.env)
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
			got := rec.Header().Get("Strict-Transport-Security") != ""
			if got != tt.wantHSTS {
				t.Errorf("ENV=%s: HSTS present = %v, want %v", tt.env, got, tt.wantHSTS)
			}
		})
	}
}
