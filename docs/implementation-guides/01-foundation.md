# Phase 0 — Foundation Kick-off

Define immutable decisions first; changing these later is painful.

## Key Implementation Steps

| Step | Task | Why It Matters |
|------|------|----------------|
| 0.01 | Choose Go version & module path | Go 1.22+ enables improved handler signatures |
| 0.02 | Select web framework | Chi/Echo align better with standard library |
| 0.03 | Decide development workflow | Consistent reloading and automation saves time |
| 0.04 | Initialize project structure | Structure affects maintainability |
| 0.05 | Setup git hooks & CI | Early quality gates prevent issues |
| 0.06 | Choose structured logger | Pick one consistent approach (zerolog OR zap) |
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
- Cloudflare Workers Classic has a 1MB WASM binary size limit
- Consider code size when selecting dependencies
- For larger applications, plan for Durable Objects or alternative deployments

## Security Strategy Hand-off

Document your secret management approach in the ADR, covering:
- Development environment: godotenv or similar for local development only
- Staging/Production: Managed secrets (Cloudflare environment variables)
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

```
my-app/
├── cmd/web/         # Application entry points
├── internal/        # Private code
│   ├── config/      # Configuration handling
│   ├── handler/     # HTTP handlers
│   ├── model/       # Domain models
│   ├── repository/  # Data access
│   └── server/      # Server configuration
├── migrations/      # Database migrations
├── templates/       # HTML templates
├── static/          # Static assets
├── .air.toml        # Hot reload config
├── .golangci.yml    # Linting rules
├── Taskfile.yml     # Task automation
└── README.md        # Documentation
```
