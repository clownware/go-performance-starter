# ADR-005: Handling Primary Organization Selection - Two-Step SQL Query

*   **Status:** Accepted
*   **Date:** 2025-05-01

## Context

The application requires users to designate one of their organization memberships as "primary". This involves atomically updating two sets of records: setting the target membership record's `is_primary_organization` flag to `true`, and setting the flag to `false` for all *other* memberships belonging to the same user.

An initial attempt was made to perform this operation within a single SQL statement using a Common Table Expression (CTE) involving two `UPDATE` statements and a `RETURNING` clause to pass the `user_id` between them. However, the `sqlc` tool (documented in ADR-003), used for generating type-safe Go code from SQL, failed to parse this complex query structure, consistently reporting ambiguous column references despite various aliasing attempts.

## Decision

Refactor the database operation into two distinct, simpler SQL queries:

1.  `-- name: SetPrimaryOrganizationStep1`: Updates the target membership, setting `is_primary_organization = true`.
2.  `-- name: SetPrimaryOrganizationStep2`: Updates all other memberships for the given user (excluding the target organization), setting `is_primary_organization = false`.

These two `sqlc`-generated query functions are executed sequentially within a single database transaction managed by the Go repository layer (`internal/repository/postgres/organization_member.go`) to ensure the overall operation remains atomic.

## Consequences

### Pros:

*   **Unblocks Development:** Resolves the `sqlc` parsing and code generation issue, allowing development to proceed.
*   **Simpler SQL:** The individual SQL queries are less complex and easier for `sqlc` and potentially developers to understand.
*   **Atomicity Maintained:** Atomicity is preserved through the use of standard database transactions managed in the Go application code.

### Cons:

*   **Application-Level Transaction:** Requires transaction management logic (begin, defer rollback, commit) within the Go application code, slightly increasing complexity there compared to a purely SQL-based atomic operation.
*   **Less SQL Elegance:** From a pure SQL perspective, the two-step approach is less elegant than the (non-functional with sqlc) single-statement CTE approach.
*   **Multiple Round Trips:** Involves two separate database calls within the transaction, although this impact is likely negligible for this operation.

## 2026-04 Review Note

Investigated whether modern sqlc versions have resolved the CTE parsing limitation. As of sqlc v1.29+, multiple related issues remain open (GitHub sqlc-dev/sqlc#3861 — ambiguous column reference in nested CTE chains). The database-backed analyzer may help with some patterns, but the specific multi-UPDATE CTE structure this ADR addresses is not confirmed fixed. The two-step transactional approach remains the correct solution and performs well within ADR-000 budget constraints (< 10ms total for both queries on indexed unique constraints).

## Enforcement
<!-- added 2026-07-12, see ADR-033 (Enforcement Architecture) -->
- **Testable consequences:**
  - TC-1: The named queries `SetPrimaryOrganizationStep1`/`SetPrimaryOrganizationStep2` exist and compile.
- **Checks:**
  - TC-1 → sqlc generation + `go build` in `task ci` (status: **block**, pre-existing)
- **Not machine-checkable:** The <10ms combined-latency budget (monitored via ADR-000 instrumentation, not gated).
- **Graduation log:** _(empty)_
