package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestAuthRecoverPost pins the anti-enumeration contract: every outcome that
// isn't a client-side validation failure produces the SAME generic response,
// so the endpoint cannot be used to probe which emails have accounts.
func TestAuthRecoverPost(t *testing.T) {
	const generic = "If that email has an account, a reset link is on its way."

	tests := []struct {
		name         string
		form         string
		gotrueStatus int // 0 = GoTrue must not be called
		gotrueBody   string
		wantStatus   int
		wantToast    string
		wantKind     string
	}{
		{
			name:       "missing email returns 400",
			form:       "email=",
			wantStatus: http.StatusBadRequest,
			wantToast:  "Email cannot be empty.",
			wantKind:   "error",
		},
		{
			name:         "unknown email still reports success (no enumeration)",
			form:         "email=nobody%40example.com",
			gotrueStatus: http.StatusBadRequest,
			gotrueBody:   `{"error":"user_not_found"}`,
			wantStatus:   http.StatusOK,
			wantToast:    generic,
			wantKind:     "success",
		},
		{
			name:         "gotrue rate limit still reports success (no enumeration)",
			form:         "email=known%40example.com",
			gotrueStatus: http.StatusTooManyRequests,
			gotrueBody:   `{"error":"over_email_send_rate_limit"}`,
			wantStatus:   http.StatusOK,
			wantToast:    generic,
			wantKind:     "success",
		},
		{
			name:         "known email sends and reports the same success",
			form:         "email=known%40example.com",
			gotrueStatus: http.StatusOK,
			gotrueBody:   `{}`,
			wantStatus:   http.StatusOK,
			wantToast:    generic,
			wantKind:     "success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			client := newFakeGoTrue(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.WriteHeader(tt.gotrueStatus)
				_, _ = w.Write([]byte(tt.gotrueBody))
			})
			w := httptest.NewRecorder()

			AuthRecoverPost(client)(w, formRequest("/auth/recover", tt.form))

			if w.Code != tt.wantStatus {
				t.Fatalf("AuthRecoverPost() status = %d, want %d", w.Code, tt.wantStatus)
			}
			if got := w.Header().Get("HX-Trigger"); !strings.Contains(got, tt.wantToast) {
				t.Errorf("HX-Trigger = %q, want it to contain %q", got, tt.wantToast)
			}
			if got := w.Header().Get("HX-Toast-Type"); got != tt.wantKind {
				t.Errorf("HX-Toast-Type = %q, want %q", got, tt.wantKind)
			}
			if !strings.Contains(w.Body.String(), tt.wantToast) {
				t.Errorf("response body missing inline message %q", tt.wantToast)
			}
			if tt.gotrueStatus != 0 && gotPath != "/auth/v1/recover" {
				t.Errorf("gotrue path = %q, want /auth/v1/recover", gotPath)
			}
			if tt.gotrueStatus == 0 && gotPath != "" {
				t.Errorf("gotrue was called (path %q) on a request that should fail validation", gotPath)
			}
		})
	}
}

// TestAuthResetPage pins the token_hash exchange: a valid hash becomes a
// session (cookies set, update form rendered); anything else renders the
// invalid-link state with a path back to /auth/recover and NO cookies.
func TestAuthResetPage(t *testing.T) {
	const sessionBody = `{"access_token":"rec-tok","refresh_token":"rec-ref","expires_in":3600,
		"user":{"id":"6d3f4c9a-92c8-4a2e-9b6e-0d6a3f1c2b4d","is_anonymous":false}}`

	tests := []struct {
		name         string
		target       string
		gotrueStatus int // 0 = GoTrue must not be called
		gotrueBody   string
		wantSession  bool
		wantForm     bool // the update-password form is present
	}{
		{
			name:        "missing token_hash renders invalid-link state without calling gotrue",
			target:      "/auth/reset",
			wantSession: false,
			wantForm:    false,
		},
		{
			name:         "expired token renders invalid-link state",
			target:       "/auth/reset?token_hash=stale&type=recovery",
			gotrueStatus: http.StatusForbidden,
			gotrueBody:   `{"error":"access_denied"}`,
			wantSession:  false,
			wantForm:     false,
		},
		{
			name:         "valid token sets session cookies and shows the form",
			target:       "/auth/reset?token_hash=good&type=recovery",
			gotrueStatus: http.StatusOK,
			gotrueBody:   sessionBody,
			wantSession:  true,
			wantForm:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			client := newFakeGoTrue(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.WriteHeader(tt.gotrueStatus)
				_, _ = w.Write([]byte(tt.gotrueBody))
			})
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)

			AuthResetPage(client, false)(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("AuthResetPage() status = %d, want 200", w.Code)
			}
			cookies := map[string]*http.Cookie{}
			for _, c := range w.Result().Cookies() {
				cookies[c.Name] = c
			}
			if tt.wantSession {
				access, ok := cookies["sb-access-token"]
				if !ok {
					t.Fatal("valid recovery did not set sb-access-token")
				}
				if access.Value != "rec-tok" || !access.HttpOnly {
					t.Errorf("sb-access-token = %q HttpOnly=%v, want rec-tok/HttpOnly", access.Value, access.HttpOnly)
				}
				if _, ok := cookies["sb-refresh-token"]; !ok {
					t.Error("valid recovery did not set sb-refresh-token")
				}
			} else if _, ok := cookies["sb-access-token"]; ok {
				t.Error("invalid recovery must not set session cookies")
			}
			body := w.Body.String()
			if tt.wantForm {
				if !strings.Contains(body, `hx-post="/auth/reset"`) {
					t.Error("update-password form missing from valid-token page")
				}
				if gotPath != "/auth/v1/verify" {
					t.Errorf("gotrue path = %q, want /auth/v1/verify", gotPath)
				}
			} else {
				if strings.Contains(body, `hx-post="/auth/reset"`) {
					t.Error("update-password form must not render for an invalid link")
				}
				if !strings.Contains(body, "/auth/recover") {
					t.Error("invalid-link state must link back to /auth/recover")
				}
			}
			if tt.gotrueStatus == 0 && gotPath != "" {
				t.Errorf("gotrue was called (path %q) without a token_hash", gotPath)
			}
		})
	}
}

// TestAuthResetPost pins the password update: it requires the recovery
// session cookie, validates the password pair, and on success redirects to
// /profile via the HTMX contract.
func TestAuthResetPost(t *testing.T) {
	tests := []struct {
		name         string
		cookie       string // sb-access-token value; "" = absent
		form         string
		gotrueStatus int // 0 = GoTrue must not be called
		gotrueBody   string
		wantStatus   int
		wantRedirect string
		wantToast    string
	}{
		{
			name:       "no session cookie returns 401 with expired-link guidance",
			cookie:     "",
			form:       "password=newpass123&password_confirm=newpass123",
			wantStatus: http.StatusUnauthorized,
			wantToast:  "Your reset link has expired. Request a new one.",
		},
		{
			name:       "empty password returns 400",
			cookie:     "rec-tok",
			form:       "password=&password_confirm=",
			wantStatus: http.StatusBadRequest,
			wantToast:  "Password cannot be empty.",
		},
		{
			name:       "mismatched confirmation returns 400",
			cookie:     "rec-tok",
			form:       "password=newpass123&password_confirm=different",
			wantStatus: http.StatusBadRequest,
			wantToast:  "Passwords do not match.",
		},
		{
			name:         "gotrue rejects weak password",
			cookie:       "rec-tok",
			form:         "password=short&password_confirm=short",
			gotrueStatus: http.StatusUnprocessableEntity,
			gotrueBody:   `{"error_description":"Password should be at least 6 characters"}`,
			wantStatus:   http.StatusUnprocessableEntity,
			wantToast:    "Could not update password",
		},
		{
			name:         "success updates password and redirects to profile",
			cookie:       "rec-tok",
			form:         "password=newpass123&password_confirm=newpass123",
			gotrueStatus: http.StatusOK,
			gotrueBody:   `{"id":"6d3f4c9a-92c8-4a2e-9b6e-0d6a3f1c2b4d","email":"a@b.com"}`,
			wantStatus:   http.StatusOK,
			wantRedirect: "/profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath, gotAuth string
			client := newFakeGoTrue(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotAuth = r.Header.Get("Authorization")
				w.WriteHeader(tt.gotrueStatus)
				_, _ = w.Write([]byte(tt.gotrueBody))
			})
			w := httptest.NewRecorder()
			req := formRequest("/auth/reset", tt.form)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "sb-access-token", Value: tt.cookie})
			}

			AuthResetPost(client)(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("AuthResetPost() status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantRedirect != "" {
				if got := w.Header().Get("HX-Redirect"); got != tt.wantRedirect {
					t.Errorf("HX-Redirect = %q, want %q", got, tt.wantRedirect)
				}
				if gotPath != "/auth/v1/user" {
					t.Errorf("gotrue path = %q, want /auth/v1/user", gotPath)
				}
				if !strings.Contains(gotAuth, tt.cookie) {
					t.Errorf("Authorization = %q, want bearer %q (the recovery session)", gotAuth, tt.cookie)
				}
			}
			if tt.wantToast != "" {
				if got := w.Header().Get("HX-Trigger"); !strings.Contains(got, tt.wantToast) {
					t.Errorf("HX-Trigger = %q, want it to contain %q", got, tt.wantToast)
				}
				if !strings.Contains(w.Body.String(), tt.wantToast) {
					t.Errorf("response body missing inline message %q", tt.wantToast)
				}
			}
			if tt.gotrueStatus == 0 && gotPath != "" {
				t.Errorf("gotrue was called (path %q) on a request that should fail validation", gotPath)
			}
		})
	}
}
