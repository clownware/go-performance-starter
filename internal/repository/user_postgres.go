package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clownware/alpine-go-performance-starter/internal/database"
)

type userRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository returns a Postgres-backed UserRepository
func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{db: db}
}

// UpdateFirstRunComplete sets the onboarding completion flag for a user
func (r *userRepository) UpdateFirstRunComplete(ctx context.Context, id uuid.UUID, complete bool) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET first_run_complete = $1, updated_at = NOW() WHERE id = $2`, complete, id)
	return err
}

// Get retrieves a user by ID (type-correct stub)
func (r *userRepository) Get(ctx context.Context, id uuid.UUID) (*database.User, error) {
	return nil, nil
}

// GetByEmail retrieves a user by email (type-correct stub)
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*database.User, error) {
	return nil, nil
}

// GetByAuthID retrieves a user by auth ID (type-correct stub)
func (r *userRepository) GetByAuthID(ctx context.Context, authID string) (*database.User, error) {
	return nil, nil
}

// List retrieves users with pagination (type-correct stub)
func (r *userRepository) List(ctx context.Context, limit, offset int32) ([]*database.User, error) {
	return nil, nil
}

// Create adds a new user (type-correct stub)
func (r *userRepository) Create(ctx context.Context, params database.CreateUserParams) (*database.User, error) {
	return nil, nil
}

// Update modifies an existing user (type-correct stub)
func (r *userRepository) Update(ctx context.Context, params database.UpdateUserParams) (*database.User, error) {
	return nil, nil
}

// Delete soft-deletes a user by setting is_active to false (type-correct stub)
func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

// RecordLogin updates the last_login_at timestamp for a user (type-correct stub)
func (r *userRepository) RecordLogin(ctx context.Context, id uuid.UUID, loginTime time.Time) error {
	return nil
}
