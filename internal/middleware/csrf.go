package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"log/slog"
	"net/http"

	"github.com/clownware/alpine-go-performance-starter/internal/webutil"
)

const (
	// CSRFCookieName is the double-submit cookie holding the CSRF token.
	CSRFCookieName = "csrf_token"
	// CSRFHeaderName is the request header HTMX sends via hx-headers.
	CSRFHeaderName = "X-CSRF-Token"
	// CSRFFormField is the hidden-input fallback for non-JS form posts.
	CSRFFormField = "csrf_token"

	csrfTokenBytes    = 32
	csrfCookieMaxAge  = 12 * 60 * 60 // 12h; reissued transparently on expiry
	csrfTokenHexChars = csrfTokenBytes * 2
)

// CSRF implements double-submit-cookie CSRF protection (ADR-014 §3).
//
// Every request gets a random token in an HttpOnly cookie; the server renders
// the same token into pages (via context → templ), where it travels back on
// state-changing requests as the X-CSRF-Token header (HTMX hx-headers) or the
// csrf_token form field. Unsafe methods are rejected with 403 unless the
// submitted token matches the cookie. secureCookie should be true in
// production (TLS terminates at the edge per ADR-025, so r.TLS is nil there).
func CSRF(secureCookie bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ""
			if c, err := r.Cookie(CSRFCookieName); err == nil && isValidTokenFormat(c.Value) {
				token = c.Value
			}
			if token == "" {
				buf := make([]byte, csrfTokenBytes)
				if _, err := rand.Read(buf); err != nil {
					slog.Error("CSRF token generation failed", "error", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				token = hex.EncodeToString(buf)
				http.SetCookie(w, &http.Cookie{
					Name:     CSRFCookieName,
					Value:    token,
					Path:     "/",
					MaxAge:   csrfCookieMaxAge,
					HttpOnly: true,
					Secure:   secureCookie,
					SameSite: http.SameSiteLaxMode,
				})
			}
			r = r.WithContext(webutil.WithCSRFToken(r.Context(), token))

			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
				next.ServeHTTP(w, r)
				return
			}

			// Unsafe method: the submitted token must match the cookie the
			// browser sent (not the one just issued — a first-contact POST
			// has no cookie and is correctly rejected).
			cookie, err := r.Cookie(CSRFCookieName)
			if err != nil || !isValidTokenFormat(cookie.Value) {
				http.Error(w, "Forbidden: missing CSRF cookie", http.StatusForbidden)
				return
			}
			sent := r.Header.Get(CSRFHeaderName)
			if sent == "" {
				sent = r.PostFormValue(CSRFFormField)
			}
			if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(sent)) != 1 {
				http.Error(w, "Forbidden: CSRF token mismatch", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isValidTokenFormat(token string) bool {
	if len(token) != csrfTokenHexChars {
		return false
	}
	_, err := hex.DecodeString(token)
	return err == nil
}
