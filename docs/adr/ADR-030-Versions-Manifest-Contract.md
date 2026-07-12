# ADR-030: Versions Manifest Contract

**Date**: 2026-07-12

## Status

Accepted

## Context

External consumers (marketing sites, template catalogs, dashboards) want to display what this template ships — language, framework, and tool versions — without cloning the repo or parsing `go.mod`. The sibling Astro starter established the pattern: a `versions.json` at the repo root, fetched raw from the default branch at build time.

Such a manifest is only useful if it cannot lie. Hand-maintained version lists drift the first time a Dependabot PR merges without anyone remembering the manifest exists. The repo already solves this class of problem for `AGENTS.md` with a generated-artifact drift check in the quality gate (ADR-022); the same approach applies here.

## Decision

1. **`versions.json` at the repo root is a public consumption contract.** One `template` key for the template's own release version, plus one key per meaningful stack pin. **Additive changes (new keys) are fine; renaming or removing a key is a breaking change** for consumers and needs a deliberate decision.
2. **Every non-`template` key has a single in-repo source of truth**, and `task versions:check` (part of `task ci`) fails when the manifest disagrees with it: `go.mod` (Go toolchain/minimum and direct dependencies), `package-lock.json` (Tailwind), the vendored JS bundles (htmx, Alpine), the sqlc header in `internal/database/db.go`, and the workflow pins (golang-migrate, Node). A key with no wired check also fails — nothing may drift silently. Keys with no single source of truth (e.g. Postgres, which diverges between docker-compose and CI) are excluded rather than guessed.
3. **The `template` field is stamped by the release workflow, not by hand.** On a `v*` tag, `release.yml` commits `.template = <tag>` to the default branch — the ref consumers fetch — so the field cannot drift from tags. CI only enforces its `vX.Y.Z` format.

## Consequences

- Dependency bumps that touch a manifested pin must update `versions.json` in the same PR or CI fails — that is the point. Dependabot PRs for manifested dependencies will need a follow-up commit.
- The manifest lags a release by one commit on the default branch (the stamp lands after the tag), which is acceptable for build-time consumers.
- Consumers can rely on key stability; this repo takes on the discipline cost of treating renames/removals as breaking.

## Alternatives Considered

- **Generate versions.json entirely instead of checking it.** Rejected — generation needs a build step consumers can't see run; a checked-in file with a drift gate is simpler and matches the ADR-022 pattern.
- **Verify the `template` field against git tags in CI.** Rejected — every pre-release commit would fail between bumping and tagging; the stamp job owns that field.

## References

- Pattern source: `clownware/astro-performance-starter` `versions.json`.
- Related: [ADR-021](ADR-021-Halt-On-Violation-Quality-Gate.md) (quality gate), [ADR-022](ADR-022-Cross-Tool-Agents-Spine.md) (drift-check precedent), [ADR-025](ADR-025-Deployment-Target.md) (release pipeline).
