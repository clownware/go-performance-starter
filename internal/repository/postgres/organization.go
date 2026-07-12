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

// OrganizationRepo implements the repository.OrganizationRepository interface
// using PostgreSQL. All methods run through inScope so RLS evaluates against
// the requester (ADR-004).
type OrganizationRepo struct {
	db      *pgxpool.Pool
	querier database.Querier
}

// NewOrganizationRepo creates a new OrganizationRepo instance.
func NewOrganizationRepo(db *pgxpool.Pool, querier database.Querier) *OrganizationRepo {
	return &OrganizationRepo{
		db:      db,
		querier: querier,
	}
}

// Get retrieves an organization by ID.
func (r *OrganizationRepo) Get(ctx context.Context, id uuid.UUID) (*database.Organization, error) {
	org, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.Organization, error) {
		return q.GetOrganization(ctx, id)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &org, nil
}

// GetBySlug retrieves an organization by its slug.
func (r *OrganizationRepo) GetBySlug(ctx context.Context, slug string) (*database.Organization, error) {
	org, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.Organization, error) {
		return q.GetOrganizationBySlug(ctx, slug)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &org, nil
}

// List retrieves all organizations with pagination.
func (r *OrganizationRepo) List(ctx context.Context, limit, offset int32) ([]database.Organization, error) {
	return inScope(ctx, r.db, r.querier, func(q database.Querier) ([]database.Organization, error) {
		return q.ListOrganizations(ctx, database.ListOrganizationsParams{
			Limit:  limit,
			Offset: offset,
		})
	})
}

// Create adds a new organization.
func (r *OrganizationRepo) Create(ctx context.Context, params database.CreateOrganizationParams) (*database.Organization, error) {
	org, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.Organization, error) {
		return q.CreateOrganization(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// Update modifies an existing organization.
func (r *OrganizationRepo) Update(ctx context.Context, params database.UpdateOrganizationParams) (*database.Organization, error) {
	org, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.Organization, error) {
		return q.UpdateOrganization(ctx, params)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &org, nil
}

// Delete marks an organization as inactive.
func (r *OrganizationRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (struct{}, error) {
		return struct{}{}, q.DeleteOrganization(ctx, id)
	})
	return err
}

// ListForUser retrieves all organizations that a user is a member of.
func (r *OrganizationRepo) ListForUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]database.Organization, error) {
	return inScope(ctx, r.db, r.querier, func(q database.Querier) ([]database.Organization, error) {
		return q.ListUserOrganizations(ctx, database.ListUserOrganizationsParams{
			UserID: userID,
			Limit:  limit,
			Offset: offset,
		})
	})
}

// GetPrimaryForUser retrieves the primary organization for a user.
func (r *OrganizationRepo) GetPrimaryForUser(ctx context.Context, userID uuid.UUID) (*database.Organization, error) {
	org, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.Organization, error) {
		return q.GetUserPrimaryOrganization(ctx, userID)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound // No primary org set or user not found
		}
		return nil, err
	}
	return &org, nil
}
