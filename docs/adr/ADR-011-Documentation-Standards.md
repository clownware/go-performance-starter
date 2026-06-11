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
