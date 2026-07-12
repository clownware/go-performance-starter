package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/clownware/go-performance-starter/internal/database"
	"github.com/clownware/go-performance-starter/internal/repository"
	"github.com/clownware/go-performance-starter/internal/webutil"
)

// fakeUserRepo implements only the UserRepository method the first-run flow
// uses; the embedded interface panics on anything else, catching unexpected
// repository calls.
type fakeUserRepo struct {
	repository.UserRepository
	firstRunErr      error
	gotUserID        uuid.UUID
	gotComplete      bool
	firstRunSeen     bool
	updateNameErr    error
	updateNameSeen   bool
	gotName          string
	anonymousFlipIDs []uuid.UUID
	updatedEmails    []string
}

func (f *fakeUserRepo) UpdateFirstRunComplete(_ context.Context, id uuid.UUID, complete bool) error {
	f.firstRunSeen = true
	f.gotUserID = id
	f.gotComplete = complete
	return f.firstRunErr
}

func firstRunRequest(target string, user *database.User, repo repository.UserRepository) *http.Request {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	ctx := req.Context()
	if user != nil {
		ctx = webutil.WithUser(ctx, user)
	}
	if repo != nil {
		ctx = webutil.WithUserRepo(ctx, repo)
	}
	return req.WithContext(ctx)
}

func TestFirstRunStepsRequireAuth(t *testing.T) {
	steps := []struct {
		name    string
		target  string
		handler http.HandlerFunc
	}{
		{name: "welcome", target: "/first-run", handler: ShowFirstRunWelcome},
		{name: "profile", target: "/first-run/profile", handler: ShowFirstRunProfile},
		{name: "ctas", target: "/first-run/ctas", handler: ShowFirstRunCTAs},
	}

	for _, tt := range steps {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.handler(w, firstRunRequest(tt.target, nil, nil))

			if w.Code != http.StatusSeeOther {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
			}
			if got := w.Header().Get("Location"); got != "/auth/page" {
				t.Errorf("Location = %q, want /auth/page", got)
			}
		})
	}
}

func TestFirstRunStepsRender(t *testing.T) {
	user := &database.User{ID: uuid.New()}
	steps := []struct {
		name    string
		target  string
		handler http.HandlerFunc
	}{
		{name: "welcome", target: "/first-run", handler: ShowFirstRunWelcome},
		{name: "profile", target: "/first-run/profile", handler: ShowFirstRunProfile},
	}

	for _, tt := range steps {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.handler(w, firstRunRequest(tt.target, user, nil))

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
			}
			if w.Body.Len() == 0 {
				t.Error("rendered an empty body")
			}
		})
	}
}

func TestShowFirstRunCTAs(t *testing.T) {
	tests := []struct {
		name       string
		repoErr    error
		wantStatus int
	}{
		{name: "marks onboarding complete and redirects", wantStatus: http.StatusSeeOther},
		{name: "repository failure re-renders with 500", repoErr: errors.New("db down"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &database.User{ID: uuid.New()}
			repo := &fakeUserRepo{firstRunErr: tt.repoErr}
			w := httptest.NewRecorder()

			ShowFirstRunCTAs(w, firstRunRequest("/first-run/ctas", user, repo))

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if !repo.firstRunSeen {
				t.Fatal("UpdateFirstRunComplete was not called")
			}
			if repo.gotUserID != user.ID || !repo.gotComplete {
				t.Errorf("UpdateFirstRunComplete(%v, %v), want (%v, true)", repo.gotUserID, repo.gotComplete, user.ID)
			}
			if tt.wantStatus == http.StatusSeeOther {
				if got := w.Header().Get("Location"); got != "/dashboard" {
					t.Errorf("Location = %q, want /dashboard", got)
				}
			} else if w.Body.Len() == 0 {
				t.Error("error path rendered an empty body")
			}
		})
	}
}
