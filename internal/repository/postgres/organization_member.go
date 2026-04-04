package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/clownware/alpine-go-performance-starter/internal/database"
	"github.com/clownware/alpine-go-performance-starter/internal/repository"
)

// OrganizationMemberRepo implements the repository.OrganizationMemberRepository interface using PostgreSQL.
type OrganizationMemberRepo struct {
	db      *pgxpool.Pool
	querier database.Querier
}

// NewOrganizationMemberRepo creates a new OrganizationMemberRepo instance.
func NewOrganizationMemberRepo(db *pgxpool.Pool, querier database.Querier) *OrganizationMemberRepo {
	return &OrganizationMemberRepo{
		db:      db,
		querier: querier,
	}
}

// Get retrieves an organization member by ID.
func (r *OrganizationMemberRepo) Get(ctx context.Context, id uuid.UUID) (*database.OrganizationMember, error) {
	member, err := r.querier.GetOrganizationMember(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &member, nil
}

// GetByUserAndOrg retrieves a member relationship by user ID and organization ID.
func (r *OrganizationMemberRepo) GetByUserAndOrg(ctx context.Context, userID, orgID uuid.UUID) (*database.OrganizationMember, error) {
	member, err := r.querier.GetOrganizationMemberByUserAndOrg(ctx, database.GetOrganizationMemberByUserAndOrgParams{
		UserID:         userID,
		OrganizationID: orgID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &member, nil
}

// List retrieves all members for an organization with pagination.
func (r *OrganizationMemberRepo) List(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]database.OrganizationMember, error) {
	return r.querier.ListOrganizationMembers(ctx, database.ListOrganizationMembersParams{
		OrganizationID: orgID,
		Limit:          limit,
		Offset:         offset,
	})
}

// Create adds a new organization member.
func (r *OrganizationMemberRepo) Create(ctx context.Context, params database.CreateOrganizationMemberParams) (*database.OrganizationMember, error) {
	member, err := r.querier.CreateOrganizationMember(ctx, params)
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// Update modifies an existing organization member.
func (r *OrganizationMemberRepo) Update(ctx context.Context, params database.UpdateOrganizationMemberParams) (*database.OrganizationMember, error) {
	member, err := r.querier.UpdateOrganizationMember(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &member, nil
}

// Delete removes an organization member.
func (r *OrganizationMemberRepo) Delete(ctx context.Context, userID, orgID uuid.UUID) error {
	return r.querier.DeleteOrganizationMember(ctx, database.DeleteOrganizationMemberParams{
		UserID:         userID,
		OrganizationID: orgID,
	})
}

// SetPrimary marks an organization as the primary one for a user.
func (r *OrganizationMemberRepo) SetPrimary(ctx context.Context, userID, orgID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }() // Rollback is a no-op if tx has already been committed.

	qtx := database.New(tx) // Create a new Querier instance with the transaction

	// Step 1: Set the target membership as primary
	err = qtx.SetPrimaryOrganizationStep1(ctx, database.SetPrimaryOrganizationStep1Params{
		OrganizationID: orgID,
		UserID:         userID,
	})
	if err != nil {
		return err
	}

	// Step 2: Set all other memberships for the user as non-primary
	err = qtx.SetPrimaryOrganizationStep2(ctx, database.SetPrimaryOrganizationStep2Params{
		UserID:         userID, // User ID from step 1
		OrganizationID: orgID,  // Org ID from step 1 (to exclude)
	})
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Count returns the number of members in an organization.
func (r *OrganizationMemberRepo) Count(ctx context.Context, orgID uuid.UUID) (int64, error) {
	return r.querier.CountOrganizationMembers(ctx, orgID)
}
