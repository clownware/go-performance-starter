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
- The forker-facing counterpart of this boundary — which pieces of the template are load-bearing vs. removable — lives in the README's ["What's Load-Bearing vs. Removable"](../../README.md#whats-load-bearing-vs-removable) section (added 2026-07 with the ADR-024 demo).

## Enforcement
<!-- added 2026-07-12, see ADR-033 (Enforcement Architecture) -->
- **Testable consequences:**
  - TC-1: Generated artefacts (`internal/database/*` via sqlc, `internal/view/*_templ.go` via templ, `AGENTS.md`) are never hand-edited.
- **Checks:**
  - TC-1 → drift checks in `task ci` (`agents:check`; build regeneration) (status: **block**, pre-existing)
  - TC-1 → PreToolUse guard (`scripts/adrguard`) denies agent edits to these paths at write time (status: **block**, hook — see ADR-033)
- **Not machine-checkable:** The "docs/ is read-only unless asked" and "don't create deployment infra/marketing" rules require knowing what the operator asked for — agent-behavioural, enforced by review.
- **Graduation log:** _(empty)_
