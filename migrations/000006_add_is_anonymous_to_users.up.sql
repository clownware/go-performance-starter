-- Anonymous guest identities (ADR-024): guests are real users rows whose
-- auth identity is a Supabase anonymous sign-in. The flag mirrors the JWT's
-- is_anonymous claim at provisioning time and flips to false on the
-- data-preserving upgrade. The TTL reaper deletes anonymous users whose
-- account age exceeds the guest TTL (upgrading exempts them).

ALTER TABLE users ADD COLUMN is_anonymous boolean NOT NULL DEFAULT false;

-- The reaper scans by age over anonymous rows only.
CREATE INDEX idx_users_anonymous_created ON users (created_at) WHERE is_anonymous;
