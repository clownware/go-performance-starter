// Package jobs holds background workers. Per the minimal-dependency ethos
// (ADR-000) these are plain goroutines with tickers, not a job framework —
// the starter's background needs are periodic maintenance, not queues.
package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/clownware/go-performance-starter/internal/auth"
	"github.com/clownware/go-performance-starter/internal/database"
)

// ReaperStore is the reaper's data-access seam (implemented by
// postgres.ReaperRepo with a service_role-scoped transaction).
type ReaperStore interface {
	// DeleteExpiredAnonymousUsers deletes anonymous users created before the
	// cutoff and returns their ids (auth_ids feed GoTrue-side cleanup).
	DeleteExpiredAnonymousUsers(ctx context.Context, olderThan time.Time) ([]database.DeleteExpiredAnonymousUsersRow, error)
	// ListExistingAuthIDs reports which of the given GoTrue auth ids have a
	// public users row — the orphan pass (#82) deletes the ones that don't.
	ListExistingAuthIDs(ctx context.Context, authIDs []string) ([]string, error)
}

// AuthUserDeleter removes the corresponding GoTrue auth user; nil disables
// auth-side cleanup (e.g. no service role key configured).
type AuthUserDeleter func(ctx context.Context, authID string) error

// AuthUserLister enumerates anonymous GoTrue identities for the orphan pass
// (#82); nil disables it. Same service-role requirement as the deleter.
type AuthUserLister func(ctx context.Context) ([]auth.AdminUser, error)

// Reaper periodically deletes anonymous guest accounts older than the TTL
// (ADR-024). Guests who upgrade become non-anonymous and are exempt; age is
// measured from account creation, so an unupgraded guest is reaped after TTL
// regardless of activity — the demo makes no retention promise to guests.
//
// Two passes per run. The row pass deletes expired public.users rows and
// their GoTrue twins. The orphan pass (#82) then sweeps the other direction:
// anonymous GoTrue identities older than the TTL that never got a public row
// — provisioning failed, or the guest signed in but never hit a UserLoader
// route — are invisible to the row pass and would otherwise linger forever.
type Reaper struct {
	store          ReaperStore
	deleteAuthUser AuthUserDeleter
	listAuthUsers  AuthUserLister
	ttl            time.Duration
	interval       time.Duration
}

// NewReaper builds a Reaper. deleteAuthUser may be nil.
func NewReaper(store ReaperStore, deleteAuthUser AuthUserDeleter, ttl, interval time.Duration) *Reaper {
	return &Reaper{store: store, deleteAuthUser: deleteAuthUser, ttl: ttl, interval: interval}
}

// WithAuthLister enables the orphan pass. Chainable. It only has effect when
// a deleter is also configured — listing orphans you cannot delete is noise.
func (r *Reaper) WithAuthLister(list AuthUserLister) *Reaper {
	r.listAuthUsers = list
	return r
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

// RunOnce performs a single reap pass and returns how many users were
// removed (public rows reaped plus auth-side orphans). A row-pass failure is
// the returned error; the orphan pass is best-effort and only logs.
func (r *Reaper) RunOnce(ctx context.Context) (int, error) {
	cutoff := time.Now().Add(-r.ttl)
	reaped, err := r.store.DeleteExpiredAnonymousUsers(ctx, cutoff)
	if err != nil {
		return 0, err
	}

	if len(reaped) > 0 && r.deleteAuthUser != nil {
		for _, row := range reaped {
			if !row.AuthID.Valid || row.AuthID.String == "" {
				continue
			}
			// Best-effort: app rows are already gone; a failed GoTrue
			// deletion just leaves an orphaned (empty) auth user — which the
			// orphan pass below will pick up on a later run.
			if err := r.deleteAuthUser(ctx, row.AuthID.String); err != nil {
				slog.Warn("Failed to delete GoTrue auth user for reaped guest", "auth_id", row.AuthID.String, "error", err)
			}
		}
	}
	if len(reaped) > 0 {
		slog.Info("Reaped expired anonymous users", "count", len(reaped), "cutoff", cutoff)
	}

	orphans := r.reapAuthOrphans(ctx, cutoff)
	return len(reaped) + orphans, nil
}

// reapAuthOrphans deletes anonymous GoTrue identities older than cutoff that
// have no public users row. Returns how many it deleted. Every failure is
// logged and stops the pass without deleting anything it is unsure about —
// an auth identity is only removed when the database positively confirmed
// it has no row.
func (r *Reaper) reapAuthOrphans(ctx context.Context, cutoff time.Time) int {
	if r.listAuthUsers == nil || r.deleteAuthUser == nil {
		return 0
	}
	users, err := r.listAuthUsers(ctx)
	if err != nil {
		slog.Warn("Reaper orphan pass: listing anonymous auth users failed", "error", err)
		return 0
	}
	var candidates []string
	for _, u := range users {
		if u.CreatedAt.Before(cutoff) {
			candidates = append(candidates, u.ID)
		}
	}
	if len(candidates) == 0 {
		return 0
	}
	existing, err := r.store.ListExistingAuthIDs(ctx, candidates)
	if err != nil {
		slog.Warn("Reaper orphan pass: checking for public rows failed", "error", err)
		return 0
	}
	hasRow := make(map[string]bool, len(existing))
	for _, id := range existing {
		hasRow[id] = true
	}

	deleted := 0
	for _, id := range candidates {
		if hasRow[id] {
			continue
		}
		if err := r.deleteAuthUser(ctx, id); err != nil {
			slog.Warn("Reaper orphan pass: failed to delete orphaned GoTrue auth user", "auth_id", id, "error", err)
			continue
		}
		deleted++
	}
	if deleted > 0 {
		slog.Info("Reaped orphaned anonymous auth users (no public row)", "count", deleted, "cutoff", cutoff)
	}
	return deleted
}
