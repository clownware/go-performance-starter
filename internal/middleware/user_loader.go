package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/clownware/go-performance-starter/internal/database"
	"github.com/clownware/go-performance-starter/internal/repository"
	"github.com/clownware/go-performance-starter/internal/webutil"
)

// UserLoader resolves the authenticated identity to its application users row
// and stores it via webutil.WithUser for handlers (e.g. first-run onboarding).
// Must run after AuthMiddleware (needs the claims + gotrue user in context).
//
// A missing row is provisioned just-in-time: the app's users table is
// populated lazily on first authenticated request rather than at signup, and
// the insert runs RLS-scoped — the users_self_access WITH CHECK proves the
// row belongs to the requester (ADR-004; ADR-024's upgrade flow relies on
// this row existing).
func UserLoader(repo repository.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := webutil.AuthClaimsFromContext(r.Context())
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			user, err := resolveUser(r, repo, claims)
			if err != nil {
				slog.Error("Failed to load user row", "sub", claims.Sub, "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			ctx := webutil.WithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// resolveUser loads (or JIT-provisions) the users row for the authenticated
// claims and heals a stale guest flag: a non-anonymous token with an
// is_anonymous row means the identity was upgraded (#68) but the row flip
// was missed — sync it here so the reaper can never delete an upgraded
// account. The reverse direction never syncs: a row is only ever promoted.
func resolveUser(r *http.Request, repo repository.UserRepository, claims webutil.AuthClaims) (*database.User, error) {
	user, err := repo.GetByAuthID(r.Context(), claims.Sub)
	if errors.Is(err, repository.ErrNotFound) {
		return provisionUser(r, repo, claims)
	}
	if err != nil {
		return nil, err
	}
	if user.IsAnonymous && !claims.IsAnonymous {
		if err := repo.SetAnonymous(r.Context(), user.ID, false); err != nil {
			return nil, fmt.Errorf("heal stale guest flag: %w", err)
		}
		user.IsAnonymous = false
		slog.Info("Healed stale guest flag after upgrade", "user_id", user.ID)
	}
	return user, nil
}

// provisionUser creates the users row for a first-time authenticated visitor,
// copying identity fields from the validated gotrue user in context. The
// is_anonymous flag mirrors the JWT claim so the TTL reaper can distinguish
// guests from registered users (ADR-024).
func provisionUser(r *http.Request, repo repository.UserRepository, claims webutil.AuthClaims) (*database.User, error) {
	sub := claims.Sub
	params := database.CreateUserParams{
		AuthID:      pgtype.Text{String: sub, Valid: true},
		IsAnonymous: claims.IsAnonymous,
	}
	if gotrueUser, ok := GetUserFromContext(r.Context()); ok {
		params.Email = gotrueUser.Email
		if name, ok := gotrueUser.UserMetadata["name"].(string); ok && name != "" {
			params.Name = pgtype.Text{String: name, Valid: true}
		}
	}
	// Anonymous identities carry no email, and users.email is NOT NULL
	// UNIQUE — a shared "" collides on the second guest ever (live 500s,
	// 2026-07-12). Use a per-identity placeholder on the reserved .invalid
	// TLD; the upgrade flow (#68) replaces it with the real address.
	if params.Email == "" {
		params.Email = sub + "@guest.invalid"
	}
	slog.Info("Provisioning users row for first authenticated request", "sub", sub)
	return repo.Create(r.Context(), params)
}
