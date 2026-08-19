package handler

import (
	"log/slog"
	"math"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/clownware/go-performance-starter/internal/database"
	"github.com/clownware/go-performance-starter/internal/repository"
	"github.com/clownware/go-performance-starter/internal/view"
	"github.com/clownware/go-performance-starter/internal/view/pages"
	"github.com/clownware/go-performance-starter/internal/view/partials"
	"github.com/clownware/go-performance-starter/internal/webutil"
)

// The dashboard is the progress surface of ADR-024 surface 3 (design brief
// priority 2, #69): quiz score history and cards to review, read through
// the same RLS-scoped repositories the quiz and flashcards write to
// (ADR-003/004) — the handler never sees SQL.
//
// Mounted in server.setupRoutes inside the /learn identity chain
// (GuestSession → OptionalAuth → OptionalUserLoader), so a guest-mode
// visitor gets an anonymous identity and sees their own widgets, a
// signed-out visitor (guest mode off) gets the browse-first teaser, and a
// registered user sees their widgets without the guest banner. When auth or
// the database is not configured the plain DashboardPage is still
// registered, so the route never 404s — it renders the teaser.
//
// Loading pattern (brief §HTMX/Alpine — skeleton loaders): the page ships
// two skeleton slots, each hx-get-ing its fragment on load and swapping
// itself out (outerHTML). Progressive enhancement (ADR-007/012): without
// JavaScript the page still renders meaningfully — heading, intro, and a
// <noscript> link inside each slot to the fragment endpoint, which returns
// the same fragment whether or not HTMX asked (a fragment endpoint is a
// fragment endpoint; the layout is never wrapped around it).

// dashboardRecentAttempts is how many rows the recent-attempts list shows —
// and, since CountAttemptsByUser owns the total (2026-08-19 found-work fix),
// all the listing the widget needs.
const dashboardRecentAttempts = 5

// DashboardRoutes registers the dashboard page and its widget fragments
// backed by the RLS-scoped quiz and flashcard repositories.
func DashboardRoutes(r chi.Router, quizRepo repository.QuizRepository, cardRepo repository.FlashcardRepository) {
	r.Get("/dashboard", DashboardPage)
	r.Get("/dashboard/widgets/quiz", dashboardQuizWidget(quizRepo))
	r.Get("/dashboard/widgets/flashcards", dashboardFlashcardWidget(cardRepo))
}

// DashboardPage renders the widgets host, or the teaser when there is no
// identity in context. It takes no repositories on purpose: the data loads
// through the widget endpoints, so the page is also the auth-less fallback.
func DashboardPage(w http.ResponseWriter, r *http.Request) {
	props := pages.DashboardPageProps{BaseProps: view.NewBaseProps("Dashboard")}
	if user := webutil.GetUserFromContext(r.Context()); user == nil {
		props.Teaser = true
	} else {
		props.GuestBanner = user.IsAnonymous
	}
	if err := view.Render(w, r, http.StatusOK, pages.DashboardPage(props)); err != nil {
		slog.Error("Failed to render dashboard page", "error", err)
	}
}

func dashboardQuizWidget(repo repository.QuizRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := webutil.GetUserFromContext(r.Context())
		if user == nil {
			renderQuiz(w, r, http.StatusOK, partials.DashboardTeaser())
			return
		}

		attempts, err := repo.ListAttemptsByUser(r.Context(), user.ID, dashboardRecentAttempts, 0)
		if err != nil {
			slog.Error("Failed to list quiz attempts", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		total, err := repo.CountAttemptsByUser(r.Context(), user.ID)
		if err != nil {
			slog.Error("Failed to count quiz attempts", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		correct, err := repo.CountCorrectByUser(r.Context(), user.ID)
		if err != nil {
			slog.Error("Failed to count correct attempts", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		// Attempts carry only question IDs; the prompts come from the (small,
		// seeded) question table. Skip the lookup when there is nothing to label.
		var questions []database.QuizQuestion
		if len(attempts) > 0 {
			if questions, err = repo.ListQuestions(r.Context()); err != nil {
				slog.Error("Failed to list quiz questions", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}

		renderQuiz(w, r, http.StatusOK, partials.QuizProgressWidget(quizProgress(attempts, total, correct, questions)))
	}
}

func dashboardFlashcardWidget(repo repository.FlashcardRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := webutil.GetUserFromContext(r.Context())
		if user == nil {
			renderQuiz(w, r, http.StatusOK, partials.DashboardTeaser())
			return
		}

		cards, err := repo.ListByUser(r.Context(), user.ID)
		if err != nil {
			slog.Error("Failed to list flashcards", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		renderQuiz(w, r, http.StatusOK, partials.FlashcardProgressWidget(flashcardProgress(cards)))
	}
}

// quizProgress maps the attempt log (most recent first), the correct count,
// and the question table to widget props. Percent rounds half-up; an attempt
// whose question is gone still renders, labelled, without a link.
func quizProgress(attempts []database.QuizAttempt, total, correct int64, questions []database.QuizQuestion) partials.QuizProgressProps {
	props := partials.QuizProgressProps{
		Attempts: int(total),
		Correct:  int(correct),
	}
	if total > 0 {
		props.Percent = int(math.Round(float64(correct) * 100 / float64(total)))
	}

	byID := make(map[uuid.UUID]database.QuizQuestion, len(questions))
	for _, q := range questions {
		byID[q.ID] = q
	}
	recent := min(len(attempts), dashboardRecentAttempts)
	props.Recent = make([]partials.QuizAttemptRowProps, 0, recent)
	for _, a := range attempts[:recent] {
		row := partials.QuizAttemptRowProps{Correct: a.IsCorrect, Prompt: "Question no longer available"}
		if q, ok := byID[a.QuestionID]; ok {
			row.Slug, row.Prompt = q.Slug, q.Prompt
		}
		props.Recent = append(props.Recent, row)
	}
	return props
}

// flashcardProgress splits the user's cards into known and still-to-review.
func flashcardProgress(cards []database.Flashcard) partials.FlashcardProgressProps {
	props := partials.FlashcardProgressProps{Total: len(cards)}
	for _, c := range cards {
		if c.IsKnown {
			props.Known++
		}
	}
	props.ToReview = props.Total - props.Known
	return props
}
