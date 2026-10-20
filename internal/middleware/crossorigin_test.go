package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCrossOrigin(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		headers  map[string]string
		wantCode int
		wantLog  bool
	}{
		{"safe method from another site passes", http.MethodGet, map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusOK, false},
		{"same-origin POST passes", http.MethodPost, map[string]string{"Sec-Fetch-Site": "same-origin"}, http.StatusOK, false},
		{"user-initiated POST passes", http.MethodPost, map[string]string{"Sec-Fetch-Site": "none"}, http.StatusOK, false},
		{"headerless POST passes (non-browser client; the token layer still applies)", http.MethodPost, nil, http.StatusOK, false},
		{"cross-site POST is rejected", http.MethodPost, map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden, true},
		{"same-site POST is rejected", http.MethodPost, map[string]string{"Sec-Fetch-Site": "same-site"}, http.StatusForbidden, true},
		{"cross-site DELETE is rejected", http.MethodDelete, map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden, true},
		{"mismatched Origin without Sec-Fetch-Site is rejected", http.MethodPost, map[string]string{"Origin": "https://evil.example"}, http.StatusForbidden, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			prev := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
			t.Cleanup(func() { slog.SetDefault(prev) })

			reached := false
			h := CrossOrigin()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reached = true
			}))
			req := httptest.NewRequest(tt.method, "http://example.com/patterns/api/favorite/1", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "secret-cookie-value"})
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			if reached != (tt.wantCode == http.StatusOK) {
				t.Errorf("next handler reached = %v, want %v", reached, tt.wantCode == http.StatusOK)
			}
			logged := strings.Contains(buf.String(), "cross-origin request rejected")
			if logged != tt.wantLog {
				t.Errorf("rejection logged = %v, want %v; log: %q", logged, tt.wantLog, buf.String())
			}
			if strings.Contains(buf.String(), "secret-cookie-value") {
				t.Errorf("rejection log leaked a cookie value: %q", buf.String())
			}
		})
	}
}
