# ADR-019: Template Scope Boundary

## Status

Accepted

## Date

2026-06-11

## Context

As a starter template developed with agents, the repo needs explicit boundaries on what an agent may change without asking. Without them, agents "improve" documentation, invent deployment infrastructure, or hand-edit generated code — all of which create noise and drift. Some directories are generated artefacts (`internal/database/` from sqlc, `*_templ.go` from templ, `AGENTS.md` from the constitution) and must never be hand-edited.

## Decision

Define a scope table (canonical copy in `.claude/workflow.md`):

| Category | Paths | Rule |
|---|---|---|
| Modify freely | `cmd/`, `internal/`, `web/`, `sql/`, `migrations/`, build/config files, `.github/workflows/`, `.windsurfrules` | Full read/write |
| Read-only | `docs/` | Don't modify unless explicitly asked |
| Generated — never hand-edit | `internal/database/*`, `internal/view/*_templ.go`, `AGENTS.md` | Edit the source, then regenerate |
| Don't create | Deployment infra, marketing content, maintenance scripts | Suggest adding to `docs/` instead |

## Consequences

- Agents stop touching `docs/` and generated files uninvited; diffs stay reviewable.
- Generated-file edits are caught (templ/sqlc/AGENTS.md regeneration is part of the workflow and the `task ci` gate).
- The boundary is advisory to humans but enforced for agents via the constitution.

## Alternatives Considered

- **No explicit boundary.** Rejected — observed agent scope creep into docs and generated files.
- **Lock generated dirs via tooling only.** Partial — CI drift checks help, but a stated rule prevents the wasted work upstream.

## References

- [ADR-018](ADR-018-Layered-AI-Constitution.md), [ADR-022](ADR-022-Cross-Tool-Agents-Spine.md)
- `.claude/workflow.md`
