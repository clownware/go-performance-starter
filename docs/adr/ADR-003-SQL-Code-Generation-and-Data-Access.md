# ADR-003: SQL Code Generation and Data Access Pattern

**Date**: 2025-04-30

**Status**: Accepted

## Context

We need a robust, maintainable, and type-safe way to interact with our PostgreSQL database (managed by Supabase) from our Go application. The approach should handle mapping database rows to Go structs, executing queries, and managing connections.

Key considerations include:
- Type safety between Go code and the database schema.
- Performance overhead of the data access layer.
- Avoiding boilerplate code for CRUD operations.
- Testability of the application logic that depends on data access.
- Developer control over the exact SQL being executed.

## Decision

1.  **SQL Code Generation**: We will use `sqlc` ([https://github.com/sqlc-dev/sqlc](https://github.com/sqlc-dev/sqlc)) to generate type-safe Go code directly from SQL queries.
    -   SQL queries (`.sql` files containing `CREATE`, `SELECT`, `INSERT`, `UPDATE`, `DELETE` statements with special `sqlc` comments) will be stored in the `sql/queries/` directory.
    -   `sqlc` will generate Go code (structs matching table schemas, functions for each query, and a `Querier` interface) into the `internal/database/` package.
    -   A `task db:generate` command will be added to `Taskfile.yml` to run `sqlc generate`.

2.  **Data Access Pattern**: We will implement the **Repository Pattern** on top of the `sqlc`-generated code.
    -   Interfaces (e.g., `UserRepository`) will be defined in the `internal/repository/` package. These interfaces will define the data access methods required by the application's business logic (e.g., `GetUserByID`, `CreateUser`).
    -   Concrete implementations of these repository interfaces will be created, embedding the `sqlc`-generated `*Queries` struct (which implements the `Querier` interface) and the `*pgxpool.Pool` for database connections.
    -   Application logic (handlers, services) will depend on the repository *interfaces*, not the concrete implementations or the `sqlc`-generated code directly.

## Consequences

**Positive:**
- **Type Safety:** `sqlc` generates Go code that reflects the SQL query parameters and return types, catching mismatches at compile time.
- **Reduced Boilerplate:** Eliminates manual `sql.Rows` scanning and struct mapping.
- **Performance:** Minimal overhead compared to full ORMs, as it generates straightforward code using the standard `database/sql` or `pgx` interfaces.
- **SQL Control:** Developers write and control the exact SQL queries.
- **Testability:** The Repository Pattern allows business logic to be tested easily by mocking the repository interfaces, decoupling it from the actual database.
- **Clear Separation:** Enforces a clean separation between data access logic (repositories) and business logic.

**Negative:**
- **Requires Writing SQL:** Developers need to write SQL queries manually in `.sql` files.
- **Extra Layer:** The Repository Pattern adds a layer of interfaces and structs compared to directly using the `sqlc`-generated `Querier`.
- **Generation Step:** Requires running `sqlc generate` after changing SQL queries or schema.

## Alternatives Considered

- **Full ORM (e.g., GORM, Ent):** Rejected due to potential performance overhead, "magic" behavior obscuring underlying SQL, and sometimes complex APIs. We prefer more explicit SQL control.
- **Lightweight SQL Wrappers (e.g., sqlx):** Considered, but `sqlc` provides stronger compile-time type safety guarantees derived directly from the SQL queries.
- **Manual `database/sql` or `pgx`:** Rejected as it involves significant boilerplate for scanning rows and mapping to structs, which `sqlc` automates.
- **Direct use of `sqlc` Querier:** Considered, but using the Repository Pattern provides better decoupling and testability for application logic.

## References

- `sqlc` Documentation: [https://docs.sqlc.dev/](https://docs.sqlc.dev/)
- Repository Pattern discussions.
## Enforcement
<!-- added 2026-07-12, see ADR-033 (Enforcement Architecture) -->
- **Testable consequences:**
  - TC-1: No hand-written SQL string literals in `internal/handler`.
  - TC-2: Handlers do not import `pgx` or `database/sql` — data access goes through repository interfaces. (Importing `internal/database` for sqlc-generated param/row *types* is legitimate.)
- **Checks:**
  - TC-1, TC-2 → `adr003-no-sql-in-handlers` in `scripts/adrcheck` (status: **warn**)
- **Not machine-checkable:** Repository-interface granularity and naming judgment. sqlc-regeneration drift (`task db:generate` output vs committed code) has no wired check — recorded as a TODO in ADR-033.
- **Graduation log:** _(empty)_
