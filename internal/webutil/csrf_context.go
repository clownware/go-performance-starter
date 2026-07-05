package webutil

import "context"

const csrfContextKey contextKey = "csrfToken"

// WithCSRFToken stores the request's CSRF token in the context.
func WithCSRFToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, csrfContextKey, token)
}

// CSRFTokenFromContext retrieves the CSRF token from the context.
// Returns "" if the CSRF middleware did not run.
func CSRFTokenFromContext(ctx context.Context) string {
	token, _ := ctx.Value(csrfContextKey).(string)
	return token
}
