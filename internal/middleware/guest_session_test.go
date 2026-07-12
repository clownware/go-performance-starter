package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clownware/go-performance-starter/internal/auth"
)

type fakeSigner struct {
	session *auth.AnonSession
	err     error
	calls   int
}

func (f *fakeSigner) SignInAnonymously(ctx context.Context) (*auth.AnonSession, error) {
	f.calls++
	return f.session, f.err
}

func anonSession() *auth.AnonSession {
	s := &auth.AnonSession{AccessToken: "guest-tok", RefreshToken: "guest-ref", ExpiresIn: 3600}
	s.User.ID = "guest-123"
	s.User.IsAnonymous = true
	return s
}

func TestGuestSession(t *testing.T) {
	tests := []struct {
		name           string
		existingCookie string
		signer         *fakeSigner
		wantCalls      int
		wantCookie     bool
		wantDownstream string // access token visible to downstream handler
	}{
		{
			name:           "cookie-less visitor gets a guest session",
			signer:         &fakeSigner{session: anonSession()},
			wantCalls:      1,
			wantCookie:     true,
			wantDownstream: "guest-tok",
		},
		{
			name:           "existing session untouched",
			existingCookie: "already-signed-in",
			signer:         &fakeSigner{session: anonSession()},
			wantCalls:      0,
			wantDownstream: "already-signed-in",
		},
		{
			name:      "gotrue failure degrades to unauthenticated",
			signer:    &fakeSigner{err: errors.New("gotrue down")},
			wantCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var downstreamToken string
			var downstreamStatus = http.StatusOK
			h := GuestSession(tt.signer, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if c, err := r.Cookie("sb-access-token"); err == nil {
					downstreamToken = c.Value
				}
				w.WriteHeader(downstreamStatus)
			}))

			req := httptest.NewRequest(http.MethodGet, "/learn", nil)
			if tt.existingCookie != "" {
				req.AddCookie(&http.Cookie{Name: "sb-access-token", Value: tt.existingCookie})
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (guest failures must not break pages)", rec.Code)
			}
			if tt.signer.calls != tt.wantCalls {
				t.Errorf("SignInAnonymously calls = %d, want %d", tt.signer.calls, tt.wantCalls)
			}
			if downstreamToken != tt.wantDownstream {
				t.Errorf("downstream access token = %q, want %q", downstreamToken, tt.wantDownstream)
			}

			var gotSetCookie bool
			for _, c := range rec.Result().Cookies() {
				if c.Name == "sb-access-token" && c.Value != "" {
					gotSetCookie = true
					if !c.HttpOnly {
						t.Error("guest access cookie must be HttpOnly")
					}
					if c.MaxAge != 3600 {
						t.Errorf("access cookie MaxAge = %d, want the session's ExpiresIn (3600)", c.MaxAge)
					}
				}
				if c.Name == "sb-refresh-token" && c.Value != "" {
					if c.MaxAge != 30*24*60*60 {
						t.Errorf("refresh cookie MaxAge = %d, want 30 days (%d)", c.MaxAge, 30*24*60*60)
					}
				}
			}
			if gotSetCookie != tt.wantCookie {
				t.Errorf("Set-Cookie sb-access-token present = %v, want %v", gotSetCookie, tt.wantCookie)
			}
		})
	}
}
