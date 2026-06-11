# Stack

Technology facts. This file updates when dependencies change or commands move; rules elsewhere are stable.

## Versions

- **Language:** Go 1.25 (`go.mod`)
- **Router:** Chi v5 (stdlib-aligned, lightweight)
- **Templating:** templ (type-safe, compiles to Go) — replaces `html/template` (ADR-017)
- **Frontend:** HTMX + Alpine.js (minimal JS), Tailwind CSS v4 (`@tailwindcss/cli`)
- **Database:** PostgreSQL via pgx/v5 + pgxpool; sqlc for type-safe queries; golang-migrate for migrations
- **Auth:** Supabase (gotrue) — JWT validation server-side; auth is optional/disabled if unconfigured
- **Observability:** Prometheus (`client_golang`) + structured logging (zerolog/slog)
- **Task runner:** Taskfile (`taskfile.dev`)
- **Node:** 20+ (Tailwind CLI only)

## Key Commands

```bash
task dev               # hot-reload dev server (air); watches .go, .templ
task ci                # quality gate: fmt + lint + test(-race -cover) + agents:check + binary-size + vuln
task test              # go test ./...
task test:coverage     # coverage report (HTML)
task lint              # golangci-lint
task fmt               # gofmt
task templ:generate    # regenerate *_templ.go from .templ
task db:generate       # regenerate internal/database from sql/ via sqlc
task db:migrate:up     # apply migrations
task css:build         # build Tailwind CSS
task test:performance  # performance budget tests + binary size
task scan:vuln         # govulncheck
task agents:build      # regenerate AGENTS.md from CLAUDE.md + .claude/*.md
task agents:check      # CI gate: fail if AGENTS.md drifts from sources
```

## Performance Budgets (ADR-000)

| Metric | Budget |
|---|---|
| P95 response time | < 100ms |
| P99 response time | < 200ms |
| Binary size (linux production, stripped) | < 20MB |
| Docker image | < 30MB |
| Memory (steady state) | < 128MB |
| Startup time | < 500ms |
| JavaScript (gzip) | < 50KB |
| CSS (gzip) | < 30KB |

Budgets are enforced in CI via `task test:performance` / `task test:binary-size`. The 20MB binary budget targets the stripped linux build (`-ldflags="-s -w"`), not local debug builds.

## Key ADRs

18+ ADRs in `docs/adr/`. The structurally important ones:

- **ADR-000:** Performance budgets and quality attributes
- **ADR-001:** Foundation (Go, Chi, logging)
- **ADR-003:** sqlc + repository pattern for data access
- **ADR-007 / ADR-012:** Frontend stack (HTMX + Alpine + Tailwind), routing & UI patterns
- **ADR-015:** Configuration via environment (twelve-factor)
- **ADR-017:** templ adoption (supersedes ADR-008)
- **ADR-018:** Layered AI constitution (this file structure)
- **ADR-019–022:** Scope boundary, agent roles, quality gate, cross-tool AGENTS.md spine
- **ADR-023:** Testing philosophy

## Deployment

- **Default port:** `4000` (`HTTP_PORT`, see `internal/config/config.go`)
- **Build:** `task build` → `./dist/app` (stripped). Multi-stage Dockerfile (Node→Tailwind, Go→templ+build, Alpine runtime).
- **Target:** Cloudflare edge / container host.

## Cross-tool spine

The cross-tool agent context lives in [`AGENTS.md`](../AGENTS.md) at the repo root. It is generated from `CLAUDE.md` plus `.claude/engineering.md`, `.claude/workflow.md`, and this file via `task agents:build`. CI fails if `AGENTS.md` drifts from its sources (see [ADR-022](../docs/adr/ADR-022-Cross-Tool-Agents-Spine.md)). Do not edit `AGENTS.md` directly; edit the source layer instead.
