# ADR-020: Agent Roles and Three-Pass Workflow

## Status

Accepted

## Date

2026-06-11

## Context

Single-shot agent runs on non-trivial features tend to conflate deciding *what* to build, *how* to build it, and *whether* it's correct — producing code that passes a happy-path check but skips the ADR, the test, or the budget. Separating these concerns into explicit passes with hand-off artefacts makes each step auditable and lets the operator refuse work that skipped a step.

## Decision

For any feature that touches multiple ADRs, has non-obvious acceptance criteria, adds a dependency, or changes a public API, use a three-pass workflow (prompts in `.claude/roles/`):

1. **Architect** — writes/updates the ADR (Proposed) and a failing table-driven test; no production code.
2. **Coder** — minimum code to make the failing test pass; no test weakening; stays in the Architect's plan.
3. **Reviewer** — runs `task ci`, reports delta vs. plan and risk flags, recommends; no commits.

Each pass ends with an explicit hand-off announcement. Trivial changes (typo, single-line, single rename) skip the pattern. The human operator enforces the hand-off by refusing to merge work that skipped a pass.

## Consequences

- Every non-trivial change has an ADR, a test, and a review on record.
- Test-first is structurally enforced (the Architect produces the failing test before any code).
- Overhead on small changes is avoided via the trivial-change escape hatch.

## Alternatives Considered

- **Single-pass agent.** Rejected — conflates concerns; ADRs and tests get skipped under happy-path pressure.
- **Heavyweight BDD/Gherkin specs per feature.** Rejected as default — table-driven tests already read as specs (see ADR-023); revisit only if prose acceptance criteria become genuinely ambiguous.

## References

- [ADR-021](ADR-021-Halt-On-Violation-Quality-Gate.md), [ADR-023](ADR-023-Testing-Philosophy.md)
- `.claude/roles/{architect,coder,reviewer}.md`

## Enforcement
<!-- added 2026-07-12, see ADR-033 (Enforcement Architecture) -->
- **Not machine-checkable:** Three-pass (Architect/Coder/Reviewer) compliance is process, enforced by the human operator refusing to merge skipped passes — by this ADR's own design. The Reviewer pass's `task ci` run is the machine-checkable part and is owned by ADR-021.
- **Graduation log:** _(empty)_
