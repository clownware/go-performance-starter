-- name: GetUser :one
SELECT id, email, name, avatar_url, auth_id, is_active, last_login_at, created_at, updated_at, first_run_complete, is_anonymous
FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT id, email, name, avatar_url, auth_id, is_active, last_login_at, created_at, updated_at, first_run_complete, is_anonymous
FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserByAuthID :one
SELECT id, email, name, avatar_url, auth_id, is_active, last_login_at, created_at, updated_at, first_run_complete, is_anonymous
FROM users
WHERE auth_id = $1 LIMIT 1;

-- name: ListUsers :many
SELECT id, email, name, avatar_url, auth_id, is_active, last_login_at, created_at, updated_at, first_run_complete, is_anonymous
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CreateUser :one
INSERT INTO users (
    email,
    name,
    avatar_url,
    auth_id,
    first_run_complete,
    is_anonymous
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING id, email, name, avatar_url, auth_id, is_active, last_login_at, created_at, updated_at, first_run_complete, is_anonymous;

-- name: UpdateUser :one
UPDATE users
SET
    email = COALESCE($2, email),
    name = COALESCE($3, name),
    avatar_url = COALESCE($4, avatar_url),
    auth_id = COALESCE($5, auth_id),
    is_active = COALESCE($6, is_active),
    last_login_at = COALESCE($7, last_login_at),
    updated_at = NOW()
WHERE id = $1
RETURNING id, email, name, avatar_url, auth_id, is_active, last_login_at, created_at, updated_at, first_run_complete, is_anonymous;

-- name: DeleteUser :exec
UPDATE users
SET is_active = false, updated_at = NOW()
WHERE id = $1;

-- name: PermanentDeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: UpdateUserName :one
-- Profile self-service rename (#70). Deliberately narrow — the generic
-- UpdateUser's COALESCE params make a name-only change thread every other
-- column through untouched.
UPDATE users
SET name = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, email, name, avatar_url, auth_id, is_active, last_login_at, created_at, updated_at, first_run_complete, is_anonymous;

-- name: SetUserIsAnonymous :exec
-- Guest → registered upgrade (#68): flipping to false exempts the row from
-- the anonymous-user reaper.
UPDATE users
SET is_anonymous = $2, updated_at = NOW()
WHERE id = $1;

-- name: SetUserFirstRunComplete :exec
UPDATE users
SET first_run_complete = $2, updated_at = NOW()
WHERE id = $1;

-- name: DeleteExpiredAnonymousUsers :many
-- Reaps anonymous guest accounts older than the TTL (ADR-024). Upgraded
-- accounts have is_anonymous=false and are exempt. Flashcards and quiz
-- attempts cascade via their user_id foreign keys.
DELETE FROM users
WHERE is_anonymous AND created_at < $1
RETURNING id, auth_id;
