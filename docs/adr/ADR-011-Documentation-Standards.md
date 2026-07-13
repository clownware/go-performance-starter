# ADR-011: Documentation Standards

**Date**: 2025-05-01

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
  - TC-1: Every `docs/adr/ADR-*.md` follows the `ADR-NNN-Title.md` naming pattern and contains a `## Status` heading.
- **Checks:**
  - TC-1 → `adr011-adr-metadata` in `scripts/adrcheck` (status: **warn**)
- **Not machine-checkable:** Comment quality ("non-obvious logic is commented") and implementation-guide freshness.
- **Graduation log:** _(empty)_
