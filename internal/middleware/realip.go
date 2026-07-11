package middleware

import (
	"net"
	"net/http"
	"strings"
)

// RealIP resolves the client IP into r.RemoteAddr for downstream middleware
// (rate limiting, request logging). It always normalizes RemoteAddr to a bare
// IP (stripping the port so keys are per-client, not per-connection) and, only
// when the direct peer is inside one of trustedProxyCIDRs, honors the
// X-Forwarded-For / X-Real-IP / True-Client-IP headers that peer set (ADR-027).
//
// With no trusted CIDRs it never trusts a forwarded header — a request hitting
// the app directly cannot spoof its client IP to evade the rate limiter or
// poison logs. A single trusted hop is assumed; the left-most XFF entry wins.
func RealIP(trustedProxyCIDRs []string) func(http.Handler) http.Handler {
	trusted := parseCIDRs(trustedProxyCIDRs)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := hostOnly(r.RemoteAddr)
			if forwarded := forwardedClientIP(r); forwarded != "" && peerIsTrusted(ip, trusted) {
				ip = forwarded
			}
			r.RemoteAddr = ip
			next.ServeHTTP(w, r)
		})
	}
}

// parseCIDRs converts CIDR strings to networks, skipping blanks. Entries are
// validated at boot by config.Validate (ADR-015), so a parse failure here is
// treated as "not trusted" rather than a panic.
func parseCIDRs(cidrs []string) []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, n, err := net.ParseCIDR(c); err == nil {
			nets = append(nets, n)
		}
	}
	return nets
}

// hostOnly strips the port from a host:port RemoteAddr, returning the input
// unchanged when it has no port.
func hostOnly(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr
}

func peerIsTrusted(peer string, trusted []*net.IPNet) bool {
	ip := net.ParseIP(peer)
	if ip == nil {
		return false
	}
	for _, n := range trusted {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// forwardedClientIP returns the left-most X-Forwarded-For entry, falling back
// to X-Real-IP then True-Client-IP. Callers must only trust the result when the
// direct peer is a trusted proxy (ADR-027).
func forwardedClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if first := strings.TrimSpace(strings.Split(xff, ",")[0]); first != "" {
			return first
		}
	}
	if xr := strings.TrimSpace(r.Header.Get("X-Real-IP")); xr != "" {
		return xr
	}
	if tc := strings.TrimSpace(r.Header.Get("True-Client-IP")); tc != "" {
		return tc
	}
	return ""
}
