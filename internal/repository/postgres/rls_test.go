package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clownware/go-performance-starter/internal/database"
	"github.com/clownware/go-performance-starter/internal/repository"
)

// seedUserWithAuth creates a user and returns both its id and its auth_id, so a
// test can authenticate as that user via request.jwt.claim.sub.
func seedUserWithAuth(ctx context.Context, t *testing.T, q *database.Queries) (uuid.UUID, string) {
	t.Helper()
	authID := uuid.NewString()
	user, err := q.CreateUser(ctx, database.CreateUserParams{
		Email:  authID + "@example.com",
		AuthID: pgtype.Text{String: authID, Valid: true},
	})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return user.ID, authID
}

// TestFlashcardRLSIsolation proves the RLS self-access policy: an authenticated
// user cannot read another user's flashcards. This is the same mechanism that
// scopes anonymous guests (a guest is just a user row whose auth_id is the
// anonymous auth.uid()).
//
// Seeding happens as the owner role (RLS bypassed); the actual assertions run
// after SET ROLE authenticated, so RLS is enforced.
func TestFlashcardRLSIsolation(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := database.New(tx)
	userA, authA := seedUserWithAuth(ctx, t, q)
	userB, _ := seedUserWithAuth(ctx, t, q)

	cardA, err := q.CreateFlashcard(ctx, database.CreateFlashcardParams{UserID: userA, Front: "a-front", Back: "a-back"})
	if err != nil {
		t.Fatalf("create card A: %v", err)
	}
	cardB, err := q.CreateFlashcard(ctx, database.CreateFlashcardParams{UserID: userB, Front: "b-front", Back: "b-back"})
	if err != nil {
		t.Fatalf("create card B: %v", err)
	}

	// Become authenticated user A. SET LOCAL is rolled back with the tx.
	if _, err := tx.Exec(ctx, "SET LOCAL ROLE authenticated"); err != nil {
		t.Fatalf("set role authenticated: %v", err)
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('request.jwt.claim.sub', $1, true)", authA); err != nil {
		t.Fatalf("set jwt claim: %v", err)
	}

	repo := NewFlashcardRepo(nil, q)

	// A can read its own card.
	if _, err := repo.Get(ctx, cardA.ID); err != nil {
		t.Errorf("user A reading own card: err = %v, want nil", err)
	}

	// A cannot read B's card — RLS hides it (returns no rows → ErrNotFound).
	if _, err := repo.Get(ctx, cardB.ID); err != repository.ErrNotFound {
		t.Errorf("user A reading B's card: err = %v, want ErrNotFound (RLS not enforced?)", err)
	}

	// Listing B's cards as A yields nothing, even though the query filters by
	// B's id — RLS denies the rows.
	list, err := repo.ListByUser(ctx, userB)
	if err != nil {
		t.Fatalf("list B's cards as A: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("user A listing B's cards = %d rows, want 0", len(list))
	}
}
