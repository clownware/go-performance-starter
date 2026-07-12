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

// Ensure FlashcardRepo satisfies the repository interface at compile time.
var _ repository.FlashcardRepository = (*FlashcardRepo)(nil)

// FlashcardRepo implements repository.FlashcardRepository using PostgreSQL.
// All methods run through inScope so RLS evaluates against the requester
// (ADR-004).
type FlashcardRepo struct {
	db      *pgxpool.Pool
	querier database.Querier
}

// NewFlashcardRepo creates a new FlashcardRepo instance.
func NewFlashcardRepo(db *pgxpool.Pool, querier database.Querier) *FlashcardRepo {
	return &FlashcardRepo{db: db, querier: querier}
}

// Create adds a flashcard for a user.
func (r *FlashcardRepo) Create(ctx context.Context, params database.CreateFlashcardParams) (*database.Flashcard, error) {
	card, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.Flashcard, error) {
		return q.CreateFlashcard(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	return &card, nil
}

// Get retrieves a flashcard by ID.
func (r *FlashcardRepo) Get(ctx context.Context, id uuid.UUID) (*database.Flashcard, error) {
	card, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.Flashcard, error) {
		return q.GetFlashcard(ctx, id)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &card, nil
}

// ListByUser returns a user's flashcards, most recent first.
func (r *FlashcardRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]database.Flashcard, error) {
	return inScope(ctx, r.db, r.querier, func(q database.Querier) ([]database.Flashcard, error) {
		return q.ListFlashcardsByUser(ctx, userID)
	})
}

// SetKnown marks a flashcard as known/unknown.
func (r *FlashcardRepo) SetKnown(ctx context.Context, id uuid.UUID, known bool) (*database.Flashcard, error) {
	card, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.Flashcard, error) {
		return q.SetFlashcardKnown(ctx, database.SetFlashcardKnownParams{
			ID:      id,
			IsKnown: known,
		})
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &card, nil
}

// Delete removes a flashcard, scoped to its owner.
func (r *FlashcardRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	_, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (struct{}, error) {
		return struct{}{}, q.DeleteFlashcard(ctx, database.DeleteFlashcardParams{
			ID:     id,
			UserID: userID,
		})
	})
	return err
}
