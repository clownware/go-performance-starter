-- Test/CI-only stub of Supabase's auth schema and roles.
--
-- Supabase provides the `auth` schema, `auth.uid()` (which reads the JWT `sub`
-- claim), and the anon/authenticated/service_role roles. Vanilla PostgreSQL —
-- the CI service container and local dev databases — does not, so the RLS
-- migrations fail (`schema "auth" does not exist`) and RLS can't be exercised.
--
-- Apply this BEFORE running migrations against a non-Supabase database. Do NOT
-- turn it into a migration: a migration would override Supabase's real
-- auth.uid()/roles in production.
--
-- RLS tests then do, on a transaction:
--   SET LOCAL ROLE authenticated;
--   SELECT set_config('request.jwt.claim.sub', '<users.auth_id>', true);
-- so auth.uid() resolves to that user and RLS is enforced (authenticated is a
-- non-superuser without BYPASSRLS).

CREATE SCHEMA IF NOT EXISTS auth;

-- Returns the current request's auth id, or NULL when unset (→ RLS denies).
-- Returns text because every call site casts auth.uid()::text and compares
-- against users.auth_id (varchar).
CREATE OR REPLACE FUNCTION auth.uid() RETURNS text
    LANGUAGE sql STABLE
    AS $$ SELECT nullif(current_setting('request.jwt.claim.sub', true), '') $$;

-- Supabase roles. service_role bypasses RLS (matches Supabase); anon and
-- authenticated are subject to it.
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'anon') THEN
        CREATE ROLE anon NOLOGIN;
    END IF;
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'authenticated') THEN
        CREATE ROLE authenticated NOLOGIN;
    END IF;
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'service_role') THEN
        CREATE ROLE service_role NOLOGIN BYPASSRLS;
    END IF;
END $$;

GRANT USAGE ON SCHEMA public, auth TO anon, authenticated, service_role;

-- Tables/sequences are created later by the migration runner (this connection's
-- user). Default privileges auto-grant them to the Supabase roles; RLS policies
-- then do the actual restriction.
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO anon, authenticated, service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO anon, authenticated, service_role;
