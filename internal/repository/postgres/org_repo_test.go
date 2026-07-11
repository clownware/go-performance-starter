package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/clownware/alpine-go-performance-starter/internal/database"
	"github.com/clownware/alpine-go-performance-starter/internal/repository"
)

func TestOrganizationRepoIntegration(t *testing.T) {
	ctx, q := withTx(t)
	repo := NewOrganizationRepo(nil, q)

	slug := "org-" + uuid.NewString()
	created, err := repo.Create(ctx, database.CreateOrganizationParams{
		Name: "Acme",
		Slug: slug,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !created.IsActive.Bool {
		t.Error("new organization should default to is_active=true")
	}

	byID, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if byID.Slug != slug {
		t.Errorf("Get slug = %q, want %q", byID.Slug, slug)
	}

	bySlug, err := repo.GetBySlug(ctx, slug)
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if bySlug.ID != created.ID {
		t.Errorf("GetBySlug id = %v, want %v", bySlug.ID, created.ID)
	}

	updated, err := repo.Update(ctx, database.UpdateOrganizationParams{
		ID:       created.ID,
		Name:     "Acme Renamed",
		Slug:     slug,
		PlanType: pgtype.Text{String: "pro", Valid: true},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Acme Renamed" || updated.PlanType.String != "pro" {
		t.Errorf("Update = (%q, %q), want (Acme Renamed, pro)", updated.Name, updated.PlanType.String)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	afterDelete, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if afterDelete.IsActive.Bool {
		t.Error("Delete should soft-delete (is_active=false)")
	}

	if _, err := repo.Get(ctx, uuid.New()); err != repository.ErrNotFound {
		t.Errorf("Get(missing) err = %v, want ErrNotFound", err)
	}
	if _, err := repo.GetBySlug(ctx, "missing-"+uuid.NewString()); err != repository.ErrNotFound {
		t.Errorf("GetBySlug(missing) err = %v, want ErrNotFound", err)
	}
	if _, err := repo.Update(ctx, database.UpdateOrganizationParams{ID: uuid.New(), Name: "x", Slug: "x-" + uuid.NewString()}); err != repository.ErrNotFound {
		t.Errorf("Update(missing) err = %v, want ErrNotFound", err)
	}
}

func seedOrg(ctx context.Context, t *testing.T, repo *OrganizationRepo, name string) *database.Organization {
	t.Helper()
	org, err := repo.Create(ctx, database.CreateOrganizationParams{
		Name: name,
		Slug: name + "-" + uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("seed org %s: %v", name, err)
	}
	return org
}

func TestOrganizationMemberRepoIntegration(t *testing.T) {
	ctx, q := withTx(t)
	orgRepo := NewOrganizationRepo(nil, q)
	memberRepo := NewOrganizationMemberRepo(nil, q)
	userID := seedUser(ctx, t, q)

	orgA := seedOrg(ctx, t, orgRepo, "org-a")
	orgB := seedOrg(ctx, t, orgRepo, "org-b")

	memberA, err := memberRepo.Create(ctx, database.CreateOrganizationMemberParams{
		OrganizationID:        orgA.ID,
		UserID:                userID,
		Role:                  "owner",
		IsPrimaryOrganization: pgtype.Bool{Bool: true, Valid: true},
	})
	if err != nil {
		t.Fatalf("Create member A: %v", err)
	}
	if _, err := memberRepo.Create(ctx, database.CreateOrganizationMemberParams{
		OrganizationID: orgB.ID,
		UserID:         userID,
		Role:           "member",
	}); err != nil {
		t.Fatalf("Create member B: %v", err)
	}

	byID, err := memberRepo.Get(ctx, memberA.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if byID.Role != "owner" {
		t.Errorf("Get role = %q, want owner", byID.Role)
	}

	byUserAndOrg, err := memberRepo.GetByUserAndOrg(ctx, userID, orgA.ID)
	if err != nil {
		t.Fatalf("GetByUserAndOrg: %v", err)
	}
	if byUserAndOrg.ID != memberA.ID {
		t.Errorf("GetByUserAndOrg id = %v, want %v", byUserAndOrg.ID, memberA.ID)
	}

	count, err := memberRepo.Count(ctx, orgA.ID)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Errorf("Count = %d, want 1", count)
	}

	members, err := memberRepo.List(ctx, orgA.ID, 10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(members) != 1 {
		t.Errorf("List len = %d, want 1", len(members))
	}

	userOrgs, err := orgRepo.ListForUser(ctx, userID, 10, 0)
	if err != nil {
		t.Fatalf("ListForUser: %v", err)
	}
	if len(userOrgs) != 2 {
		t.Errorf("ListForUser len = %d, want 2", len(userOrgs))
	}

	// Flip the primary org and verify both sides of the two-step update (ADR-005).
	if err := memberRepo.SetPrimary(ctx, userID, orgB.ID); err != nil {
		t.Fatalf("SetPrimary: %v", err)
	}
	primary, err := orgRepo.GetPrimaryForUser(ctx, userID)
	if err != nil {
		t.Fatalf("GetPrimaryForUser: %v", err)
	}
	if primary.ID != orgB.ID {
		t.Errorf("GetPrimaryForUser = %v, want %v (org B)", primary.ID, orgB.ID)
	}
	demoted, err := memberRepo.GetByUserAndOrg(ctx, userID, orgA.ID)
	if err != nil {
		t.Fatalf("GetByUserAndOrg after SetPrimary: %v", err)
	}
	if demoted.IsPrimaryOrganization.Bool {
		t.Error("org A membership should no longer be primary after SetPrimary(org B)")
	}

	if err := memberRepo.Delete(ctx, userID, orgA.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := memberRepo.GetByUserAndOrg(ctx, userID, orgA.ID); err != repository.ErrNotFound {
		t.Errorf("GetByUserAndOrg after delete err = %v, want ErrNotFound", err)
	}

	if _, err := memberRepo.Get(ctx, uuid.New()); err != repository.ErrNotFound {
		t.Errorf("Get(missing) err = %v, want ErrNotFound", err)
	}
}
