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
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/clownware/go-performance-starter/internal/database"
)

type fakeReaperStore struct {
	rows      []database.DeleteExpiredAnonymousUsersRow
	err       error
	gotCutoff time.Time
}

func (f *fakeReaperStore) DeleteExpiredAnonymousUsers(ctx context.Context, olderThan time.Time) ([]database.DeleteExpiredAnonymousUsersRow, error) {
	f.gotCutoff = olderThan
	return f.rows, f.err
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

// countingReaperStore is race-safe so TestReaperStart can poll it while the
// reap goroutine runs under -race.
type countingReaperStore struct {
	calls atomic.Int32
}

func (s *countingReaperStore) DeleteExpiredAnonymousUsers(ctx context.Context, olderThan time.Time) ([]database.DeleteExpiredAnonymousUsersRow, error) {
	s.calls.Add(1)
	return nil, nil
}

// TestReaperStart pins the loop contract: an immediate first pass on start
// (restarts must not postpone overdue cleanup), ticker-driven passes after,
// and a clean stop on context cancellation.
func TestReaperStart(t *testing.T) {
	store := &countingReaperStore{}
	r := NewReaper(store, nil, time.Hour, 2*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())

	r.Start(ctx)

	deadline := time.Now().Add(5 * time.Second)
	for store.calls.Load() < 3 {
		if time.Now().After(deadline) {
			t.Fatalf("reaper ran %d passes, want >= 3 (immediate + ticks)", store.calls.Load())
		}
		time.Sleep(time.Millisecond)
	}

	cancel()
	// Let any in-flight pass drain, then assert the loop has stopped.
	time.Sleep(20 * time.Millisecond)
	settled := store.calls.Load()
	time.Sleep(50 * time.Millisecond)
	if got := store.calls.Load(); got != settled {
		t.Errorf("reaper kept running after cancel: %d passes after settling at %d", got, settled)
	}
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
// is the only signal an operator gets.
func TestReaperLogsFailures(t *testing.T) {
	sink := &lockedBuffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(sink, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	waitForFailures := func(want int, phase string) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for strings.Count(sink.String(), "reaper pass failed") < want {
			if time.Now().After(deadline) {
				t.Fatalf("%s: want >= %d logged pass failures, log:\n%s", phase, want, sink.String())
			}
			time.Sleep(time.Millisecond)
		}
	}

	store := &fakeReaperStore{err: errors.New("db down")}

	// Hour-long interval: the only pass that can log inside the deadline is
	// the immediate one, so this isolates the on-start failure log.
	ctxImmediate, cancelImmediate := context.WithCancel(context.Background())
	defer cancelImmediate()
	NewReaper(store, nil, time.Hour, time.Hour).Start(ctxImmediate)
	waitForFailures(1, "immediate pass")
	cancelImmediate()

	// Fast ticker: failures must keep logging on every tick, not just once.
	ctxTicks, cancelTicks := context.WithCancel(context.Background())
	defer cancelTicks()
	NewReaper(store, nil, time.Hour, 2*time.Millisecond).Start(ctxTicks)
	waitForFailures(3, "ticker passes")
	cancelTicks()

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
