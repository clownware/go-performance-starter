# Data access

Every row this application reads or writes goes through one pipeline: a raw-SQL migration ([ADR-002](../adr/ADR-002-Database-Migration-Tool.md)), a query file compiled to typed Go by sqlc and wrapped in a repository interface ([ADR-003](../adr/ADR-003-SQL-Code-Generation-and-Data-Access.md)), and a per-request transaction that carries the caller's identity so Postgres Row Level Security decides what is visible ([ADR-004](../adr/ADR-004-Authorization-Strategy-RLS.md)). Multi-statement operations stay atomic inside that transaction ([ADR-005](../adr/ADR-005-Handling-Primary-Organization-Selection.md)). The demo domain — quiz questions, attempts and flashcards — is built the same way ([ADR-024](../adr/ADR-024-Demo-Application-Direction.md)), and its public instance is seeded and reset through gated Taskfile targets ([ADR-031](../adr/ADR-031-Public-Demo-Operations.md)). This guide walks that pipeline for a developer adding a table, a query and a repository method.

## The pipeline

1. **Migration.** `task db:migrate:create -- add_widgets` runs `migrate create -ext sql -dir migrations -seq`, producing an `.up.sql`/`.down.sql` pair. Both are hand-written; `adr002-migration-pairs` in `scripts/adrcheck` warns if one is missing. Indexes, constraints, triggers and RLS policies live here and only here.
2. **Schema for sqlc.** Mirror the new columns in `sql/schema/schema.sql`. As its header says, it is a type source for sqlc, not a migration: no indexes, constraints or triggers, but column names and types must match the migrated table or the generated structs will be wrong.
3. **Queries.** Add named queries to a file in `sql/queries/` (`-- name: GetWidget :one`). `sqlc.yaml` points sqlc at `sql/queries/` and `sql/schema/`, generates into `internal/database` with `sql_package: "pgx/v5"`, emits the `Querier` interface, and maps `uuid` to `github.com/google/uuid.UUID` and `timestamptz` to `time.Time`.
4. **Generate.** `task db:generate` runs `sqlc generate`. The output is committed; `task check:generated` (part of `task ci`) regenerates into a scratch copy and diffs it, so stale generated code is caught before merge.
5. **Repository.** Declare the method on an interface in `internal/repository/` and implement it in `internal/repository/postgres/`, calling the generated function through `inScope` (see below). Map `pgx.ErrNoRows` to `repository.ErrNotFound`.
6. **Handler.** Handlers depend on the interface, never on `internal/database.Queries` or `pgx`; `adr003-no-sql-in-handlers` in `scripts/adrcheck` flags SQL strings or pgx imports there. Repositories are constructed once in `internal/server/server.go` (`postgres.NewFlashcardRepo(s.db, database.New(s.db))`) and passed to route registrars such as `handler.FlashcardRoutes`.

## Adding a query: the flashcard "known" toggle

The existing `SetFlashcardKnown` chain is the model. The query:

```sql
-- sql/queries/flashcards.sql
-- name: SetFlashcardKnown :one
UPDATE flashcards
SET is_known = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, user_id, question_id, front, back, is_known, created_at, updated_at;
```

After `task db:generate`, `internal/database/querier.go` gains `SetFlashcardKnown(ctx, SetFlashcardKnownParams) (Flashcard, error)`. The interface in `internal/repository/flashcard.go` exposes it as `SetKnown(ctx, id uuid.UUID, known bool) (*database.Flashcard, error)`, and the implementation is:

```go
// internal/repository/postgres/flashcard.go
func (r *FlashcardRepo) SetKnown(ctx context.Context, id uuid.UUID, known bool) (*database.Flashcard, error) {
	card, err := inScope(ctx, r.db, r.querier, func(q database.Querier) (database.Flashcard, error) {
		return q.SetFlashcardKnown(ctx, database.SetFlashcardKnownParams{
			ID:      id,
			IsKnown: known,
		})
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &card, nil
}
```

`flashcardSetKnown` in `internal/handler/flashcard_handlers.go` calls `repo.SetKnown` and turns `repository.ErrNotFound` into a 404. Note the query has no `user_id` predicate: ownership is enforced by the `flashcards_self_access` policy, not by the SQL. Handler tests substitute `fakeFlashcardRepo` (`internal/handler/flashcard_handlers_test.go`) for the interface, so no database is needed there.

Two conventions from `sql/queries/users.sql` are worth copying. Narrow updates (`UpdateUserName`, `SetUserLastLogin`) exist because the generic `COALESCE`-based `UpdateUser` silently overwrote columns when a caller passed `''` instead of `NULL`. And ADR-005's `SetPrimaryOrganizationStep1`/`Step2` show how to split a statement sqlc cannot parse: `SetPrimary` in `internal/repository/postgres/organization_member.go` runs both inside the single transaction `inScope` opens.

## RLS scoping

Every table has `ENABLE` and `FORCE ROW LEVEL SECURITY` (migrations `000002` and `000003`), so even the table owner is policy-checked. The policies resolve identity through `auth.uid()`: `users_self_access` compares `auth_id = auth.uid()::text`; `quiz_attempts_self_access` and `flashcards_self_access` look the owner up through `users`; the organization policies (`organizations_member_select`, `organizations_owner_or_admin_update`, `org_members_select`, `org_members_owner_modify`) use the `SECURITY DEFINER` helper `public.user_is_organization_member(uuid)` or an `EXISTS` join; `quiz_questions_public_read` is `USING (true)`. Each table also has a `service_role_bypass` policy scoped `TO service_role`.

`inScope` in `internal/repository/postgres/scope.go` is what makes application queries pass those policies. With a pool it begins a transaction, and when `webutil.AuthClaimsFromContext` finds claims it validates them (`Role` must be `authenticated` or `anon`, the allowlist in `internal/webutil/auth_claims.go`), runs `SET LOCAL ROLE`, then sets both `request.jwt.claims` (JSON) and `request.jwt.claim.sub` via `set_config(..., true)` because Supabase's `auth.uid()` coalesces the two. Without claims the transaction runs as the connection role; with a nil pool (unit tests injecting a transaction-bound querier) it runs `fn` against the fallback querier. Multi-statement methods stay atomic because the whole callback executes in that one transaction.

Two migrations patch identity handling on already-deployed databases. `000005_rescope_service_role_bypass` drops and recreates the three original bypass policies scoped `TO service_role`; its `.down.sql` is deliberately a no-op (`SELECT 1`) so a rollback cannot reopen the unscoped bypass. `000006_add_is_anonymous_to_users` adds `users.is_anonymous`, mirrored from the JWT `is_anonymous` claim at provisioning and flipped to `false` by `SetUserIsAnonymous` on upgrade, which is what exempts a row from the reaper.

The reaper is the one caller that runs as `service_role`: `ReaperRepo` (`internal/repository/postgres/reaper.go`) opens its own transaction and executes `SET LOCAL ROLE service_role` before `DeleteExpiredAnonymousUsers` and `ListExistingAuthIDs`. That role is intentionally absent from `inScope`'s allowlist; request paths must never assume it.

Vanilla Postgres has no `auth` schema or Supabase roles, so migrations `000002` onward fail on it. `sql/test/auth_stub.sql` creates `auth.uid()` (the same coalesce as Supabase), the `anon`, `authenticated` and `service_role` roles (`service_role` with `BYPASSRLS`), and default grants. It is applied by `task db:auth-stub` and must never become a migration — on Supabase it would shadow the real function.

## Local and CI bootstrap

`task db:up` starts the `postgres:16-alpine` container from `docker-compose.yml` (database `alpine_saas` on port 5432; `task db:down` stops it). With `DATABASE_URL` pointing at it, `task db:test:setup` applies the auth stub and then `task db:migrate:up`; `task db:migrate:down` reverts one step and `task db:migrate:force` resets the version marker when a migration half-applied.

CI (`.github/workflows/ci.yml`) provides the same environment: a `postgres:16-alpine` service, `DATABASE_URL` in the job env, `task db:test:setup`, then `task ci`. Repository tests in `internal/repository/postgres/*_test.go` call `t.Skip` when `DATABASE_URL` is unset, so `go test ./...` passes without a database locally and exercises Postgres in CI. `integration_test.go` wraps each test in a rolled-back transaction as the connection role, which is superuser in CI, so those tests verify CRUD behaviour and not RLS; `rls_test.go` and `scope_rls_test.go` are the isolation proofs — the first sets the role and claims by hand, the second gives `NewFlashcardRepo` the real pool and context claims exactly as production does. Fixtures for the second must be committed, since `inScope` cannot see another transaction's rows.

Production migrations run from `.github/workflows/db-migrate.yml`: on a push to `master` touching `migrations/**`, a `validate` job applies the stub and the full chain to a fresh Postgres, and only then does the `migrate` job run `migrate ... up` against `SUPABASE_DATABASE_URL`, skipping cleanly when the secret is absent.

## Schema, constraints and indexes

Six tables, all UUID-keyed via `uuid_generate_v4()` from the `uuid-ossp` extension: `organizations`, `users`, `organization_members` (`000001`), `quiz_questions`, `quiz_attempts`, `flashcards` (`000003`). Later migrations add `users.first_run_complete` (`000004`, idempotent `ADD COLUMN IF NOT EXISTS`) and `users.is_anonymous` (`000006`).

Constraints are the integrity story; there are no `CHECK` constraints anywhere, and `organization_members.role` accepts any string — the `'owner'`/`'admin'`/`'member'` vocabulary is a column comment enforced by application code. Uniqueness: `users.email`, `users.auth_id`, `organizations.slug`, `quiz_questions.slug`, and the composite `unique_org_user (organization_id, user_id)`. Foreign keys: `organization_members` cascades from both `users` and `organizations`; `quiz_attempts.user_id` and `question_id` cascade; `flashcards.user_id` cascades while `flashcards.question_id` is `ON DELETE SET NULL`, so deleting a seeded question keeps the card. Deleting a `users` row therefore removes that user's attempts and flashcards, which is what the reaper's `DeleteExpiredAnonymousUsers` relies on. `updated_at` is maintained by the `update_updated_at_column()` trigger on `organizations`, `users`, `organization_members` and `flashcards`.

Explicit indexes, beyond what `PRIMARY KEY` and `UNIQUE` create implicitly:

| Index | Definition | Serves |
|---|---|---|
| `idx_organizations_slug` | `organizations(slug)` | redundant with the `UNIQUE` on `slug` |
| `idx_users_email` | `users(email)` | redundant with the `UNIQUE` on `email` |
| `idx_org_members_org_id` | `organization_members(organization_id)` | FK joins, cascades, `ListOrganizationMembers` |
| `idx_org_members_user_id` | `organization_members(user_id)` | `ListUserOrganizations`, membership policies |
| `idx_quiz_questions_topic` | `quiz_questions(topic)` | topic filtering |
| `idx_quiz_attempts_user_id` | `quiz_attempts(user_id)` | `ListQuizAttemptsByUser`, counts, `quiz_attempts_self_access` |
| `idx_flashcards_user_id` | `flashcards(user_id)` | `ListFlashcardsByUser`, `flashcards_self_access` |
| `idx_users_anonymous_created` | `users (created_at) WHERE is_anonymous` | partial index for the reaper's age scan |

Postgres does not index the referencing side of a foreign key, so every new FK column that appears in a `WHERE`, a join or an RLS policy needs its own index in the migration that creates it — the RLS `EXISTS` subqueries run on every row a policy inspects.

## Demo data

Quiz questions are reference content shipped as migrations, not fixtures: `000007_seed_quiz_questions` inserts the ten explainer questions and `000008_seed_quiz_proof_questions` the three ADR-034 proof-surface questions, both `ON CONFLICT (slug) DO NOTHING` so re-running is harmless; their `.down.sql` files delete by slug. `sql/demo/seed.sql` simply `\ir`-includes both migration files, so there is one source of truth. `sql/demo/reset.sql` deletes `flashcards` and `quiz_attempts` belonging to `is_anonymous` users and nothing else — never `users` rows (identity deletion belongs to the reaper), never registered users' data, never `quiz_questions`.

`task demo:seed` runs the seed and `task demo:reset` runs the reset then re-seeds; both carry a Taskfile `preconditions` check that refuses unless `DEMO_MODE=1` is in the environment. The nightly `.github/workflows/demo-reset.yml` adds a second gate on the repo variable before it ever reaches the Taskfile. Anything a real deployment needs belongs in a migration, not in `sql/demo/`.

## Pool tuning

`internal/config/config.go` reads four variables and applies them in `Config.PoolConfig()`, which parses `DATABASE_URL` with `pgxpool.ParseConfig` and overrides the pgx defaults (whose `MaxConns` is only `max(4, CPUs)`):

| Variable | Default | pgxpool field |
|---|---|---|
| `DB_MAX_CONNS` | `25` | `MaxConns` (must be at least 1) |
| `DB_MIN_CONNS` | `2` | `MinConns` (0 to `DB_MAX_CONNS`) |
| `DB_MAX_CONN_LIFETIME` | `30m` | `MaxConnLifetime` |
| `DB_MAX_CONN_IDLE_TIME` | `5m` | `MaxConnIdleTime` |

`Config.Validate` rejects out-of-range values at boot. Because `inScope` opens a transaction per repository call, a request that touches several repositories briefly holds several connections; size `DB_MAX_CONNS` against the database's connection limit with that in mind. See [Auth and security](auth-and-security.md) for where the claims that `inScope` applies come from, and [Background jobs](background-jobs.md) for the reaper that consumes the partial index above.
