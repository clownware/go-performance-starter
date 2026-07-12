package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/clownware/go-performance-starter/internal/database"
)

// OrganizationMemberRepository defines the interface for organization member data access operations.
type OrganizationMemberRepository interface {
	// Get retrieves an organization member by ID
	Get(ctx context.Context, id uuid.UUID) (*database.OrganizationMember, error)

	// GetByUserAndOrg retrieves a member relationship by user ID and organization ID
	GetByUserAndOrg(ctx context.Context, userID, orgID uuid.UUID) (*database.OrganizationMember, error)

	// List retrieves all members for an organization with pagination
	List(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]*database.OrganizationMember, error)

	// Create adds a new organization member
	Create(ctx context.Context, params database.CreateOrganizationMemberParams) (*database.OrganizationMember, error)

	// Update modifies an existing organization member
	Update(ctx context.Context, params database.UpdateOrganizationMemberParams) (*database.OrganizationMember, error)

	// Delete removes an organization member
	Delete(ctx context.Context, userID, orgID uuid.UUID) error

	// SetPrimary marks an organization as the primary one for a user
	SetPrimary(ctx context.Context, userID, orgID uuid.UUID) error

	// Count returns the number of members in an organization
	Count(ctx context.Context, orgID uuid.UUID) (int64, error)
}
