package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/clownware/alpine-go-performance-starter/internal/database"
)

// OrganizationRepository defines the interface for organization data access operations.
type OrganizationRepository interface {
	// Get retrieves an organization by ID
	Get(ctx context.Context, id uuid.UUID) (*database.Organization, error)

	// GetBySlug retrieves an organization by slug
	GetBySlug(ctx context.Context, slug string) (*database.Organization, error)

	// List retrieves organizations with pagination
	List(ctx context.Context, limit, offset int32) ([]*database.Organization, error)

	// Create adds a new organization
	Create(ctx context.Context, params database.CreateOrganizationParams) (*database.Organization, error)

	// Update modifies an existing organization
	Update(ctx context.Context, params database.UpdateOrganizationParams) (*database.Organization, error)

	// Delete soft-deletes an organization by setting is_active to false
	Delete(ctx context.Context, id uuid.UUID) error

	// ListForUser retrieves all organizations that a user is a member of
	ListForUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*database.Organization, error)

	// GetPrimaryForUser retrieves the primary organization for a user
	GetPrimaryForUser(ctx context.Context, userID uuid.UUID) (*database.Organization, error)
}
