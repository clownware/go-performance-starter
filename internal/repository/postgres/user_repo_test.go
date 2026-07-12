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

func TestUserRepoIntegration(t *testing.T) {
	ctx, q := withTx(t)
	repo := NewUserRepo(nil, q)

	authID := uuid.NewString()
	email := authID + "@example.com"
	created, err := repo.Create(ctx, database.CreateUserParams{
		Email:  email,
		AuthID: pgtype.Text{String: authID, Valid: true},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	byID, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if byID.Email != email {
		t.Errorf("Get email = %q, want %q", byID.Email, email)
	}

	byEmail, err := repo.GetByEmail(ctx, email)
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if byEmail.ID != created.ID {
		t.Errorf("GetByEmail id = %v, want %v", byEmail.ID, created.ID)
	}

	byAuth, err := repo.GetByAuthID(ctx, authID)
	if err != nil {
		t.Fatalf("GetByAuthID: %v", err)
	}
	if byAuth.ID != created.ID {
		t.Errorf("GetByAuthID id = %v, want %v", byAuth.ID, created.ID)
	}

	if _, err := repo.GetByEmail(ctx, "missing-"+uuid.NewString()+"@example.com"); err != repository.ErrNotFound {
		t.Errorf("GetByEmail(missing) err = %v, want ErrNotFound", err)
	}
}

// TestUserRLSIsolation proves the users_self_access policy: an authenticated
// user can read only its own user row. This is what keeps an anonymous guest's
// identity (a users row keyed by the anonymous auth.uid()) private.
func TestUserRLSIsolation(t *testing.T) {
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
	userB, authB := seedUserWithAuth(ctx, t, q)

	if _, err := tx.Exec(ctx, "SET LOCAL ROLE authenticated"); err != nil {
		t.Fatalf("set role authenticated: %v", err)
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('request.jwt.claim.sub', $1, true)", authA); err != nil {
		t.Fatalf("set jwt claim: %v", err)
	}

	repo := NewUserRepo(nil, q)

	if _, err := repo.Get(ctx, userA); err != nil {
		t.Errorf("A reading own user row: err = %v, want nil", err)
	}
	if _, err := repo.Get(ctx, userB); err != repository.ErrNotFound {
		t.Errorf("A reading B's user row: err = %v, want ErrNotFound (RLS not enforced?)", err)
	}
	if _, err := repo.GetByAuthID(ctx, authA); err != nil {
		t.Errorf("A GetByAuthID(self): err = %v, want nil", err)
	}
	if _, err := repo.GetByAuthID(ctx, authB); err != repository.ErrNotFound {
		t.Errorf("A GetByAuthID(B): err = %v, want ErrNotFound", err)
	}
}
