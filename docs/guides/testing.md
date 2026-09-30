# Testing

How tests are written, which layer a new test belongs in, and how to run them. The rules come from [ADR-023](../adr/ADR-023-Testing-Philosophy.md) (testing philosophy), [ADR-032](../adr/ADR-032-Mutation-Testing.md) (mutation testing) and [ADR-021](../adr/ADR-021-Halt-On-Violation-Quality-Gate.md) (`task ci` is the one definition of done). Tooling is [ADR-010](../adr/ADR-010-Testing-and-Code-Quality.md): the standard `testing` package, table-driven cases, golangci-lint, gofmt. The `## Testing` section of [`.claude/engineering.md`](../../.claude/engineering.md) is the short form of this page.

## The rule: failing test first

Constitution rule 8 ([`CLAUDE.md`](../../CLAUDE.md)): non-trivial production code follows a failing test. In the three-pass workflow ([ADR-020](../adr/ADR-020-Agent-Roles.md), [`.claude/workflow.md`](../../.claude/workflow.md)) the **Architect** pass writes the failing table-driven test and no production code; the **Coder** pass writes the minimum code to pass it and does not edit the test beyond what the Architect scaffolded; the **Reviewer** pass runs `task ci`. If you are working solo, play all three roles in that order. ADR-023 adds the style rules: one behaviour per case, cases named for the behaviour (not the implementation), and both success and error paths covered.

## Test layers you have

| Layer | What it proves | Real examples |
|-------|----------------|---------------|
| Unit, table-driven | Pure logic: config validation, percentile maths, budget checks | `internal/config/config_validate_test.go`, `internal/performance/observed_test.go`, `internal/validate/validate_test.go` |
| Handler via `httptest` + repository fake | Status, redirects, HTMX headers, and what the handler asked the repository to do — no Postgres | `internal/handler/flashcard_handlers_test.go` (`fakeFlashcardRepo`), `quiz_handlers_test.go`, `upgrade_handlers_test.go` |
| Middleware | Each middleware in isolation through `httptest` | `internal/middleware/ratelimit_test.go`, `metrics_test.go`, `csrf_test.go` |
| Repository integration (gated on `DATABASE_URL`) | sqlc queries and RLS policies against real Postgres; skipped when the variable is unset | `internal/repository/postgres/integration_test.go`, `rls_test.go`, `scope_rls_test.go` |
| Server wiring | The assembled router: middleware order as observable behaviour, route table (`TestServer_RouteTable`), static cache headers, CSRF/HSTS/metrics gating | `internal/server/wiring_test.go`, `server_test.go` (uses a `fakeGoTrue` httptest server instead of Supabase) |
| Design tokens | ADR-029 discipline over `.templ` sources (no `dark:` variants, no raw palette steps) and WCAG contrast computed from `input.css` | `internal/view/tokens_test.go` (`TestTemplTokenDiscipline`), `tokens_contrast_test.go` (`TestTokenContrast`) |
| Performance budgets | The ADR-000 constants and the ADR-034 observer (nearest-rank percentiles, shipped-asset lists shared with the CI gate) | `internal/performance/budgets_test.go`, `observed_test.go` (`TestShippedAssetPaths`) |
| Background jobs | The reaper against a `fakeReaperStore` | `internal/jobs/reaper_test.go` |
| ADR enforcement suite | Each ADR's testable consequences, checked by `scripts/adrcheck` from `checks/enforcement.config.json` (ADR-033) | `task check:adr` |
| Mutation | Whether the tests *assert*, not merely execute, in five pure-logic packages (ADR-032) | `task test:mutation` |

There are no browser or end-to-end tests. A templ component is tested through the handler or server test that renders it (assert on `data-testid` attributes and body text, as `TestFlashcardCreate` does).

## Writing a handler test

Handlers take a repository interface ([ADR-003](../adr/ADR-003-SQL-Code-Generation-and-Data-Access.md)), so the test hands them a hand-rolled fake and drives them through `httptest`. The fake records every mutation so the test can assert on *what the handler asked for*, not just the response:

```go
// internal/handler/flashcard_handlers_test.go
type fakeFlashcardRepo struct {
	cards []database.Flashcard
	// scoped makes ListByUser behave like RLS: rows come back only when the
	// context claims name the owner (asFlashcardUser's "auth-<id>" sub).
	scoped bool

	createErr error
	listErr   error
	setErr    error
	deleteErr error

	created []database.CreateFlashcardParams
	known   []setKnownCall
	deleted []deleteCall
}

var _ repository.FlashcardRepository = (*fakeFlashcardRepo)(nil)
```

Cases are a struct slice with the inputs and every expectation; the body builds the request, injects identity the way the real middleware chain would (`asFlashcardUser` puts the user and auth claims on the context), serves it through a chi router, and checks both the response and the fake:

```go
// internal/handler/flashcard_handlers_test.go (TestFlashcardCreate, abridged)
		{
			name:       "blank front is a 422 and creates nothing",
			form:       url.Values{"front": {"   "}, "back": {"something"}},
			repo:       &fakeFlashcardRepo{},
			user:       user,
			htmx:       true,
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "repository failure is a 500",
			form:       url.Values{"front": {"a"}, "back": {"b"}},
			repo:       &fakeFlashcardRepo{createErr: errFake},
			user:       user,
			htmx:       true,
			wantStatus: http.StatusInternalServerError,
		},
	// ...
			req := httptest.NewRequest(http.MethodPost, "/learn/flashcards", strings.NewReader(tt.form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req = asFlashcardUser(req, tt.user)
			if tt.htmx {
				req.Header.Set("HX-Request", "true")
			}
			w := httptest.NewRecorder()

			newFlashcardRouter(tt.repo).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("POST /learn/flashcards status = %d, want %d", w.Code, tt.wantStatus)
			}
```

For a new handler, copy this shape: a fake per repository interface (with injectable errors and a call record), a `newXRouter` helper that mounts the real routes, and cases for the HTMX path, the non-JS redirect path, each validation failure, the repository error, and the missing user.

## Writing a repository test

Repository tests in `internal/repository/postgres/` run the real sqlc queries against real Postgres and skip themselves when `DATABASE_URL` is unset:

```go
// internal/repository/postgres/integration_test.go
func withTx(t *testing.T) (context.Context, *database.Queries) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
```

`withTx` opens a transaction and rolls it back on cleanup, so CRUD tests leave no rows behind. RLS tests ([ADR-004](../adr/ADR-004-Authorization-Strategy-RLS.md)) seed as the owner role, then switch identity inside the same transaction before asserting:

```go
// internal/repository/postgres/rls_test.go
	// Become authenticated user A. SET LOCAL is rolled back with the tx.
	if _, err := tx.Exec(ctx, "SET LOCAL ROLE authenticated"); err != nil {
		t.Fatalf("set role authenticated: %v", err)
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('request.jwt.claim.sub', $1, true)", authA); err != nil {
		t.Fatalf("set jwt claim: %v", err)
	}
```

That works on vanilla Postgres only because of the **auth stub**: `sql/test/auth_stub.sql` creates the `auth` schema, an `auth.uid()` that reads the JWT claim settings, and the `anon`/`authenticated`/`service_role` roles that Supabase provides in production. It is applied before migrations and must never become a migration itself. Locally:

```bash
task db:up             # docker-compose Postgres
task db:test:setup     # auth stub + migrations against $DATABASE_URL
task test              # the postgres package now runs instead of skipping
```

`TestScopedRepoRLSIsolation` in `scope_rls_test.go` is the production-path counterpart: it goes through the repository's own `inScope` (which applies the role and claims from context) rather than injecting them by hand, and its stranger-identity case is what ADR-034 cites as proof that RLS, not a `WHERE` clause, hides other users' rows.

## Running things

| Command | What it runs |
|---------|--------------|
| `task test` | `go test -v ./...` — the inner loop |
| `task test:coverage` | `go test -coverprofile=coverage.out ./...` then `go tool cover -html` |
| `task test:performance` | `go test -v ./internal/performance/...` + `test:binary-size` + `test:asset-budgets` |
| `task test:mutation` | `mutation:install` (gremlins v0.6.0), then `gremlins unleash --timeout-coefficient 20` on each scoped package |
| `task check:adr` | `go run ./scripts/adrcheck` — warn-status checks report, block-status checks fail |
| `task ci` | The gate: `fmt:check` → `lint` → `go test -race -covermode=atomic -coverprofile=coverage.txt ./...` → `agents:check` → `versions:check` → `check:adr` → `check:generated` → `test:binary-size` → `test:asset-budgets` → `scan:vuln` |

Run `task ci` before claiming any change complete (Constitution rule 9). If it exits non-zero, fix the failure; do not lower a threshold, exclude a file, or push with `--no-verify`.

Agents get a second net. `.claude/settings.json` registers a `Stop` hook, [`.claude/hooks/stop-gate.sh`](../../.claude/hooks/stop-gate.sh), that runs `go test ./...` and then `go run ./scripts/adrcheck` whenever an agent tries to end its turn. A test failure or a BLOCKER exits 2, which blocks completion and feeds the tail of the output back into the loop; `WARNING` lines pass through as information. It honours `stop_hook_active` so it cannot ping-pong, and `STOP_GATE_OFF=1` disables it for a session. Note the Stop-gate runs plain `go test`, not the race-enabled run: `task ci` is still the gate.

## Mutation testing

gremlins is configured entirely in `Taskfile.yml` (there is no `.gremlins.yaml`). `task test:mutation` runs `internal/middleware`, `internal/auth`, `internal/config`, `internal/validate` and `internal/jobs` — the well-covered, pure-logic packages ADR-032 scopes. Generated code, the `DATABASE_URL`-gated repository suites and render-only view code are out of scope, and a package only joins the list after it has first-order coverage. The run is manual; it is deliberately not in `task ci`, and ADR-032's TC-1 makes any addition a public diff.

Read a surviving mutant as a finding, not a score: no `--threshold-efficacy` is set, so the question for each `LIVED` line is "which test case would have killed this?" Either add that case or record the survivor with a reason. Timed-out mutants count as killed (`--timeout-coefficient 20` keeps slow-but-legitimate kills from being misreported).

Worked example: [#138](https://github.com/clownware/go-performance-starter/issues/138). The run leaves one survivor in `internal/middleware/metrics.go`, the slow-request condition `duration > performance.MaxP95ResponseTime`. Mutating `>` to `>=` survives because `TestMetrics_SlowRequestLogged` can only produce real wall-clock durations (a no-sleep request and a budget+20 ms request), never a request that takes exactly the budget. The options on the issue are to accept it as a documented survivor or to thread a `now func() time.Time` seam through the middleware so the boundary becomes a table-driven row — `TestLimiterStore_Evict` in `ratelimit_test.go` shows the seam pattern already used for the limiter's staleness boundary.

## What CI does differently

[`.github/workflows/ci.yml`](../../.github/workflows/ci.yml) does not re-implement the checks; it provides an environment and calls `task ci`, so the gate cannot drift from the local one (ADR-021's 2026-07 amendment). The differences from your laptop:

- A `postgres:16-alpine` service container with `DATABASE_URL` and `ENV=test` set, bootstrapped by `task db:test:setup` before the gate — so the repository and RLS suites that skip locally without a database always run in CI.
- Tool installs pinned to the repo's own sources (templ from `go.mod`, sqlc from `versions.json`, govulncheck via `go install tool`).
- After the gate, `coverage.txt` is uploaded to Codecov with `fail_ci_if_error: false`. That upload is informational; it is not a check and cannot fail the job.
- On pull requests a comment reports the binary size against the 20 MB budget (`continue-on-error: true`).
- A separate `docker` job builds the image and fails if it exceeds the 30 MB ADR-000 budget — the one check that only exists in CI.

## Thresholds

There is no enforced coverage floor. ADR-000 still lists "Test Coverage: 80%+" under its code-quality metrics, but ADR-023 rejected a coverage percentage as the rule (a number invites gaming) and nothing in `task ci`, the workflow or Codecov gates on one. The thresholds that do exist are the [ADR-000](../adr/ADR-000-Performance-Budgets-and-Quality-Attributes.md) Enforced budgets (binary 20 MB, Docker image 30 MB, JS 50 KB and CSS 30 KB gzipped), the WCAG ratios in `TestTokenContrast`, and the warn/block statuses in `checks/enforcement.config.json`. Never lower any of them to make a test pass: fix the code, or open an ADR that documents the exception (ADR-023 rule 4, `.claude/engineering.md`). Promoting an enforcement check from warn to block follows [ADR-033](../adr/ADR-033-ADR-Enforcement-Architecture.md)'s graduation rule; demoting one that false-positives is always allowed.
