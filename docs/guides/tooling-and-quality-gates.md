# Tooling and quality gates

`task ci` is the single definition of done ([ADR-021](../adr/ADR-021-Halt-On-Violation-Quality-Gate.md)): a change is not complete until it exits 0, and nobody — human or agent — lowers a threshold, excludes a file, or skips a hook to get there (CLAUDE.md rule 9). This guide is what the gate runs and the tools around it. Run `task --list` for everything.

## What `task ci` runs, in order

| Leg | What it checks | Decided in |
|---|---|---|
| `fmt:check` | `golangci-lint fmt` diff is empty (gofmt + goimports with the module as local prefix) | ADR-010 |
| `lint` | `.golangci.yml`: errcheck, govet, ineffassign, misspell, staticcheck, unused, whitespace; `generated: lax` | ADR-010 |
| `go mod verify` | The module cache matches `go.sum` (supply-chain tamper check) | ADR-014 §8 |
| `go test -race -covermode=atomic ./...` | Every test, race detector on; the `internal/performance` budget tests run here too | ADR-023, ADR-000 |
| `agents:check` | `AGENTS.md` equals what `task agents:build` would generate | ADR-022 |
| `versions:check` | `versions.json` equals the repo's real pins | ADR-030 |
| `check:adr` | The ADR enforcement suite (`scripts/adrcheck`); block-status findings fail, warn-status report. All eleven checks are block since 2026-09-30 | ADR-033 |
| `check:generated` | sqlc and templ regenerated into a scratch copy equal the committed output (block since 2026-09-30) | ADR-003, ADR-017 |
| `test:binary-size` | Stripped build (`-ldflags="-s -w"`) under 20MB | ADR-000 |
| `test:asset-budgets` | Built CSS and shipped JS under 30KB / 50KB gzipped (`task css:build` runs first) | ADR-000 |
| `scan:vuln` | `govulncheck ./...` | ADR-014 |

CI ([`.github/workflows/ci.yml`](../../.github/workflows/ci.yml)) runs exactly this gate as the **Quality Gate** job after bootstrapping a Postgres service with `task db:test:setup`, then a separate **Docker Image** job builds the image and enforces the 30MB budget. Coverage is uploaded for information; there is no coverage floor. `vuln-scan.yml` re-runs govulncheck on a schedule so a new advisory surfaces without a push.

## Tools and how they install

`golangci-lint` (pinned in `Taskfile.yml`'s `GOLANGCI_LINT_VERSION`, mirrored in `versions.json`) and `gremlins` install themselves through `lint:install` / `mutation:install` on first use. `templ`, `sqlc`, `migrate`, `air` and `psql` are prerequisites — the README's prerequisites block has the pinned `go install` lines; CI's "Install tools" step is the reference. `govulncheck` is a `tool` directive in `go.mod`, so `go run`/`task scan:vuln` resolve it without a global install.

## Inner loop

- `task dev` — air, rebuilding on `.go` and `.templ` changes (`.air.toml` runs `templ generate` before `go build`). CSS is not watched by air; `task css:build` after editing `input.css`, or run `task css:watch` alongside.
- `task test`, `task lint` — the fast loop. `task test:coverage` writes an HTML report.
- `task templ:generate`, `task db:generate` — after editing `.templ` or `sql/`.
- `task db:up`, `task db:test:setup` — local Postgres with the `auth.uid()` stub and migrations, which also unlocks the `DATABASE_URL`-gated integration tests ([testing.md](testing.md)).

## Mutation testing

`task test:mutation` runs go-gremlins over the packages scoped in [ADR-032](../adr/ADR-032-Mutation-Testing.md). It is not part of `task ci` (too slow for every push) but is the tool for asking whether a test would notice the code being wrong; a surviving mutant is a real finding ([#138](https://github.com/clownware/go-performance-starter/issues/138) is a worked example).

## The two hooks

Configured in [`.claude/settings.json`](../../.claude/settings.json), these are the only *blocking* layer of [ADR-033](../adr/ADR-033-ADR-Enforcement-Architecture.md) and they act on agents, not on `git`:

- **Stop-gate** (`.claude/hooks/stop-gate.sh`): when an agent tries to end its turn, `go test ./...` and `scripts/adrcheck` run; a failing test or a block-status finding sends it back. Kill-switch `STOP_GATE_OFF=1`.
- **PreToolUse guard** (`scripts/adrguard`): denies `Edit`/`Write` to existing ADRs, `AGENTS.md`, `internal/database/*.go` and `*_templ.go`, naming the legal move in each case. Kill-switch `ADR_GUARD_OFF=1`, meant for operator-reviewed ADR amendments.

There are no client-side git hooks in the repo; the constitution's "never `--no-verify`" applies to any you add.

## Graduating a check

Checks in `checks/enforcement.config.json` start at **warn** and move to **block** after 7+ clean days or one real catch; the promotion is logged in the owning ADR's graduation log and in `CHANGELOG.md`. `check:generated` has its own `-mode` flag in the Taskfile with the same rule. The first graduation happened on 2026-09-30: all eleven adrcheck checks and `check:generated`, after 42–80 clean days. Never fix a red gate by demoting a check.

## Dependencies

Dependabot ([`.github/dependabot.yml`](../../.github/dependabot.yml)) opens PRs for Go modules, npm and Actions. A bump to a pinned tool must be followed by `task versions:sync` on that branch or `versions:check` fails ([ADR-030](../adr/ADR-030-Versions-Manifest-Contract.md)).
