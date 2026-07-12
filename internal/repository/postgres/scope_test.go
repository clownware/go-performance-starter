package postgres

import (
	"context"
	"testing"

	"github.com/clownware/go-performance-starter/internal/database"
	"github.com/clownware/go-performance-starter/internal/webutil"
)

// scopeFakeQuerier lets unit tests observe which querier inScope hands to fn.
// Embedding the interface satisfies it without implementing every method.
type scopeFakeQuerier struct {
	database.Querier
	id string
}

func TestInScope_FallbackWithoutClaims(t *testing.T) {
	fallback := &scopeFakeQuerier{id: "fallback"}

	got, err := inScope(context.Background(), nil, fallback, func(q database.Querier) (string, error) {
		fq, ok := q.(*scopeFakeQuerier)
		if !ok {
			return "", nil
		}
		return fq.id, nil
	})
	if err != nil {
		t.Fatalf("inScope error: %v", err)
	}
	if got != "fallback" {
		t.Errorf("querier = %q, want the fallback querier when no claims are set", got)
	}
}

func TestInScope_FallbackWithClaimsButNilPool(t *testing.T) {
	// Tests inject a tx-bound querier with a nil pool; claims must not
	// cause a nil-pool panic — the fallback path wins.
	ctx := webutil.WithAuthClaims(context.Background(), webutil.AuthClaims{
		Sub: "user-1", Role: webutil.RoleAuthenticated,
	})
	fallback := &scopeFakeQuerier{id: "fallback"}

	got, err := inScope(ctx, nil, fallback, func(q database.Querier) (string, error) {
		fq, _ := q.(*scopeFakeQuerier)
		return fq.id, nil
	})
	if err != nil {
		t.Fatalf("inScope error: %v", err)
	}
	if got != "fallback" {
		t.Errorf("querier = %q, want fallback with nil pool", got)
	}
}
