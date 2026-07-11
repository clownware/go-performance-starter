package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRealIP(t *testing.T) {
	tests := []struct {
		name       string
		trusted    []string
		remoteAddr string
		xff        string
		xRealIP    string
		want       string
	}{
		{
			name:       "no trusted proxies ignores forwarded header",
			trusted:    nil,
			remoteAddr: "203.0.113.7:5555",
			xff:        "1.2.3.4",
			want:       "203.0.113.7",
		},
		{
			name:       "trusted peer honors left-most XFF",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "10.0.0.9:443",
			xff:        "1.2.3.4, 10.0.0.9",
			want:       "1.2.3.4",
		},
		{
			name:       "untrusted peer ignores XFF",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "203.0.113.7:5555",
			xff:        "1.2.3.4",
			want:       "203.0.113.7",
		},
		{
			name:       "trusted peer falls back to X-Real-IP",
			trusted:    []string{"192.168.0.0/16"},
			remoteAddr: "192.168.1.1:80",
			xRealIP:    "8.8.8.8",
			want:       "8.8.8.8",
		},
		{
			name:       "no headers strips port to bare IP",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "10.0.0.9:443",
			want:       "10.0.0.9",
		},
		{
			name:       "malformed CIDR config trusts nothing",
			trusted:    []string{"not-a-cidr"},
			remoteAddr: "10.0.0.9:443",
			xff:        "1.2.3.4",
			want:       "10.0.0.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			h := RealIP(tt.trusted)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got = r.RemoteAddr
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			h.ServeHTTP(httptest.NewRecorder(), req)

			if got != tt.want {
				t.Errorf("RemoteAddr = %q, want %q", got, tt.want)
			}
		})
	}
}
