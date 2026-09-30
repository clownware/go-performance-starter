# Guides

Topic guides for working in this codebase. Each one explains the paved path, links the ADRs that decided it, and names the real tasks, files and environment variables — nothing here is a build-order tutorial. The ADRs are the constraints; the guides are how to satisfy them. If a guide and an ADR disagree, the ADR wins and the guide is the bug.

## Reading order for a new contributor

1. [`../../README.md`](../../README.md) — what the template is, the quick start, what is load-bearing vs removable.
2. [foundation.md](foundation.md) — the immutable choices: Go, Chi, module layout, config, secrets, logging, the constitution.
3. [tooling-and-quality-gates.md](tooling-and-quality-gates.md) — `task ci` and everything it runs; the two agent hooks.
4. [data-access.md](data-access.md) — migrations → sqlc → repositories → RLS scoping.
5. [view-layer.md](view-layer.md) — templ pages, partials, components, role tokens.
6. [routing-and-middleware.md](routing-and-middleware.md) — the middleware stack and route groups.
7. [htmx-and-alpine.md](htmx-and-alpine.md) — the interaction patterns and the live `/patterns` catalogue.
8. [auth-and-security.md](auth-and-security.md) — Supabase JWTs, anonymous guests, RLS as authorization, CSRF, rate limits, headers.
9. [testing.md](testing.md) — failing test first, the test layers, mutation testing.
10. [performance.md](performance.md) — the budgets, what fails the build, what is only measured.
11. [background-jobs.md](background-jobs.md) — in-process tickers vs GitHub Actions cron.
12. [deployment.md](deployment.md) — the Docker image, Fly.io worked example, Cloudflare as proxy, the workflows.
13. [documentation.md](documentation.md) — where things live, writing and amending ADRs, CHANGELOG, `versions.json`.

## The stack in one table

| Layer | Choice | Decided in |
|---|---|---|
| Language / router | Go 1.26, Chi v5 | ADR-001 |
| Templates | templ (typed props; `html/template` forbidden) | ADR-017 |
| Frontend | HTMX + Alpine.js, Tailwind CSS v4, role-based tokens | ADR-007, ADR-012, ADR-029 |
| Database | PostgreSQL via pgx/v5; sqlc; golang-migrate | ADR-002, ADR-003 |
| Authorization | Postgres Row Level Security scoped by Supabase JWT claims | ADR-004, ADR-014 |
| Auth | Supabase GoTrue, server-side validation, anonymous guests | ADR-024 |
| Config | Environment variables only | ADR-015 |
| Logging / metrics | `log/slog` JSON in production; Prometheus | ADR-026, ADR-013 |
| Background work | Plain goroutines with tickers; GitHub Actions cron for scheduled jobs | ADR-024, ADR-031 |
| Deployment | One stateless container (Fly.io worked example) behind the Cloudflare proxy | ADR-025 |
| Quality gate | `task ci`, halt-on-violation | ADR-021, ADR-033 |
| Budgets | Binary 20MB, image 30MB, JS 50KB / CSS 30KB gzip enforced; latency, memory, startup monitored | ADR-000 |

Stack facts that churn (versions, commands) live in [`.claude/stack.md`](../../.claude/stack.md) and [`versions.json`](../../versions.json), not here.
