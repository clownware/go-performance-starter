package webutil

import (
	"context"
	"encoding/json"
	"fmt"
)

const claimsContextKey contextKey = "authClaims"

// Roles the app is allowed to assume on a database connection. The connection
// role can never be a bind parameter, so it is restricted to this set.
const (
	RoleAuthenticated = "authenticated"
	RoleAnon          = "anon"
)

// AuthClaims is the validated JWT identity carried from the auth middleware
// to the repository layer, where it is applied to the database connection so
// RLS policies evaluate against the requester (ADR-004).
type AuthClaims struct {
	// Sub is the Supabase auth user id (auth.uid()).
	Sub string
	// Role is the Postgres role to assume: RoleAuthenticated or RoleAnon.
	Role string
	// IsAnonymous mirrors the GoTrue is_anonymous claim (ADR-024 gating).
	IsAnonymous bool
}

// Validate checks the claims are safe to apply to a connection.
func (c AuthClaims) Validate() error {
	if c.Sub == "" {
		return fmt.Errorf("auth claims: sub is empty")
	}
	if c.Role != RoleAuthenticated && c.Role != RoleAnon {
		return fmt.Errorf("auth claims: role %q not allowed", c.Role)
	}
	return nil
}

// JSON renders the claims as the request.jwt.claims JSON Supabase's
// auth.uid() and policy expressions read.
func (c AuthClaims) JSON() (string, error) {
	b, err := json.Marshal(map[string]any{
		"sub":          c.Sub,
		"role":         c.Role,
		"is_anonymous": c.IsAnonymous,
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// WithAuthClaims stores the validated JWT claims in the context.
func WithAuthClaims(ctx context.Context, claims AuthClaims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

// AuthClaimsFromContext retrieves the JWT claims; ok is false when the
// request is unauthenticated (or auth is disabled).
func AuthClaimsFromContext(ctx context.Context) (AuthClaims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(AuthClaims)
	return claims, ok
}
