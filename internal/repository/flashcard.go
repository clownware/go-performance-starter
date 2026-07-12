package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/clownware/go-performance-starter/internal/database"
)

// FlashcardRepository defines data access for user-owned flashcards (ADR-024).
type FlashcardRepository interface {
	// Create adds a flashcard for a user (typically from a wrong quiz answer).
	Create(ctx context.Context, params database.CreateFlashcardParams) (*database.Flashcard, error)

	// Get retrieves a flashcard by ID.
	Get(ctx context.Context, id uuid.UUID) (*database.Flashcard, error)

	// ListByUser returns a user's flashcards, most recent first.
	ListByUser(ctx context.Context, userID uuid.UUID) ([]database.Flashcard, error)

	// SetKnown marks a flashcard as known/unknown.
	SetKnown(ctx context.Context, id uuid.UUID, known bool) (*database.Flashcard, error)

	// Delete removes a flashcard, scoped to its owner.
	Delete(ctx context.Context, id, userID uuid.UUID) error
}
