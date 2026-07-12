package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clownware/go-performance-starter/internal/database"
	"github.com/clownware/go-performance-starter/internal/repository"
	"github.com/clownware/go-performance-starter/internal/webutil"
)

// fakeUserRepo implements repository.UserRepository for UserLoader tests.
type fakeUserRepo struct {
	byAuthID       map[string]*database.User
	created        []database.CreateUserParams
	anonymousFlips []bool
}

func (f *fakeUserRepo) GetByAuthID(ctx context.Context, authID string) (*database.User, error) {
	if u, ok := f.byAuthID[authID]; ok {
		return u, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakeUserRepo) Create(ctx context.Context, params database.CreateUserParams) (*database.User, error) {
	f.created = append(f.created, params)
	u := &database.User{ID: uuid.New(), Email: params.Email, AuthID: params.AuthID}
	return u, nil
}

func (f *fakeUserRepo) Get(ctx context.Context, id uuid.UUID) (*database.User, error) {
	return nil, repository.ErrNotFound
}
func (f *fakeUserRepo) GetByEmail(ctx context.Context, email string) (*database.User, error) {
	return nil, repository.ErrNotFound
}
func (f *fakeUserRepo) List(ctx context.Context, limit, offset int32) ([]database.User, error) {
	return nil, nil
}
func (f *fakeUserRepo) Update(ctx context.Context, params database.UpdateUserParams) (*database.User, error) {
	return nil, repository.ErrNotFound
}
func (f *fakeUserRepo) UpdateName(ctx context.Context, id uuid.UUID, name string) (*database.User, error) {
	return nil, repository.ErrNotFound
}
func (f *fakeUserRepo) SetAnonymous(ctx context.Context, id uuid.UUID, anonymous bool) error {
	f.anonymousFlips = append(f.anonymousFlips, anonymous)
	return nil
}
func (f *fakeUserRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeUserRepo) SetLastLogin(ctx context.Context, id uuid.UUID, loginTime time.Time) error {
	return nil
}
func (f *fakeUserRepo) UpdateFirstRunComplete(ctx context.Context, id uuid.UUID, complete bool) error {
	return nil
}

func TestUserLoader(t *testing.T) {
	existing := &database.User{ID: uuid.New(), Email: "known@example.com"}

	tests := []struct {
		name          string
		claims        *webutil.AuthClaims
		repo          *fakeUserRepo
		wantStatus    int
		wantUserEmail string
		wantCreated   int
		wantAnonFlip  bool
		wantEmail     string // email the provisioned row must carry
	}{
		{
			name:       "no claims: unauthorized",
			claims:     nil,
			repo:       &fakeUserRepo{byAuthID: map[string]*database.User{}},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:          "existing user loaded into context",
			claims:        &webutil.AuthClaims{Sub: "known-sub", Role: webutil.RoleAuthenticated},
			repo:          &fakeUserRepo{byAuthID: map[string]*database.User{"known-sub": existing}},
			wantStatus:    http.StatusOK,
			wantUserEmail: "known@example.com",
		},
		{
			name:        "missing user is JIT-provisioned",
			claims:      &webutil.AuthClaims{Sub: "new-sub", Role: webutil.RoleAuthenticated},
			repo:        &fakeUserRepo{byAuthID: map[string]*database.User{}},
			wantStatus:  http.StatusOK,
			wantCreated: 1,
		},
		{
			// Anonymous identities have no email, and users.email is NOT
			// NULL UNIQUE — a shared "" collides on the second guest ever
			// (live 500s, 2026-07-12). Provision a per-identity placeholder.
			name:        "anonymous provision gets a unique placeholder email",
			claims:      &webutil.AuthClaims{Sub: "guest-sub-2", Role: webutil.RoleAuthenticated, IsAnonymous: true},
			repo:        &fakeUserRepo{byAuthID: map[string]*database.User{}},
			wantStatus:  http.StatusOK,
			wantCreated: 1,
			wantEmail:   "guest-sub-2@guest.invalid",
		},
		{
			// Upgrade self-heal (#68): a non-anonymous token with a stale
			// guest row flips the row, so the reaper can never eat an
			// upgraded account even if the upgrade handler's flip failed.
			name:   "stale guest row is healed from non-anonymous claims",
			claims: &webutil.AuthClaims{Sub: "upgraded-sub", Role: webutil.RoleAuthenticated, IsAnonymous: false},
			repo: &fakeUserRepo{byAuthID: map[string]*database.User{
				"upgraded-sub": {ID: uuid.New(), Email: "upgraded@example.com", IsAnonymous: true},
			}},
			wantStatus:    http.StatusOK,
			wantUserEmail: "upgraded@example.com",
			wantAnonFlip:  true,
		},
		{
			// An anonymous token must never flip the row the other way.
			name:   "anonymous claims leave the row untouched",
			claims: &webutil.AuthClaims{Sub: "guest-sub", Role: webutil.RoleAuthenticated, IsAnonymous: true},
			repo: &fakeUserRepo{byAuthID: map[string]*database.User{
				"guest-sub": {ID: uuid.New(), Email: "guest@example.com", IsAnonymous: true},
			}},
			wantStatus:    http.StatusOK,
			wantUserEmail: "guest@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUser *database.User
			h := UserLoader(tt.repo)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotUser = webutil.GetUserFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/first-run", nil)
			if tt.claims != nil {
				req = req.WithContext(webutil.WithAuthClaims(req.Context(), *tt.claims))
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantUserEmail != "" && (gotUser == nil || gotUser.Email != tt.wantUserEmail) {
				t.Errorf("context user = %+v, want email %q", gotUser, tt.wantUserEmail)
			}
			if len(tt.repo.created) != tt.wantCreated {
				t.Errorf("created rows = %d, want %d", len(tt.repo.created), tt.wantCreated)
			}
			if tt.wantCreated > 0 {
				if gotUser == nil {
					t.Error("JIT-provisioned user not placed in context")
				}
				if got := tt.repo.created[0].AuthID.String; got != tt.claims.Sub {
					t.Errorf("provisioned auth_id = %q, want claims sub %q", got, tt.claims.Sub)
				}
				if tt.wantEmail != "" {
					if got := tt.repo.created[0].Email; got != tt.wantEmail {
						t.Errorf("provisioned email = %q, want %q", got, tt.wantEmail)
					}
				}
			}
			if tt.wantAnonFlip {
				if len(tt.repo.anonymousFlips) != 1 || tt.repo.anonymousFlips[0] != false {
					t.Errorf("SetAnonymous calls = %v, want one flip to false", tt.repo.anonymousFlips)
				}
				if gotUser != nil && gotUser.IsAnonymous {
					t.Error("context user still marked anonymous after heal")
				}
			} else if len(tt.repo.anonymousFlips) != 0 {
				t.Errorf("unexpected SetAnonymous calls: %v", tt.repo.anonymousFlips)
			}
		})
	}
}
