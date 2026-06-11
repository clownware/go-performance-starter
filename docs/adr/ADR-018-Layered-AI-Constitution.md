# ADR-018: Layered AI Constitution

## Status

Accepted

## Date

2026-06-11

## Context

This starter is built to be developed with AI coding agents. Until now the only agent-facing guidance was a single `.windsurfrules` file, which had drifted stale (it referenced Go 1.22, Tailwind 3, `web/templates/`, and `html/template` — all superseded). A single flat rules file mixes three concerns with different volatility: hard prohibitions, engineering defaults, and ephemeral stack facts (versions/commands). When they live together, version churn forces edits to the whole file and the hard rules lose force amid the noise.

Our sibling project (`astro-performance-starter`) converged on a layered "constitution" structure that has proven durable. We adopt the same pattern here, adapted to Go.

## Decision

Split agent guidance into a layered constitution:

- **`CLAUDE.md`** (repo root) — the constitution: ~10 halt-on-violation rules only. Stable, rarely edited.
- **`.claude/engineering.md`** — strong engineering defaults (package layout, view layer, data access, testing style). Constitutional force; named exceptions need an ADR.
- **`.claude/workflow.md`** — how work moves: scope boundaries, three-pass workflow, the quality gate, ADR discipline, git conventions.
- **`.claude/stack.md`** — ephemeral facts: versions, commands, performance budgets. Updated freely as dependencies move.
- **`.claude/roles/`**, **`.claude/skills/`**, **`.claude/agents/`** — role prompts, custom skills, and subagent definitions.

`CLAUDE.md` carries a precedence clause stating the sublayers apply with the same force; the layering is for organisation and volatility isolation, not for softening.

## Consequences

- Hard rules stay short and legible; stack churn touches only `stack.md`.
- Claude Code reads `CLAUDE.md` and `.claude/` natively; other tools read the generated `AGENTS.md` spine (ADR-022).
- New contributors (human or agent) have one obvious entry point.
- Cost: more files to keep coherent, mitigated by the generated spine and its drift check.

## Alternatives Considered

- **Keep a single `.windsurfrules`/`CLAUDE.md`.** Rejected — mixes volatile facts with hard rules and had already drifted stale.
- **Put everything in `AGENTS.md` by hand.** Rejected — hand-maintained cross-tool mirrors drift; we generate it instead (ADR-022).

## References

- [ADR-019](ADR-019-Template-Scope-Boundary.md), [ADR-020](ADR-020-Agent-Roles.md), [ADR-021](ADR-021-Halt-On-Violation-Quality-Gate.md), [ADR-022](ADR-022-Cross-Tool-Agents-Spine.md)
- `astro-performance-starter` ADR-036 (layered constitution)
