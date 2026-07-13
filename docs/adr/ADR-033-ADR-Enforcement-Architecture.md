# ADR-033: ADR Enforcement Architecture

## Status

Accepted

## Date

2026-07-12

## Context

Thirty-two ADRs govern this repo, but until now their constraints were enforced unevenly: some had dedicated tests wired into `task ci` (ADR-028's inline-script pin, ADR-029's token scan), some were enforced as a side effect of the build (ADR-005's named queries), and many stated rules no machine ever verified (ADR-026's `log.Printf` ban, ADR-002's migration pairing). An agent or contributor had no single place to learn which rules are checked, which are review territory, and which are aspirational.

This ADR applies the Clownware ADR-enforcement pattern: every ADR declares its testable consequences, a deterministic check suite verifies them, and enforcement graduates from warning to blocking only after a check earns trust. The pattern was specified for the Clownware repo family alongside `astro-performance-starter`; at implementation time the Astro repo had not yet landed its enforcement ADR, so this repo is the pattern's reference implementation, not a port.

An honest scale note: this repo needed less new machinery than the pattern budgets for. Roughly half of the machine-checkable surface was already blocking in `task ci` (ADR-021) before this ADR existed. The new value is the mapping, the gap-filling checks, and the graduation mechanism — not a new wall of gates.

## Decision

### 1. Every ADR carries an Enforcement section

Appended as an amendment (never rewriting existing prose), in a fixed schema:

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

Constraints are classified into three buckets: **A — structural invariant** (deterministic: imports, file layout, forbidden patterns, config), **B — semantic constraint** (intent-level; gets only the "Not machine-checkable" line), **C — process rule** (metadata-checkable). An ADR too vague to yield a testable assertion is reported as such, never creatively patched.

### 2. A warn-only check suite: `scripts/adrcheck`

A single dependency-free Go program (matching the repo's `scripts/<name>/main.go` convention), run as `task check:adr` inside `task ci`. Checks are driven by `checks/enforcement.config.json`:

```json
{ "id": "adr026-slog-only", "adr": "ADR-026", "tc": "TC-1/TC-2",
  "status": "warn", "added": "2026-07-12" }
```

Semantics: a failing **warn** check reports `WARNING` and never fails the run; a failing **block** check reports `BLOCKER` and exits non-zero; a config entry naming no wired check is always a `BLOCKER` (the ADR-030 "key with no wired check fails" rule). Failure messages name the ADR, the TC, and the remedy. Output is human-readable by default, `--json` for machines. The suite is deterministic and runs in under a second.

Checks that an existing tool already enforces are not re-implemented: their Enforcement sections cite the tool (`go test`, golangci-lint, `agents:check`, budget scripts) with status **block, pre-existing**. Restarting already-blocking enforcement at warn would have been a downgrade dressed as rigour.

### 3. Two blocking hooks — the only blocking layer at launch

Configured in `.claude/settings.json` (project-shared, checked in), verified against the Claude Code hooks documentation as of 2026-07-12:

1. **Stop-gate** (`.claude/hooks/stop-gate.sh`): when an agent finishes a turn, run `go test ./...` and the check suite. A test failure or BLOCKER exits 2, which prevents completion and feeds the report back into the agent's loop; warnings pass through as information. Respects `stop_hook_active` to avoid ping-pong. Kill-switch: `STOP_GATE_OFF=1`.
2. **PreToolUse guard** (`scripts/adrguard`): denies agent Edit/Write on protected paths — existing ADRs (the record is append-only; legal moves are a graduation-log append or supersession), `AGENTS.md` (generated, ADR-022), and sqlc/templ-generated code (ADR-019). The denial message names the governing ADR and the legal move. Kill-switch: `ADR_GUARD_OFF=1` for one operator-authorized amendment.

### 4. Graduation rule

A check is promoted from **warn** to **block** after **7+ days with no false positives OR one real catch** (it flagged a change that was genuinely wrong). Promotion is three lines: flip `status` in `checks/enforcement.config.json` and set `graduated`, append a dated entry to the owning ADR's Graduation log, and note it in `CHANGELOG.md`. **Demotion is always allowed** — a check producing false positives goes back to warn immediately, with the same three-line trail. Nothing was promoted in the launch session; every new check starts at warn.

### 5. Deltas from the briefed pattern

- **Go-native suite in `scripts/`, not shell in `checks/`** — a check written in the repo's own language is a feature consumers of a starter template inherit; only the config lives in `checks/`.
- **Pre-existing enforcement mapped, not migrated** — `task ci`'s existing gates keep their blocking status and are recorded as such, rather than being rebuilt inside adrcheck.
- **Import checks parse Go, not text** — `go/parser` (ImportsOnly), after the text version flagged its own detection literal on first run.

## Consequences

- Every ADR now answers "how would I know this rule was broken?" in a fixed place, including the honest answer "you wouldn't — it's review territory."
- Warn-only launch means the suite can be wrong without hurting anyone; trust is earned per-check through the graduation log, and the log is public evidence either way.
- Two new `scripts/` programs and a hooks config join the template; consumers who don't want enforcement delete `scripts/adrcheck`, `scripts/adrguard`, `checks/`, `.claude/hooks/`, the `check:adr` task, and the `hooks` block in `.claude/settings.json`.
- The Stop-gate adds a `go test ./...` run each time an agent finishes a turn — seconds with a warm build cache.
- First run produced one genuine finding: `internal/handler/health_handler.go` imports `pgxpool` to ping the pool for `/health`, a gray zone against ADR-003's letter. It stays a warning pending a triage decision (allowlist as infrastructure vs. refactor behind an interface) — the graduation decision for that check will force the call.

### TODOs (deferred, not forgotten)

- sqlc and templ regeneration-drift checks (`task db:generate` / `templ generate` output vs committed code) — ADR-003/ADR-017.
- A secret scanner for ADR-014/ADR-015's "no hardcoded secrets" beyond structural patterns.
- Lighthouse-CI accessibility/performance gates (ADR-000, ADR-009) — deliberately unwired at this repo's scale.

## Alternatives Considered

- **Encode the new checks as golangci-lint rules (forbidigo/depguard).** Rejected — lint findings are already blocking, which contradicts the warn-only launch; and most of the suite (migration pairing, package.json, workflow structure, ADR metadata) isn't lintable Go anyway. Splitting one policy across two engines with different severity semantics costs more than it saves.
- **Block everything from day one.** Rejected — an unproven check that false-positives inside a blocking gate erodes trust in the whole gate (the ADR-021 lesson in reverse). Warn-first is how a check earns its place.
- **Agent-as-judge review of semantic (bucket B) constraints.** Out of scope by decision — deterministic checks only; bucket B stays human.

## Enforcement
<!-- added 2026-07-12, see ADR-033 (Enforcement Architecture) -->
- **Testable consequences:**
  - TC-1: Every ADR file contains an `## Enforcement` section.
  - TC-2: Every entry in `checks/enforcement.config.json` names a wired check.
- **Checks:**
  - TC-1 → covered by `adr011-adr-metadata`'s scope going forward; verified 33/33 at launch (status: **warn**)
  - TC-2 → adrcheck itself — an unwired config entry is a BLOCKER (status: **block**, structural)
- **Not machine-checkable:** Whether a TC honestly captures its ADR's intent, and whether graduation decisions follow the 7-day/real-catch rule — operator judgment, evidenced by the public graduation logs.
- **Graduation log:** _(empty)_

## References

- [ADR-018](ADR-018-Layered-AI-Constitution.md), [ADR-019](ADR-019-Template-Scope-Boundary.md), [ADR-021](ADR-021-Halt-On-Violation-Quality-Gate.md), [ADR-022](ADR-022-Cross-Tool-Agents-Spine.md), [ADR-030](ADR-030-Versions-Manifest-Contract.md)
- Claude Code hooks reference: https://code.claude.com/docs/en/hooks (verified 2026-07-12; the docs carry no version identifier)
