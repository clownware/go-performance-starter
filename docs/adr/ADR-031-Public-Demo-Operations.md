# ADR-031: Public Demo Operations

**Date**: 2026-07-12

## Status

Accepted

## Context

The demo application (ADR-024) is going public: the repo self-deploys its own demo instance (ADR-025's Fly.io worked example), which anonymous guests can write to. That raises three operational questions the ADRs don't yet answer: how the demo stays current between tagged releases, how demo data is seeded and kept clean under open anonymous writes, and how none of this leaks into what a template cloner inherits. The hard constraint: nothing deployment-specific may be hardcoded in workflows — cloners must inherit only generic, parameterized machinery that stays inert until they add their own secrets.

## Decision

1. **Continuous deploy on merge** (`.github/workflows/deploy.yml`). Every push to the default branch runs migrations (when `SUPABASE_DATABASE_URL` is set) and `flyctl deploy --remote-only`. The app name comes from `fly.toml`; the workflow is gated on the `FLY_API_TOKEN` secret via a gate job (the secrets context is unavailable in job-level `if:`), so a clone without the secret sees a skipped job. Tagged releases keep the careful path (`release.yml`: budget-checked image, mirror, deploy).
2. **Seed and reset are Taskfile targets hard-gated on `DEMO_MODE=1`.** `task demo:seed` loads `sql/demo/seed.sql`, which re-includes the idempotent quiz-question seed migration (one source of truth, no copied fixtures). `task demo:reset` runs `sql/demo/reset.sql` then re-seeds. Both refuse to run without `DEMO_MODE=1` in the environment, so they are impossible to trigger against a real deployment by accident.
3. **Nightly reset via scheduled workflow** (`.github/workflows/demo-reset.yml`), double-gated: the job only runs when the `SUPABASE_DATABASE_URL` secret is present *and* the repo variable `DEMO_MODE` is `1`, and the Taskfile precondition re-checks `DEMO_MODE` at execution.
4. **Division of labor with the reaper.** The reset deletes only anonymous users' *content* (`flashcards`, `quiz_attempts`). Identity deletion stays with the TTL reaper (ADR-024), which owns both the GoTrue auth record and the `users` row — deleting `users` rows in SQL would orphan the auth-side records the reaper finds through them. The demo sets `GUEST_TTL=48h` in `fly.toml` so identity cleanup runs on the same rhythm as the nightly content reset. Registered (upgraded) accounts are never touched — keeping progress is the upgrade flow's promise.

### Abuse surface inventory (what going public opens)

- **Anonymous flashcard writes** (`POST /learn/flashcards`): arbitrary user text, capped at 500 chars/field, RLS-scoped so only the author sees it, templ-escaped on render. Bounded by the stricter `/learn` rate-limit tier and the request-body cap; the nightly reset bounds accumulation. Accepted.
- **Anonymous identity creation**: every new visitor mints a GoTrue user. Rate-limited; reaped after `GUEST_TTL`. Accepted.
- **Open email signup via the upgrade flow**: lets anyone use the demo's Supabase project to send confirmation emails. Mitigated by rate limits and Supabase's own email throttling; residual risk accepted for the demo — a fork running real traffic should reconsider.
- **No file uploads, no publicly-visible shared writes** — quiz questions are read-only under RLS, so cross-user vandalism has no surface.

## Consequences

- The public demo is self-healing: vandalized or bloated guest content lives at most one day; identities at most 48h idle.
- A cloner inherits three inert workflows; adding `FLY_API_TOKEN` (and optionally `SUPABASE_DATABASE_URL`, `DEMO_MODE=1`) turns on their own demo, not ours.
- Merges to the default branch deploy immediately; anything merged is live. The PR quality gate (`task ci`) is the deploy gate.
- `scripts/` gains no new runtime dependency; the reset path needs only `psql` and Task on the runner.

## Alternatives Considered

- **Fly cron (scheduled machines) for the reset.** Rejected — a second machine config to parameterize and for cloners to pay for; GitHub's scheduler is free, visible in the Actions tab, and uses the same secret plumbing as the other workflows.
- **Reset by deleting anonymous `users` rows (cascade).** Rejected — orphans GoTrue auth records; the reaper already owns identity deletion end-to-end.
- **Full database re-create (drop schema, re-migrate, re-seed).** Rejected — destroys upgraded users' progress and breaks live sessions mid-reset for no additional protection, since shared data is read-only.

## References

- Related: [ADR-024](ADR-024-Demo-Application-Direction.md) (demo app, guest mode, reaper), [ADR-025](ADR-025-Deployment-Target.md) (deployment target, release pipeline), [ADR-030](ADR-030-Versions-Manifest-Contract.md) (public manifest).
