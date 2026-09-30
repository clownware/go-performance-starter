# Troubleshooting

Things that have bitten more than once. Each entry names the symptom, the cause, and the fix; if you hit something new, add it here in the PR that fixes it.

## pgx fails at boot with a parse error on `DATABASE_URL`

The database password contains characters that are not URL-safe (`@`, `#`, `/`, `%`, `?`). URL-encode the password portion of the connection string (`@` → `%40` and so on). Supabase-generated passwords do this to you regularly.

## `task db:migrate:up` fails with `schema "auth" does not exist`

You ran the migrations against a vanilla Postgres (the local Docker one, or a CI service). The RLS migrations call Supabase's `auth.uid()`. Run `task db:test:setup` instead: it applies `sql/test/auth_stub.sql` and then migrates. Never apply the stub to a real Supabase database.

## Repository or integration tests are skipped

They run only when `DATABASE_URL` is set. `task db:up && task db:test:setup && task test` is the local sequence; CI does the same with a Postgres service ([testing.md](guides/testing.md)).

## `task dev` cannot find `templ`, `sqlc`, `migrate`, `air`, or `psql`

They are prerequisites, not auto-installed. The pinned `go install` lines are in the README's prerequisites block; `psql` ships with Postgres. `golangci-lint`, `govulncheck` and `gremlins` install themselves on first use.

## `check:generated` fails after a templ or sqlc version bump

The committed generated files were produced by a different generator version than the one you have. Regenerate (`task templ:generate`, `task db:generate`) and commit the output together with the bump. The Docker builder derives the templ CLI version from `go.mod`, so the image and the repo cannot drift from each other.

## `versions:check` fails on a Dependabot branch

Dependabot bumped a pin but not the manifest. Run `task versions:sync` on that branch and push ([ADR-030](adr/ADR-030-Versions-Manifest-Contract.md)). Adding keys to `versions.json` is fine; renaming or removing one is a breaking change for consumers.

## `agents:check` fails

You edited `CLAUDE.md` or a file under `.claude/` without regenerating. Run `task agents:build` and commit `AGENTS.md`. Never edit `AGENTS.md` directly. Links inside `.claude/*.md` must be written relative to that directory (`../docs/...`, `../.claude/roles/...`) so the generator can rewrite them for the repo root.

## An agent's edit was denied by the ADR guard, or its turn was blocked by the Stop-gate

Both are ADR-033 hooks working as designed. The guard denies `Edit`/`Write` to existing ADRs, `AGENTS.md`, `internal/database/*.go` and `*_templ.go`, and names the legal move in its message. The Stop-gate runs `go test ./...` and `scripts/adrcheck` when an agent tries to finish; fix the failure rather than bypass it. Kill-switches for operator-reviewed exceptions: `ADR_GUARD_OFF=1` and `STOP_GATE_OFF=1`, set in the environment Claude Code runs in.

## `task ci` passes locally but the CI docker job fails the 30MB budget

The binary budget (20MB) and the image budget (30MB) are separate gates; a new dependency can pass the first and fail the second. `task docker:build` then `docker image inspect go-performance-starter --format '{{.Size}}'` reproduces it locally ([performance.md](guides/performance.md)).

## Rate limited (429) while developing

Every 429 carries `Retry-After`. The tiers are per client IP; behind a proxy with `TRUSTED_PROXY_CIDRS` unset every request shares one bucket, which is the fail-closed default ([ADR-027](adr/ADR-027-Trusted-Proxy-Client-IP.md)). Locally there is no proxy, so this is usually the strict `/auth` tier (5 attempts per minute) doing its job.

## The local launch config wants `op`

`.claude/launch.json`'s `api` entry injects secrets from 1Password (`op run --env-file=.env.tpl`). If you keep secrets in `.env` instead, use the `api-dotenv` entry, which runs `go run ./cmd/api` and lets `godotenv` load `.env`.

## Password reset emails link to Supabase, not the app

The reset flow verifies the `token_hash` server-side, so the Supabase *Reset Password* email template must link to `{{ .SiteURL }}/auth/reset?token_hash={{ .TokenHash }}&type=recovery`. The README's Quick Start has the exact setting.
