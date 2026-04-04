package middleware

import "net/http"

// SecurityHeaders adds security-related HTTP headers to all responses.
// Per ADR-014 Security Patterns and Threat Model.
//
// CSP note: 'unsafe-inline' is required for style-src because Tailwind CSS
// generates utility classes that may be applied via style attributes.
// script-src is locked to 'self' only — HTMX and Alpine.js are loaded as
// script files, not inline scripts, so unsafe-inline is not needed for scripts.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// X-XSS-Protection set to 0 per OWASP recommendation — CSP supersedes it,
		// and non-zero values can introduce XSS vulnerabilities in older browsers.
		w.Header().Set("X-XSS-Protection", "0")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data:; "+
				"font-src 'self'; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'")
		next.ServeHTTP(w, r)
	})
}
