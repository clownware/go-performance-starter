package postgres

import (
	"context"
	"os"
	"testing"
	"time"

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

	renamed, err := repo.UpdateName(ctx, created.ID, "Ada Lovelace")
	if err != nil {
		t.Fatalf("UpdateName: %v", err)
	}
	if !renamed.Name.Valid || renamed.Name.String != "Ada Lovelace" {
		t.Errorf("UpdateName name = %+v, want Ada Lovelace", renamed.Name)
	}
	if renamed.Email != email {
		t.Errorf("UpdateName must not touch email: got %q, want %q", renamed.Email, email)
	}
	if _, err := repo.UpdateName(ctx, uuid.New(), "Nobody"); err != repository.ErrNotFound {
		t.Errorf("UpdateName(missing) err = %v, want ErrNotFound", err)
	}

	if err := repo.SetAnonymous(ctx, created.ID, false); err != nil {
		t.Fatalf("SetAnonymous: %v", err)
	}
	promoted, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get after promotion: %v", err)
	}
	if promoted.IsAnonymous {
		t.Error("SetAnonymous(false) did not promote the row")
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

// TestUserRepoLifecycleIntegration covers the mutation methods the demo's
// login/onboarding/reaper paths call: List, Update (the upgrade flow's email
// sync, #68), SetLastLogin, UpdateFirstRunComplete, and soft Delete.
func TestUserRepoLifecycleIntegration(t *testing.T) {
	ctx, q := withTx(t)
	repo := NewUserRepo(nil, q)

	first := seedUser(ctx, t, q)
	second := seedUser(ctx, t, q)

	all, err := repo.List(ctx, 1000, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := map[uuid.UUID]bool{}
	for _, u := range all {
		found[u.ID] = true
	}
	if !found[first] || !found[second] {
		t.Errorf("List missing seeded users (got %d rows)", len(all))
	}
	one, err := repo.List(ctx, 1, 0)
	if err != nil {
		t.Fatalf("List(limit 1): %v", err)
	}
	if len(one) != 1 {
		t.Errorf("List(limit 1) len = %d, want 1", len(one))
	}

	newEmail := "upgraded-" + uuid.NewString() + "@example.com"
	updated, err := repo.Update(ctx, database.UpdateUserParams{ID: first, Email: newEmail})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Email != newEmail {
		t.Errorf("Update email = %q, want %q", updated.Email, newEmail)
	}
	if _, err := repo.Update(ctx, database.UpdateUserParams{ID: uuid.New(), Email: "x@example.com"}); err != repository.ErrNotFound {
		t.Errorf("Update(missing) err = %v, want ErrNotFound", err)
	}

	loginAt := time.Now().Truncate(time.Second)
	if err := repo.SetLastLogin(ctx, first, loginAt); err != nil {
		t.Fatalf("SetLastLogin: %v", err)
	}
	afterLogin, err := repo.Get(ctx, first)
	if err != nil {
		t.Fatalf("Get after SetLastLogin: %v", err)
	}
	if !afterLogin.LastLoginAt.Valid || afterLogin.LastLoginAt.Time.Unix() != loginAt.Unix() {
		t.Errorf("LastLoginAt = %+v, want %v", afterLogin.LastLoginAt, loginAt)
	}
	if afterLogin.Email != newEmail {
		t.Errorf("SetLastLogin must not touch email: got %q", afterLogin.Email)
	}
	if err := repo.SetLastLogin(ctx, uuid.New(), loginAt); err != repository.ErrNotFound {
		t.Errorf("SetLastLogin(missing) err = %v, want ErrNotFound", err)
	}

	if err := repo.UpdateFirstRunComplete(ctx, first, true); err != nil {
		t.Fatalf("UpdateFirstRunComplete: %v", err)
	}
	onboarded, err := repo.Get(ctx, first)
	if err != nil {
		t.Fatalf("Get after UpdateFirstRunComplete: %v", err)
	}
	if !onboarded.FirstRunComplete {
		t.Error("UpdateFirstRunComplete(true) did not set the flag")
	}

	if err := repo.Delete(ctx, first); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	deleted, err := repo.Get(ctx, first)
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if deleted.IsActive.Bool {
		t.Error("Delete should soft-delete (is_active=false), not remove the row")
	}
}
