# ADR-026: Standardize Structured Logging on log/slog

**Date**: 2026-07-05

## Status

Accepted (supersedes ADR-001 §3; amends ADR-013 §2)

## Context

ADR-001 chose zerolog, but the codebase drifted: `cmd/api/main.go`, `internal/config`, and all handlers use stdlib `log/slog`; only `internal/middleware/metrics.go` uses zerolog; and 13 call sites (auth handlers, auth middleware, profile handler) still use unstructured `log.Printf`. Three logging APIs in one codebase means inconsistent output formats, broken log aggregation queries, and two dependencies doing one job. Issue #16 tracks the inconsistency.

When ADR-001 was written (2025-04), `log/slog` was "newer and less mature". That is no longer true: slog has been the stdlib structured-logging standard since Go 1.21, the ecosystem (chi middleware, OTel bridges) supports it, and the codebase already voted with its feet.

## Decision

Standardize all logging on **`log/slog`** (stdlib):

- Migrate `internal/middleware/metrics.go` off zerolog; remove `github.com/rs/zerolog` from `go.mod`.
- Replace all `log.Printf` call sites with leveled `slog` calls carrying structured fields.
- One logger setup in `main.go`: JSON handler when `ENV=production`, text handler otherwise; level from env (`LOG_LEVEL`, default `info`); `slog.SetDefault` remains the wiring mechanism.
- The log-level semantics, required context fields (`request_id`, `user_id`, `error`, `duration_ms`), and scrubbing rules in ADR-013 are unchanged — only the library changes.

Rationale for slog over migrating everything to zerolog:

- **Zero dependency weight** — aligns with ADR-000 budgets and the starter's minimal-dependency ethos; zerolog is the only thing `go.mod` keeps it for.
- **The codebase is already ~95% slog** — this direction is a small refactor of one middleware file plus the `log.Printf` stragglers; the reverse is a full-codebase migration.
- **Performance is not the bottleneck**: zerolog wins microbenchmarks, but logging is nowhere near the P95 < 100ms budget's critical path, and slog's JSON handler allocation profile is more than adequate at this request volume.

Implementation lands in the consistency phase (issue #16) as a table-driven-tested refactor per ADR-023.

## Consequences

- One log format across the app; aggregation queries and the ADR-013 context standards actually hold.
- `go.mod` loses a dependency; binary size budget gets marginal headroom.
- `metrics.go` needs a small refactor; ADR-013's zerolog code samples become illustrative rather than literal (amendment note added there).
- Anyone forking the template who prefers zerolog/zap swaps one `main.go` setup block rather than untangling three APIs.

## Alternatives Considered

- **Standardize on zerolog (enforce ADR-001 as written).** Rejected: migrates ~30 files instead of one, keeps a dependency stdlib now covers, and fights the codebase's existing direction.
- **Keep both, document a boundary.** Rejected: there is no principled boundary; "middleware logs differently than handlers" is drift with a permission slip.
- **zap.** Rejected: same dependency-weight argument as zerolog with a heavier API.

## References

- [ADR-001](ADR-001-Foundation.md) §3 (superseded), [ADR-013](ADR-013-Error-Handling-and-Observability.md) §2 (amended), [ADR-000](ADR-000-Performance-Budgets-and-Quality-Attributes.md) (budgets), [ADR-023](ADR-023-Testing-Philosophy.md) (test-first refactor)
- Issue #16 — zerolog vs slog inconsistency
- [Go blog: Structured Logging with slog](https://go.dev/blog/slog)

## Enforcement
<!-- added 2026-07-12, see ADR-033 (Enforcement Architecture) -->
- **Testable consequences:**
  - TC-1: `github.com/rs/zerolog` is absent from `go.mod`.
  - TC-2: No `log.Printf`/`log.Print`/`log.Println` call sites in `cmd/` or `internal/` — logging goes through `log/slog`.
- **Checks:**
  - TC-1, TC-2 → `adr026-slog-only` in `scripts/adrcheck` (status: **warn**)
- **Not machine-checkable:** Structured-field completeness (`request_id`, `user_id`, `error`, `duration_ms` on relevant entries) and scrubbing rules — semantic, per ADR-013.
- **Graduation log:** _(empty)_
