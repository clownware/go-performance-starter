package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/clownware/alpine-go-performance-starter/internal/auth"
)

// anonSigner is the slice of AuthClient GuestSession needs; an interface so
// tests can fake the GoTrue call.
type anonSigner interface {
	SignInAnonymously(ctx context.Context) (*auth.AnonSession, error)
}

// GuestSession issues an anonymous Supabase identity to cookie-less visitors
// (ADR-024 guest mode): the server performs the GoTrue anonymous sign-in and
// sets the same httpOnly session cookies the login flow uses, so downstream
// AuthMiddleware and RLS treat guests exactly like registered users.
//
// Failure is non-fatal by design — the page still renders unauthenticated
// and a warning is logged (a GoTrue outage must not take down public pages).
func GuestSession(signer anonSigner, secureCookie bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if c, err := r.Cookie("sb-access-token"); err == nil && c.Value != "" {
				next.ServeHTTP(w, r) // already has a session (guest or registered)
				return
			}

			session, err := signer.SignInAnonymously(r.Context())
			if err != nil {
				slog.Warn("Guest sign-in failed; continuing unauthenticated", "error", err)
				next.ServeHTTP(w, r)
				return
			}

			access := &http.Cookie{
				Name:     "sb-access-token",
				Value:    session.AccessToken,
				Path:     "/",
				MaxAge:   session.ExpiresIn,
				HttpOnly: true,
				Secure:   secureCookie,
				SameSite: http.SameSiteLaxMode,
			}
			refresh := &http.Cookie{
				Name:     "sb-refresh-token",
				Value:    session.RefreshToken,
				Path:     "/",
				MaxAge:   int((30 * 24 * time.Hour).Seconds()),
				HttpOnly: true,
				Secure:   secureCookie,
				SameSite: http.SameSiteLaxMode,
			}
			http.SetCookie(w, access)
			http.SetCookie(w, refresh)

			// Make the new session visible to downstream middleware in THIS
			// request (AuthMiddleware reads the request cookie jar).
			r.AddCookie(access)
			r.AddCookie(refresh)

			slog.Info("Issued anonymous guest session", "sub", session.User.ID)
			next.ServeHTTP(w, r)
		})
	}
}
