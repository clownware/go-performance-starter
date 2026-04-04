# Alpine Go Performance Starter

An opinionated, performance-first SaaS starter kit built with Go, HTMX, Alpine.js, and Tailwind CSS. Designed for solo developers and small teams who want production-ready infrastructure without the bloat.

**Stack:** Go (Chi) | HTMX + Alpine.js | Tailwind CSS | Supabase (Auth + PostgreSQL) | Cloudflare

## What You Get

- **Authentication** -- Supabase email/password auth with server-side JWT validation
- **User-scoped CRUD** -- "Items" resource with HTMX forms and optimistic UI
- **Row Level Security** -- PostgreSQL RLS policies enforced at the database layer
- **Type-safe SQL** -- sqlc code generation with repository pattern
- **Performance budgets** -- CI-enforced binary size, response time, and memory limits
- **Observability** -- Prometheus metrics, structured logging (zerolog), health checks
- **Developer experience** -- Hot reload (air), Taskfile automation, golangci-lint, CI/CD

## Prerequisites

- Go 1.24+
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

The application runs at [http://localhost:8080](http://localhost:8080) by default.

## Available Tasks

Run `task --list` to see all available tasks. Key ones:

| Task | Description |
|------|-------------|
| `task dev` | Start dev server with hot reload |
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
  config/             Environment-based configuration
  database/           sqlc-generated types and queries
  handler/            HTTP handlers
  middleware/         Auth, metrics, logging, request ID
  performance/        Performance budget definitions
  repository/         Data access interfaces + implementations
  server/             Router setup and middleware stack
  view/               View models
  webutil/            Template rendering, HTMX helpers
web/
  templates/          Go html/template files
  static/             CSS, JS, images, fonts
migrations/           golang-migrate SQL files
sql/                  sqlc query and schema definitions
docs/                 ADRs, implementation guides, product docs
```

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
