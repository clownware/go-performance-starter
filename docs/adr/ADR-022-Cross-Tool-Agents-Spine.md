# ADR-022: Cross-Tool AGENTS.md Spine

## Status

Accepted

## Date

2026-06-11

## Context

Claude Code reads `CLAUDE.md` and `.claude/` natively, but other agent tools (Cursor, Codex CLI, Copilot, Windsurf, Aider, Zed, Continue, and others) look for a root `AGENTS.md`. Maintaining a hand-written `AGENTS.md` alongside the layered constitution guarantees drift: the two copies disagree within a few commits, and an agent reading the stale one applies wrong rules. The same drift already happened with the old `.windsurfrules`.

## Decision

Treat `AGENTS.md` as a **generated artefact**, never hand-edited. `scripts/agentsmd` concatenates `CLAUDE.md` + `.claude/engineering.md` + `.claude/workflow.md` + `.claude/stack.md`, demoting each source's headings one level (code fences left intact) under a single H1 with a "do not edit" banner.

- `task agents:build` regenerates `AGENTS.md`.
- `task agents:check` exits non-zero if `AGENTS.md` differs from a fresh generation; it is part of `task ci` and runs as its own CI job.
- `.windsurfrules` is reduced to a thin pointer at `AGENTS.md` plus one Cascade-specific directive.

The generator is written in Go (own package `scripts/agentsmd`) to match the existing `scripts/check-binary-size.go` precedent and keep the toolchain Go-native.

## Consequences

- One source of truth (the constitution layers); every tool reads a consistent spine.
- Editing a source layer without regenerating fails the gate — drift is impossible to merge.
- Cost: one generation step and one CI job; negligible.

## Alternatives Considered

- **Hand-maintained `AGENTS.md`.** Rejected — guaranteed drift (the problem we are solving).
- **Symlink/include.** Rejected — `AGENTS.md` must be a single self-contained file spanning four sources with adjusted heading levels.
- **Shell script generator.** Viable, but heading demotion with code-fence awareness is cleaner and testable in Go, and avoids `sed` portability issues.

## References

- [ADR-018](ADR-018-Layered-AI-Constitution.md), [ADR-021](ADR-021-Halt-On-Violation-Quality-Gate.md)
- `scripts/agentsmd/main.go`, `AGENTS.md`
- `astro-performance-starter` ADR-045 (cross-tool spine)
