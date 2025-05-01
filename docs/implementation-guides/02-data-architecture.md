# Phase 1 — Data Architecture & Schema with Supabase

Lock the data schema before writing handlers.

## Key Implementation Steps

| Step | Task | Why It Matters |
|------|------|----------------|
| 1.01 | Inventory all data entities | Defines application boundaries |
| 1.02 | Design database schema | Schema drives application structure |
| 1.03 | Create index & constraint checklist | Ensures data integrity and performance |
| 1.04 | Set up Supabase project | Provides PostgreSQL database and auth |
| 1.05 | Configure type generation | Enables type safety from database to code |
| 1.06 | Create repository interfaces | Separates data access from business logic |
| 1.07 | Setup migration strategy | Enables reliable schema evolution |
| 1.08 | Configure Row Level Security | Controls data access at the database level |
| 1.09 | Create test fixtures | Enables consistent testing |

## Core Principles

- Use Supabase as primary database and authentication provider
- Generate Go types from SQL definitions (sqlc)
- Implement repository pattern for clean separation of concerns 
- Document all schema changes in a changelog
- Configure Row Level Security (RLS) for data protection
- Use appropriate connection pooling parameters
- Create meaningful test fixtures for database testing

## Common Pitfalls

- **N+1 query problems**: Use joins and proper pagination
- **Schema drift**: Always use migrations, never manual schema changes
- **Poor RLS policies**: Default to deny, explicitly allow access
- **Connection exhaustion**: Tune max open/idle connections appropriately
- **Poor transaction handling**: Use consistent error handling patterns
- **Missing indexes**: Ensure proper indexes for common query patterns
- **Inadequate constraints**: Use foreign keys, uniqueness constraints, and NOT NULL

## Supabase Integration

- Use Supabase's PostgreSQL database for data storage
- Configure appropriate RLS policies for data security (will be tested in Phase 6)
- Leverage PostgREST for optimized queries when possible
- Use Supabase Storage for file uploads when needed
- Implement appropriate indexing and constraints

## Index & Constraint Checklist

Common indexes to consider:
- Primary keys (automatically indexed)
- Foreign keys (often need manual indexing)
- Columns used in WHERE clauses
- Columns used in ORDER BY
- Columns used in JOIN conditions

Essential constraints:
- NOT NULL for required fields
- UNIQUE for values that should not duplicate
- CHECK constraints for value validation
- Foreign keys for referential integrity
- Exclusion constraints for complex uniqueness rules

## Exit Criteria

- Schema fully defined with versioned migrations
- Supabase project configured properly
- SQL queries defined for core operations
- Repository interfaces established
- Types generated correctly from schema
- Migration process documented and tested
- RLS policies designed (implementation tested in Phase 6)
- Test fixtures created for key entities
- Index and constraint strategy documented

