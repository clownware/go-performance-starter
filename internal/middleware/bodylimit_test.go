package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaxBodyBytes(t *testing.T) {
	tests := []struct {
		name       string
		limit      int64
		bodyLen    int
		wantStatus int
	}{
		{"body under limit is read", 100, 50, http.StatusOK},
		{"body at limit is read", 100, 100, http.StatusOK},
		{"body over limit is rejected", 100, 101, http.StatusRequestEntityTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := MaxBodyBytes(tt.limit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, err := io.ReadAll(r.Body); err != nil {
					http.Error(w, "too large", http.StatusRequestEntityTooLarge)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("a", tt.bodyLen)))
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
