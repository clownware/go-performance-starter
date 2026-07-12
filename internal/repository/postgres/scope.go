package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clownware/go-performance-starter/internal/database"
	"github.com/clownware/go-performance-starter/internal/webutil"
)

// inScope runs fn against a querier carrying the request's identity, engaging
// RLS (ADR-004). This is the runtime counterpart of the pattern the RLS tests
// prove: on a transaction, SET LOCAL ROLE + request.jwt.claims so auth.uid()
// resolves to the requester and the *_self_access policies apply. The tables
// use FORCE ROW LEVEL SECURITY, so on Supabase (non-superuser connection)
// unscoped queries are denied — this wrapper is what makes app queries work
// in production at all.
//
// Without a pool (tests inject a tx-bound querier), fn runs on the fallback
// querier. With a pool, fn always runs inside a transaction — so multi-
// statement methods (e.g. SetPrimary, ADR-005) stay atomic — and the identity
// is applied only when claims are present (auth disabled → connection role).
func inScope[T any](ctx context.Context, db *pgxpool.Pool, fallback database.Querier, fn func(database.Querier) (T, error)) (T, error) {
	var zero T

	if db == nil {
		return fn(fallback)
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return zero, fmt.Errorf("begin scoped tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after commit

	if claims, ok := webutil.AuthClaimsFromContext(ctx); ok {
		if err := claims.Validate(); err != nil {
			return zero, err
		}
		claimsJSON, err := claims.JSON()
		if err != nil {
			return zero, fmt.Errorf("marshal jwt claims: %w", err)
		}
		// claims.Validate() restricts Role to the authenticated/anon
		// allowlist, so interpolating it is safe (SET ROLE cannot take a
		// bind parameter).
		if _, err := tx.Exec(ctx, "SET LOCAL ROLE "+claims.Role); err != nil {
			return zero, fmt.Errorf("set local role: %w", err)
		}
		// Set both the modern claims JSON and the legacy per-claim setting:
		// Supabase's auth.uid() coalesces request.jwt.claim.sub with
		// request.jwt.claims->>'sub' (the test stub mirrors this).
		if _, err := tx.Exec(ctx,
			"SELECT set_config('request.jwt.claims', $1, true), set_config('request.jwt.claim.sub', $2, true)",
			claimsJSON, claims.Sub,
		); err != nil {
			return zero, fmt.Errorf("set jwt claims: %w", err)
		}
	}

	out, err := fn(database.New(tx))
	if err != nil {
		return zero, err
	}
	if err := tx.Commit(ctx); err != nil {
		return zero, fmt.Errorf("commit scoped tx: %w", err)
	}
	return out, nil
}
