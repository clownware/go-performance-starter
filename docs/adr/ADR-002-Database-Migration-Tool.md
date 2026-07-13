# ADR-002: Database Migration Tool

**Date**: 2025-04-30

**Status**: Accepted

## Context

As the application evolves, the database schema will need to change. We need a reliable and consistent way to manage and apply these schema changes across different environments (development, testing, production).

Key requirements for a migration tool include:
- Versioning of schema changes.
- Ability to apply migrations sequentially (up).
- Ability to revert migrations (down).
- Integration with our development and CI/CD workflows.
- Compatibility with PostgreSQL (used by Supabase).

## Decision

We will use `golang-migrate` ([https://github.com/golang-migrate/migrate](https://github.com/golang-migrate/migrate)) as our database migration tool.

Migrations will be written as pairs of raw SQL files (`.up.sql` and `.down.sql`) stored in the `migrations/` directory.

The `Taskfile.yml` will include tasks to create new migration files and apply/revert migrations using the `golang-migrate` CLI against the configured `DATABASE_URL`.

## Consequences

**Positive:**
- **SQL-First:** Allows direct use of PostgreSQL features and syntax in migrations.
- **Simplicity:** `golang-migrate` is a focused tool with a clear CLI interface.
- **Widely Adopted:** It's a standard and well-understood tool in the Go ecosystem.
- **Language Agnostic:** Migrations are plain SQL, not tied to Go code constructs.
- **Explicit Control:** Schema changes are explicit and version-controlled.

**Negative:**
- **External Tool:** Requires installing and managing the `golang-migrate` CLI tool.
- **Manual SQL:** Requires writing raw SQL for both `up` and `down` migrations (no automatic `down` generation).
- **No Go Integration:** Migrations are managed outside the Go application binary itself (though can be embedded if desired, we opt for CLI usage initially).

## Alternatives Considered

- **ORM Auto-Migration (e.g., GORM):** Rejected because we are not using a full ORM, and auto-migration can be opaque and risky in production environments.
- **Atlas:** A powerful, modern schema management tool. Considered more complex than needed for our initial setup, but could be revisited later.
- **sqlc Migrations (Experimental):** `sqlc` has some experimental migration features, but `golang-migrate` is more mature and purpose-built for this task.
- **dbmate:** Similar to `golang-migrate`, also a viable option, but `golang-migrate` is slightly more prevalent in the Go ecosystem.

## References

- `golang-migrate` Repository: [https://github.com/golang-migrate/migrate](https://github.com/golang-migrate/migrate)

## Enforcement
<!-- added 2026-07-12, see ADR-033 (Enforcement Architecture) -->
- **Testable consequences:**
  - TC-1: Every `migrations/*.up.sql` has a matching `*.down.sql`, and vice versa; migrations are raw SQL files.
- **Checks:**
  - TC-1 → `adr002-migration-pairs` in `scripts/adrcheck` (status: **warn**)
- **Not machine-checkable:** That a down migration actually reverts its up migration — revert safety is review territory.
- **Graduation log:** _(empty)_
