# Background jobs

The starter's background needs are periodic maintenance, not queues, so it ships two mechanisms and no job framework: an in-process goroutine with a ticker, and a GitHub Actions cron. Redis-backed queues (Asynq and friends) are ruled out by the stateless-container topology ([ADR-025](../adr/ADR-025-Deployment-Target.md) §3) and the minimal-dependency ethos ([ADR-000](../adr/ADR-000-Performance-Budgets-and-Quality-Attributes.md)); if a real queue is ever needed it is an ADR, not a `go get`.

## In-process: the guest reaper

`internal/jobs/reaper.go` is the worked example ([ADR-024](../adr/ADR-024-Demo-Application-Direction.md)). Anonymous guest identities older than `GUEST_TTL` (default 720h; 48h on the public demo per `fly.toml`) are deleted every `REAPER_INTERVAL` (default 1h) in two passes: the row pass deletes expired `public.users` rows and their GoTrue twins; the orphan pass lists anonymous GoTrue users via the admin API and deletes the ones with no row (#82). It is wired in `cmd/api/main.go` only when `GUEST_MODE_ENABLED` is true, and the auth-side passes need `SUPABASE_SERVICE_ROLE_KEY`.

What makes it a good template for the next job:

- **Seams, not globals.** `ReaperStore` (data access, implemented by `postgres.NewReaperRepo` inside a `service_role`-scoped transaction), `AuthUserDeleter` and `AuthUserLister` (nil disables that pass). The tests in `internal/jobs/` drive it with fakes and a controllable clock.
- **Context-bound lifecycle.** `reaper.Start(ctx)` runs until the server's shutdown context is cancelled; no leaked goroutines on `SIGTERM`.
- **Failures are logged, never guessed through.** A pass that cannot confirm a user is orphaned does nothing to it.
- **Bounded work per tick.** The admin listing is paginated and page-capped so one tick cannot run unbounded.

## Outside the process: GitHub Actions cron

Work that should not depend on the app being up runs as a scheduled workflow. `.github/workflows/demo-reset.yml` resets the public demo's guest content nightly at 07:00 UTC ([ADR-031](../adr/ADR-031-Public-Demo-Operations.md)), double-gated on the `SUPABASE_DATABASE_URL` secret and the `DEMO_MODE=1` repository variable so a clone never runs it by accident. `vuln-scan.yml` is the other scheduled job. Cloudflare is proxy and CDN only, not a scheduled-task runtime.

## Choosing

| Need | Use |
|---|---|
| Periodic cleanup that needs the app's pool and config | Ticker goroutine started from `main.go`, modelled on the reaper |
| Scheduled work against the database or an API that can run from a clean runner | Actions cron with secrets, gated like `demo-reset.yml` |
| Durable delivery, retries, fan-out, minutes-long tasks | Not in the starter; write an ADR first ([ADR-019](../adr/ADR-019-Template-Scope-Boundary.md) says don't invent infrastructure inline) |

Every job needs a table-driven test through its seams before the production code (CLAUDE.md rule 8), structured `slog` lines with what it did and how many, and a metric if it can fail silently.
