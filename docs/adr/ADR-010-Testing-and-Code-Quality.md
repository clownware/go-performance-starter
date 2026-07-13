# ADR-010: Testing, Linting, and Code Quality

**Date**: 2025-05-01

## Status
Accepted

## Context

A robust testing and code quality strategy is essential for reliability and maintainability. The team needs to standardize on tools and patterns for Go code quality, testing, and CI.

## Decision

- Use Go's built-in `testing` package for all tests.
- Favor table-driven tests for handler and utility logic.
- Use integration tests for database flows.
- Add `golangci-lint` for linting, and enforce `gofmt` for formatting.
- Run tests and linting in CI (see Taskfile and GitHub Actions).

## Consequences

- Codebase is more robust and maintainable.
- Onboarding is easier for new devs.
- CI failures catch issues before production.

---

# ADR-011: Documentation Standards

## Status
Accepted

## Context

Clear and up-to-date documentation is critical for onboarding and handoff. The team must decide where and how documentation is maintained.

## Decision

- All major architectural decisions are documented as ADRs in `/docs/adr`.
- User/developer guides live in `/docs/implementation-guides`.
- Code should be commented for non-obvious logic.
- `.env.example` is maintained as the canonical template for environment variables.

## Consequences

- Team can onboard quickly.
- Less knowledge is lost over time.

## Enforcement
<!-- added 2026-07-12, see ADR-033 (Enforcement Architecture) -->
- **Testable consequences:**
  - TC-1: `golangci-lint run ./...` passes.
  - TC-2: All Go files are gofmt-clean.
  - TC-3: `go test` runs in the gate with race detection and coverage.
- **Checks:**
  - TC-1 → `task lint` in `task ci` (status: **block**, pre-existing)
  - TC-2 → `task fmt:check` in `task ci` (status: **block**, pre-existing)
  - TC-3 → `go test -race -covermode=atomic` in `task ci` (status: **block**, pre-existing)
- **Not machine-checkable:** Table-driven style preference and integration-test judgment (see ADR-023).
- **Graduation log:** _(empty)_
