package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ErrEmailInUse marks an upgrade rejected because the email already belongs
// to another identity — the one upgrade failure the user can act on.
var ErrEmailInUse = errors.New("email address already registered")

// UpgradeResult reports the outcome of attaching credentials to an anonymous
// user (ADR-024 upgrade flow, #68).
type UpgradeResult struct {
	// ConfirmationSent is true when GoTrue requires the email to be
	// confirmed before it becomes active (the Supabase default).
	ConfirmationSent bool
}

// UpgradeAnonymousUser attaches an email/password identity to the current
// anonymous session via PUT /auth/v1/user — the same auth.uid() keeps every
// RLS-scoped row, which is the whole point of the upgrade flow (ADR-024).
// accessToken is the session token of the anonymous user being upgraded.
func (a *AuthClient) UpgradeAnonymousUser(ctx context.Context, accessToken, email, password string) (*UpgradeResult, error) {
	if a.baseURL == "" || a.anonKey == "" {
		return nil, fmt.Errorf("upgrade user: auth client not configured")
	}

	payload, err := json.Marshal(map[string]string{"email": email, "password": password})
	if err != nil {
		return nil, fmt.Errorf("upgrade user: encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		strings.TrimRight(a.baseURL, "/")+"/auth/v1/user", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("upgrade user: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", a.anonKey)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := anonHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upgrade user: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("upgrade user: read response: %w", err)
	}
	if resp.StatusCode == http.StatusUnprocessableEntity &&
		strings.Contains(strings.ToLower(string(body)), "already been registered") {
		return nil, ErrEmailInUse
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upgrade user: gotrue returned %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var user struct {
		EmailConfirmedAt string `json:"email_confirmed_at"`
	}
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("upgrade user: decode response: %w", err)
	}
	return &UpgradeResult{ConfirmationSent: user.EmailConfirmedAt == ""}, nil
}

// RefreshSession exchanges a refresh token for fresh session tokens. Used
// right after an upgrade so the access token's is_anonymous claim reflects
// the new identity instead of waiting out the old token's lifetime.
func (a *AuthClient) RefreshSession(ctx context.Context, refreshToken string) (*AnonSession, error) {
	if a.baseURL == "" || a.anonKey == "" {
		return nil, fmt.Errorf("refresh session: auth client not configured")
	}

	payload, err := json.Marshal(map[string]string{"refresh_token": refreshToken})
	if err != nil {
		return nil, fmt.Errorf("refresh session: encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(a.baseURL, "/")+"/auth/v1/token?grant_type=refresh_token", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("refresh session: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", a.anonKey)
	req.Header.Set("Authorization", "Bearer "+a.anonKey)

	resp, err := anonHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh session: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("refresh session: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh session: gotrue returned %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var session AnonSession
	if err := json.Unmarshal(body, &session); err != nil {
		return nil, fmt.Errorf("refresh session: decode response: %w", err)
	}
	if session.AccessToken == "" {
		return nil, fmt.Errorf("refresh session: response contained no access token")
	}
	return &session, nil
}
