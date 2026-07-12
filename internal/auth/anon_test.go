package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignInAnonymously(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantErr    bool
		wantSub    string
		checkAnon  bool
		wantIsAnon bool
	}{
		{
			name:   "successful anonymous session",
			status: http.StatusOK,
			body: `{"access_token":"tok-abc","refresh_token":"ref-abc","expires_in":3600,
				"user":{"id":"user-123","is_anonymous":true}}`,
			wantSub:    "user-123",
			checkAnon:  true,
			wantIsAnon: true,
		},
		{
			name:    "gotrue error (anonymous sign-ins disabled)",
			status:  http.StatusUnprocessableEntity,
			body:    `{"msg":"Anonymous sign-ins are disabled"}`,
			wantErr: true,
		},
		{
			name:    "missing access token",
			status:  http.StatusOK,
			body:    `{"user":{"id":"user-123"}}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath, gotAPIKey string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotAPIKey = r.Header.Get("apikey")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := &AuthClient{baseURL: srv.URL, anonKey: "anon-key"}
			session, err := client.SignInAnonymously(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Fatal("SignInAnonymously() = nil error, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("SignInAnonymously() error: %v", err)
			}
			if gotPath != "/auth/v1/signup" {
				t.Errorf("request path = %q, want /auth/v1/signup", gotPath)
			}
			if gotAPIKey != "anon-key" {
				t.Errorf("apikey header = %q, want anon-key", gotAPIKey)
			}
			if session.User.ID != tt.wantSub {
				t.Errorf("user id = %q, want %q", session.User.ID, tt.wantSub)
			}
			if tt.checkAnon && session.User.IsAnonymous != tt.wantIsAnon {
				t.Errorf("is_anonymous = %v, want %v", session.User.IsAnonymous, tt.wantIsAnon)
			}
		})
	}
}

func TestSignInAnonymously_Unconfigured(t *testing.T) {
	client := &AuthClient{}
	if _, err := client.SignInAnonymously(context.Background()); err == nil {
		t.Error("unconfigured client should error, not panic or call out")
	}
}

func TestServiceRoleKey(t *testing.T) {
	client := &AuthClient{}
	if client.HasServiceRoleKey() {
		t.Error("HasServiceRoleKey() = true before a key is attached")
	}
	if got := client.WithServiceRoleKey("sr-key"); got != client {
		t.Error("WithServiceRoleKey() must return the same client for chaining")
	}
	if !client.HasServiceRoleKey() {
		t.Error("HasServiceRoleKey() = false after attaching a key")
	}
}

func TestAdminDeleteUser(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{"200 deletes", http.StatusOK, `{}`, false},
		{"204 deletes", http.StatusNoContent, "", false},
		{"404 is tolerated (already gone, reaper retry-safe)", http.StatusNotFound, `{"msg":"User not found"}`, false},
		{"403 is an error (bad service role key)", http.StatusForbidden, `{"msg":"forbidden"}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod, gotPath, gotAuth string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotPath = r.URL.Path
				gotAuth = r.Header.Get("Authorization")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := (&AuthClient{baseURL: srv.URL, anonKey: "anon-key"}).WithServiceRoleKey("sr-key")
			err := client.AdminDeleteUser(context.Background(), "user-123")

			if tt.wantErr {
				if err == nil {
					t.Fatal("AdminDeleteUser() = nil error, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("AdminDeleteUser() error: %v", err)
			}
			if gotMethod != http.MethodDelete {
				t.Errorf("method = %q, want DELETE", gotMethod)
			}
			if gotPath != "/auth/v1/admin/users/user-123" {
				t.Errorf("path = %q, want /auth/v1/admin/users/user-123", gotPath)
			}
			if gotAuth != "Bearer sr-key" {
				t.Errorf("Authorization = %q, want the service role key, never the anon key", gotAuth)
			}
		})
	}
}

func TestAdminDeleteUser_NoServiceRoleKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("admin delete without a service role key must not call GoTrue")
	}))
	defer srv.Close()

	client := &AuthClient{baseURL: srv.URL, anonKey: "anon-key"}
	if err := client.AdminDeleteUser(context.Background(), "user-123"); err == nil {
		t.Error("AdminDeleteUser() without service role key = nil error, want error")
	}
}

// makeJWT builds an unsigned JWT with the given payload for claim parsing tests.
func makeJWT(t *testing.T, payload map[string]any) string {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	head := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
	return head + "." + base64.RawURLEncoding.EncodeToString(b) + ".sig"
}

func TestTokenClaims(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		wantSub  string
		wantAnon bool
		wantErr  bool
	}{
		{
			name:     "anonymous token",
			token:    "", // filled below
			wantSub:  "abc-123",
			wantAnon: true,
		},
		{
			name:    "not a jwt",
			token:   "just-an-opaque-string",
			wantErr: true,
		},
		{
			name:    "garbage payload",
			token:   "a.!!!.c",
			wantErr: true,
		},
	}
	tests[0].token = makeJWT(t, map[string]any{"sub": "abc-123", "is_anonymous": true})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub, isAnon, err := TokenClaims(tt.token)
			if tt.wantErr {
				if err == nil {
					t.Fatal("TokenClaims() = nil error, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("TokenClaims() error: %v", err)
			}
			if sub != tt.wantSub || isAnon != tt.wantAnon {
				t.Errorf("TokenClaims() = (%q, %v), want (%q, %v)", sub, isAnon, tt.wantSub, tt.wantAnon)
			}
		})
	}
}
