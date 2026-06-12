-- name: ListOrganizationMembers :many
SELECT au.id, au.email, om.role, om.created_at, au.last_login_at
FROM organization_members om
JOIN admin_users au ON om.admin_user_id = au.id
WHERE om.organization_id = $1
ORDER BY om.created_at;

-- name: DeleteOrganizationMember :exec
DELETE FROM organization_members
WHERE organization_id = $1 AND admin_user_id = $2;

-- name: CreateInvite :one
INSERT INTO organization_invites (organization_id, email, role, token_hash, invited_by, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetInviteByTokenHash :one
SELECT
    i.id,
    i.organization_id,
    i.email,
    i.role,
    i.invited_by,
    i.expires_at,
    i.accepted_at,
    i.created_at,
    o.name AS organization_name,
    o.slug AS organization_slug
FROM organization_invites i
JOIN organizations o ON i.organization_id = o.id
WHERE i.token_hash = $1;

-- name: AcceptInvite :one
UPDATE organization_invites
SET accepted_at = now()
WHERE id = $1 AND accepted_at IS NULL
RETURNING *;

-- name: ListPendingInvites :many
SELECT id, email, role, invited_by, expires_at, created_at
FROM organization_invites
WHERE organization_id = $1 AND accepted_at IS NULL
ORDER BY created_at DESC;

-- name: DeletePendingInvitesByEmail :exec
DELETE FROM organization_invites
WHERE organization_id = $1 AND lower(email) = lower($2) AND accepted_at IS NULL;

-- name: DeleteInvite :exec
DELETE FROM organization_invites
WHERE id = $1 AND organization_id = $2;
