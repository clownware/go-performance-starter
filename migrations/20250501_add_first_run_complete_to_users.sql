-- Add first_run_complete to users table
ALTER TABLE users ADD COLUMN first_run_complete BOOLEAN NOT NULL DEFAULT FALSE;
