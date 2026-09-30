# Documentation

How a change gets documented in this repo, and where. The rules come from four ADRs: [ADR-011](../adr/ADR-011-Documentation-Standards.md) (what is documented and where), [ADR-033](../adr/ADR-033-ADR-Enforcement-Architecture.md) (every ADR carries an Enforcement section; the record is append-only), [ADR-022](../adr/ADR-022-Cross-Tool-Agents-Spine.md) (`AGENTS.md` is generated) and [ADR-030](../adr/ADR-030-Versions-Manifest-Contract.md) (`versions.json` is a public contract). This guide tells you which file to touch and what the checks will hold you to; it does not restate the ADRs.

One boundary applies to agents before anything else: `docs/` is read-only unless the operator asks for a documentation change ([ADR-019](../adr/ADR-019-Template-Scope-Boundary.md), scope table in [`.claude/workflow.md`](../../.claude/workflow.md)). The single standing exception is appending a graduation-log entry to an ADR's Enforcement section.

## Where things live

| Place | What belongs there |
|---|---|
| [`README.md`](../../README.md) | Orientation for a forker: prerequisites, Quick Start, the task list, project structure, the agentic-discipline summary, the `versions.json` contract, budgets. Deep material links into `docs/`. |
| [`CHANGELOG.md`](../../CHANGELOG.md) | Every user-visible change, per release, Keep a Changelog format (see below). |
| [`CLAUDE.md`](../../CLAUDE.md) + `.claude/*.md` | The layered constitution ([ADR-018](../adr/ADR-018-Layered-AI-Constitution.md)): halt-on-violation rules, engineering defaults ([`engineering.md`](../../.claude/engineering.md)), process ([`workflow.md`](../../.claude/workflow.md)), ephemeral stack facts ([`stack.md`](../../.claude/stack.md)). |
| [`AGENTS.md`](../../AGENTS.md) | **Generated** from the four layers above. Never hand-edited. |
| [`docs/adr/`](../adr/README.md) | Architecture decisions, ADR-000 onward, plus [`TEMPLATE.md`](../adr/TEMPLATE.md). The only place a decision is recorded. |
| [`docs/guides/`](README.md) | Topic-named how-to guides (this folder). They explain how to work inside the decisions; they link ADRs rather than restating them. |
| [`docs/design-system.md`](../design-system.md) | The role-token system (ADR-029): the roles, the rules the tests enforce, how to restyle. |
| [`docs/personalization-guide.md`](../personalization-guide.md) | Everything that carries the template's own identity and what a forker changes. |
| [`docs/README.md`](../README.md) | The folder map for `docs/`. |
| [`versions.json`](../../versions.json) | Machine-readable manifest of what the template ships; fetched raw by external consumers. |
| `.env.example` | The canonical template for environment variables (ADR-011, ADR-015). Add a variable there when you add one to `internal/config`. |
| Code comments | Non-obvious logic only (ADR-011). Everything else is named, typed, or tested. |

## Writing an ADR

**When.** Any time a change picks a tool, library, pattern or convention, or conflicts with an Accepted ADR — say so and write the ADR instead of deciding inline ([`.claude/workflow.md`](../../.claude/workflow.md), "ADR Discipline"). In the three-pass workflow the Architect pass owns this: it produces the ADR in `Proposed` status and the failing test, and no production code ([ADR-020](../adr/ADR-020-Agent-Roles.md), [`.claude/roles/architect.md`](../../.claude/roles/architect.md)).

**Numbering and filename.** Sequential; check the highest number in `docs/adr/` first (ADR-034 at the time of writing). The filename is `ADR-NNN-Title.md` with hyphenated title words — `adr011-adr-metadata` rejects anything that does not match `^ADR-\d{3}-[A-Za-z0-9-]+\.md$`. Creating a new ADR file is always allowed by the PreToolUse guard; editing an existing one is not (next section).

**Template and metadata.** Start from [`docs/adr/TEMPLATE.md`](../adr/TEMPLATE.md). The canonical layout for ADR-035 onward is the one ADR-030, ADR-031 and ADR-034 already use:

```markdown
# ADR-NNN: Title

**Date**: YYYY-MM-DD

## Status
## Context
## Decision
## Consequences
## Alternatives Considered
## References
## Enforcement
```

(Some earlier ADRs put a `## Date` heading after `## Status`; the check accepts both, but new ADRs use the `**Date**:` line.)

**Status vocabulary.** `Proposed` until accepted; then `Accepted`. Relationships go in the Status line, as the existing record does: `Accepted (supersedes ADR-008)` (ADR-017), `Superseded by ADR-017` (ADR-008), `Accepted (§3 superseded by ADR-026; §5 superseded by ADR-025)` (ADR-001), `Accepted (amends ADR-014 §4; refines ADR-025 §2)` (ADR-027). The template also lists `Deprecated`.

**The Enforcement block is mandatory.** ADR-033 §1 fixes the schema:

```markdown
## Enforcement
<!-- added YYYY-MM-DD, see ADR-033 (Enforcement Architecture) -->
- **Testable consequences:**
  - TC-1: <verifiable assertion>
- **Checks:**
  - TC-1 → <tool/rule or scripts/adrcheck check> (status: **warn** | **block**)
- **Not machine-checkable:** <named honestly>
- **Graduation log:** _(empty)_
```

Classify each constraint before you write it down: a structural invariant gets a check (new checks start at **warn**); an intent-level constraint gets only the "Not machine-checkable" line; a rule an existing tool already enforces (`go test`, golangci-lint, `agents:check`, the budget scripts) cites that tool with `status: **block**, pre-existing`. An ADR too vague to yield a testable assertion says so rather than inventing one. If you add a check, it lives in [`scripts/adrcheck/main.go`](../../scripts/adrcheck/main.go) and needs an entry in [`checks/enforcement.config.json`](../../checks/enforcement.config.json) (`id`, `adr`, `tc`, `status`, `added`); a config entry with no wired check is a BLOCKER on its own.

**What the metadata check actually verifies.** `adr011-adr-metadata` (warn, in `task check:adr`) checks two things per file: the filename pattern above, and the presence of a Status marker — a `Status` heading of one to four `#`, or `**Status**`/`**Status:**` in bold. It does not check that an Enforcement section exists; ADR-033's own Enforcement section records that gap. Reviewers hold the line the machine does not.

## Amending an ADR

Existing ADRs are **append-only** (ADR-033 §3, `.claude/workflow.md`). Three legal moves:

- **Dated amendment note.** Append a `> **Amended YYYY-MM-DD**: …` blockquote at the section it corrects (ADR-007, ADR-012, ADR-015, ADR-016 do this), or an `- **Amendment YYYY-MM-DD (#NN):**` bullet inside the Enforcement section when the change is a new TC or check (ADR-003, ADR-015, ADR-017). The original prose stays.
- **Graduation-log entry.** See the next section.
- **Supersede.** When the decision itself changes, write a new ADR. Both sides carry the relationship: the new ADR's Status line says `supersedes ADR-NNN` (or `amends ADR-NNN §N`), and the old ADR gets a dated note at the affected section plus a Status line update — `Superseded by ADR-NNN`, or `Accepted (§N superseded by ADR-NNN)` for a partial supersession. ADR-001 ↔ ADR-025/026 and ADR-008 ↔ ADR-017 are the worked examples.

The PreToolUse guard ([`scripts/adrguard/main.go`](../../scripts/adrguard/main.go), wired in [`.claude/settings.json`](../../.claude/settings.json)) denies agent Edit/Write on any existing `docs/adr/ADR-*.md`, on `AGENTS.md`, and on sqlc/templ-generated code, naming the governing ADR and the legal move. `ADR_GUARD_OFF=1` disables it for one operator-reviewed amendment — it is the operator's call, not the agent's. The companion Stop-gate ([`.claude/hooks/stop-gate.sh`](../../.claude/hooks/stop-gate.sh)) runs `go test ./...` and the check suite when an agent finishes a turn; `STOP_GATE_OFF=1` is its kill-switch.

## Graduating a check

A check moves from **warn** to **block** after 7+ days with no false positives or one real catch (ADR-033 §4). Promotion is three edits, all in one PR:

1. In [`checks/enforcement.config.json`](../../checks/enforcement.config.json), flip `"status"` to `"block"` and set `"graduated": "YYYY-MM-DD"`.
2. Append a dated entry to the **Graduation log** of the owning ADR's Enforcement section (the ADR-019 exception to read-only `docs/`).
3. Note it under `### Changed` in the `[Unreleased]` section of `CHANGELOG.md`.

`task check:adr` prints every check with its current status and any findings; `go run ./scripts/adrcheck --json` emits the same for machines. Use the human output to confirm the clean-days claim before promoting.

Demotion is always allowed and leaves the same three-line trail. `task check:generated` graduates differently — by flipping `-mode=block` in `Taskfile.yml`, as ADR-003 and ADR-017's amendments record. Nothing has been promoted yet; ADR-003's graduation log shows what a not-yet-promoted entry looks like.

## Constitution edits

Edit the source layer — `CLAUDE.md`, `.claude/engineering.md`, `.claude/workflow.md` or `.claude/stack.md` — then run `task agents:build` and commit the regenerated `AGENTS.md` with it. `task agents:check` (inside `task ci`, which the CI workflow invokes) fails when `AGENTS.md` differs from a fresh generation, so a forgotten rebuild cannot merge.

The generator ([`scripts/agentsmd/main.go`](../../scripts/agentsmd/main.go)) concatenates the four layers in that order, demotes every heading one level (code fences untouched), and rewrites link targets that start with `../` to be relative to the repo root — because `AGENTS.md` lives at the root while the `.claude/` layers live one directory down. Consequence: a link in `.claude/*.md` to a repo file must be written as `../docs/adr/ADR-NNN-Title.md` or `../README.md`, exactly as the existing layers do. A root-relative or absolute link would resolve in the layer but not in the generated spine, or vice versa.

`CLAUDE.md` stays at roughly ten halt-on-violation rules (ADR-018). Facts that churn — versions, commands, budgets — go in `stack.md`, not the constitution.

## CHANGELOG

[`CHANGELOG.md`](../../CHANGELOG.md) follows [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/) and SemVer. Entries land in `## [Unreleased]` as PRs merge, under `### Added`, `### Changed`, `### Fixed`, `### Removed` or `### Security` — one of each heading per release at most, so a new entry joins the existing heading rather than opening a second one. Each bullet references its PR or issue in parentheses, `(#142)`. At release the section becomes `## [X.Y.Z] - YYYY-MM-DD`, a fresh empty `[Unreleased]` goes above it, and the link references at the foot of the file are extended: `[X.Y.Z]: …/compare/vPREV...vX.Y.Z` and `[Unreleased]: …/compare/vX.Y.Z...HEAD`. Version links point at tags, never at branches.

## versions.json

[`versions.json`](../../versions.json) is a public consumption contract (ADR-030): adding a key is fine, renaming or removing one is a breaking change for consumers and needs a deliberate decision. Every non-`template` key has one in-repo source of truth, and `task versions:check` (in `task ci`) fails when the file disagrees with it. After bumping a dependency or toolchain pin, run `task versions:sync` (`go run ./scripts/checkversions -write`) and commit the manifest with the bump. A new key needs a wired source in `scripts/checkversions`; a key with no wired check fails the gate. Do not touch `template` by hand — the release workflow ([`.github/workflows/release.yml`](../../.github/workflows/release.yml)) stamps it from the `v*` tag on the default branch.

## Guides

Guides in `docs/guides/` are topic-named (`data-access.md`, `view-layer.md`, `deployment.md`, …) and indexed in [`README.md`](README.md). A guide explains how to work within the decisions the ADRs already made; when it would restate a decision, it links the ADR as `../adr/ADR-NNN-Title.md` instead. Every command, task name, path and check name in a guide must exist in the repo as written — run `task --list` and `ls` before citing, and re-verify when the thing you cite changes. Guides describe the current tree, not a build order or a project phase; if a guide only makes sense read in sequence with another, the sequence belongs in the index, not in the prose. Adding a guide means adding its row to the index and, if it changes the folder shape, to [`docs/README.md`](../README.md).

## Code comments

ADR-011 sets the bar: comment non-obvious logic, and only that. A comment explains *why* — the constraint, the ADR, the bug number — when the code cannot; it does not narrate what a well-named function already says. Package doc comments on the `scripts/*/main.go` programs are the house example: each states what the program enforces, which ADR owns it, and how to run it. When a comment cites an ADR, cite it by number so `grep ADR-0NN` finds every place a decision touches the code.
