package fixtures

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourusername/go-alpine-saas-starter/internal/database"
)

// TestFixtures provides utilities for setting up test data.
type TestFixtures struct {
	DB         *pgxpool.Pool
	Queries    *database.Queries
	ctx        context.Context
	cleanupFns []func() error
}

// NewFixtures creates a new TestFixtures instance.
// The returned TestFixtures will use the provided database connection.
func NewFixtures(ctx context.Context, db *pgxpool.Pool) (*TestFixtures, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	// Ping the database to ensure it's reachable
	if err := db.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &TestFixtures{
		DB:      db,
		Queries: database.New(db),
		ctx:     ctx,
	}, nil
}

// CreateUser creates a test user with given or generated data.
func (f *TestFixtures) CreateUser(params database.CreateUserParams) (*database.User, error) {
	// Provide defaults for required fields if not specified
	if params.Email == "" {
		params.Email = fmt.Sprintf("test-%s@example.com", uuid.New().String()[:8])
	}

	user, err := f.Queries.CreateUser(f.ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create test user: %w", err)
	}

	// Register cleanup function to delete this user after tests
	f.cleanupFns = append(f.cleanupFns, func() error {
		return f.Queries.PermanentDeleteUser(f.ctx, user.ID)
	})

	return &user, nil
}

// CreateOrganization creates a test organization with given or generated data.
func (f *TestFixtures) CreateOrganization(params database.CreateOrganizationParams) (*database.Organization, error) {
	// Provide defaults for required fields if not specified
	if params.Name == "" {
		params.Name = fmt.Sprintf("Test Org %s", uuid.New().String()[:8])
	}
	if params.Slug == "" {
		params.Slug = fmt.Sprintf("test-org-%s", uuid.New().String()[:8])
	}

	org, err := f.Queries.CreateOrganization(f.ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create test organization: %w", err)
	}

	// Register cleanup function to delete this organization after tests
	f.cleanupFns = append(f.cleanupFns, func() error {
		return f.Queries.PermanentDeleteOrganization(f.ctx, org.ID)
	})

	return &org, nil
}

// CreateOrganizationMember creates a test membership between a user and organization.
func (f *TestFixtures) CreateOrganizationMember(params database.CreateOrganizationMemberParams) (*database.OrganizationMember, error) {
	member, err := f.Queries.CreateOrganizationMember(f.ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create test organization member: %w", err)
	}

	// Register cleanup function to delete this membership after tests
	f.cleanupFns = append(f.cleanupFns, func() error {
		return f.Queries.DeleteOrganizationMember(f.ctx, database.DeleteOrganizationMemberParams{
			OrganizationID: params.OrganizationID,
			UserID:         params.UserID,
		})
	})

	return &member, nil
}

// CreateFullTestAccount creates a user, organization, and membership in one step.
func (f *TestFixtures) CreateFullTestAccount(role string) (*database.User, *database.Organization, *database.OrganizationMember, error) {
	// Create user
	user, err := f.CreateUser(database.CreateUserParams{
		Email:     fmt.Sprintf("test-%s@example.com", uuid.New().String()[:8]),
		Name:      pgtype.Text{String: "Test User", Valid: true},
		AuthID:    pgtype.Text{String: uuid.New().String(), Valid: true},
		AvatarUrl: pgtype.Text{Valid: false},
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create test user: %w", err)
	}

	// Create organization
	org, err := f.CreateOrganization(database.CreateOrganizationParams{
		Name:              fmt.Sprintf("Test Org %s", uuid.New().String()[:8]),
		Slug:              fmt.Sprintf("test-org-%s", uuid.New().String()[:8]),
		PlanType:          pgtype.Text{String: "free", Valid: true},
		BillingCustomerID: pgtype.Text{Valid: false},
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create test organization: %w", err)
	}

	// Create membership
	if role == "" {
		role = "owner"
	}
	member, err := f.CreateOrganizationMember(database.CreateOrganizationMemberParams{
		OrganizationID:        org.ID,
		UserID:                user.ID,
		Role:                  role,
		IsPrimaryOrganization: pgtype.Bool{Bool: true, Valid: true},
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create test organization member: %w", err)
	}

	return user, org, member, nil
}

// Cleanup cleans up all test data created by this fixture.
func (f *TestFixtures) Cleanup() error {
	// Execute cleanup functions in reverse order
	for i := len(f.cleanupFns) - 1; i >= 0; i-- {
		if err := f.cleanupFns[i](); err != nil {
			return fmt.Errorf("cleanup failed: %w", err)
		}
	}

	// Reset the cleanup functions slice
	f.cleanupFns = nil
	return nil
}

// End of fixtures package
