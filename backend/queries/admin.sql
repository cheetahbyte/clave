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

-- name: InsertAdminEmailCode :exec
INSERT INTO admin_email_codes (admin_user_id, code_hash, expires_at)
VALUES (sqlc.arg(admin_user_id), sqlc.arg(code_hash), sqlc.arg(expires_at));

-- name: GetLatestAdminEmailCode :one
SELECT * FROM admin_email_codes
WHERE admin_user_id = sqlc.arg(admin_user_id)
  AND used_at IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: MarkAdminEmailCodeUsed :exec
UPDATE admin_email_codes
SET used_at = now()
WHERE id = sqlc.arg(id);

-- name: IncrementAdminEmailCodeAttempts :one
UPDATE admin_email_codes
SET attempts = attempts + 1
WHERE id = sqlc.arg(id)
RETURNING attempts;

-- name: InvalidateAdminEmailCodes :exec
UPDATE admin_email_codes
SET used_at = now()
WHERE admin_user_id = sqlc.arg(admin_user_id)
  AND used_at IS NULL;

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

-- Retention maintenance queries are centralized here so one generated query
-- adapter can run the database lifecycle worker.

-- name: DeleteStaleSelfServiceTokens :execrows
DELETE FROM self_service_tokens
WHERE expires_at < now() - INTERVAL '1 day'
   OR used_at < now() - INTERVAL '1 day';

-- name: DeleteStaleOrganizationInvites :execrows
DELETE FROM organization_invites
WHERE (accepted_at IS NOT NULL AND accepted_at < now() - INTERVAL '30 days')
   OR (accepted_at IS NULL AND expires_at < now() - INTERVAL '30 days');

-- name: DeleteStaleAdminEmailCodes :execrows
DELETE FROM admin_email_codes
WHERE expires_at < now() - INTERVAL '1 day'
   OR used_at < now() - INTERVAL '1 day';

-- name: ScrubStaleAuditSecurityMetadata :execrows
UPDATE admin_audit_log
SET ip = NULL,
    user_agent = NULL
WHERE created_at < now() - make_interval(days => sqlc.arg('retention_days')::int)
  AND (ip IS NOT NULL OR user_agent IS NOT NULL);

-- name: DeleteStaleAuditLogs :execrows
DELETE FROM admin_audit_log
WHERE created_at < now() - make_interval(days => sqlc.arg('retention_days')::int);

-- name: DeleteStaleUpdateChecks :execrows
DELETE FROM update_checks
WHERE created_at < now() - make_interval(days => sqlc.arg('retention_days')::int);
