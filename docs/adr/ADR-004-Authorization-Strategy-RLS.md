# ADR-004: Authorization Strategy - Row Level Security (RLS)

*   **Status:** Accepted
*   **Date:** 2025-05-01

## Context

The application is multi-tenant, requiring strict data isolation between different organizations and users. Authorization rules must ensure users can only access resources (like organizations, memberships, etc.) they are permitted to see, typically those associated with their own account or organization memberships. Implementing this purely at the application layer can be complex and prone to errors, potentially leading to data leaks between tenants.

## Decision

Implement primary authorization logic directly within the PostgreSQL database using Row Level Security (RLS) policies. Leverage Supabase's built-in `auth.uid()` function within these policies to identify the currently authenticated user making the request. Application-level checks might still be used for finer-grained permissions, but the core "can this user see this row" logic resides in the database.

## Consequences

### Pros:

*   **Strong Security:** Provides robust security guarantees enforced at the data layer, reducing the risk of application-level bugs bypassing authorization checks.
*   **Simplified Application Logic:** Reduces the need for repetitive authorization boilerplate code within the Go application handlers and repositories.
*   **Centralized Policies:** Authorization rules are defined centrally within the database schema (managed via migrations), making them easier to audit and manage.
*   **Supabase Integration:** Easily leverages Supabase Auth (`auth.uid()`) for identifying the user within policies.

### Cons:

*   **SQL Complexity:** Shifts some complexity to writing and managing SQL policies within database migrations.
*   **Debugging:** Diagnosing authorization issues may require inspecting database logs and policy definitions in addition to application code.
*   **Performance:** Requires careful policy design and appropriate indexing to avoid performance degradation on queries.
*   **SECURITY DEFINER:** May necessitate the use of `SECURITY DEFINER` functions for specific scenarios (e.g., checking permissions based on related tables), which must be implemented carefully to avoid security vulnerabilities.
