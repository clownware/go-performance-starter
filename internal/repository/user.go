package repository

import (
	"context"
	"github.com/google/uuid"
	"github.com/yourusername/go-alpine-saas-starter/internal/database"
	"time"
)

// UserRepository defines the interface for user data access operations.
type UserRepository interface {
	// Get retrieves a user by ID
	Get(ctx context.Context, id uuid.UUID) (*database.User, error)
	
	// GetByEmail retrieves a user by email
	GetByEmail(ctx context.Context, email string) (*database.User, error)
	
	// GetByAuthID retrieves a user by auth ID (Supabase Auth integration)
	GetByAuthID(ctx context.Context, authID string) (*database.User, error)
	
	// List retrieves users with pagination
	List(ctx context.Context, limit, offset int32) ([]*database.User, error)
	
	// Create adds a new user
	Create(ctx context.Context, params database.CreateUserParams) (*database.User, error)
	
	// Update modifies an existing user
	Update(ctx context.Context, params database.UpdateUserParams) (*database.User, error)
	
	// Delete soft-deletes a user by setting is_active to false
	Delete(ctx context.Context, id uuid.UUID) error
	
	// RecordLogin updates the last_login_at timestamp for a user
	RecordLogin(ctx context.Context, id uuid.UUID, loginTime time.Time) error
}

// RecordLoginParams represents the parameters for recording a user login
type RecordLoginParams struct {
	ID         uuid.UUID
	LastLoginAt time.Time
}
