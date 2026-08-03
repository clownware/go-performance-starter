# Phase 0 — Foundation Kick-off

Define immutable decisions first; changing these later is painful.

## Key Implementation Steps

| Step | Task | Why It Matters |
|------|------|----------------|
| 0.01 | Choose Go version & module path | Go 1.22+ enables improved handler signatures |
| 0.02 | Select web framework | Chi aligns better with standard library |
| 0.03 | Decide development workflow | Consistent reloading and automation saves time |
| 0.04 | Initialize project structure | Structure affects maintainability |
| 0.05 | Setup git hooks & CI | Early quality gates prevent issues |
| 0.06 | Choose structured logger | This starter uses the standard library's log/slog (ADR-026) |
| 0.07 | Create Architecture Decision Record | Document foundational choices |
| 0.08 | Pick base linting tool | Select golangci-lint for code quality |
| 0.09 | Define secret management strategy | Never use .env files in production |
| 0.10 | Configure environment variables | Separate dev/prod strategies |
| 0.11 | Branch & release workflow | Clear deployment pipeline |
| 0.12 | Set up multi-module workflow | Consider go work for larger projects |

## Core Principles

- Document foundational decisions in an Architecture Decision Record (ADR)
- Follow standard Go project layout for consistency
- Configure simple linting from day one (detailed configuration comes in Phase 3)
- Implement a clear secret management strategy
- Establish automated quality gates
- Consider multi-module structure for larger projects using go work

## Deployment Considerations in ADR

When creating your ADR, document deployment considerations:
- This starter deploys as a single stateless container on Fly.io behind the Cloudflare proxy (ADR-025)
- Consider binary size when selecting dependencies — `task ci` enforces a 20MB binary budget (`test:binary-size`), and CI's docker job adds a 30MB image budget
- Cloudflare terminates TLS and serves as CDN; Workers is not an application runtime

## Security Strategy Hand-off

Document your secret management approach in the ADR, covering:
- Development environment: 1Password injection via `op run --env-file=.env.tpl` — no plaintext secrets on disk
- Staging/Production: Fly.io secrets (`fly secrets`) and GitHub Actions secrets
- Runtime injection: How environment variables will be loaded at runtime
- Rotation strategy: How secrets will be rotated in production

This documentation will be referenced in the Deployment phase (Phase 10).

## Exit Criteria

- Core decisions documented in ADR and pushed to repository
- Initial structure with working go.mod
- Git hooks and linting configured
- Environment strategy established
- Project passes basic build test
- Secret management strategy documented

## Recommended Directory Structure

This starter kit uses the following structure:

```
go-performance-starter/
├── cmd/
│   └── api/         # Application entry point (main.go)
├── internal/        # Private application code
│   ├── auth/        # Authentication related code
│   ├── cache/       # In-memory caching
│   ├── config/      # Configuration handling
│   ├── database/    # Database connection and sqlc-generated models
│   ├── handler/     # HTTP handlers
│   ├── jobs/        # Background jobs
│   ├── middleware/  # Application middleware
│   ├── performance/ # Performance budget tests
│   ├── repository/  # Data access layer
│   ├── server/      # Server configuration
│   ├── validate/    # Input validation
│   ├── view/        # templ pages, partials, and components (ADR-017)
│   └── webutil/     # Shared HTTP helpers
├── migrations/      # Database migrations
├── sql/             # SQLC query files
│   ├── queries/     # SQL queries for SQLC
│   └── schema/      # Database schema
├── web/             # Web assets
│   └── static/      # Static assets (css, js, images)
├── .air.toml        # Hot reload config
├── .golangci.yml    # Linting rules
├── Taskfile.yml     # Task automation
└── README.md        # Documentation
```
