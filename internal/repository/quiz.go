package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/clownware/go-performance-starter/internal/database"
)

// QuizRepository defines data access for the architecture quiz (ADR-024).
type QuizRepository interface {
	// ListQuestions returns all quiz questions in display order.
	ListQuestions(ctx context.Context) ([]database.QuizQuestion, error)

	// GetQuestion retrieves a question by ID.
	GetQuestion(ctx context.Context, id uuid.UUID) (*database.QuizQuestion, error)

	// GetQuestionBySlug retrieves a question by its stable slug.
	GetQuestionBySlug(ctx context.Context, slug string) (*database.QuizQuestion, error)

	// RecordAttempt persists a user's answer to a question.
	RecordAttempt(ctx context.Context, params database.CreateQuizAttemptParams) (*database.QuizAttempt, error)

	// ListAttemptsByUser returns a user's attempts, most recent first.
	ListAttemptsByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]database.QuizAttempt, error)

	// CountCorrectByUser returns how many questions the user has answered correctly.
	CountCorrectByUser(ctx context.Context, userID uuid.UUID) (int64, error)
}
