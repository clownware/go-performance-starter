-- name: GetOrganization :one
SELECT * FROM organizations
WHERE id = $1 LIMIT 1;

-- name: GetOrganizationBySlug :one
SELECT * FROM organizations
WHERE slug = $1 LIMIT 1;

-- name: ListOrganizations :many
SELECT * FROM organizations
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CreateOrganization :one
INSERT INTO organizations (
    name,
    slug,
    plan_type,
    billing_customer_id
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: UpdateOrganization :one
UPDATE organizations
SET 
    name = COALESCE($2, name),
    slug = COALESCE($3, slug),
    plan_type = COALESCE($4, plan_type),
    billing_customer_id = COALESCE($5, billing_customer_id),
    is_active = COALESCE($6, is_active),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteOrganization :exec
UPDATE organizations
SET is_active = false, updated_at = NOW()
WHERE id = $1;

-- name: PermanentDeleteOrganization :exec
DELETE FROM organizations
WHERE id = $1;
