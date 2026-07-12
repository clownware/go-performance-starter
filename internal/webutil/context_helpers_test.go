package webutil

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/clownware/go-performance-starter/internal/database"
	"github.com/clownware/go-performance-starter/internal/repository"
)

func TestUserContextRoundTrip(t *testing.T) {
	if got := GetUserFromContext(context.Background()); got != nil {
		t.Errorf("GetUserFromContext(empty) = %v, want nil", got)
	}

	user := &database.User{ID: uuid.New(), Email: "u@example.com"}
	ctx := WithUser(context.Background(), user)
	if got := GetUserFromContext(ctx); got != user {
		t.Errorf("GetUserFromContext = %v, want the stored user", got)
	}
}

func TestUserRepoContextRoundTrip(t *testing.T) {
	if got := GetUserRepoFromContext(context.Background()); got != nil {
		t.Errorf("GetUserRepoFromContext(empty) = %v, want nil", got)
	}

	var repo repository.UserRepository // typed nil interface value round-trips as stored
	ctx := WithUserRepo(context.Background(), repo)
	if got := GetUserRepoFromContext(ctx); got != nil {
		t.Errorf("GetUserRepoFromContext = %v, want nil for a nil repo", got)
	}
}

func TestCSRFTokenContextRoundTrip(t *testing.T) {
	if got := CSRFTokenFromContext(context.Background()); got != "" {
		t.Errorf("CSRFTokenFromContext(empty) = %q, want empty (middleware did not run)", got)
	}

	ctx := WithCSRFToken(context.Background(), "tok-123")
	if got := CSRFTokenFromContext(ctx); got != "tok-123" {
		t.Errorf("CSRFTokenFromContext = %q, want tok-123", got)
	}
}
