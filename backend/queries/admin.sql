-- name: GetAdminByEmail :one
SELECT * FROM admin_users WHERE email = $1;

-- name: GetAdminById :one
SELECT * FROM admin_users WHERE id = $1;

-- name: CreateAdmin :one
INSERT INTO admin_users (email, password_hash, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateLastLogin :exec
UPDATE admin_users
SET last_login_at = now(), updated_at = now()
WHERE id = $1;

-- name: EnableTOTP :exec
UPDATE admin_users
SET totp_enabled = true,
    totp_secret_enc = sqlc.arg(secret_enc),
    totp_secret_nonce = sqlc.arg(secret_nonce),
    updated_at = now()
WHERE id = sqlc.arg(id);

-- name: DisableTOTP :exec
UPDATE admin_users
SET totp_enabled = false,
    totp_secret_enc = NULL,
    totp_secret_nonce = NULL,
    updated_at = now()
WHERE id = $1;

-- name: InsertRecoveryCode :exec
INSERT INTO admin_recovery_codes (admin_user_id, code_hash)
VALUES (sqlc.arg(admin_user_id), sqlc.arg(code_hash));

-- name: ConsumeRecoveryCode :one
UPDATE admin_recovery_codes
SET used_at = now()
WHERE admin_user_id = sqlc.arg(admin_user_id)
  AND code_hash = sqlc.arg(code_hash)
  AND used_at IS NULL
RETURNING id;

-- name: InvalidateRecoveryCodes :exec
DELETE FROM admin_recovery_codes
WHERE admin_user_id = sqlc.arg(admin_user_id);

-- name: InsertAuditLog :exec
INSERT INTO admin_audit_log (admin_user_id, organization_id, action, resource_type, resource_id, metadata, ip, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetAdminOrganizations :many
SELECT o.id, o.name, o.slug, o.created_at, o.updated_at, om.role as member_role
FROM organizations o
JOIN organization_members om ON o.id = om.organization_id
WHERE om.admin_user_id = $1;

-- name: GetOrganizationById :one
SELECT * FROM organizations WHERE id = $1;

-- name: CreateOrganization :one
INSERT INTO organizations (name, slug) VALUES ($1, $2) RETURNING *;

-- name: CreateOrganizationMember :exec
INSERT INTO organization_members (organization_id, admin_user_id, role) VALUES ($1, $2, $3);

-- name: GetOrganizationMembership :one
SELECT organization_id, admin_user_id, role, created_at
FROM organization_members
WHERE organization_id = $1 AND admin_user_id = $2;

-- name: GetOrganizationBySlug :one
SELECT * FROM organizations WHERE slug = $1;
