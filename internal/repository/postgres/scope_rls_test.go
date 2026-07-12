package postgres

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clownware/go-performance-starter/internal/database"
	"github.com/clownware/go-performance-starter/internal/repository"
	"github.com/clownware/go-performance-starter/internal/webutil"
)

// TestScopedRepoRLSIsolation proves the repositories engage RLS at runtime
// through context claims — the production path (inScope opens its own
// transaction from the pool and applies SET LOCAL ROLE + request.jwt.claims).
// This is the end-to-end counterpart of TestFlashcardRLSIsolation, which
// exercises the policies but injects the role/claims by hand.
//
// Fixtures must be committed (inScope's transactions can't see another
// transaction's uncommitted rows), so cleanup deletes them explicitly.
func TestScopedRepoRLSIsolation(t *testing.T) {
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

	// Seed committed fixtures as the connection role (superuser in CI).
	q := database.New(pool)
	userA, authA := seedUserWithAuth(ctx, t, q)
	userB, _ := seedUserWithAuth(ctx, t, q)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM flashcards WHERE user_id = ANY($1)", []uuid.UUID{userA, userB})
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = ANY($1)", []uuid.UUID{userA, userB})
	})

	cardA, err := q.CreateFlashcard(ctx, database.CreateFlashcardParams{UserID: userA, Front: "a-front", Back: "a-back"})
	if err != nil {
		t.Fatalf("create card A: %v", err)
	}
	cardB, err := q.CreateFlashcard(ctx, database.CreateFlashcardParams{UserID: userB, Front: "b-front", Back: "b-back"})
	if err != nil {
		t.Fatalf("create card B: %v", err)
	}

	// The repo gets the real pool — claims come only from the context,
	// exactly as in production.
	repo := NewFlashcardRepo(pool, q)
	ctxA := webutil.WithAuthClaims(ctx, webutil.AuthClaims{Sub: authA, Role: webutil.RoleAuthenticated})

	// A can read its own card through the scoped repo.
	if _, err := repo.Get(ctxA, cardA.ID); err != nil {
		t.Errorf("user A reading own card: err = %v, want nil", err)
	}

	// A cannot read B's card — RLS hides it.
	if _, err := repo.Get(ctxA, cardB.ID); err != repository.ErrNotFound {
		t.Errorf("user A reading B's card: err = %v, want ErrNotFound (claims not applied?)", err)
	}

	// Listing B's cards as A yields nothing despite filtering by B's id.
	list, err := repo.ListByUser(ctxA, userB)
	if err != nil {
		t.Fatalf("list B's cards as A: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("user A listing B's cards = %d rows, want 0", len(list))
	}

	// A cannot delete B's card; B still has it afterwards.
	if err := repo.Delete(ctxA, cardB.ID, userB); err != nil {
		t.Fatalf("delete B's card as A: %v", err)
	}
	var cardBExists bool
	if err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM flashcards WHERE id = $1)", cardB.ID).Scan(&cardBExists); err != nil {
		t.Fatalf("check card B: %v", err)
	}
	if !cardBExists {
		t.Error("user A deleted B's card — RLS write protection failed")
	}
}

// TestScopedUserProvisioning proves the users_self_access WITH CHECK under the
// scoped path: an authenticated identity can insert its own users row (the
// UserLoader JIT-provisioning path) but not a row for someone else.
func TestScopedUserProvisioning(t *testing.T) {
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

	repo := NewUserRepo(pool, database.New(pool))
	sub := uuid.NewString()
	ctxUser := webutil.WithAuthClaims(ctx, webutil.AuthClaims{Sub: sub, Role: webutil.RoleAuthenticated})

	// Creating one's own row passes WITH CHECK.
	created, err := repo.Create(ctxUser, database.CreateUserParams{
		Email:  sub + "@example.com",
		AuthID: pgtype.Text{String: sub, Valid: true},
	})
	if err != nil {
		t.Fatalf("self-provisioning own users row: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", created.ID)
	})

	// Creating a row for a different auth identity violates WITH CHECK.
	otherSub := uuid.NewString()
	if _, err := repo.Create(ctxUser, database.CreateUserParams{
		Email:  otherSub + "@example.com",
		AuthID: pgtype.Text{String: otherSub, Valid: true},
	}); err == nil || !strings.Contains(err.Error(), "row-level security") {
		t.Errorf("creating another identity's row: err = %v, want RLS violation", err)
	}
}
