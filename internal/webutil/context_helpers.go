package webutil

import (
	"context"

	"github.com/clownware/alpine-go-performance-starter/internal/database"
	"github.com/clownware/alpine-go-performance-starter/internal/repository"
)

// context keys (unexported)
type contextKey string

const (
	userContextKey contextKey = "user"
	repoContextKey contextKey = "userRepo"
)

// WithUser stores the user in the context.
func WithUser(ctx context.Context, user *database.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// GetUserFromContext retrieves the user from the context.
func GetUserFromContext(ctx context.Context) *database.User {
	user, _ := ctx.Value(userContextKey).(*database.User)
	return user
}

// WithUserRepo stores the user repo in the context.
func WithUserRepo(ctx context.Context, repo repository.UserRepository) context.Context {
	return context.WithValue(ctx, repoContextKey, repo)
}

// GetUserRepoFromContext retrieves the user repo from the context.
func GetUserRepoFromContext(ctx context.Context) repository.UserRepository {
	repo, _ := ctx.Value(repoContextKey).(repository.UserRepository)
	return repo
}
