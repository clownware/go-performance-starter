package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clownware/go-performance-starter/internal/database"
)

// ReaperRepo performs system-job deletions of expired anonymous users
// (ADR-024). Unlike the request-scoped repos, it runs as service_role — the
// system identity the service_role_bypass policies exist for — because no
// user's RLS scope may delete other users' rows. This is deliberately NOT
// part of inScope's role allowlist: request paths must never assume
// service_role.
type ReaperRepo struct {
	db *pgxpool.Pool
}

// NewReaperRepo creates a ReaperRepo.
func NewReaperRepo(db *pgxpool.Pool) *ReaperRepo {
	return &ReaperRepo{db: db}
}

// DeleteExpiredAnonymousUsers removes anonymous users created before the
// cutoff and returns the deleted rows (auth_ids feed GoTrue-side cleanup).
// Flashcards and quiz attempts cascade via foreign keys.
func (r *ReaperRepo) DeleteExpiredAnonymousUsers(ctx context.Context, olderThan time.Time) ([]database.DeleteExpiredAnonymousUsersRow, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("reaper: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "SET LOCAL ROLE service_role"); err != nil {
		return nil, fmt.Errorf("reaper: set service_role: %w", err)
	}

	rows, err := database.New(tx).DeleteExpiredAnonymousUsers(ctx, olderThan)
	if err != nil {
		return nil, fmt.Errorf("reaper: delete expired anonymous users: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("reaper: commit: %w", err)
	}
	return rows, nil
}

// ListExistingAuthIDs returns the subset of authIDs that have a public users
// row (orphan pass, #82). Runs as service_role for the same reason the
// delete does: RLS scopes users_self_access to one identity, and this has
// to see every row to answer "does anyone own this auth id" truthfully —
// a false "no" would delete a live guest's identity.
func (r *ReaperRepo) ListExistingAuthIDs(ctx context.Context, authIDs []string) ([]string, error) {
	if len(authIDs) == 0 {
		return nil, nil
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("reaper: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "SET LOCAL ROLE service_role"); err != nil {
		return nil, fmt.Errorf("reaper: set service_role: %w", err)
	}
	rows, err := database.New(tx).ListExistingAuthIDs(ctx, authIDs)
	if err != nil {
		return nil, fmt.Errorf("reaper: list existing auth ids: %w", err)
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.Valid {
			out = append(out, row.String)
		}
	}
	return out, nil
}
