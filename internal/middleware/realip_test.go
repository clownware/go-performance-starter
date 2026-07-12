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
		ipHeader   string            // CLIENT_IP_HEADER config value
		headers    map[string]string // extra request headers
		remoteAddr string
		xff        string
		xRealIP    string
		want       string
	}{
		{
			name: "configured client-IP header wins over XFF from a trusted peer",
			// Fly appends its own edge IP to XFF, so the right-most
			// untrusted entry is the edge, not the client — Fly-Client-IP
			// is the authoritative value (ADR-027 amendment, issue #72).
			trusted:    []string{"172.16.0.0/12"},
			ipHeader:   "Fly-Client-IP",
			headers:    map[string]string{"Fly-Client-IP": "72.251.219.37"},
			remoteAddr: "172.19.9.185:39154",
			xff:        "72.251.219.37, 66.241.125.15",
			want:       "72.251.219.37",
		},
		{
			name:       "configured header from an untrusted peer is ignored",
			trusted:    []string{"172.16.0.0/12"},
			ipHeader:   "Fly-Client-IP",
			headers:    map[string]string{"Fly-Client-IP": "6.6.6.6"},
			remoteAddr: "203.0.113.7:5555",
			want:       "203.0.113.7",
		},
		{
			name:       "missing configured header falls back to XFF resolution",
			trusted:    []string{"172.16.0.0/12"},
			ipHeader:   "Fly-Client-IP",
			remoteAddr: "172.19.9.185:39154",
			xff:        "203.0.113.9",
			want:       "203.0.113.9",
		},
		{
			name:       "garbage in configured header falls back to XFF resolution",
			trusted:    []string{"172.16.0.0/12"},
			ipHeader:   "Fly-Client-IP",
			headers:    map[string]string{"Fly-Client-IP": "not-an-ip"},
			remoteAddr: "172.19.9.185:39154",
			xff:        "203.0.113.9",
			want:       "203.0.113.9",
		},
		{
			name:       "no trusted proxies ignores forwarded header",
			trusted:    nil,
			remoteAddr: "203.0.113.7:5555",
			xff:        "1.2.3.4",
			want:       "203.0.113.7",
		},
		{
			name:       "trusted peer resolves the right-most untrusted XFF entry",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "10.0.0.9:443",
			xff:        "1.2.3.4, 10.0.0.9",
			want:       "1.2.3.4",
		},
		{
			name: "client-spoofed XFF prefix is ignored behind an appending proxy",
			// Fly/Cloudflare APPEND the real client to any inbound XFF, so
			// the left-most entry is attacker-controlled: the client sent
			// "6.6.6.6" and the proxy appended the true 203.0.113.9.
			trusted:    []string{"172.16.0.0/12"},
			remoteAddr: "172.19.9.185:39154",
			xff:        "6.6.6.6, 203.0.113.9",
			want:       "203.0.113.9",
		},
		{
			name:       "multi-hop chain skips trusted proxies right-to-left",
			trusted:    []string{"172.16.0.0/12", "10.0.0.0/8"},
			remoteAddr: "172.19.9.185:39154",
			xff:        "203.0.113.9, 10.0.0.4",
			want:       "203.0.113.9",
		},
		{
			name:       "XFF of only trusted addresses falls back to the peer",
			trusted:    []string{"172.16.0.0/12"},
			remoteAddr: "172.19.9.185:39154",
			xff:        "172.19.0.1, 172.20.0.2",
			want:       "172.19.9.185",
		},
		{
			name:       "garbage XFF entries are skipped",
			trusted:    []string{"172.16.0.0/12"},
			remoteAddr: "172.19.9.185:39154",
			xff:        "203.0.113.9, not-an-ip",
			want:       "203.0.113.9",
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
			name:       "trusted peer falls back to True-Client-IP",
			trusted:    []string{"192.168.0.0/16"},
			remoteAddr: "192.168.1.1:80",
			headers:    map[string]string{"True-Client-IP": "9.9.9.9"},
			want:       "9.9.9.9",
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
			h := RealIP(tt.trusted, tt.ipHeader)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
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
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			h.ServeHTTP(httptest.NewRecorder(), req)

			if got != tt.want {
				t.Errorf("RemoteAddr = %q, want %q", got, tt.want)
			}
		})
	}
}
