package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clownware/alpine-go-performance-starter/internal/auth"
)

// newFakeGoTrue stands up an httptest server faking the GoTrue REST API and
// returns an AuthClient pointed at it (the seam noted in ADR-023: the handlers
// talk to a real supabase client, which talks HTTP to the fake).
func newFakeGoTrue(t *testing.T, handler http.HandlerFunc) *auth.AuthClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client, err := auth.NewAuthClient(srv.URL, "test-anon-key")
	if err != nil {
		t.Fatalf("NewAuthClient: %v", err)
	}
	return client
}

func formRequest(target, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func TestAuthPage(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/auth/page", nil)
	w := httptest.NewRecorder()

	AuthPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("AuthPage() status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.Len() == 0 {
		t.Error("AuthPage() rendered an empty body")
	}
}

func TestAuthLoginPost(t *testing.T) {
	// user.id must be a valid UUID for gotrue-go's response decoding.
	const sessionBody = `{"access_token":"tok","refresh_token":"ref","expires_in":3600,
		"token_type":"bearer","user":{"id":"6d3f4c9a-92c8-4a2e-9b6e-0d6a3f1c2b4d"}}`

	tests := []struct {
		name         string
		form         string
		malformed    bool
		gotrueStatus int
		gotrueBody   string
		wantStatus   int
		wantRedirect string
		wantToast    string
	}{
		{
			name:       "malformed form body returns 400",
			form:       "%zz",
			malformed:  true,
			wantStatus: http.StatusBadRequest,
			wantToast:  "Failed to process form.",
		},
		{
			name:       "missing email returns 400",
			form:       "password=hunter22",
			wantStatus: http.StatusBadRequest,
			wantToast:  "Email and password cannot be empty.",
		},
		{
			name:       "missing password returns 400",
			form:       "email=a%40b.com",
			wantStatus: http.StatusBadRequest,
			wantToast:  "Email and password cannot be empty.",
		},
		{
			name:         "invalid credentials return 401 with generic error",
			form:         "email=a%40b.com&password=wrong",
			gotrueStatus: http.StatusBadRequest,
			gotrueBody:   `{"error":"invalid_grant","error_description":"Invalid login credentials"}`,
			wantStatus:   http.StatusUnauthorized,
			wantToast:    "Invalid login credentials.",
		},
		{
			name:         "successful login redirects to profile",
			form:         "email=a%40b.com&password=correct",
			gotrueStatus: http.StatusOK,
			gotrueBody:   sessionBody,
			wantStatus:   http.StatusOK,
			wantRedirect: "/profile",
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

			AuthLoginPost(client)(w, formRequest("/auth/login", tt.form))

			if w.Code != tt.wantStatus {
				t.Fatalf("AuthLoginPost() status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantRedirect != "" {
				if got := w.Header().Get("HX-Redirect"); got != tt.wantRedirect {
					t.Errorf("HX-Redirect = %q, want %q", got, tt.wantRedirect)
				}
			}
			if tt.wantToast != "" {
				if got := w.Header().Get("HX-Trigger"); !strings.Contains(got, tt.wantToast) {
					t.Errorf("HX-Trigger = %q, want it to contain %q", got, tt.wantToast)
				}
			}
			if tt.gotrueStatus != 0 && gotPath != "/auth/v1/token" {
				t.Errorf("gotrue request path = %q, want /auth/v1/token", gotPath)
			}
			// Validation failures must short-circuit before any network call.
			if tt.gotrueStatus == 0 && gotPath != "" {
				t.Errorf("gotrue was called (path %q) on a request that should fail validation", gotPath)
			}
		})
	}
}

func TestAuthSignupPost(t *testing.T) {
	tests := []struct {
		name         string
		form         string
		gotrueStatus int
		gotrueBody   string
		wantStatus   int
		wantToast    string
	}{
		{
			name:       "malformed form body returns 400",
			form:       "%zz",
			wantStatus: http.StatusBadRequest,
			wantToast:  "Failed to process form.",
		},
		{
			name:       "missing fields return 400",
			form:       "email=&password=",
			wantStatus: http.StatusBadRequest,
			wantToast:  "Email and password cannot be empty.",
		},
		{
			name:         "gotrue rejection returns 409",
			form:         "email=a%40b.com&password=short",
			gotrueStatus: http.StatusUnprocessableEntity,
			gotrueBody:   `{"msg":"User already registered"}`,
			wantStatus:   http.StatusConflict,
			wantToast:    "Signup failed.",
		},
		{
			name:         "successful signup returns 200 with success toast",
			form:         "email=new%40b.com&password=long-enough-pw",
			gotrueStatus: http.StatusOK,
			gotrueBody:   `{"id":"6d3f4c9a-92c8-4a2e-9b6e-0d6a3f1c2b4d","email":"new@b.com"}`,
			wantStatus:   http.StatusOK,
			wantToast:    "Signup successful!",
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

			AuthSignupPost(client)(w, formRequest("/auth/signup", tt.form))

			if w.Code != tt.wantStatus {
				t.Fatalf("AuthSignupPost() status = %d, want %d", w.Code, tt.wantStatus)
			}
			if got := w.Header().Get("HX-Trigger"); !strings.Contains(got, tt.wantToast) {
				t.Errorf("HX-Trigger = %q, want it to contain %q", got, tt.wantToast)
			}
			if tt.gotrueStatus != 0 && gotPath != "/auth/v1/signup" {
				t.Errorf("gotrue request path = %q, want /auth/v1/signup", gotPath)
			}
		})
	}
}

func TestAuthLogoutPost(t *testing.T) {
	tests := []struct {
		name         string
		gotrueStatus int
		secureCookie bool
	}{
		{name: "successful logout clears cookies", gotrueStatus: http.StatusNoContent},
		{name: "gotrue failure still logs out client-side", gotrueStatus: http.StatusInternalServerError},
		{name: "production marks cleared cookies Secure", gotrueStatus: http.StatusNoContent, secureCookie: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newFakeGoTrue(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.gotrueStatus)
			})
			w := httptest.NewRecorder()

			AuthLogoutPost(client, tt.secureCookie)(w, httptest.NewRequest(http.MethodPost, "/auth/logout", nil))

			if w.Code != http.StatusOK {
				t.Fatalf("AuthLogoutPost() status = %d, want %d", w.Code, http.StatusOK)
			}
			if got := w.Header().Get("HX-Redirect"); got != "/auth/page" {
				t.Errorf("HX-Redirect = %q, want /auth/page", got)
			}

			cleared := map[string]bool{}
			for _, c := range w.Result().Cookies() {
				if c.MaxAge >= 0 {
					t.Errorf("cookie %q MaxAge = %d, want negative (expired)", c.Name, c.MaxAge)
				}
				if c.Secure != tt.secureCookie {
					t.Errorf("cookie %q Secure = %v, want %v", c.Name, c.Secure, tt.secureCookie)
				}
				cleared[c.Name] = true
			}
			for _, name := range []string{"sb-access-token", "sb-refresh-token"} {
				if !cleared[name] {
					t.Errorf("cookie %q was not cleared", name)
				}
			}
		})
	}
}
