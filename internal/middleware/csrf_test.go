package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/clownware/go-performance-starter/internal/webutil"
)

// okHandler records that it ran and exposes the CSRF token from context.
func csrfTestHandler(t *testing.T, gotToken *string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*gotToken = webutil.CSRFTokenFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
}

func csrfCookieFrom(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == CSRFCookieName {
			return c
		}
	}
	return nil
}

func TestCSRF_GetIssuesCookieAndContextToken(t *testing.T) {
	var gotToken string
	h := CSRF(false)(csrfTestHandler(t, &gotToken))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", rec.Code)
	}
	cookie := csrfCookieFrom(t, rec)
	if cookie == nil {
		t.Fatal("GET did not set CSRF cookie")
	}
	if !cookie.HttpOnly {
		t.Error("CSRF cookie must be HttpOnly")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("CSRF cookie SameSite = %v, want Lax", cookie.SameSite)
	}
	if gotToken == "" || gotToken != cookie.Value {
		t.Errorf("context token = %q, want cookie value %q", gotToken, cookie.Value)
	}
}

func TestCSRF_SecureCookieInProduction(t *testing.T) {
	var gotToken string
	h := CSRF(true)(csrfTestHandler(t, &gotToken))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	cookie := csrfCookieFrom(t, rec)
	if cookie == nil {
		t.Fatal("GET did not set CSRF cookie")
	}
	if !cookie.Secure {
		t.Error("CSRF cookie must be Secure when production")
	}
}

func TestCSRF_UnsafeMethods(t *testing.T) {
	const token = "a1b2c3d4e5f6a7b8a1b2c3d4e5f6a7b8a1b2c3d4e5f6a7b8a1b2c3d4e5f6a7b8"

	tests := []struct {
		name       string
		method     string
		cookie     string
		header     string
		formField  string
		wantStatus int
	}{
		{"POST with no token", http.MethodPost, token, "", "", http.StatusForbidden},
		{"POST with matching header", http.MethodPost, token, token, "", http.StatusOK},
		{"POST with matching form field", http.MethodPost, token, "", token, http.StatusOK},
		{"POST with mismatched header", http.MethodPost, token, "wrong-token", "", http.StatusForbidden},
		{"POST with no cookie at all", http.MethodPost, "", token, "", http.StatusForbidden},
		{"DELETE with matching header", http.MethodDelete, token, token, "", http.StatusOK},
		{"PUT with mismatched header", http.MethodPut, token, "nope", "", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotToken string
			h := CSRF(false)(csrfTestHandler(t, &gotToken))

			var body *strings.Reader
			if tt.formField != "" {
				body = strings.NewReader(url.Values{CSRFFormField: {tt.formField}}.Encode())
			} else {
				body = strings.NewReader("")
			}
			req := httptest.NewRequest(tt.method, "/submit", body)
			if tt.formField != "" {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: tt.cookie})
			}
			if tt.header != "" {
				req.Header.Set(CSRFHeaderName, tt.header)
			}

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestCSRF_ExistingCookieReused(t *testing.T) {
	const token = "a1b2c3d4e5f6a7b8a1b2c3d4e5f6a7b8a1b2c3d4e5f6a7b8a1b2c3d4e5f6a7b8"

	var gotToken string
	h := CSRF(false)(csrfTestHandler(t, &gotToken))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: token})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if c := csrfCookieFrom(t, rec); c != nil {
		t.Errorf("valid existing cookie must not be reissued, got new cookie %q", c.Value)
	}
	if gotToken != token {
		t.Errorf("context token = %q, want existing cookie value", gotToken)
	}
}
