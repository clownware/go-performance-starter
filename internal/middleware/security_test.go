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

// TestSecurityHeaders_CSPStrictScriptSrc pins ADR-036 (superseding ADR-028):
// Alpine ships as the @alpinejs/csp build, whose expression parser needs no
// eval, so script-src is 'self' alone — no 'unsafe-eval', and inline scripts
// stay forbidden (no 'unsafe-inline').
func TestSecurityHeaders_CSPStrictScriptSrc(t *testing.T) {
	h := SecurityHeaders(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	csp := rec.Header().Get("Content-Security-Policy")
	scriptSrc := ""
	for directive := range strings.SplitSeq(csp, ";") {
		if d := strings.TrimSpace(directive); strings.HasPrefix(d, "script-src ") {
			scriptSrc = d
		}
	}
	if scriptSrc == "" {
		t.Fatalf("CSP %q has no script-src directive", csp)
	}
	if scriptSrc != "script-src 'self'" {
		t.Errorf("script-src = %q, want exactly \"script-src 'self'\" (ADR-036: no 'unsafe-eval', no 'unsafe-inline')", scriptSrc)
	}
}
