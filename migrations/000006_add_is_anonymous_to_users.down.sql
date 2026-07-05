DROP INDEX IF EXISTS idx_users_anonymous_created;
ALTER TABLE users DROP COLUMN IF EXISTS is_anonymous;
