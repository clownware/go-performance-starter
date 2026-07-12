package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clownware/go-performance-starter/internal/database"
	"github.com/clownware/go-performance-starter/internal/repository"
)

// UserRepo implements the repository.UserRepository interface using PostgreSQL.
// All methods run through inScope so RLS evaluates against the requester
// (ADR-004); see scope.go.
type UserRepo struct {
	db      *pgxpool.Pool
	querier database.Querier
}

var _ repository.UserRepository = (*UserRepo)(nil)

// NewUserRepo creates a new UserRepo instance.
func NewUserRepo(db *pgxpool.Pool, querier database.Querier) *UserRepo {
	return &UserRepo{
		db:      db,
		querier: querier,
	}
}

// Get retrieves a user by ID.
func (r *UserRepo) Get(ctx context.Context, id uuid.UUID) (*database.User, error) {
	user, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.User, error) {
		return q.GetUser(ctx, id)
	})
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
	user, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.User, error) {
		return q.GetUserByEmail(ctx, email)
	})
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
	user, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.User, error) {
		return q.GetUserByAuthID(ctx, pgtype.Text{String: authID, Valid: true})
	})
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
	return inScope(ctx, r.db, r.querier, func(q database.Querier) ([]database.User, error) {
		return q.ListUsers(ctx, database.ListUsersParams{
			Limit:  limit,
			Offset: offset,
		})
	})
}

// Create adds a new user.
func (r *UserRepo) Create(ctx context.Context, params database.CreateUserParams) (*database.User, error) {
	user, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.User, error) {
		return q.CreateUser(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update modifies an existing user.
func (r *UserRepo) Update(ctx context.Context, params database.UpdateUserParams) (*database.User, error) {
	user, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.User, error) {
		return q.UpdateUser(ctx, params)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// UpdateName sets the user's display name (profile self-service, #70).
func (r *UserRepo) UpdateName(ctx context.Context, id uuid.UUID, name string) (*database.User, error) {
	user, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.User, error) {
		return q.UpdateUserName(ctx, database.UpdateUserNameParams{
			ID:   id,
			Name: pgtype.Text{String: name, Valid: true},
		})
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// SetAnonymous flips the guest flag (upgrade flow, #68).
func (r *UserRepo) SetAnonymous(ctx context.Context, id uuid.UUID, anonymous bool) error {
	_, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (struct{}, error) {
		return struct{}{}, q.SetUserIsAnonymous(ctx, database.SetUserIsAnonymousParams{
			ID:          id,
			IsAnonymous: anonymous,
		})
	})
	return err
}

// Delete soft-deletes a user by setting is_active to false.
func (r *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (struct{}, error) {
		return struct{}{}, q.DeleteUser(ctx, id)
	})
	return err
}

// SetLastLogin updates the last login timestamp for a user.
func (r *UserRepo) SetLastLogin(ctx context.Context, id uuid.UUID, loginTime time.Time) error {
	_, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.User, error) {
		return q.UpdateUser(ctx, database.UpdateUserParams{
			ID:          id,
			LastLoginAt: pgtype.Timestamptz{Time: loginTime, Valid: true},
		})
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repository.ErrNotFound
		}
		return err
	}
	return nil
}

// UpdateFirstRunComplete sets the onboarding completion flag for a user.
func (r *UserRepo) UpdateFirstRunComplete(ctx context.Context, id uuid.UUID, complete bool) error {
	_, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (struct{}, error) {
		return struct{}{}, q.SetUserFirstRunComplete(ctx, database.SetUserFirstRunCompleteParams{
			ID:               id,
			FirstRunComplete: complete,
		})
	})
	return err
}
