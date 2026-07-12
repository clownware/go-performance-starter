package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/supabase-community/gotrue-go/types"

	"github.com/clownware/go-performance-starter/internal/database"
	"github.com/clownware/go-performance-starter/internal/middleware"
	"github.com/clownware/go-performance-starter/internal/webutil"
)

// UpdateName extends the shared fakeUserRepo for the profile handlers.
func (f *fakeUserRepo) UpdateName(_ context.Context, id uuid.UUID, name string) (*database.User, error) {
	f.updateNameSeen = true
	f.gotUserID = id
	f.gotName = name
	if f.updateNameErr != nil {
		return nil, f.updateNameErr
	}
	return &database.User{ID: id, Name: pgtype.Text{String: name, Valid: true}}, nil
}

// withDBUser stores a users row (and the repo) in the context the way
// UserLoader + UserRepoMiddleware do for the profile routes.
func withDBUser(r *http.Request, user *database.User, repo *fakeUserRepo) *http.Request {
	ctx := webutil.WithUser(r.Context(), user)
	if repo != nil {
		ctx = webutil.WithUserRepo(ctx, repo)
	}
	return r.WithContext(ctx)
}

// withAuthUser stores a gotrue user in the request context the way
// middleware.AuthMiddleware does.
func withAuthUser(r *http.Request, user *types.UserResponse) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), middleware.ContextUserKey, user))
}

func TestUserNameFromContext(t *testing.T) {
	tests := []struct {
		name string
		user *types.UserResponse
		want string
	}{
		{name: "no user in context", user: nil, want: "User"},
		{
			name: "full_name metadata wins",
			user: &types.UserResponse{User: types.User{
				Email:        "ada@example.com",
				UserMetadata: map[string]interface{}{"full_name": "Ada Lovelace", "name": "Ada"},
			}},
			want: "Ada Lovelace",
		},
		{
			name: "name metadata second",
			user: &types.UserResponse{User: types.User{
				Email:        "grace@example.com",
				UserMetadata: map[string]interface{}{"name": "Grace Hopper"},
			}},
			want: "Grace Hopper",
		},
		{
			name: "email fallback",
			user: &types.UserResponse{User: types.User{Email: "fallback@example.com"}},
			want: "fallback@example.com",
		},
		{name: "empty user falls back to User", user: &types.UserResponse{}, want: "User"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/profile", nil)
			if tt.user != nil {
				req = withAuthUser(req, tt.user)
			}
			if got := userNameFromContext(req); got != tt.want {
				t.Errorf("userNameFromContext() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProfileView(t *testing.T) {
	tests := []struct {
		name     string
		user     *types.UserResponse
		dbUser   *database.User
		wantName string
	}{
		{name: "anonymous context"},
		{name: "authenticated user", user: &types.UserResponse{User: types.User{Email: "ada@example.com"}}},
		{
			// The persisted row is the source of truth (#70): a saved name
			// must survive reloads instead of reverting to token metadata.
			name:     "users row name wins over token metadata",
			user:     &types.UserResponse{User: types.User{Email: "ada@example.com", UserMetadata: map[string]interface{}{"full_name": "Token Name"}}},
			dbUser:   &database.User{ID: uuid.New(), Name: pgtype.Text{String: "Persisted Name", Valid: true}},
			wantName: "Persisted Name",
		},
		{
			name:     "empty users row name falls back to token metadata",
			user:     &types.UserResponse{User: types.User{Email: "ada@example.com", UserMetadata: map[string]interface{}{"full_name": "Token Name"}}},
			dbUser:   &database.User{ID: uuid.New()},
			wantName: "Token Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/profile", nil)
			if tt.user != nil {
				req = withAuthUser(req, tt.user)
			}
			if tt.dbUser != nil {
				req = withDBUser(req, tt.dbUser, nil)
			}
			w := httptest.NewRecorder()

			ProfileView(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("ProfileView() status = %d, want %d", w.Code, http.StatusOK)
			}
			if w.Body.Len() == 0 {
				t.Error("ProfileView() rendered an empty body")
			}
			if tt.wantName != "" && !strings.Contains(w.Body.String(), tt.wantName) {
				t.Errorf("ProfileView() body missing %q", tt.wantName)
			}
		})
	}
}

func TestProfileUpdate(t *testing.T) {
	userID := uuid.New()
	tests := []struct {
		name          string
		form          string
		htmx          bool
		noUser        bool // no users row in context (loader missing/failed)
		repoErr       error
		wantStatus    int
		wantLocation  string
		wantTrigger   bool
		wantPersisted string // name the repository must receive
	}{
		{name: "malformed form body returns 400", form: "%zz", wantStatus: http.StatusBadRequest},
		{name: "empty name via htmx re-renders form with 422", form: "name=+", htmx: true, wantStatus: http.StatusUnprocessableEntity},
		{name: "empty name without htmx re-renders page with 422", form: "name=", wantStatus: http.StatusUnprocessableEntity},
		{name: "valid name via htmx persists and triggers", form: "name=Ada", htmx: true, wantStatus: http.StatusOK, wantTrigger: true, wantPersisted: "Ada"},
		{name: "valid name without htmx persists and redirects", form: "name=Ada", wantStatus: http.StatusSeeOther, wantLocation: "/profile", wantPersisted: "Ada"},
		{name: "persisted name is trimmed", form: "name=+Ada+Lovelace+", htmx: true, wantStatus: http.StatusOK, wantTrigger: true, wantPersisted: "Ada Lovelace"},
		{name: "no users row redirects to auth", form: "name=Ada", noUser: true, wantStatus: http.StatusSeeOther, wantLocation: "/auth/page"},
		{name: "repository failure returns 500", form: "name=Ada", repoErr: errors.New("boom"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeUserRepo{updateNameErr: tt.repoErr}
			req := formRequest("/profile", tt.form)
			if !tt.noUser {
				req = withDBUser(req, &database.User{ID: userID}, repo)
			}
			if tt.htmx {
				req.Header.Set("HX-Request", "true")
			}
			w := httptest.NewRecorder()

			ProfileUpdate(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("ProfileUpdate() status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantLocation != "" {
				if got := w.Header().Get("Location"); got != tt.wantLocation {
					t.Errorf("Location = %q, want %q", got, tt.wantLocation)
				}
			}
			if tt.wantTrigger && w.Header().Get("HX-Trigger") == "" {
				t.Error("HX-Trigger not set on successful htmx update")
			}
			if tt.wantStatus == http.StatusUnprocessableEntity && w.Body.Len() == 0 {
				t.Error("validation failure rendered an empty body")
			}
			if tt.wantPersisted != "" {
				if !repo.updateNameSeen {
					t.Fatal("successful update never called UpdateName — the write does not persist (#70)")
				}
				if repo.gotName != tt.wantPersisted || repo.gotUserID != userID {
					t.Errorf("UpdateName(%v, %q), want (%v, %q)", repo.gotUserID, repo.gotName, userID, tt.wantPersisted)
				}
			}
			if tt.wantPersisted == "" && tt.repoErr == nil && repo.updateNameSeen {
				t.Error("UpdateName called on a request that must not persist")
			}
		})
	}
}
