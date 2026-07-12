package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpgradeAnonymousUser(t *testing.T) {
	tests := []struct {
		name             string
		status           int
		body             string
		wantErr          bool
		wantEmailInUse   bool
		wantConfirmation bool
	}{
		{
			name:   "upgrade with confirmation pending",
			status: http.StatusOK,
			body:   `{"id":"user-123","email":"","new_email":"ada@example.com","email_confirmed_at":null,"is_anonymous":false}`,
			// The email is not active until the confirmation link is clicked.
			wantConfirmation: true,
		},
		{
			name:   "upgrade with confirmations disabled",
			status: http.StatusOK,
			body:   `{"id":"user-123","email":"ada@example.com","email_confirmed_at":"2026-07-12T20:00:00Z","is_anonymous":false}`,
		},
		{
			name:           "email already registered maps to ErrEmailInUse",
			status:         http.StatusUnprocessableEntity,
			body:           `{"code":422,"msg":"A user with this email address has already been registered"}`,
			wantErr:        true,
			wantEmailInUse: true,
		},
		{
			name:    "gotrue 401 (stale session token)",
			status:  http.StatusUnauthorized,
			body:    `{"msg":"invalid JWT"}`,
			wantErr: true,
		},
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

			client := &AuthClient{baseURL: srv.URL, anonKey: "anon-key"}
			result, err := client.UpgradeAnonymousUser(context.Background(), "user-token", "ada@example.com", "hunter2hunter2")

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", result)
				}
				if got := errors.Is(err, ErrEmailInUse); got != tt.wantEmailInUse {
					t.Errorf("errors.Is(ErrEmailInUse) = %v, want %v", got, tt.wantEmailInUse)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotMethod != http.MethodPut || gotPath != "/auth/v1/user" {
				t.Errorf("request = %s %s, want PUT /auth/v1/user", gotMethod, gotPath)
			}
			if gotAuth != "Bearer user-token" {
				t.Errorf("Authorization = %q, want the user's session token", gotAuth)
			}
			if result.ConfirmationSent != tt.wantConfirmation {
				t.Errorf("ConfirmationSent = %v, want %v", result.ConfirmationSent, tt.wantConfirmation)
			}
		})
	}
}

func TestRefreshSession(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "successful refresh",
			status: http.StatusOK,
			body:   `{"access_token":"tok-new","refresh_token":"ref-new","expires_in":3600,"user":{"id":"user-123","is_anonymous":false}}`,
		},
		{
			name:    "revoked refresh token",
			status:  http.StatusBadRequest,
			body:    `{"msg":"Invalid Refresh Token"}`,
			wantErr: true,
		},
		{
			name:    "missing access token in response",
			status:  http.StatusOK,
			body:    `{"user":{"id":"user-123"}}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath, gotQuery string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotQuery = r.URL.Query().Get("grant_type")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := &AuthClient{baseURL: srv.URL, anonKey: "anon-key"}
			session, err := client.RefreshSession(context.Background(), "ref-old")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotPath != "/auth/v1/token" || gotQuery != "refresh_token" {
				t.Errorf("request = %s?grant_type=%s, want /auth/v1/token?grant_type=refresh_token", gotPath, gotQuery)
			}
			if session.AccessToken != "tok-new" || session.User.IsAnonymous {
				t.Errorf("session = %+v, want fresh non-anonymous tokens", session)
			}
		})
	}
}
