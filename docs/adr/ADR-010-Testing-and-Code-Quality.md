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
