package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	tests := []struct {
		name     string
		isProd   bool
		wantHSTS bool
	}{
		{"development omits HSTS", false, false},
		{"production sets HSTS", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := SecurityHeaders(tt.isProd)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

			// Baseline headers apply in every environment (ADR-014 §6).
			baseline := map[string]string{
				"X-Frame-Options":        "DENY",
				"X-Content-Type-Options": "nosniff",
				"X-XSS-Protection":       "0",
				"Referrer-Policy":        "strict-origin-when-cross-origin",
			}
			for header, want := range baseline {
				if got := rec.Header().Get(header); got != want {
					t.Errorf("%s = %q, want %q", header, got, want)
				}
			}
			if rec.Header().Get("Content-Security-Policy") == "" {
				t.Error("Content-Security-Policy missing")
			}

			hsts := rec.Header().Get("Strict-Transport-Security")
			if tt.wantHSTS && hsts == "" {
				t.Error("Strict-Transport-Security missing in production (ADR-025 §2)")
			}
			if !tt.wantHSTS && hsts != "" {
				t.Errorf("Strict-Transport-Security = %q, want unset outside production", hsts)
			}
		})
	}
}

// TestSecurityHeaders_CSPAllowsAlpineEval pins the ADR-028 decision: Alpine 3
// evaluates x-data/x-show expressions with the Function constructor, so
// script-src must include 'unsafe-eval' or every Alpine behavior in the app
// (dark mode, menus, toasts, tabs) fails silently. Inline scripts stay
// forbidden — script-src must NOT gain 'unsafe-inline'.
func TestSecurityHeaders_CSPAllowsAlpineEval(t *testing.T) {
	h := SecurityHeaders(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	csp := rec.Header().Get("Content-Security-Policy")
	scriptSrc := ""
	for _, directive := range strings.Split(csp, ";") {
		if d := strings.TrimSpace(directive); strings.HasPrefix(d, "script-src ") {
			scriptSrc = d
		}
	}
	if scriptSrc == "" {
		t.Fatalf("CSP %q has no script-src directive", csp)
	}
	if !strings.Contains(scriptSrc, "'unsafe-eval'") {
		t.Errorf("script-src = %q, must include 'unsafe-eval' (Alpine expression engine, ADR-028)", scriptSrc)
	}
	if strings.Contains(scriptSrc, "'unsafe-inline'") {
		t.Errorf("script-src = %q, must NOT include 'unsafe-inline' (ADR-028 keeps inline scripts forbidden)", scriptSrc)
	}
}
