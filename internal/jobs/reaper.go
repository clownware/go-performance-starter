// Package jobs holds background workers. Per the minimal-dependency ethos
// (ADR-000) these are plain goroutines with tickers, not a job framework —
// the starter's background needs are periodic maintenance, not queues.
package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/clownware/go-performance-starter/internal/database"
)

// ReaperStore deletes expired anonymous users (implemented by
// postgres.ReaperRepo with a service_role-scoped transaction).
type ReaperStore interface {
	DeleteExpiredAnonymousUsers(ctx context.Context, olderThan time.Time) ([]database.DeleteExpiredAnonymousUsersRow, error)
}

// AuthUserDeleter removes the corresponding GoTrue auth user; nil disables
// auth-side cleanup (e.g. no service role key configured).
type AuthUserDeleter func(ctx context.Context, authID string) error

// Reaper periodically deletes anonymous guest accounts older than the TTL
// (ADR-024). Guests who upgrade become non-anonymous and are exempt; age is
// measured from account creation, so an unupgraded guest is reaped after TTL
// regardless of activity — the demo makes no retention promise to guests.
type Reaper struct {
	store          ReaperStore
	deleteAuthUser AuthUserDeleter
	ttl            time.Duration
	interval       time.Duration
}

// NewReaper builds a Reaper. deleteAuthUser may be nil.
func NewReaper(store ReaperStore, deleteAuthUser AuthUserDeleter, ttl, interval time.Duration) *Reaper {
	return &Reaper{store: store, deleteAuthUser: deleteAuthUser, ttl: ttl, interval: interval}
}

// Start runs the reap loop until ctx is cancelled. An immediate first pass
// runs on start so restarts don't postpone overdue cleanup by a full interval.
func (r *Reaper) Start(ctx context.Context) {
	go func() {
		if _, err := r.RunOnce(ctx); err != nil {
			slog.Error("Anonymous-user reaper pass failed", "error", err)
		}
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := r.RunOnce(ctx); err != nil {
					slog.Error("Anonymous-user reaper pass failed", "error", err)
				}
			}
		}
	}()
}

// RunOnce performs a single reap pass and returns how many users were removed.
func (r *Reaper) RunOnce(ctx context.Context) (int, error) {
	cutoff := time.Now().Add(-r.ttl)
	reaped, err := r.store.DeleteExpiredAnonymousUsers(ctx, cutoff)
	if err != nil {
		return 0, err
	}
	if len(reaped) == 0 {
		return 0, nil
	}

	if r.deleteAuthUser != nil {
		for _, row := range reaped {
			if !row.AuthID.Valid || row.AuthID.String == "" {
				continue
			}
			// Best-effort: app rows are already gone; a failed GoTrue
			// deletion just leaves an orphaned (empty) auth user.
			if err := r.deleteAuthUser(ctx, row.AuthID.String); err != nil {
				slog.Warn("Failed to delete GoTrue auth user for reaped guest", "auth_id", row.AuthID.String, "error", err)
			}
		}
	}

	slog.Info("Reaped expired anonymous users", "count", len(reaped), "cutoff", cutoff)
	return len(reaped), nil
}
