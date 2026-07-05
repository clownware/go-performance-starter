package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clownware/alpine-go-performance-starter/internal/database"
	"github.com/clownware/alpine-go-performance-starter/internal/repository"
)

// Ensure QuizRepo satisfies the repository interface at compile time.
var _ repository.QuizRepository = (*QuizRepo)(nil)

// QuizRepo implements repository.QuizRepository using PostgreSQL. All methods
// run through inScope so RLS evaluates against the requester (ADR-004).
type QuizRepo struct {
	db      *pgxpool.Pool
	querier database.Querier
}

// NewQuizRepo creates a new QuizRepo instance.
func NewQuizRepo(db *pgxpool.Pool, querier database.Querier) *QuizRepo {
	return &QuizRepo{db: db, querier: querier}
}

// ListQuestions returns all quiz questions in display order.
func (r *QuizRepo) ListQuestions(ctx context.Context) ([]database.QuizQuestion, error) {
	return inScope(ctx, r.db, r.querier, func(q database.Querier) ([]database.QuizQuestion, error) {
		return q.ListQuizQuestions(ctx)
	})
}

// GetQuestion retrieves a question by ID.
func (r *QuizRepo) GetQuestion(ctx context.Context, id uuid.UUID) (*database.QuizQuestion, error) {
	question, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.QuizQuestion, error) {
		return q.GetQuizQuestion(ctx, id)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &question, nil
}

// GetQuestionBySlug retrieves a question by its stable slug.
func (r *QuizRepo) GetQuestionBySlug(ctx context.Context, slug string) (*database.QuizQuestion, error) {
	question, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.QuizQuestion, error) {
		return q.GetQuizQuestionBySlug(ctx, slug)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &question, nil
}

// RecordAttempt persists a user's answer to a question.
func (r *QuizRepo) RecordAttempt(ctx context.Context, params database.CreateQuizAttemptParams) (*database.QuizAttempt, error) {
	attempt, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.QuizAttempt, error) {
		return q.CreateQuizAttempt(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	return &attempt, nil
}

// ListAttemptsByUser returns a user's attempts, most recent first.
func (r *QuizRepo) ListAttemptsByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]database.QuizAttempt, error) {
	return inScope(ctx, r.db, r.querier, func(q database.Querier) ([]database.QuizAttempt, error) {
		return q.ListQuizAttemptsByUser(ctx, database.ListQuizAttemptsByUserParams{
			UserID: userID,
			Limit:  limit,
			Offset: offset,
		})
	})
}

// CountCorrectByUser returns how many questions the user has answered correctly.
func (r *QuizRepo) CountCorrectByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	return inScope(ctx, r.db, r.querier, func(q database.Querier) (int64, error) {
		return q.CountCorrectAttemptsByUser(ctx, userID)
	})
}
