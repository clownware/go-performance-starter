package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clownware/go-performance-starter/internal/database"
	"github.com/clownware/go-performance-starter/internal/repository"
)

// These tests round-trip the repositories against a real PostgreSQL. They run
// in CI (where DATABASE_URL is set and migrations are applied) and skip locally
// when DATABASE_URL is unset.
//
// NOTE: they verify query/CRUD behavior, NOT RLS isolation — the test connects
// as the owner role and the service_role_bypass policy nullifies RLS for it.
// RLS-scoping tests require the Supabase role model (anon/authenticated/
// service_role) in the test harness; tracked as a follow-up.

// withTx opens a transaction and rolls it back on cleanup so each test is
// isolated and leaves no rows behind. The returned Queries run on the tx.
func withTx(t *testing.T) (context.Context, *database.Queries) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		pool.Close()
		t.Fatalf("begin: %v", err)
	}
	t.Cleanup(func() {
		_ = tx.Rollback(ctx)
		pool.Close()
	})
	return ctx, database.New(tx)
}

func seedUser(ctx context.Context, t *testing.T, q *database.Queries) uuid.UUID {
	t.Helper()
	authID := uuid.NewString()
	user, err := q.CreateUser(ctx, database.CreateUserParams{
		Email:  authID + "@example.com",
		AuthID: pgtype.Text{String: authID, Valid: true},
	})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return user.ID
}

func seedQuestion(ctx context.Context, t *testing.T, q *database.Queries) database.QuizQuestion {
	t.Helper()
	question, err := q.CreateQuizQuestion(ctx, database.CreateQuizQuestionParams{
		Slug:         "q-" + uuid.NewString(),
		Topic:        "middleware",
		Prompt:       "What runs first?",
		Choices:      []byte(`["handler","middleware"]`),
		CorrectIndex: 1,
		Explanation:  "The router",
	})
	if err != nil {
		t.Fatalf("seed question: %v", err)
	}
	return question
}

func TestQuizRepoIntegration(t *testing.T) {
	ctx, q := withTx(t)
	repo := NewQuizRepo(nil, q)
	userID := seedUser(ctx, t, q)
	question := seedQuestion(ctx, t, q)

	// Record one correct and one incorrect attempt.
	for _, correct := range []bool{true, false} {
		if _, err := repo.RecordAttempt(ctx, database.CreateQuizAttemptParams{
			UserID:        userID,
			QuestionID:    question.ID,
			SelectedIndex: 1,
			IsCorrect:     correct,
		}); err != nil {
			t.Fatalf("RecordAttempt(correct=%v): %v", correct, err)
		}
	}

	attempts, err := repo.ListAttemptsByUser(ctx, userID, 10, 0)
	if err != nil {
		t.Fatalf("ListAttemptsByUser: %v", err)
	}
	if len(attempts) != 2 {
		t.Errorf("ListAttemptsByUser len = %d, want 2", len(attempts))
	}

	correctCount, err := repo.CountCorrectByUser(ctx, userID)
	if err != nil {
		t.Fatalf("CountCorrectByUser: %v", err)
	}
	if correctCount != 1 {
		t.Errorf("CountCorrectByUser = %d, want 1", correctCount)
	}

	got, err := repo.GetQuestionBySlug(ctx, question.Slug)
	if err != nil {
		t.Fatalf("GetQuestionBySlug: %v", err)
	}
	if got.ID != question.ID {
		t.Errorf("GetQuestionBySlug ID = %v, want %v", got.ID, question.ID)
	}

	questions, err := repo.ListQuestions(ctx)
	if err != nil {
		t.Fatalf("ListQuestions: %v", err)
	}
	listed := false
	for _, qn := range questions {
		if qn.ID == question.ID {
			listed = true
			break
		}
	}
	if !listed {
		t.Errorf("ListQuestions missing seeded question (got %d rows)", len(questions))
	}
}

func TestFlashcardRepoIntegration(t *testing.T) {
	ctx, q := withTx(t)
	repo := NewFlashcardRepo(nil, q)
	userID := seedUser(ctx, t, q)
	question := seedQuestion(ctx, t, q)

	created, err := repo.Create(ctx, database.CreateFlashcardParams{
		UserID:     userID,
		QuestionID: pgtype.UUID{Bytes: question.ID, Valid: true},
		Front:      "What runs first?",
		Back:       "The router/middleware stack",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.IsKnown {
		t.Error("new flashcard should default to is_known=false")
	}

	list, err := repo.ListByUser(ctx, userID)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListByUser len = %d, want 1", len(list))
	}

	updated, err := repo.SetKnown(ctx, created.ID, true)
	if err != nil {
		t.Fatalf("SetKnown: %v", err)
	}
	if !updated.IsKnown {
		t.Error("SetKnown(true) did not set is_known")
	}

	if err := repo.Delete(ctx, created.ID, userID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.Get(ctx, created.ID); err != repository.ErrNotFound {
		t.Errorf("Get after delete err = %v, want ErrNotFound", err)
	}
}
