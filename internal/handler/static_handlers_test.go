package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestStaticPages smoke-tests the render-only pages: 200, HTML, and the page
// title present — enough to catch a props/render regression without coupling
// to markup.
func TestStaticPages(t *testing.T) {
	tests := []struct {
		name      string
		handler   http.HandlerFunc
		wantTitle string
	}{
		{"dashboard", DashboardPage, "Dashboard"},
		{"terms", TermsPage, "Terms of Service"},
		{"privacy", PrivacyPage, "Privacy Policy"},
		{"logout confirmation", LogoutPage, "Sign Out"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()

			tt.handler(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
				t.Errorf("Content-Type = %q, want text/html", ct)
			}
			if !strings.Contains(rec.Body.String(), tt.wantTitle) {
				t.Errorf("body does not contain page title %q", tt.wantTitle)
			}
		})
	}
}

func TestLogoutPage_NonGETRedirectsHome(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	rec := httptest.NewRecorder()

	LogoutPage(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/" {
		t.Errorf("Location = %q, want /", got)
	}
}

// TestHealthDetailHandler covers the ADR-013 detail probe without a database:
// "not configured" is a healthy state (auth-less demo boot), not degraded.
func TestHealthDetailHandler_NoDatabase(t *testing.T) {
	InitHealth(nil)

	req := httptest.NewRequest(http.MethodGet, "/health/detail", nil)
	rec := httptest.NewRecorder()

	HealthDetailHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp struct {
		Status string            `json:"status"`
		Uptime string            `json:"uptime"`
		Checks map[string]string `json:"checks"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("status = %q, want ok", resp.Status)
	}
	if resp.Checks["database"] != "not configured" {
		t.Errorf("checks.database = %q, want %q", resp.Checks["database"], "not configured")
	}
	if resp.Uptime == "" {
		t.Error("uptime missing — InitHealth did not record a start time")
	}
}

// TestFirstRunHandlers_RoutesRegistered proves the onboarding routes are
// mounted; sessionless requests get the login redirect, not a 404.
func TestFirstRunHandlers_RoutesRegistered(t *testing.T) {
	router := chi.NewRouter()
	FirstRunHandlers(router)

	for _, path := range []string{"/first-run", "/first-run/profile", "/first-run/ctas"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusSeeOther {
			t.Errorf("GET %s = %d, want 303 to login (route missing?)", path, rec.Code)
		}
	}
}
