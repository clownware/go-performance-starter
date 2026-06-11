# Constitution — Alpine Go Performance Starter

These rules apply with halt-on-violation force. If a rule fires and you cannot satisfy it, halt and report the conflict — do not work around it.

## Rules

1. Architectural changes require checking `docs/adr/` first. If an Accepted ADR contradicts your proposal, halt and update the ADR or revise the proposal.
2. Server-rendered HTML uses templ. Raw `html/template` is forbidden (ADR-017). New UI is a templ page, partial, or component in `internal/view/` with a typed props struct — never `map[string]interface{}`.
3. All database access goes through sqlc-generated queries behind the repository interfaces (ADR-003). Hand-written SQL strings in handlers are forbidden. If you need a new query, add it to `sql/queries/` and run `task db:generate`.
4. Prefer server-rendered HTMX over client JavaScript (ADR-007, ADR-012). Reach for Alpine.js only for light client-only interactivity. Pages must work as progressive enhancement.
5. Do not disable or `//nolint` golangci-lint findings to make the build pass — fix the code. Formatting is `gofmt`; do not introduce a different formatter.
6. Configuration comes from environment variables only (ADR-015). Never hardcode secrets, tokens, connection strings, or keys.
7. Respect the performance budgets in `.claude/stack.md` (ADR-000). Do not exceed the binary, memory, JS, or CSS budgets; if a change must, halt and open an ADR.
8. Non-trivial production code follows a failing test. If you write non-trivial logic without a corresponding table-driven test, halt and write the test first (ADR-023).
9. Before claiming a change is complete, run `task ci`. If it exits non-zero, halt and fix the failure. Do not propose the change as complete, lower a budget/threshold, or skip git hooks with `--no-verify`.
10. `AGENTS.md` is generated. Never hand-edit it — edit the source layers and run `task agents:build` (ADR-022).

## Precedence

Rules in [`.claude/engineering.md`](.claude/engineering.md) and [`.claude/workflow.md`](.claude/workflow.md) apply with constitutional force; the layering exists for organisation, not for softening. Stack facts (commands, versions, dependencies, budgets) live in [`.claude/stack.md`](.claude/stack.md). The layering itself is established by [ADR-018](docs/adr/ADR-018-Layered-AI-Constitution.md); scope boundaries by [ADR-019](docs/adr/ADR-019-Template-Scope-Boundary.md).
