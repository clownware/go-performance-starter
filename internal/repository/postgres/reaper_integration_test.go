package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clownware/alpine-go-performance-starter/internal/database"
)

// TestReaperRepoIntegration proves the reaper deletes only expired anonymous
// users — cascading their flashcards — while sparing recent guests and
// registered users, running under the service_role identity.
func TestReaperRepoIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	q := database.New(pool)

	seed := func(isAnonymous bool, age time.Duration) uuid.UUID {
		t.Helper()
		authID := uuid.NewString()
		user, err := q.CreateUser(ctx, database.CreateUserParams{
			Email:       authID + "@example.com",
			AuthID:      pgtype.Text{String: authID, Valid: true},
			IsAnonymous: isAnonymous,
		})
		if err != nil {
			t.Fatalf("seed user: %v", err)
		}
		if _, err := pool.Exec(ctx, "UPDATE users SET created_at = NOW() - $1::interval WHERE id = $2",
			age.String(), user.ID); err != nil {
			t.Fatalf("age user: %v", err)
		}
		if _, err := q.CreateFlashcard(ctx, database.CreateFlashcardParams{
			UserID: user.ID, Front: "f", Back: "b",
		}); err != nil {
			t.Fatalf("seed flashcard: %v", err)
		}
		return user.ID
	}

	expiredGuest := seed(true, 40*24*time.Hour)   // reaped
	recentGuest := seed(true, 1*24*time.Hour)     // survives (younger than TTL)
	oldRegistered := seed(false, 90*24*time.Hour) // survives (not anonymous)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = ANY($1)",
			[]uuid.UUID{expiredGuest, recentGuest, oldRegistered})
	})

	repo := NewReaperRepo(pool)
	reaped, err := repo.DeleteExpiredAnonymousUsers(ctx, time.Now().Add(-30*24*time.Hour))
	if err != nil {
		t.Fatalf("DeleteExpiredAnonymousUsers: %v", err)
	}

	reapedIDs := map[uuid.UUID]bool{}
	for _, row := range reaped {
		reapedIDs[row.ID] = true
	}
	if !reapedIDs[expiredGuest] {
		t.Error("expired guest was not reaped")
	}
	if reapedIDs[recentGuest] {
		t.Error("recent guest was reaped — TTL not respected")
	}
	if reapedIDs[oldRegistered] {
		t.Error("registered user was reaped — is_anonymous not respected")
	}

	var userExists, cardExists bool
	if err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", expiredGuest).Scan(&userExists); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM flashcards WHERE user_id = $1)", expiredGuest).Scan(&cardExists); err != nil {
		t.Fatal(err)
	}
	if userExists {
		t.Error("expired guest row still present")
	}
	if cardExists {
		t.Error("expired guest's flashcards did not cascade")
	}
}
