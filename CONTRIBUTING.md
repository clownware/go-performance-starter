# Contributing

This repo is developed by humans and coding agents under the same rules. The rules live in [`CLAUDE.md`](CLAUDE.md) (ten halt-on-violation rules) and [`.claude/`](.claude/) (engineering defaults, workflow, stack facts); [`AGENTS.md`](AGENTS.md) is the generated copy other tools read. This page is the short human version and points at the record for everything else.

## Before you start

- `git fetch` and look at open PRs and branches; parallel sessions work on this repo.
- Read [`docs/adr/README.md`](docs/adr/README.md). Every Accepted ADR is a constraint. If your change conflicts with one, the change needs an ADR first, not a workaround.
- The paved paths are in [`docs/guides/`](docs/guides/README.md). If a guide and the code disagree, the guide is the bug; fix it in the same PR.

## The definition of done

```bash
task ci
```

A change is complete when that exits 0 ([ADR-021](docs/adr/ADR-021-Halt-On-Violation-Quality-Gate.md)). Nobody lowers a budget, demotes a check, excludes a file, or passes `--no-verify` to get there. What it runs and how to install the tools is in [tooling-and-quality-gates.md](docs/guides/tooling-and-quality-gates.md). The fast loop is `task test` and `task lint`.

Repository and integration tests need a database and run only when `DATABASE_URL` is set:

```bash
task db:up && task db:test:setup && task test
```

## Non-trivial work runs in three passes

For anything that touches several ADRs, adds a dependency, changes a public surface, or has non-obvious acceptance criteria ([ADR-020](docs/adr/ADR-020-Agent-Roles.md)):

1. **Architect** writes or amends the ADR and a failing table-driven test. No production code.
2. **Coder** writes the minimum that makes the test pass.
3. **Reviewer** runs `task ci`, reports the delta from the plan, and recommends. No commits.

Trivial changes (typo, single line, rename) skip this. Non-trivial production code always follows a failing test ([ADR-023](docs/adr/ADR-023-Testing-Philosophy.md)); [testing.md](docs/guides/testing.md) shows what each layer looks like.

## ADRs

- Start from [`docs/adr/TEMPLATE.md`](docs/adr/TEMPLATE.md); the number is one above the last row in the index. The `## Enforcement` block is mandatory ([ADR-033](docs/adr/ADR-033-ADR-Enforcement-Architecture.md)).
- Existing ADRs are **append-only**: add a dated amendment note or a graduation-log entry, or supersede with a new ADR. Both sides of a supersession or amendment carry a note. A guard hook denies in-place edits by agents; `ADR_GUARD_OFF=1` is the operator-reviewed kill-switch.
- Checks start at warn and are promoted to block in `checks/enforcement.config.json` after 7+ clean days or one real catch, logged in the ADR and the CHANGELOG.

Details in [documentation.md](docs/guides/documentation.md).

## Commits and pull requests

- Conventional commits: `feat`, `fix`, `perf`, `docs`, `style`, `refactor`, `test`, `chore`, `ci`, with an optional scope: `fix(health): …`. Lowercase summary; reference issues.
- Branch, PR, CI green, merge. The PR template asks for the ADRs touched, the `task ci` result, and a CHANGELOG line under `[Unreleased]` for anything user-visible.
- Generated files are never hand-edited: `internal/database/` (`task db:generate`), `*_templ.go` (`task templ:generate`), `AGENTS.md` (`task agents:build` after any `CLAUDE.md` or `.claude/*.md` change). After bumping a pinned tool, `task versions:sync` ([ADR-030](docs/adr/ADR-030-Versions-Manifest-Contract.md)).
- Found work that does not belong in the current change becomes an issue with the `found-work` label, never a note left in chat or a TODO.

## Scope for agents

`docs/` is read-only unless the change is asked for; deployment infrastructure, marketing content, and maintenance scripts are not created uninvited ([ADR-019](docs/adr/ADR-019-Template-Scope-Boundary.md)). The canonical scope table is in [`.claude/workflow.md`](.claude/workflow.md).

## Labels

Severity `P0`–`P3`, type `bug` / `enhancement` / `docs` / `chore` / `decision` / `security`, status `blocked` / `found-work` / `launch` / `triaged`. The issue templates set the type label; a maintainer sets severity.

## Reporting a vulnerability

See [`SECURITY.md`](SECURITY.md). Do not open a public issue for a security problem.
