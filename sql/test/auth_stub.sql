-- Test/CI-only stub of Supabase's auth schema.
--
-- Supabase provides the `auth` schema and `auth.uid()` (which reads the JWT
-- `sub` claim). Vanilla PostgreSQL — the CI service container and local dev
-- databases — does not, so the RLS migrations (000002, 000003) fail with
-- `schema "auth" does not exist`.
--
-- Apply this BEFORE running migrations against a non-Supabase database. Do NOT
-- turn it into a migration: a migration would override Supabase's real
-- auth.uid() in production and break RLS there.
--
-- Tests set the current user with:
--   SET LOCAL request.jwt.claim.sub = '<users.auth_id>';
-- and reset with RESET request.jwt.claim.sub; (or a fresh connection).

CREATE SCHEMA IF NOT EXISTS auth;

-- Returns the current request's auth id, or NULL when unset (→ RLS denies).
-- Returns text because every call site casts auth.uid()::text and compares
-- against users.auth_id (varchar).
CREATE OR REPLACE FUNCTION auth.uid() RETURNS text
    LANGUAGE sql STABLE
    AS $$ SELECT nullif(current_setting('request.jwt.claim.sub', true), '') $$;
