package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supabase-community/gotrue-go/types"

	"github.com/clownware/alpine-go-performance-starter/internal/middleware"
)

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
		name string
		user *types.UserResponse
	}{
		{name: "anonymous context"},
		{name: "authenticated user", user: &types.UserResponse{User: types.User{Email: "ada@example.com"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/profile", nil)
			if tt.user != nil {
				req = withAuthUser(req, tt.user)
			}
			w := httptest.NewRecorder()

			ProfileView(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("ProfileView() status = %d, want %d", w.Code, http.StatusOK)
			}
			if w.Body.Len() == 0 {
				t.Error("ProfileView() rendered an empty body")
			}
		})
	}
}

func TestProfileUpdate(t *testing.T) {
	tests := []struct {
		name         string
		form         string
		htmx         bool
		wantStatus   int
		wantLocation string
		wantTrigger  bool
	}{
		{name: "malformed form body returns 400", form: "%zz", wantStatus: http.StatusBadRequest},
		{name: "empty name via htmx re-renders form with 422", form: "name=+", htmx: true, wantStatus: http.StatusUnprocessableEntity},
		{name: "empty name without htmx re-renders page with 422", form: "name=", wantStatus: http.StatusUnprocessableEntity},
		{name: "valid name via htmx returns form with trigger", form: "name=Ada", htmx: true, wantStatus: http.StatusOK, wantTrigger: true},
		{name: "valid name without htmx redirects", form: "name=Ada", wantStatus: http.StatusSeeOther, wantLocation: "/profile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := formRequest("/profile", tt.form)
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
		})
	}
}
