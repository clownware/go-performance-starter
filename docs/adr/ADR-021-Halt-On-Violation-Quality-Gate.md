# ADR-021: Halt-On-Violation Quality Gate

## Status

Accepted

## Date

2026-06-11

## Context

"Run the tests before you call it done" is only meaningful if there is one command that defines "done" and it fails loudly. Previously the repo had separate `task test`, `task lint`, `task test:binary-size`, and `task scan:vuln` with no single gate, and no rule binding agents to run them. Agents would declare work complete on a green happy-path build while lint, race, budget, or vulnerability checks were never run.

## Decision

Define a single halt-on-violation gate, `task ci`, that an agent must run before claiming any change complete (Constitution rule 9). It chains:

```
fmt:check → lint → go test -race -cover ./... → agents:check → test:binary-size → test:asset-budgets → scan:vuln
```

If it exits non-zero, the agent halts and fixes the failure. It must not lower a threshold, exclude files, or bypass git hooks with `--no-verify`.

**2026-07 Amendment**: the GitHub Actions workflow originally re-implemented these checks as parallel jobs, and they drifted (`fmt:check` ran locally but never in CI). The workflow now installs the environment (Postgres, toolchain, Task) and invokes `task ci` itself — the same single-source-of-truth fix ADR-022 applied to AGENTS.md. If the gate needs a new check, add it to the `ci` task; the workflow inherits it.

## Consequences

- "Done" has one definition, locally and in CI.
- Formatting, race conditions, binary-size budget, agent-spine drift, and known vulnerabilities are all caught before review.
- Slightly slower inner loop — mitigated by running `task test`/`task lint` individually during iteration and reserving `task ci` for the final gate.

## Alternatives Considered

- **Rely on CI only.** Rejected — pushes failures to the slowest feedback point and lets agents claim completion prematurely.
- **A pre-push hook running everything.** Complementary, not a replacement — the constitution rule makes the agent run it before claiming done, not just before pushing.

## References

- [ADR-000](ADR-000-Performance-Budgets-and-Quality-Attributes.md), [ADR-010](ADR-010-Testing-and-Code-Quality.md), [ADR-022](ADR-022-Cross-Tool-Agents-Spine.md)
- `Taskfile.yml` (`ci` task), `.github/workflows/ci.yml`

## Enforcement
<!-- added 2026-07-12, see ADR-033 (Enforcement Architecture) -->
- **Testable consequences:**
  - TC-1: `task ci` chains the full gate (fmt, lint, race+coverage tests, agents:check, versions:check, binary/asset budgets, vuln scan).
  - TC-2: `.github/workflows/ci.yml` invokes `task ci` rather than re-implementing its steps.
- **Checks:**
  - TC-1 → the Taskfile definition itself; any removal is a public diff (status: **block**, pre-existing)
  - TC-2 → `adr021-ci-invokes-gate` in `scripts/adrcheck` (status: **warn**)
- **Not machine-checkable:** That an agent *actually ran* the gate before claiming done — approximated by the Stop-gate hook (ADR-033), which runs tests and the check suite when an agent tries to finish.
- **Graduation log:** _(empty)_
