-- name: GetOrganizationMember :one
SELECT * FROM organization_members
WHERE id = $1 LIMIT 1;

-- name: GetOrganizationMemberByUserAndOrg :one
SELECT * FROM organization_members
WHERE user_id = $1 AND organization_id = $2 LIMIT 1;

-- name: ListOrganizationMembers :many
SELECT * FROM organization_members
WHERE organization_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListUserOrganizations :many
SELECT o.* FROM organizations o
JOIN organization_members om ON o.id = om.organization_id
WHERE om.user_id = $1 AND o.is_active = true
ORDER BY o.name
LIMIT $2 OFFSET $3;

-- name: GetUserPrimaryOrganization :one
SELECT o.* FROM organizations o
JOIN organization_members om ON o.id = om.organization_id
WHERE om.user_id = $1 AND om.is_primary_organization = true AND o.is_active = true
LIMIT 1;

-- name: CreateOrganizationMember :one
INSERT INTO organization_members (
    organization_id,
    user_id,
    role,
    is_primary_organization
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: UpdateOrganizationMember :one
UPDATE organization_members
SET 
    role = COALESCE($3, role),
    is_primary_organization = COALESCE($4, is_primary_organization),
    updated_at = NOW()
WHERE organization_id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteOrganizationMember :exec
DELETE FROM organization_members
WHERE organization_id = $1 AND user_id = $2;

-- name: ListOrganizationMembershipsForUser :many
SELECT om.*, o.name as organization_name
FROM organization_members om
JOIN organizations o ON om.organization_id = o.id
WHERE om.user_id = $1
ORDER BY o.name;

-- name: SetPrimaryOrganizationStep1 :exec
-- Step 1: Set the target membership as primary
UPDATE organization_members
SET is_primary_organization = true,
    updated_at = NOW()
WHERE organization_id = $1 AND user_id = $2;

-- name: SetPrimaryOrganizationStep2 :exec
-- Step 2: Set all other memberships for the user as non-primary
UPDATE organization_members
SET is_primary_organization = false,
    updated_at = NOW()
WHERE user_id = $1 -- The user_id from step 1
  AND organization_id != $2; -- The organization_id from step 1

-- name: CountOrganizationMembers :one
SELECT COUNT(*) FROM organization_members
WHERE organization_id = $1;
