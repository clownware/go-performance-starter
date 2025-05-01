-- name: GetUser :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserByAuthID :one
SELECT * FROM users
WHERE auth_id = $1 LIMIT 1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CreateUser :one
INSERT INTO users (
    email,
    name,
    avatar_url,
    auth_id
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

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
RETURNING *;

-- name: DeleteUser :exec
UPDATE users
SET is_active = false, updated_at = NOW()
WHERE id = $1;

-- name: PermanentDeleteUser :exec
DELETE FROM users
WHERE id = $1;
