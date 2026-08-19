package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

// TestAdminListAnonymousUsers pins the reaper's auth-side discovery (#82):
// it pages through GET /auth/v1/admin/users with the service role key, keeps
// only anonymous identities, and stops when a page comes back short.
func TestAdminListAnonymousUsers(t *testing.T) {
	type page struct {
		status int
		body   string
	}
	anon := func(id, created string) string {
		return fmt.Sprintf(`{"id":%q,"created_at":%q,"is_anonymous":true}`, id, created)
	}
	registered := func(id string) string {
		return fmt.Sprintf(`{"id":%q,"created_at":"2026-01-01T00:00:00Z","email":"a@b.c","is_anonymous":false}`, id)
	}

	tests := []struct {
		name      string
		perPage   int
		pages     map[int]page // by page number (1-based)
		wantIDs   []string
		wantPages int
		wantErr   bool
	}{
		{
			name:    "single short page, registered users filtered out",
			perPage: 10,
			pages: map[int]page{
				1: {http.StatusOK, `{"users":[` + anon("g1", "2026-07-01T00:00:00Z") + `,` + registered("r1") + `]}`},
			},
			wantIDs:   []string{"g1"},
			wantPages: 1,
		},
		{
			name:    "full page triggers a second fetch; short second page ends the walk",
			perPage: 2,
			pages: map[int]page{
				1: {http.StatusOK, `{"users":[` + anon("g1", "2026-07-01T00:00:00Z") + `,` + anon("g2", "2026-07-02T00:00:00Z") + `]}`},
				2: {http.StatusOK, `{"users":[` + anon("g3", "2026-07-03T00:00:00Z") + `]}`},
			},
			wantIDs:   []string{"g1", "g2", "g3"},
			wantPages: 2,
		},
		{
			name:    "exactly-full last page is followed by an empty page, then stops",
			perPage: 1,
			pages: map[int]page{
				1: {http.StatusOK, `{"users":[` + anon("g1", "2026-07-01T00:00:00Z") + `]}`},
				2: {http.StatusOK, `{"users":[]}`},
			},
			wantIDs:   []string{"g1"},
			wantPages: 2,
		},
		{
			name:    "no users at all",
			perPage: 10,
			pages:   map[int]page{1: {http.StatusOK, `{"users":[]}`}},
			wantIDs: nil, wantPages: 1,
		},
		{
			name:    "gotrue error surfaces",
			perPage: 10,
			pages:   map[int]page{1: {http.StatusForbidden, `{"msg":"forbidden"}`}},
			wantErr: true,
		},
		{
			name:    "malformed body surfaces",
			perPage: 10,
			pages:   map[int]page{1: {http.StatusOK, `{"users":`}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPaths []string
			var gotAuth string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
				gotPaths = append(gotPaths, r.URL.Path)
				n, _ := strconv.Atoi(r.URL.Query().Get("page"))
				if pp := r.URL.Query().Get("per_page"); pp != strconv.Itoa(tt.perPage) {
					t.Errorf("per_page = %q, want %d", pp, tt.perPage)
				}
				p, ok := tt.pages[n]
				if !ok {
					t.Errorf("unexpected fetch of page %d", n)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(p.status)
				_, _ = w.Write([]byte(p.body))
			}))
			defer srv.Close()

			client := (&AuthClient{baseURL: srv.URL, anonKey: "anon-key"}).WithServiceRoleKey("sr-key")
			users, err := client.adminListAnonymousUsers(context.Background(), tt.perPage)

			if tt.wantErr {
				if err == nil {
					t.Fatal("adminListAnonymousUsers() = nil error, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("adminListAnonymousUsers() error: %v", err)
			}
			if gotAuth != "Bearer sr-key" {
				t.Errorf("Authorization = %q, want the service role key", gotAuth)
			}
			if len(gotPaths) != tt.wantPages {
				t.Errorf("fetched %d pages (%v), want %d", len(gotPaths), gotPaths, tt.wantPages)
			}
			for _, p := range gotPaths {
				if p != "/auth/v1/admin/users" {
					t.Errorf("path = %q, want /auth/v1/admin/users", p)
				}
			}
			if len(users) != len(tt.wantIDs) {
				t.Fatalf("got %d users, want %d: %+v", len(users), len(tt.wantIDs), users)
			}
			for i, want := range tt.wantIDs {
				if users[i].ID != want {
					t.Errorf("user %d = %q, want %q", i, users[i].ID, want)
				}
				if users[i].CreatedAt.IsZero() {
					t.Errorf("user %d has zero CreatedAt — the reaper needs it for the TTL cut", i)
				}
			}
		})
	}
}

func TestAdminListAnonymousUsers_NoServiceRoleKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("admin list without a service role key must not call GoTrue")
	}))
	defer srv.Close()

	client := &AuthClient{baseURL: srv.URL, anonKey: "anon-key"}
	if _, err := client.AdminListAnonymousUsers(context.Background()); err == nil {
		t.Error("AdminListAnonymousUsers() without service role key = nil error, want error")
	}
}

// TestAdminListAnonymousUsers_PageCap proves a GoTrue that never returns a
// short page cannot spin the reaper forever.
func TestAdminListAnonymousUsers_PageCap(t *testing.T) {
	var fetches int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetches++
		_, _ = fmt.Fprintf(w, `{"users":[{"id":"g%d","created_at":%q,"is_anonymous":true}]}`, fetches, time.Now().UTC().Format(time.RFC3339))
	}))
	defer srv.Close()

	client := (&AuthClient{baseURL: srv.URL, anonKey: "anon-key"}).WithServiceRoleKey("sr-key")
	if _, err := client.adminListAnonymousUsers(context.Background(), 1); err == nil {
		t.Fatal("expected an error once the page cap is hit, got nil")
	}
	if fetches > maxAdminListPages {
		t.Errorf("fetched %d pages, cap is %d", fetches, maxAdminListPages)
	}
}
