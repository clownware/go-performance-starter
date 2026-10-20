package jobs

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/clownware/go-performance-starter/internal/auth"
	"github.com/clownware/go-performance-starter/internal/database"
)

type fakeReaperStore struct {
	rows      []database.DeleteExpiredAnonymousUsersRow
	err       error
	gotCutoff time.Time

	// Orphan pass (#82): auth ids that DO have a public row, and the ids the
	// reaper asked about.
	existing    []string
	existingErr error
	gotAsked    []string
}

func (f *fakeReaperStore) DeleteExpiredAnonymousUsers(ctx context.Context, olderThan time.Time) ([]database.DeleteExpiredAnonymousUsersRow, error) {
	f.gotCutoff = olderThan
	return f.rows, f.err
}

func (f *fakeReaperStore) ListExistingAuthIDs(ctx context.Context, authIDs []string) ([]string, error) {
	f.gotAsked = append([]string(nil), authIDs...)
	if f.existingErr != nil {
		return nil, f.existingErr
	}
	var out []string
	for _, id := range authIDs {
		for _, e := range f.existing {
			if id == e {
				out = append(out, id)
			}
		}
	}
	return out, nil
}

func reapedRow(authID string) database.DeleteExpiredAnonymousUsersRow {
	return database.DeleteExpiredAnonymousUsersRow{
		ID:     uuid.New(),
		AuthID: pgtype.Text{String: authID, Valid: authID != ""},
	}
}

func TestReaperRunOnce(t *testing.T) {
	tests := []struct {
		name          string
		store         *fakeReaperStore
		withDeleter   bool
		deleterErr    error
		wantCount     int
		wantErr       bool
		wantAuthCalls []string
	}{
		{
			name:      "nothing expired",
			store:     &fakeReaperStore{},
			wantCount: 0,
		},
		{
			name: "reaps and cleans auth users",
			store: &fakeReaperStore{rows: []database.DeleteExpiredAnonymousUsersRow{
				reapedRow("auth-1"), reapedRow("auth-2"), reapedRow(""),
			}},
			withDeleter:   true,
			wantCount:     3,
			wantAuthCalls: []string{"auth-1", "auth-2"}, // empty auth_id skipped
		},
		{
			name: "auth deletion failure is best-effort, not fatal",
			store: &fakeReaperStore{rows: []database.DeleteExpiredAnonymousUsersRow{
				reapedRow("auth-1"),
			}},
			withDeleter:   true,
			deleterErr:    errors.New("gotrue down"),
			wantCount:     1,
			wantAuthCalls: []string{"auth-1"},
		},
		{
			name: "no deleter configured still reaps app rows",
			store: &fakeReaperStore{rows: []database.DeleteExpiredAnonymousUsersRow{
				reapedRow("auth-1"),
			}},
			wantCount: 1,
		},
		{
			name:    "store failure propagates",
			store:   &fakeReaperStore{err: errors.New("db down")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var authCalls []string
			var deleter AuthUserDeleter
			if tt.withDeleter {
				deleter = func(ctx context.Context, authID string) error {
					authCalls = append(authCalls, authID)
					return tt.deleterErr
				}
			}

			ttl := 30 * 24 * time.Hour
			r := NewReaper(tt.store, deleter, ttl, time.Hour)
			count, err := r.RunOnce(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Fatal("RunOnce() = nil error, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("RunOnce() error: %v", err)
			}
			if count != tt.wantCount {
				t.Errorf("count = %d, want %d", count, tt.wantCount)
			}
			if len(authCalls) != len(tt.wantAuthCalls) {
				t.Fatalf("auth deletions = %v, want %v", authCalls, tt.wantAuthCalls)
			}
			for i, want := range tt.wantAuthCalls {
				if authCalls[i] != want {
					t.Errorf("auth deletion %d = %q, want %q", i, authCalls[i], want)
				}
			}

			// Cutoff must be ~now-ttl.
			wantCutoff := time.Now().Add(-ttl)
			if diff := tt.store.gotCutoff.Sub(wantCutoff); diff < -time.Minute || diff > time.Minute {
				t.Errorf("cutoff = %v, want ~%v", tt.store.gotCutoff, wantCutoff)
			}
		})
	}
}

// countingReaperStore is race-safe so TestReaperStart can read it while the
// reap goroutine runs under -race.
type countingReaperStore struct {
	calls atomic.Int32
}

func (s *countingReaperStore) DeleteExpiredAnonymousUsers(ctx context.Context, olderThan time.Time) ([]database.DeleteExpiredAnonymousUsersRow, error) {
	s.calls.Add(1)
	return nil, nil
}

func (s *countingReaperStore) ListExistingAuthIDs(ctx context.Context, authIDs []string) ([]string, error) {
	return nil, nil
}

// TestReaperStart pins the loop contract: an immediate first pass on start
// (restarts must not postpone overdue cleanup), one pass per interval after,
// and a clean stop on context cancellation. It runs in a synctest bubble
// (Go 1.25): synctest.Sleep (Go 1.27) advances a fake clock and returns
// once the reap goroutine has finished reacting to it, so pass counts are
// exact and the hour-long interval costs nothing.
func TestReaperStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store := &countingReaperStore{}
		ctx, cancel := context.WithCancel(t.Context())
		NewReaper(store, nil, time.Hour, time.Hour).Start(ctx)

		want := func(n int32, phase string) {
			t.Helper()
			if got := store.calls.Load(); got != n {
				t.Fatalf("%s: reaper ran %d passes, want %d", phase, got, n)
			}
		}

		synctest.Wait() // the immediate pass
		want(1, "on start")
		synctest.Sleep(time.Hour - time.Nanosecond)
		want(1, "just before the first tick")
		synctest.Sleep(time.Nanosecond)
		want(2, "first tick")
		synctest.Sleep(time.Hour)
		want(3, "second tick")

		cancel()
		synctest.Sleep(10 * time.Hour)
		want(3, "after cancel") // synctest.Test also fails if the loop is still alive
	})
}

// lockedBuffer lets the reap goroutine and the test share a log sink under -race.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// TestReaperLogsFailures pins the failure-observability contract: pass
// failures (immediate and ticker-driven) and best-effort auth deletion
// failures are logged, never swallowed — on a background job the log line
// is the only signal an operator gets. The loop phases run in a synctest
// bubble, so the logged-failure counts are exact rather than polled for.
func TestReaperLogsFailures(t *testing.T) {
	sink := &lockedBuffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(sink, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	wantFailures := func(t *testing.T, want int, phase string) {
		t.Helper()
		synctest.Wait()
		if got := strings.Count(sink.String(), "reaper pass failed"); got != want {
			t.Fatalf("%s: %d logged pass failures, want %d, log:\n%s", phase, got, want, sink.String())
		}
	}

	store := &fakeReaperStore{err: errors.New("db down")}

	synctest.Test(t, func(t *testing.T) {
		// Only the immediate pass can run before the hour-long interval
		// elapses, which isolates the on-start failure log.
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		NewReaper(store, nil, time.Hour, time.Hour).Start(ctx)
		wantFailures(t, 1, "immediate pass")
	})

	synctest.Test(t, func(t *testing.T) {
		// Failures must keep logging on every tick, not just once.
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		NewReaper(store, nil, time.Hour, time.Hour).Start(ctx)
		synctest.Sleep(2 * time.Hour)
		wantFailures(t, 4, "immediate + two ticker passes")
	})

	// Failing auth deleter: best-effort must warn, not error out the pass.
	okStore := &fakeReaperStore{rows: []database.DeleteExpiredAnonymousUsersRow{reapedRow("auth-1")}}
	deleter := func(ctx context.Context, authID string) error { return errors.New("gotrue down") }
	if _, err := NewReaper(okStore, deleter, time.Hour, time.Hour).RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce with failing deleter must stay best-effort, got error: %v", err)
	}
	if !strings.Contains(sink.String(), "Failed to delete GoTrue auth user") {
		t.Errorf("auth deletion failure was not logged, log:\n%s", sink.String())
	}
}

// TestReaperOrphanPass pins the auth-side sweep (#82): anonymous GoTrue
// identities older than the TTL with no public.users row are invisible to
// the row-driven reap and must be deleted from the auth side; everything
// else — young orphans, identities that do have a row, non-anonymous users
// (already filtered by the lister) — is left alone. The pass is best-effort:
// lister/store failures are logged and never fail the row pass.
func TestReaperOrphanPass(t *testing.T) {
	ttl := 30 * 24 * time.Hour
	old := time.Now().Add(-ttl - time.Hour)
	young := time.Now().Add(-time.Hour)

	tests := []struct {
		name        string
		store       *fakeReaperStore
		withLister  bool
		listerUsers []auth.AdminUser
		listerErr   error
		withDeleter bool
		wantCount   int
		wantDeleted []string
		wantAsked   []string
	}{
		{
			name:        "no lister configured — row pass only",
			store:       &fakeReaperStore{rows: []database.DeleteExpiredAnonymousUsersRow{reapedRow("auth-1")}},
			withDeleter: true,
			wantCount:   1,
			wantDeleted: []string{"auth-1"},
		},
		{
			name:        "lister without deleter cannot act, so it is not consulted",
			store:       &fakeReaperStore{},
			withLister:  true,
			listerUsers: []auth.AdminUser{{ID: "orphan-old", CreatedAt: old}},
			wantCount:   0,
		},
		{
			name:       "old orphan deleted; young orphan and row-backed identity spared",
			store:      &fakeReaperStore{existing: []string{"has-row-old"}},
			withLister: true,
			listerUsers: []auth.AdminUser{
				{ID: "orphan-old", CreatedAt: old},
				{ID: "orphan-young", CreatedAt: young},
				{ID: "has-row-old", CreatedAt: old},
			},
			withDeleter: true,
			wantCount:   1,
			wantDeleted: []string{"orphan-old"},
			wantAsked:   []string{"orphan-old", "has-row-old"}, // young ones never hit the DB
		},
		{
			name:        "nothing older than the TTL — store not queried",
			store:       &fakeReaperStore{},
			withLister:  true,
			listerUsers: []auth.AdminUser{{ID: "orphan-young", CreatedAt: young}},
			withDeleter: true,
			wantCount:   0,
		},
		{
			name:        "row pass and orphan pass counts add up",
			store:       &fakeReaperStore{rows: []database.DeleteExpiredAnonymousUsersRow{reapedRow("auth-1")}},
			withLister:  true,
			listerUsers: []auth.AdminUser{{ID: "orphan-old", CreatedAt: old}},
			withDeleter: true,
			wantCount:   2,
			wantDeleted: []string{"auth-1", "orphan-old"},
			wantAsked:   []string{"orphan-old"},
		},
		{
			name:        "lister failure is logged, row pass result stands",
			store:       &fakeReaperStore{rows: []database.DeleteExpiredAnonymousUsersRow{reapedRow("auth-1")}},
			withLister:  true,
			listerErr:   errors.New("gotrue down"),
			withDeleter: true,
			wantCount:   1,
			wantDeleted: []string{"auth-1"},
		},
		{
			name:        "existence lookup failure deletes nothing (never guess)",
			store:       &fakeReaperStore{existingErr: errors.New("db down")},
			withLister:  true,
			listerUsers: []auth.AdminUser{{ID: "orphan-old", CreatedAt: old}},
			withDeleter: true,
			wantCount:   0,
			wantAsked:   []string{"orphan-old"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var deleted []string
			var deleter AuthUserDeleter
			if tt.withDeleter {
				deleter = func(ctx context.Context, authID string) error {
					deleted = append(deleted, authID)
					return nil
				}
			}
			r := NewReaper(tt.store, deleter, ttl, time.Hour)
			listerCalled := false
			if tt.withLister {
				r.WithAuthLister(func(ctx context.Context) ([]auth.AdminUser, error) {
					listerCalled = true
					return tt.listerUsers, tt.listerErr
				})
			}

			count, err := r.RunOnce(context.Background())
			if err != nil {
				t.Fatalf("RunOnce() error: %v", err)
			}
			if count != tt.wantCount {
				t.Errorf("count = %d, want %d", count, tt.wantCount)
			}
			if tt.withLister && !tt.withDeleter && listerCalled {
				t.Error("lister was called although no deleter is configured")
			}
			if strings.Join(deleted, ",") != strings.Join(tt.wantDeleted, ",") {
				t.Errorf("deleted = %v, want %v", deleted, tt.wantDeleted)
			}
			if strings.Join(tt.store.gotAsked, ",") != strings.Join(tt.wantAsked, ",") {
				t.Errorf("asked store about %v, want %v", tt.store.gotAsked, tt.wantAsked)
			}
		})
	}
}
