package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiter(t *testing.T) {
	tests := []struct {
		name         string
		rps          float64
		burst        int
		requests     int
		remoteAddrs  []string
		wantStatuses []int
	}{
		{
			name:  "same IP exceeding burst is limited",
			rps:   0.001, // effectively no refill during the test
			burst: 2,
			remoteAddrs: []string{
				"10.0.0.1:1234", "10.0.0.1:1234", "10.0.0.1:1234",
			},
			wantStatuses: []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests},
		},
		{
			name:  "distinct IPs get independent buckets",
			rps:   0.001,
			burst: 1,
			remoteAddrs: []string{
				"10.0.0.1:1234", "10.0.0.2:1234", "10.0.0.1:1234",
			},
			wantStatuses: []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := RateLimiter(tt.rps, tt.burst)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			for i, addr := range tt.remoteAddrs {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.RemoteAddr = addr
				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, req)

				if rec.Code != tt.wantStatuses[i] {
					t.Errorf("request %d from %s: status = %d, want %d", i+1, addr, rec.Code, tt.wantStatuses[i])
				}
			}
		})
	}
}
