package auth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifyRecovery(t *testing.T) {
	tests := []struct {
		name      string
		tokenHash string
		status    int
		body      string
		wantErr   bool
		wantSub   string
	}{
		{
			name:      "valid token hash returns a session",
			tokenHash: "pkce-hash-abc",
			status:    http.StatusOK,
			body: `{"access_token":"tok-rec","refresh_token":"ref-rec","expires_in":3600,
				"user":{"id":"user-456","is_anonymous":false}}`,
			wantSub: "user-456",
		},
		{
			name:      "expired or invalid token",
			tokenHash: "stale-hash",
			status:    http.StatusForbidden,
			body:      `{"error":"access_denied","error_description":"Email link is invalid or has expired"}`,
			wantErr:   true,
		},
		{
			name:      "empty token hash short-circuits without a network call",
			tokenHash: "",
			wantErr:   true,
		},
		{
			name:      "missing access token in response",
			tokenHash: "hash-no-token",
			status:    http.StatusOK,
			body:      `{"user":{"id":"user-456"}}`,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			var gotBody map[string]string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				raw, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(raw, &gotBody)
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := &AuthClient{baseURL: srv.URL, anonKey: "anon-key"}
			session, err := client.VerifyRecovery(context.Background(), tt.tokenHash)

			if tt.wantErr {
				if err == nil {
					t.Fatal("VerifyRecovery() = nil error, want error")
				}
				if tt.tokenHash == "" && gotPath != "" {
					t.Errorf("empty token hash reached GoTrue (path %q); must short-circuit", gotPath)
				}
				return
			}
			if err != nil {
				t.Fatalf("VerifyRecovery() error = %v", err)
			}
			if gotPath != "/auth/v1/verify" {
				t.Errorf("gotrue path = %q, want /auth/v1/verify", gotPath)
			}
			if gotBody["type"] != "recovery" {
				t.Errorf(`verify body type = %q, want "recovery"`, gotBody["type"])
			}
			if gotBody["token_hash"] != tt.tokenHash {
				t.Errorf("verify body token_hash = %q, want %q", gotBody["token_hash"], tt.tokenHash)
			}
			if session.User.ID != tt.wantSub {
				t.Errorf("session user id = %q, want %q", session.User.ID, tt.wantSub)
			}
			if session.AccessToken == "" {
				t.Error("session access token is empty")
			}
		})
	}
}
