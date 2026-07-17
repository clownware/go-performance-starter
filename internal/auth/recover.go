package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// RecoverySession is the GoTrue session minted by verifying a recovery
// token_hash — the same shape every GoTrue sign-in returns.
type RecoverySession = AnonSession

// VerifyRecovery exchanges a recovery token_hash (from the password-reset
// email link) for a session via POST /auth/v1/verify. gotrue-go v1.2.1's
// VerifyForUser only supports token+email, not token_hash, so this uses the
// direct REST pattern established by SignInAnonymously. The token_hash
// arrives in the URL query — visible server-side, unlike the default
// fragment-based flow — which is what lets this app stay server-rendered
// with no JS in the reset path.
func (a *AuthClient) VerifyRecovery(ctx context.Context, tokenHash string) (*RecoverySession, error) {
	if a.baseURL == "" || a.anonKey == "" {
		return nil, fmt.Errorf("verify recovery: auth client not configured")
	}
	if tokenHash == "" {
		return nil, fmt.Errorf("verify recovery: empty token hash")
	}

	payload, err := json.Marshal(map[string]string{
		"type":       "recovery",
		"token_hash": tokenHash,
	})
	if err != nil {
		return nil, fmt.Errorf("verify recovery: encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(a.baseURL, "/")+"/auth/v1/verify", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("verify recovery: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", a.anonKey)
	req.Header.Set("Authorization", "Bearer "+a.anonKey)

	resp, err := anonHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("verify recovery: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("verify recovery: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("verify recovery: gotrue returned %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var session RecoverySession
	if err := json.Unmarshal(body, &session); err != nil {
		return nil, fmt.Errorf("verify recovery: decode response: %w", err)
	}
	if session.AccessToken == "" {
		return nil, fmt.Errorf("verify recovery: response contained no access token")
	}
	return &session, nil
}
