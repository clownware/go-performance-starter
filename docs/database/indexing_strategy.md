# Database Indexing and Constraint Strategy

*   **Date:** 2025-05-01

This document outlines the initial strategy for database indexes and constraints as established in Phase 1.

## Constraints

The following constraints are applied to ensure data integrity:

*   **Primary Keys:** Each table (`users`, `organizations`, `organization_members`) has a UUID primary key (`id`) ensuring entity uniqueness.
*   **NOT NULL:** Columns essential for basic functionality or identification (e.g., `users.email`, `organizations.name`, `organization_members.user_id`, `organization_members.organization_id`, `*.created_at`, `*.updated_at`) are marked `NOT NULL`.
*   **Foreign Keys:** Referential integrity is maintained using foreign key constraints:
    *   `organization_members.user_id` references `users(id)` with `ON DELETE CASCADE`.
    *   `organization_members.organization_id` references `organizations(id)` with `ON DELETE CASCADE`.
*   **Uniqueness:**
    *   `users.email` has a UNIQUE constraint.
    *   `users.auth_id` has a UNIQUE constraint (linking to the external auth provider ID).
    *   `organizations.slug` has a UNIQUE constraint.
    *   A composite UNIQUE constraint exists on `organization_members(user_id, organization_id)` to prevent duplicate memberships.
*   **Check Constraints:**
    *   `organizations.plan_type` likely has constraints (defined implicitly by application logic or to be added based on specific plan types).
    *   `organization_members.role` has a CHECK constraint (`role IN ('owner', 'admin', 'member')`).
*   **Defaults:** `created_at`, `updated_at`, and `is_active` fields typically have default values.

## Indexing

*   **Primary Keys:** Automatically indexed by PostgreSQL.
*   **Foreign Keys:** Foreign key columns (`organization_members.user_id`, `organization_members.organization_id`) are automatically indexed by Supabase/PostgreSQL by default, which is beneficial for joins and cascading deletes.
*   **Unique Constraints:** Columns with UNIQUE constraints (`users.email`, `users.auth_id`, `organizations.slug`, and the composite key on `organization_members`) are automatically indexed.
*   **RLS Indexes:** Indexes are implicitly necessary for efficient RLS policy evaluation, particularly on columns used in `WHERE` clauses within policies (e.g., `user_id` in `organization_members`).
*   **Future Considerations:** As application query patterns emerge, additional indexes will be added based on:
    *   Columns frequently used in `WHERE` clauses.
    *   Columns used in `ORDER BY` clauses.
    *   Columns involved in `JOIN` conditions (beyond FKs if necessary).

This strategy provides a solid foundation for data integrity and basic query performance. It will be reviewed and potentially expanded in later phases as application usage patterns become clearer.
