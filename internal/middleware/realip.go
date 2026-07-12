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
// poison logs. X-Forwarded-For is resolved right-to-left, skipping trusted
// proxy addresses: edge proxies (Fly, Cloudflare) APPEND the peer they saw to
// any client-supplied value, so the left-most entry is attacker-controlled
// and the right-most untrusted entry is the real client (ADR-027, amended
// 2026-07-12).
//
// clientIPHeader (CLIENT_IP_HEADER) optionally names the edge's authoritative
// client-IP header — Fly-Client-IP, CF-Connecting-IP — consulted first, still
// only from a trusted peer. XFF alone cannot recover the client when the
// edge's own hop IPs aren't enumerable as trusted CIDRs (Fly's anycast edge
// ranges are undocumented), which live verification on Fly confirmed.
func RealIP(trustedProxyCIDRs []string, clientIPHeader string) func(http.Handler) http.Handler {
	trusted := parseCIDRs(trustedProxyCIDRs)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := hostOnly(r.RemoteAddr)
			if peerIsTrusted(ip, trusted) {
				if configured := configuredClientIP(r, clientIPHeader); configured != "" {
					ip = configured
				} else if forwarded := forwardedClientIP(r, trusted); forwarded != "" {
					ip = forwarded
				}
			}
			r.RemoteAddr = ip
			next.ServeHTTP(w, r)
		})
	}
}

// configuredClientIP returns the value of the operator-configured client-IP
// header when it parses as an IP, "" otherwise (fall back to XFF resolution).
func configuredClientIP(r *http.Request, header string) string {
	if header == "" {
		return ""
	}
	v := strings.TrimSpace(r.Header.Get(header))
	if v == "" || net.ParseIP(v) == nil {
		return ""
	}
	return v
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

// forwardedClientIP resolves X-Forwarded-For right-to-left — skipping trusted
// proxy addresses and invalid entries — and returns the first remaining IP:
// the nearest hop that is not our own infrastructure. An XFF consisting only
// of trusted/invalid entries yields "" (callers fall back to the direct peer,
// failing closed). Falls back to X-Real-IP then True-Client-IP, which are
// single-valued headers set by the trusted peer. Callers must only trust the
// result when the direct peer is a trusted proxy (ADR-027).
func forwardedClientIP(r *http.Request, trusted []*net.IPNet) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		entries := strings.Split(xff, ",")
		for i := len(entries) - 1; i >= 0; i-- {
			entry := strings.TrimSpace(entries[i])
			if net.ParseIP(entry) == nil {
				continue
			}
			if !peerIsTrusted(entry, trusted) {
				return entry
			}
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
