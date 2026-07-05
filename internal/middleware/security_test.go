package middleware

import (
	"net/http"
	"net/http/httptest"
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
