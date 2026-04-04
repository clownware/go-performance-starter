package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		wantStatus     int
		wantBody       string
	}{
		{
			name:       "GET returns 200 OK",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			wantBody:   "OK",
		},
		{
			name:       "HEAD returns 200",
			method:     http.MethodHead,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/healthz", nil)
			w := httptest.NewRecorder()

			HealthHandler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("HealthHandler() status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantBody != "" && w.Body.String() != tt.wantBody {
				t.Errorf("HealthHandler() body = %q, want %q", w.Body.String(), tt.wantBody)
			}
		})
	}
}
