package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AnonSession is the GoTrue session returned by an anonymous sign-in.
type AnonSession struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	User         struct {
		ID          string `json:"id"`
		IsAnonymous bool   `json:"is_anonymous"`
	} `json:"user"`
}

// anonHTTPClient bounds the server-side GoTrue call so a slow auth service
// cannot hold guest page loads indefinitely.
var anonHTTPClient = &http.Client{Timeout: 10 * time.Second}

// SignInAnonymously creates a real but anonymous Supabase identity via the
// GoTrue REST endpoint — a credential-less POST /auth/v1/signup, the same
// call supabase-js signInAnonymously() makes (ADR-024; gotrue-go v1.2.1 has
// no anonymous method). Requires "anonymous sign-ins" enabled in Supabase.
func (a *AuthClient) SignInAnonymously(ctx context.Context) (*AnonSession, error) {
	if a.baseURL == "" || a.anonKey == "" {
		return nil, fmt.Errorf("anonymous sign-in: auth client not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(a.baseURL, "/")+"/auth/v1/signup", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		return nil, fmt.Errorf("anonymous sign-in: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", a.anonKey)
	req.Header.Set("Authorization", "Bearer "+a.anonKey)

	resp, err := anonHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anonymous sign-in: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("anonymous sign-in: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anonymous sign-in: gotrue returned %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var session AnonSession
	if err := json.Unmarshal(body, &session); err != nil {
		return nil, fmt.Errorf("anonymous sign-in: decode response: %w", err)
	}
	if session.AccessToken == "" {
		return nil, fmt.Errorf("anonymous sign-in: response contained no access token")
	}
	return &session, nil
}

// WithServiceRoleKey attaches the service-role key, enabling admin REST
// calls (GoTrue-side deletion of reaped guests). Chainable.
func (a *AuthClient) WithServiceRoleKey(key string) *AuthClient {
	a.serviceRoleKey = key
	return a
}

// HasServiceRoleKey reports whether admin operations are available.
func (a *AuthClient) HasServiceRoleKey() bool {
	return a.serviceRoleKey != ""
}

// AdminDeleteUser deletes a GoTrue auth user by id via the admin REST API.
// Used by the anonymous-user reaper (ADR-024); requires the service role key.
func (a *AuthClient) AdminDeleteUser(ctx context.Context, userID string) error {
	if a.serviceRoleKey == "" {
		return fmt.Errorf("admin delete user: service role key not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		strings.TrimRight(a.baseURL, "/")+"/auth/v1/admin/users/"+userID, nil)
	if err != nil {
		return fmt.Errorf("admin delete user: build request: %w", err)
	}
	req.Header.Set("apikey", a.serviceRoleKey)
	req.Header.Set("Authorization", "Bearer "+a.serviceRoleKey)

	resp, err := anonHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("admin delete user: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("admin delete user: gotrue returned %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	return nil
}

// TokenClaims extracts the sub and is_anonymous claims from a JWT payload
// WITHOUT verifying the signature — callers must only use it on tokens
// already validated (AuthMiddleware validates via GoTrue GetUser first).
func TokenClaims(accessToken string) (sub string, isAnonymous bool, err error) {
	parts := strings.Split(accessToken, ".")
	if len(parts) != 3 {
		return "", false, fmt.Errorf("token claims: not a JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", false, fmt.Errorf("token claims: decode payload: %w", err)
	}
	var claims struct {
		Sub         string `json:"sub"`
		IsAnonymous bool   `json:"is_anonymous"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", false, fmt.Errorf("token claims: parse payload: %w", err)
	}
	return claims.Sub, claims.IsAnonymous, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
