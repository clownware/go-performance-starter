package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/clownware/alpine-go-performance-starter/internal/database"
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
