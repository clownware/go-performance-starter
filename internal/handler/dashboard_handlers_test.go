package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/clownware/go-performance-starter/internal/database"
	"github.com/clownware/go-performance-starter/internal/repository"
)

func newDashboardRouter(quiz repository.QuizRepository, cards repository.FlashcardRepository) http.Handler {
	r := chi.NewRouter()
	DashboardRoutes(r, quiz, cards)
	return r
}

// attemptFixtures builds an attempt log, most recent first, over the seeded
// questions: results[i] is the correctness of the i-th most recent attempt,
// cycling through the questions so every row has a prompt to show.
func attemptFixtures(userID uuid.UUID, questions []database.QuizQuestion, results ...bool) []database.QuizAttempt {
	log := make([]database.QuizAttempt, 0, len(results))
	for i, correct := range results {
		q := questions[i%len(questions)]
		log = append(log, database.QuizAttempt{
			ID:         uuid.New(),
			UserID:     userID,
			QuestionID: q.ID,
			IsCorrect:  correct,
		})
	}
	return log
}

// TestQuizProgress pins the widget math as a pure function: totals come from
// the attempt log, correct from the repository count, the percentage rounds
// half-up, and the recent list is capped and resolved to question prompts.
func TestQuizProgress(t *testing.T) {
	questions := quizFixtures()
	userID := uuid.New()

	tests := []struct {
		name         string
		results      []bool
		correct      int64
		wantAttempts int
		wantCorrect  int
		wantPercent  int
		wantRecent   int
	}{
		{name: "no attempts is zero percent, not a divide by zero", results: nil, correct: 0, wantAttempts: 0, wantCorrect: 0, wantPercent: 0, wantRecent: 0},
		{name: "all correct is one hundred percent", results: []bool{true, true, true}, correct: 3, wantAttempts: 3, wantCorrect: 3, wantPercent: 100, wantRecent: 3},
		{name: "one of three rounds down to 33", results: []bool{true, false, false}, correct: 1, wantAttempts: 3, wantCorrect: 1, wantPercent: 33, wantRecent: 3},
		{name: "two of three rounds up to 67", results: []bool{true, true, false}, correct: 2, wantAttempts: 3, wantCorrect: 2, wantPercent: 67, wantRecent: 3},
		{name: "exact half rounds up", results: []bool{true, false, false, false, false, false, false, false}, correct: 1, wantAttempts: 8, wantCorrect: 1, wantPercent: 13, wantRecent: 5},
		{name: "recent list is capped at five", results: []bool{true, false, true, false, true, false, true}, correct: 4, wantAttempts: 7, wantCorrect: 4, wantPercent: 57, wantRecent: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := quizProgress(attemptFixtures(userID, questions, tt.results...), tt.correct, questions)

			if got.Attempts != tt.wantAttempts {
				t.Errorf("Attempts = %d, want %d", got.Attempts, tt.wantAttempts)
			}
			if got.Correct != tt.wantCorrect {
				t.Errorf("Correct = %d, want %d", got.Correct, tt.wantCorrect)
			}
			if got.Percent != tt.wantPercent {
				t.Errorf("Percent = %d, want %d", got.Percent, tt.wantPercent)
			}
			if len(got.Recent) != tt.wantRecent {
				t.Fatalf("len(Recent) = %d, want %d", len(got.Recent), tt.wantRecent)
			}
			for i, row := range got.Recent {
				q := questions[i%len(questions)]
				if row.Slug != q.Slug || row.Prompt != q.Prompt {
					t.Errorf("Recent[%d] = {%q %q}, want question %q", i, row.Slug, row.Prompt, q.Slug)
				}
				if row.Correct != tt.results[i] {
					t.Errorf("Recent[%d].Correct = %v, want %v", i, row.Correct, tt.results[i])
				}
			}
		})
	}
}

// TestQuizProgress_UnknownQuestion proves an attempt whose question is gone
// still renders a row (labelled, not dropped) — the history is the user's.
func TestQuizProgress_UnknownQuestion(t *testing.T) {
	orphan := []database.QuizAttempt{{ID: uuid.New(), QuestionID: uuid.New(), IsCorrect: true}}
	got := quizProgress(orphan, 1, quizFixtures())
	if len(got.Recent) != 1 {
		t.Fatalf("len(Recent) = %d, want 1", len(got.Recent))
	}
	if got.Recent[0].Prompt == "" {
		t.Error("orphaned attempt rendered with an empty prompt")
	}
	if got.Recent[0].Slug != "" {
		t.Errorf("orphaned attempt Slug = %q, want empty (no question to link to)", got.Recent[0].Slug)
	}
}

// TestFlashcardProgress pins the to-review split.
func TestFlashcardProgress(t *testing.T) {
	tests := []struct {
		name       string
		known      []bool
		wantTotal  int
		wantKnown  int
		wantReview int
	}{
		{"no cards", nil, 0, 0, 0},
		{"all unknown are all to review", []bool{false, false}, 2, 0, 2},
		{"all known leaves nothing to review", []bool{true, true, true}, 3, 3, 0},
		{"mixed splits by is_known", []bool{true, false, false, true, false}, 5, 2, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cards := make([]database.Flashcard, 0, len(tt.known))
			for _, k := range tt.known {
				cards = append(cards, database.Flashcard{ID: uuid.New(), IsKnown: k})
			}
			got := flashcardProgress(cards)
			if got.Total != tt.wantTotal || got.Known != tt.wantKnown || got.ToReview != tt.wantReview {
				t.Errorf("flashcardProgress = {Total %d Known %d ToReview %d}, want {%d %d %d}",
					got.Total, got.Known, got.ToReview, tt.wantTotal, tt.wantKnown, tt.wantReview)
			}
		})
	}
}

func TestDashboardPage(t *testing.T) {
	guest := &database.User{ID: uuid.New(), IsAnonymous: true}
	member := &database.User{ID: uuid.New(), IsAnonymous: false}

	tests := []struct {
		name         string
		user         *database.User
		wantStatus   int
		wantContains []string
		wantAbsent   []string
	}{
		{
			// The page is the widgets host: each slot ships a skeleton and
			// loads its fragment over HTMX on load (design brief §HTMX/Alpine
			// patterns — skeleton loaders on dashboard widgets).
			name:       "guest sees the widget slots with skeleton loaders and the upgrade banner",
			user:       guest,
			wantStatus: http.StatusOK,
			wantContains: []string{
				"<!doctype",
				`data-testid="widget-quiz-skeleton"`,
				`hx-get="/dashboard/widgets/quiz"`,
				`data-testid="widget-flashcards-skeleton"`,
				`hx-get="/dashboard/widgets/flashcards"`,
				`hx-trigger="load"`,
				`hx-swap="outerHTML"`,
				`data-testid="guest-banner"`,
			},
			wantAbsent: []string{
				`data-testid="dashboard-teaser"`,
				"Create Project", // pre-pivot SaaS empty state is gone
			},
		},
		{
			name:       "registered user sees the widget slots without the guest banner",
			user:       member,
			wantStatus: http.StatusOK,
			wantContains: []string{
				`data-testid="widget-quiz-skeleton"`,
				`data-testid="widget-flashcards-skeleton"`,
			},
			wantAbsent: []string{
				`data-testid="guest-banner"`,
			},
		},
		{
			// Browse-first, like the quiz and flashcards pages: a signed-out
			// visit previews what the dashboard shows and why it needs an
			// identity instead of bouncing to the login page.
			name:       "signed-out visitor sees the teaser with a sign-in call to action",
			user:       nil,
			wantStatus: http.StatusOK,
			wantContains: []string{
				`data-testid="dashboard-teaser"`,
				`href="/auth/page"`,
			},
			wantAbsent: []string{
				`data-testid="widget-quiz-skeleton"`,
				`hx-get="/dashboard/widgets/quiz"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := asQuizUser(httptest.NewRequest(http.MethodGet, "/dashboard", nil), tt.user)
			w := httptest.NewRecorder()

			newDashboardRouter(&fakeQuizRepo{}, &fakeFlashcardRepo{}).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("GET /dashboard status = %d, want %d", w.Code, tt.wantStatus)
			}
			body := strings.ToLower(w.Body.String())
			for _, want := range tt.wantContains {
				if !strings.Contains(body, strings.ToLower(want)) {
					t.Errorf("GET /dashboard body missing %q", want)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(body, strings.ToLower(absent)) {
					t.Errorf("GET /dashboard body unexpectedly contains %q", absent)
				}
			}
		})
	}
}

func TestDashboardQuizWidget(t *testing.T) {
	questions := quizFixtures()
	user := &database.User{ID: uuid.New(), IsAnonymous: true}

	tests := []struct {
		name         string
		repo         *fakeQuizRepo
		user         *database.User
		htmx         bool
		wantStatus   int
		wantContains []string
		wantAbsent   []string
		wantRows     int // data-testid="quiz-attempt" occurrences; -1 = don't care
	}{
		{
			name: "renders the score, the percentage, and the five most recent attempts",
			repo: &fakeQuizRepo{
				questions:    questions,
				correctCount: 4,
				history:      attemptFixtures(user.ID, questions, true, false, true, false, true, false, true),
			},
			user:       user,
			htmx:       true,
			wantStatus: http.StatusOK,
			wantContains: []string{
				`data-testid="widget-quiz"`,
				`data-testid="quiz-correct"`,
				`data-testid="quiz-attempts"`,
				`data-testid="quiz-percent"`,
				"57%",
				"Which router assembles the middleware stack?",
				`href="/learn/quiz?q=middleware-stack"`,
				"text-success", // correct rows ride the success role (ADR-029)
				"text-danger",  // incorrect rows ride the danger role
				`href="/learn/quiz"`,
			},
			wantAbsent: []string{
				"<!doctype", // fragment, never the layout
				`data-testid="widget-quiz-skeleton"`,
			},
			wantRows: 5,
		},
		{
			name:       "no attempts renders the empty state with a link to the quiz",
			repo:       &fakeQuizRepo{questions: questions},
			user:       user,
			htmx:       true,
			wantStatus: http.StatusOK,
			wantContains: []string{
				`data-testid="widget-quiz-empty"`,
				"No quiz attempts yet",
				`href="/learn/quiz"`,
			},
			wantAbsent: []string{
				`data-testid="quiz-attempt"`,
			},
			wantRows: 0,
		},
		{
			// A fragment endpoint returns the fragment whether or not HTMX
			// asked: the page's <noscript> fallback links straight here.
			name: "non-HTMX request still gets the fragment",
			repo: &fakeQuizRepo{
				questions:    questions,
				correctCount: 1,
				history:      attemptFixtures(user.ID, questions, true),
			},
			user:       user,
			htmx:       false,
			wantStatus: http.StatusOK,
			wantContains: []string{
				`data-testid="widget-quiz"`,
				"100%",
			},
			wantAbsent: []string{
				"<!doctype",
			},
			wantRows: 1,
		},
		{
			name:       "signed-out request gets the teaser, not a widget",
			repo:       &fakeQuizRepo{questions: questions},
			user:       nil,
			htmx:       true,
			wantStatus: http.StatusOK,
			wantContains: []string{
				`data-testid="dashboard-teaser"`,
			},
			wantAbsent: []string{
				`data-testid="widget-quiz"`,
			},
			wantRows: 0,
		},
		{
			name:       "attempt list failure is a 500",
			repo:       &fakeQuizRepo{questions: questions, listAttemptsErr: errFake},
			user:       user,
			htmx:       true,
			wantStatus: http.StatusInternalServerError,
			wantRows:   -1,
		},
		{
			name:       "count failure is a 500",
			repo:       &fakeQuizRepo{questions: questions, countErr: errFake},
			user:       user,
			htmx:       true,
			wantStatus: http.StatusInternalServerError,
			wantRows:   -1,
		},
		{
			name:       "question lookup failure is a 500",
			repo:       &fakeQuizRepo{listErr: errFake, history: attemptFixtures(user.ID, questions, true)},
			user:       user,
			htmx:       true,
			wantStatus: http.StatusInternalServerError,
			wantRows:   -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := asQuizUser(httptest.NewRequest(http.MethodGet, "/dashboard/widgets/quiz", nil), tt.user)
			if tt.htmx {
				req.Header.Set("HX-Request", "true")
			}
			w := httptest.NewRecorder()

			newDashboardRouter(tt.repo, &fakeFlashcardRepo{}).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("GET /dashboard/widgets/quiz status = %d, want %d", w.Code, tt.wantStatus)
			}
			body := strings.ToLower(w.Body.String())
			for _, want := range tt.wantContains {
				if !strings.Contains(body, strings.ToLower(want)) {
					t.Errorf("body missing %q", want)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(body, strings.ToLower(absent)) {
					t.Errorf("body unexpectedly contains %q", absent)
				}
			}
			if tt.wantRows >= 0 {
				if got := strings.Count(body, `data-testid="quiz-attempt"`); got != tt.wantRows {
					t.Errorf("rendered %d attempt rows, want %d", got, tt.wantRows)
				}
			}
		})
	}
}

func TestDashboardFlashcardWidget(t *testing.T) {
	user := &database.User{ID: uuid.New(), IsAnonymous: true}

	tests := []struct {
		name         string
		repo         *fakeFlashcardRepo
		user         *database.User
		wantStatus   int
		wantContains []string
		wantAbsent   []string
	}{
		{
			name:       "renders total, known, and to-review counts with a link to the cards",
			repo:       &fakeFlashcardRepo{cards: flashcardFixtures(user.ID)}, // 2 cards, 1 known
			user:       user,
			wantStatus: http.StatusOK,
			wantContains: []string{
				`data-testid="widget-flashcards"`,
				`data-testid="flashcards-to-review">1<`,
				`data-testid="flashcards-known">1<`,
				`data-testid="flashcards-total">2<`,
				`href="/learn/flashcards"`,
			},
			wantAbsent: []string{
				"<!doctype",
				`data-testid="widget-flashcards-skeleton"`,
			},
		},
		{
			name:       "no cards renders the empty state with a link to the quiz",
			repo:       &fakeFlashcardRepo{},
			user:       user,
			wantStatus: http.StatusOK,
			wantContains: []string{
				`data-testid="widget-flashcards-empty"`,
				`href="/learn/quiz"`,
			},
			wantAbsent: []string{
				`data-testid="flashcards-to-review"`,
			},
		},
		{
			name:       "signed-out request gets the teaser, not a widget",
			repo:       &fakeFlashcardRepo{cards: flashcardFixtures(user.ID)},
			user:       nil,
			wantStatus: http.StatusOK,
			wantContains: []string{
				`data-testid="dashboard-teaser"`,
			},
			wantAbsent: []string{
				`data-testid="widget-flashcards"`,
			},
		},
		{
			name:       "repository failure is a 500",
			repo:       &fakeFlashcardRepo{listErr: errFake},
			user:       user,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := asQuizUser(httptest.NewRequest(http.MethodGet, "/dashboard/widgets/flashcards", nil), tt.user)
			req.Header.Set("HX-Request", "true")
			w := httptest.NewRecorder()

			newDashboardRouter(&fakeQuizRepo{}, tt.repo).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("GET /dashboard/widgets/flashcards status = %d, want %d", w.Code, tt.wantStatus)
			}
			body := strings.ToLower(w.Body.String())
			for _, want := range tt.wantContains {
				if !strings.Contains(body, strings.ToLower(want)) {
					t.Errorf("body missing %q", want)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(body, strings.ToLower(absent)) {
					t.Errorf("body unexpectedly contains %q", absent)
				}
			}
		})
	}
}
