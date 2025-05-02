-- Add first_run_complete to all relevant queries for onboarding flow

-- Create user (include first_run_complete, default false)
INSERT INTO users (
    email,
    name,
    avatar_url,
    auth_id,
    first_run_complete
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING id, email, name, avatar_url, auth_id, is_active, last_login_at, created_at, updated_at, first_run_complete;

-- Update user (set first_run_complete)
UPDATE users
SET first_run_complete = $1, updated_at = NOW()
WHERE id = $2;

-- Get user by id
SELECT id, email, name, avatar_url, auth_id, is_active, last_login_at, created_at, updated_at, first_run_complete
FROM users
WHERE id = $1 LIMIT 1;

-- Get user by email
SELECT id, email, name, avatar_url, auth_id, is_active, last_login_at, created_at, updated_at, first_run_complete
FROM users
WHERE email = $1 LIMIT 1;

-- Get user by auth_id
SELECT id, email, name, avatar_url, auth_id, is_active, last_login_at, created_at, updated_at, first_run_complete
FROM users
WHERE auth_id = $1 LIMIT 1;
