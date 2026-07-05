package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetricsGuard(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		isProd     bool
		authHeader string
		wantStatus int
	}{
		{"dev, no token configured: open", "", false, "", http.StatusOK},
		{"prod, no token configured: hidden", "", true, "", http.StatusNotFound},
		{"token configured, correct bearer", "s3cret", true, "Bearer s3cret", http.StatusOK},
		{"token configured, wrong bearer", "s3cret", true, "Bearer nope", http.StatusUnauthorized},
		{"token configured, missing header", "s3cret", false, "", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := MetricsGuard(tt.token, tt.isProd)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
