package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clownware/go-performance-starter/internal/database"
	"github.com/clownware/go-performance-starter/internal/repository"
)

// OrganizationMemberRepo implements the repository.OrganizationMemberRepository
// interface using PostgreSQL. All methods run through inScope so RLS evaluates
// against the requester (ADR-004).
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
	member, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.OrganizationMember, error) {
		return q.GetOrganizationMember(ctx, id)
	})
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
	member, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.OrganizationMember, error) {
		return q.GetOrganizationMemberByUserAndOrg(ctx, database.GetOrganizationMemberByUserAndOrgParams{
			UserID:         userID,
			OrganizationID: orgID,
		})
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
	return inScope(ctx, r.db, r.querier, func(q database.Querier) ([]database.OrganizationMember, error) {
		return q.ListOrganizationMembers(ctx, database.ListOrganizationMembersParams{
			OrganizationID: orgID,
			Limit:          limit,
			Offset:         offset,
		})
	})
}

// Create adds a new organization member.
func (r *OrganizationMemberRepo) Create(ctx context.Context, params database.CreateOrganizationMemberParams) (*database.OrganizationMember, error) {
	member, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.OrganizationMember, error) {
		return q.CreateOrganizationMember(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// Update modifies an existing organization member.
func (r *OrganizationMemberRepo) Update(ctx context.Context, params database.UpdateOrganizationMemberParams) (*database.OrganizationMember, error) {
	member, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.OrganizationMember, error) {
		return q.UpdateOrganizationMember(ctx, params)
	})
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
	_, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (struct{}, error) {
		return struct{}{}, q.DeleteOrganizationMember(ctx, database.DeleteOrganizationMemberParams{
			UserID:         userID,
			OrganizationID: orgID,
		})
	})
	return err
}

// SetPrimary marks an organization as the primary one for a user. Both steps
// of the two-step pattern (ADR-005) run in the single transaction inScope
// opens, preserving atomicity.
func (r *OrganizationMemberRepo) SetPrimary(ctx context.Context, userID, orgID uuid.UUID) error {
	_, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (struct{}, error) {
		// Step 1: Set the target membership as primary
		if err := q.SetPrimaryOrganizationStep1(ctx, database.SetPrimaryOrganizationStep1Params{
			OrganizationID: orgID,
			UserID:         userID,
		}); err != nil {
			return struct{}{}, err
		}
		// Step 2: Set all other memberships for the user as non-primary
		return struct{}{}, q.SetPrimaryOrganizationStep2(ctx, database.SetPrimaryOrganizationStep2Params{
			UserID:         userID, // User ID from step 1
			OrganizationID: orgID,  // Org ID from step 1 (to exclude)
		})
	})
	return err
}

// Count returns the number of members in an organization.
func (r *OrganizationMemberRepo) Count(ctx context.Context, orgID uuid.UUID) (int64, error) {
	return inScope(ctx, r.db, r.querier, func(q database.Querier) (int64, error) {
		return q.CountOrganizationMembers(ctx, orgID)
	})
}
