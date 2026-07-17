package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clownware/go-performance-starter/internal/auth"
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

// TestAuthPage pins the tabbed auth card: one panel active at a time (login
// by default), the other reachable both by Alpine tab switch and by a plain
// ?mode link so the page works without JS. Both forms are always in the
// markup — "hidden" is a server-rendered class the client enhancement
// toggles — and the #auth-messages target the forms post into must exist.
func TestAuthPage(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		wantHidden  string // panel that must render with the hidden class
		wantVisible string // panel that must NOT be hidden
	}{
		{
			name:        "defaults to the login tab",
			target:      "/auth/page",
			wantHidden:  "panel-signup",
			wantVisible: "panel-login",
		},
		{
			name:        "mode=signup activates the signup tab without JS",
			target:      "/auth/page?mode=signup",
			wantHidden:  "panel-login",
			wantVisible: "panel-signup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			w := httptest.NewRecorder()

			AuthPage(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("AuthPage() status = %d, want %d", w.Code, http.StatusOK)
			}
			body := w.Body.String()

			// Both forms ship in the markup regardless of active tab.
			for _, want := range []string{`hx-post="/auth/login"`, `hx-post="/auth/signup"`, `id="auth-messages"`} {
				if !strings.Contains(body, want) {
					t.Errorf("auth page missing %q", want)
				}
			}
			// Tab links carry the no-JS fallback.
			if !strings.Contains(body, `href="/auth/page?mode=signup"`) || !strings.Contains(body, `href="/auth/page?mode=login"`) {
				t.Error("auth page tab links missing the ?mode fallback hrefs")
			}
			if !strings.Contains(body, `id="`+tt.wantHidden+`" class="hidden"`) {
				t.Errorf("panel %s should render hidden on %s", tt.wantHidden, tt.target)
			}
			if strings.Contains(body, `id="`+tt.wantVisible+`" class="hidden"`) {
				t.Errorf("panel %s should be visible on %s", tt.wantVisible, tt.target)
			}
		})
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
		wantSession  bool // asserts sb-access-token/sb-refresh-token cookies are issued
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
			wantSession:  true,
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

			AuthLoginPost(client, false)(w, formRequest("/auth/login", tt.form))

			if w.Code != tt.wantStatus {
				t.Fatalf("AuthLoginPost() status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantRedirect != "" {
				if got := w.Header().Get("HX-Redirect"); got != tt.wantRedirect {
					t.Errorf("HX-Redirect = %q, want %q", got, tt.wantRedirect)
				}
			}
			// A successful login must persist the session as cookies — without
			// them AuthMiddleware rejects /profile and bounces the user straight
			// back to the login form (the "spinner then flash back" regression).
			cookies := map[string]*http.Cookie{}
			for _, c := range w.Result().Cookies() {
				cookies[c.Name] = c
			}
			if tt.wantSession {
				access, ok := cookies["sb-access-token"]
				if !ok {
					t.Fatal("successful login did not set sb-access-token cookie")
				}
				if access.Value != "tok" {
					t.Errorf("sb-access-token = %q, want %q", access.Value, "tok")
				}
				if !access.HttpOnly {
					t.Error("sb-access-token must be HttpOnly")
				}
				if access.MaxAge != 3600 {
					t.Errorf("sb-access-token MaxAge = %d, want 3600 (session expires_in)", access.MaxAge)
				}
				refresh, ok := cookies["sb-refresh-token"]
				if !ok {
					t.Fatal("successful login did not set sb-refresh-token cookie")
				}
				if refresh.Value != "ref" {
					t.Errorf("sb-refresh-token = %q, want %q", refresh.Value, "ref")
				}
				if !refresh.HttpOnly {
					t.Error("sb-refresh-token must be HttpOnly")
				}
			} else if _, ok := cookies["sb-access-token"]; ok {
				t.Error("failed login must not set a session cookie")
			}
			if tt.wantToast != "" {
				// The layout's toast listener treats HX-Trigger as the plain
				// message and HX-Toast-Type as the level — no JSON envelopes
				// (that mismatch rendered raw JSON in a success-green toast).
				got := w.Header().Get("HX-Trigger")
				if !strings.Contains(got, tt.wantToast) {
					t.Errorf("HX-Trigger = %q, want it to contain %q", got, tt.wantToast)
				}
				if strings.Contains(got, "{") {
					t.Errorf("HX-Trigger = %q must be a plain message, not JSON", got)
				}
				if w.Header().Get("HX-Toast-Type") != "error" {
					t.Errorf("HX-Toast-Type = %q, want %q", w.Header().Get("HX-Toast-Type"), "error")
				}
				// The form swaps the response into #auth-messages, so failures
				// must also carry a visible inline alert — a 4-second toast is
				// not the only feedback.
				if !strings.Contains(w.Body.String(), tt.wantToast) {
					t.Errorf("response body missing inline alert %q", tt.wantToast)
				}
				if !strings.Contains(w.Body.String(), `role="alert"`) {
					t.Error("inline auth feedback missing role=\"alert\"")
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
			// Plain-message toast contract (no JSON envelope) with the level
			// in HX-Toast-Type, and a visible inline message in the body the
			// form swaps into #auth-messages — signup previously returned an
			// empty body and a mis-typed JSON toast, so "nothing happened".
			got := w.Header().Get("HX-Trigger")
			if !strings.Contains(got, tt.wantToast) {
				t.Errorf("HX-Trigger = %q, want it to contain %q", got, tt.wantToast)
			}
			if strings.Contains(got, "{") {
				t.Errorf("HX-Trigger = %q must be a plain message, not JSON", got)
			}
			wantType := "error"
			wantRole := `role="alert"`
			if tt.wantStatus == http.StatusOK {
				wantType = "success"
				wantRole = `role="status"`
			}
			if w.Header().Get("HX-Toast-Type") != wantType {
				t.Errorf("HX-Toast-Type = %q, want %q", w.Header().Get("HX-Toast-Type"), wantType)
			}
			if !strings.Contains(w.Body.String(), tt.wantToast) {
				t.Errorf("response body missing inline message %q", tt.wantToast)
			}
			if !strings.Contains(w.Body.String(), wantRole) {
				t.Errorf("inline auth feedback missing %s", wantRole)
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
