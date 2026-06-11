# Alpine Go Performance Starter

An opinionated, performance-first SaaS starter kit built with Go, templ, HTMX, Alpine.js, and Tailwind CSS. Designed for solo developers and small teams who want production-ready infrastructure without the bloat.

**Stack:** Go (Chi) | templ | HTMX + Alpine.js | Tailwind CSS | Supabase (Auth + PostgreSQL) | Cloudflare

**Why this starter:** the bet is server-rendered Go with a minimal-JS frontend, where performance budgets and AI-agent guardrails are enforced in CI rather than aspired to in a README. It's a deliberately narrow, opinionated foundation — fewer choices, proven defaults — not a framework buffet.

## What You Get

- **Authentication** -- Supabase email/password auth with server-side JWT validation
- **User-scoped CRUD** -- "Items" resource with HTMX forms and optimistic UI
- **Row Level Security** -- PostgreSQL RLS policies enforced at the database layer
- **Type-safe templates** -- templ compiles HTML to Go; typed props, no `map[string]interface{}` (see [ADR-017](docs/adr/ADR-017-Templ-Adoption.md))
- **Type-safe SQL** -- sqlc code generation with repository pattern
- **Performance budgets** -- CI-enforced binary size, response time, and memory limits
- **Agentic discipline** -- a layered AI constitution and halt-on-violation quality gate (see below)
- **Observability** -- Prometheus metrics, structured logging (zerolog), health checks
- **Developer experience** -- Hot reload (air), Taskfile automation, golangci-lint, CI/CD

## Prerequisites

- Go 1.25+
- Docker & Docker Compose (for local development database)
- [Task](https://taskfile.dev) (task runner)
- Node.js 20+ (for Tailwind CSS build)

## Quick Start

```bash
git clone https://github.com/clownware/alpine-go-performance-starter.git
cd alpine-go-performance-starter

cp .env.example .env
# Edit .env with your Supabase credentials and DATABASE_URL

task db:up              # Start local Postgres
task db:migrate:up      # Run migrations
task db:generate        # Generate sqlc types
go mod tidy             # Install Go dependencies
npm install             # Install Tailwind tooling
task dev                # Start dev server with hot reload
```

The application runs at [http://localhost:4000](http://localhost:4000) by default (`HTTP_PORT`).

## Available Tasks

Run `task --list` to see all available tasks. Key ones:

| Task | Description |
|------|-------------|
| `task dev` | Start dev server with hot reload |
| `task ci` | Halt-on-violation quality gate (fmt, lint, race tests, agent-spine drift, binary size, vuln scan) |
| `task build` | Compile optimized binary to `./dist/app` |
| `task test` | Run test suite with coverage |
| `task test:performance` | Check performance budgets |
| `task test:binary-size` | Validate binary size < 20MB |
| `task lint` | Run golangci-lint |
| `task css:build` | Build Tailwind CSS |
| `task docker:build` | Build production Docker image |
| `task scan:vuln` | Run govulncheck |

## Project Structure

```
cmd/api/              Entry point
internal/
  auth/               Supabase auth client
  cache/              In-memory TTL cache
  config/             Environment-based configuration
  database/           sqlc-generated types and queries (generated)
  handler/            HTTP handlers
  middleware/         Auth, metrics, logging, request ID
  performance/        Performance budget definitions
  repository/         Data access interfaces + implementations
  server/             Router setup and middleware stack
  view/               templ UI: layouts/, pages/, partials/, components/ (+ render, props)
  webutil/            HTMX + context helpers
web/
  static/             CSS, JS, images, fonts
migrations/           golang-migrate SQL files
sql/                  sqlc query and schema definitions
docs/                 ADRs, implementation guides, product docs
.claude/              Layered AI constitution (engineering, workflow, stack, roles, skills, agents)
```

## Agentic Discipline

This starter is built to be developed with AI coding agents — and it holds the agent to the same rules you follow. The discipline is operationalized, not aspirational:

- **Layered AI constitution.** [`CLAUDE.md`](CLAUDE.md) holds ~10 halt-on-violation rules; [`.claude/engineering.md`](.claude/engineering.md), [`.claude/workflow.md`](.claude/workflow.md), and [`.claude/stack.md`](.claude/stack.md) carry engineering defaults, process, and ephemeral stack facts ([ADR-018](docs/adr/ADR-018-Layered-AI-Constitution.md)).
- **Cross-tool spine.** [`AGENTS.md`](AGENTS.md) is **generated** from those layers via `task agents:build` and read natively by Cursor, Copilot, Codex, Windsurf, and others. CI fails if it drifts from its sources ([ADR-022](docs/adr/ADR-022-Cross-Tool-Agents-Spine.md)).
- **Role-separated workflow.** Non-trivial features run a three-pass Architect → Coder → Reviewer flow, each pass producing an ADR, a failing test, or a review ([ADR-020](docs/adr/ADR-020-Agent-Roles.md)).
- **Halt-on-violation gate.** `task ci` is the single definition of "done." An agent must clear it before claiming a change complete — no lowering thresholds, no `--no-verify` ([ADR-021](docs/adr/ADR-021-Halt-On-Violation-Quality-Gate.md)).

## Architecture Decisions

This project uses Architecture Decision Records (ADRs) to document key technical choices. See [docs/adr/](docs/adr/) for the full set, including:

- [ADR-001](docs/adr/ADR-001-Foundation.md) -- Foundation (Go, Chi, zerolog)
- [ADR-000](docs/adr/ADR-000-Performance-Budgets-and-Quality-Attributes.md) -- Performance budgets
- [ADR-007](docs/adr/ADR-007-Frontend-Stack-Selection.md) -- Frontend stack (HTMX + Alpine + Tailwind)
- [ADR-014](docs/adr/ADR-014-Security-Patterns-and-Threat-Model.md) -- Security patterns

## Performance Targets

| Metric | Budget |
|--------|--------|
| P95 response time | < 100ms |
| Binary size | < 20MB |
| Docker image | < 30MB |
| Memory (steady state) | < 128MB |
| Startup time | < 500ms |

These are enforced in CI via `task test:performance` and `task test:binary-size`.

## License

MIT -- see [LICENSE](LICENSE).
