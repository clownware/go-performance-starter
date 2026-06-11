package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/clownware/alpine-go-performance-starter/internal/database"
	"github.com/clownware/alpine-go-performance-starter/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepo implements the repository.UserRepository interface using PostgreSQL.
type UserRepo struct {
	db      *pgxpool.Pool
	querier database.Querier
}

// NewUserRepo creates a new UserRepo instance.
func NewUserRepo(db *pgxpool.Pool, querier database.Querier) *UserRepo {
	return &UserRepo{
		db:      db,
		querier: querier,
	}
}

// Get retrieves a user by ID.
func (r *UserRepo) Get(ctx context.Context, id uuid.UUID) (*database.User, error) {
	user, err := r.querier.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail retrieves a user by email.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*database.User, error) {
	user, err := r.querier.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetByAuthID retrieves a user by authentication provider ID.
func (r *UserRepo) GetByAuthID(ctx context.Context, authID string) (*database.User, error) {
	user, err := r.querier.GetUserByAuthID(ctx, pgtype.Text{String: authID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// List retrieves users with pagination.
func (r *UserRepo) List(ctx context.Context, limit, offset int32) ([]database.User, error) {
	return r.querier.ListUsers(ctx, database.ListUsersParams{
		Limit:  limit,
		Offset: offset,
	})
}

// Create adds a new user.
func (r *UserRepo) Create(ctx context.Context, params database.CreateUserParams) (*database.User, error) {
	user, err := r.querier.CreateUser(ctx, params)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update modifies an existing user.
func (r *UserRepo) Update(ctx context.Context, params database.UpdateUserParams) (*database.User, error) {
	user, err := r.querier.UpdateUser(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// Delete soft-deletes a user by setting is_active to false.
func (r *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.querier.DeleteUser(ctx, id)
}

// SetLastLogin updates the last login timestamp for a user.
func (r *UserRepo) SetLastLogin(ctx context.Context, id uuid.UUID, loginTime time.Time) error {
	_, err := r.querier.UpdateUser(ctx, database.UpdateUserParams{
		ID:          id,
		LastLoginAt: pgtype.Timestamptz{Time: loginTime, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repository.ErrNotFound
		}
		return err
	}
	return nil
}
