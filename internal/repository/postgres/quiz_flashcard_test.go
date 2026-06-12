package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/clownware/alpine-go-performance-starter/internal/database"
	"github.com/clownware/alpine-go-performance-starter/internal/repository"
)

// fakeQuerier embeds database.Querier so it satisfies the full interface; tests
// override only the methods they exercise. Calling an un-overridden method panics
// (nil embedded interface), which keeps tests honest about what they touch.
type fakeQuerier struct {
	database.Querier
	getQuizQuestion func(ctx context.Context, id uuid.UUID) (database.QuizQuestion, error)
	getFlashcard    func(ctx context.Context, id uuid.UUID) (database.Flashcard, error)
}

func (f fakeQuerier) GetQuizQuestion(ctx context.Context, id uuid.UUID) (database.QuizQuestion, error) {
	return f.getQuizQuestion(ctx, id)
}

func (f fakeQuerier) GetFlashcard(ctx context.Context, id uuid.UUID) (database.Flashcard, error) {
	return f.getFlashcard(ctx, id)
}

func TestQuizRepoGetQuestion(t *testing.T) {
	wantID := uuid.New()
	tests := []struct {
		name        string
		queryErr    error
		queryResult database.QuizQuestion
		wantErr     error
		wantNil     bool
	}{
		{
			name:        "found returns the question",
			queryResult: database.QuizQuestion{ID: wantID, Slug: "middleware-stack"},
		},
		{
			name:     "no rows maps to ErrNotFound",
			queryErr: pgx.ErrNoRows,
			wantErr:  repository.ErrNotFound,
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := fakeQuerier{
				getQuizQuestion: func(_ context.Context, _ uuid.UUID) (database.QuizQuestion, error) {
					return tt.queryResult, tt.queryErr
				},
			}
			repo := NewQuizRepo(nil, q)

			got, err := repo.GetQuestion(context.Background(), wantID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetQuestion() err = %v, want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("GetQuestion() unexpected err = %v", err)
			}
			if tt.wantNil && got != nil {
				t.Errorf("GetQuestion() = %v, want nil", got)
			}
			if !tt.wantNil && (got == nil || got.ID != wantID) {
				t.Errorf("GetQuestion() = %v, want question with ID %v", got, wantID)
			}
		})
	}
}

func TestFlashcardRepoGet(t *testing.T) {
	wantID := uuid.New()
	tests := []struct {
		name     string
		queryErr error
		wantErr  error
		wantNil  bool
	}{
		{name: "found returns the flashcard"},
		{name: "no rows maps to ErrNotFound", queryErr: pgx.ErrNoRows, wantErr: repository.ErrNotFound, wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := fakeQuerier{
				getFlashcard: func(_ context.Context, _ uuid.UUID) (database.Flashcard, error) {
					return database.Flashcard{ID: wantID}, tt.queryErr
				},
			}
			repo := NewFlashcardRepo(nil, q)

			got, err := repo.Get(context.Background(), wantID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Get() err = %v, want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("Get() unexpected err = %v", err)
			}
			if tt.wantNil && got != nil {
				t.Errorf("Get() = %v, want nil", got)
			}
			if !tt.wantNil && (got == nil || got.ID != wantID) {
				t.Errorf("Get() = %v, want flashcard with ID %v", got, wantID)
			}
		})
	}
}
