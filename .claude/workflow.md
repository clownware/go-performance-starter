# Workflow

How work moves through the repo. These rules apply with the same force as `CLAUDE.md`; the layering exists for organisation, not for softening.

## Scope Boundaries (ADR-019)

| Category | Paths | Rule |
|---|---|---|
| Modify freely | `cmd/`, `internal/`, `web/`, `sql/`, `migrations/`, `scripts/` (except the two below), `Taskfile.yml`, `sqlc.yaml`, `.golangci.yml`, `.air.toml`, `package.json`, `.github/workflows/`, `.windsurfrules` | Full read/write |
| Governed layer | `CLAUDE.md`, `.claude/`, `checks/`, `scripts/adrcheck/`, `scripts/adrguard/` | Editable, but never to make a change comply: a rule, check, or status changes only for its own reason, and the PR names the ADR or ADR note that authorises it. `adr019-governed-layer` lists these edits as a warning on every branch that makes one — expected, not a failure |
| Read-only | `docs/` | Don't modify unless explicitly asked to update documentation. Exceptions: adding a new ADR file under `docs/adr/` with status `Proposed` (and its row in `docs/adr/README.md`); appending a graduation-log entry to an ADR's Enforcement section when promoting/demoting a check (ADR-033 §4) |
| Append-only | `docs/adr/ADR-*.md` (existing) | Supersede with a new `Proposed` ADR (an agent may do this), or append a dated note with the operator's approval; never rewrite (ADR-033). The PreToolUse guard asks the operator before any Edit/Write on these files, and `adr033-append-only` reports removed or rewritten text |
| Generated — never hand-edit | `internal/database/*` (sqlc), `internal/view/*_templ.go` (templ), `AGENTS.md` (agents:build) | Edit the source, then regenerate |
| Don't create | Deployment infra, marketing content, maintenance scripts | Suggest adding to `docs/` instead. Exception: `fly.toml` at the repo root is the one permitted worked-example deploy config (ADR-025 §6) |

Full rationale in [ADR-019](../docs/adr/ADR-019-Template-Scope-Boundary.md).

## Non-trivial Feature Workflow (ADR-020)

For any feature that touches multiple ADRs, has non-obvious acceptance criteria, adds a dependency, or changes a public API, use the three-pass workflow:

1. **Architect pass** ([`.claude/roles/architect.md`](../.claude/roles/architect.md)) — write or update the relevant ADR; write the failing table-driven test; no production code.
2. **Coder pass** ([`.claude/roles/coder.md`](../.claude/roles/coder.md)) — minimum implementation to make the failing test pass; no test edits beyond what the Architect scaffolded.
3. **Reviewer pass** ([`.claude/roles/reviewer.md`](../.claude/roles/reviewer.md)) — run `task ci`; report delta vs. the Architect plan; recommend (no commits).

Each pass produces a concrete artefact and announces hand-off explicitly. The operator (human) enforces the hand-off: refuse to merge work that skipped a pass. Trivial changes (typo, single-line, single rename) can skip the pattern.

Full rationale in [ADR-020](../docs/adr/ADR-020-Agent-Roles.md).

## Quality Gate (ADR-021)

Before claiming a change is complete, run:

```bash
task ci
```

It runs `fmt` (check) + `lint` + `go mod verify` + `test` (`-race -cover`) + `agents:check` + `versions:check` + `check:adr` + `check:generated` + `test:binary-size` + `test:asset-budgets` + `scan:vuln`. If it exits non-zero, halt and fix the failure. Do not work around it by lowering thresholds, excluding files, or skipping git hooks with `--no-verify`.

The fast inner loop is `task test` and `task lint` individually. Reserve `task ci` for the final gate before claiming done.

Full rationale in [ADR-021](../docs/adr/ADR-021-Halt-On-Violation-Quality-Gate.md).

## ADR Discipline

- Check `docs/adr/` before proposing architectural changes.
- Every Accepted ADR is a constraint — if your proposal conflicts, halt and either revise the proposal or write a superseding ADR with status `Proposed`. When you halt, name the rule or ADR that stopped you and the legal move that would unblock the work.
- Existing ADRs are append-only (ADR-033): add a dated amendment note or a graduation-log entry, or write a superseding ADR. The PreToolUse guard (`scripts/adrguard`) asks the operator before any edit to an existing ADR; with nobody to approve, the edit is refused, which is intended. It denies hand-edits of generated files (`ADR_GUARD_OFF=1` is the operator's kill-switch for those).
- Every ADR carries an `## Enforcement` section (testable consequences → checks with a warn/block status, what is not machine-checkable, a graduation log). New checks start at **warn** and are promoted in `checks/enforcement.config.json` after 7+ clean days or one real catch.
- If a decision should be an ADR (picking a tool, library, pattern, or convention), say so — don't make architectural calls inline.
- ADR template: `docs/adr/TEMPLATE.md` (includes the mandatory `## Enforcement` block). Naming: `docs/adr/ADR-NNN-Title.md`. Numbering is sequential — check the highest existing number first. Index: `docs/adr/README.md`.

## Git Conventions

Conventional commits with these prefixes: `feat`, `fix`, `perf`, `docs`, `style`, `refactor`, `test`, `chore`, `ci`. Lowercase summary, concise, reference issues where applicable. Respect existing git hooks — never bypass with `--no-verify`.
