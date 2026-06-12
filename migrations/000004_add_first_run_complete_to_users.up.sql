-- Add first_run_complete to users table.
-- IF NOT EXISTS keeps this idempotent: in environments where the column was
-- already added out-of-band (it predates this versioned migration), applying
-- this is a safe no-op.
ALTER TABLE users ADD COLUMN IF NOT EXISTS first_run_complete BOOLEAN NOT NULL DEFAULT FALSE;
