# Foundation

The choices that are painful to change later, and where each one is decided. Nothing here is optional reading for a forker: [`../personalization-guide.md`](../personalization-guide.md) is the checklist for changing the ones that carry the template's identity.

## Language, module, router

- **Go 1.26** (`go.mod`; the Dockerfile builds on `golang:1.26-alpine`, CI reads `go-version-file`). The pin is mirrored in [`versions.json`](../../versions.json) and checked by `task versions:check` ([ADR-030](../adr/ADR-030-Versions-Manifest-Contract.md)).
- Module path `github.com/clownware/go-performance-starter`, one module, no `go work`. `.golangci.yml` uses it as the goimports local prefix, so renaming the module means editing that file too.
- **Chi v5** for routing, chosen for stdlib alignment ([ADR-001](../adr/ADR-001-Foundation.md)). The middleware stack and route groups are described in [routing-and-middleware.md](routing-and-middleware.md).

## Layout

`cmd/api/main.go` is the only entrypoint: it loads config, connects the pool, builds the server, starts the guest reaper, and handles shutdown. Everything else is under `internal/` — the package list with one-line purposes is the "Project Structure" tree in the [README](../../README.md#project-structure), and [`.claude/engineering.md`](../../.claude/engineering.md) states the dependency direction (handlers depend on repository *interfaces*, never on postgres types). Three paths are generated and never hand-edited: `internal/database/` (sqlc), `internal/view/*_templ.go` (templ), `AGENTS.md` (`task agents:build`); `task check:generated` and `task agents:check` catch drift, and the PreToolUse guard denies agent edits ([ADR-019](../adr/ADR-019-Template-Scope-Boundary.md), [ADR-033](../adr/ADR-033-ADR-Enforcement-Architecture.md)).

## Configuration and secrets

Configuration is environment variables only ([ADR-015](../adr/ADR-015-Configuration-Management-Strategy.md)), parsed into `internal/config/config.go` with defaults and validation. `.env.example` is the canonical list; `cmd/api/main.go` loads `.env` via godotenv and the Taskfile loads it too, so `task dev` and `go run ./cmd/api` see the same values.

Secrets never touch a tracked file:

- Locally, 1Password injection: `op run --env-file=.env.tpl -- task dev` (the `.env.tpl` is gitignored; `.claude/launch.json` ships both this and a plain `go run` config).
- In production, the container host's secret store: `fly secrets set` in the worked example ([ADR-025](../adr/ADR-025-Deployment-Target.md)). Non-secret tuning lives in `fly.toml`'s `[env]`, versioned.
- `task check:adr` runs `adr015-no-hardcoded-secrets` over shipped source and config.

## Logging and observability

`log/slog` only, JSON in production, `LOG_LEVEL` from the environment ([ADR-026](../adr/ADR-026-Logging-Standardization.md)); `adr026-slog-only` fails on any other logger import. Prometheus metrics on `/metrics` behind `METRICS_TOKEN`, `/healthz` for liveness and `/health` for readiness ([ADR-013](../adr/ADR-013-Error-Handling-and-Observability.md)).

## Deployment shape

One stateless container — the Docker image is the contract — behind the Cloudflare proxy, which terminates TLS and serves as CDN. Cloudflare Workers is not an application runtime; there is no multi-region story and sessions are stateless JWT cookies ([ADR-025](../adr/ADR-025-Deployment-Target.md)). Binary size matters from day one: `task ci` fails above 20MB stripped and the CI docker job above a 30MB image ([ADR-000](../adr/ADR-000-Performance-Budgets-and-Quality-Attributes.md)). Details in [deployment.md](deployment.md).

## The constitution

The repo is built to be developed with coding agents and holds them to the same rules as humans. [`CLAUDE.md`](../../CLAUDE.md) carries ten halt-on-violation rules; [`.claude/engineering.md`](../../.claude/engineering.md), [`.claude/workflow.md`](../../.claude/workflow.md) and [`.claude/stack.md`](../../.claude/stack.md) carry defaults, process and stack facts ([ADR-018](../adr/ADR-018-Layered-AI-Constitution.md)). `AGENTS.md` is generated from those four files for other tools ([ADR-022](../adr/ADR-022-Cross-Tool-Agents-Spine.md)). If you do not develop with agents the whole apparatus is removable — see the README's load-bearing table.

## Decisions are ADRs

Anything that would go in this guide as a new "we chose X" belongs in `docs/adr/` first ([ADR-011](../adr/ADR-011-Documentation-Standards.md)); see [documentation.md](documentation.md) for the template and the append-only rule.
