-- name: ListAuditLogsByOrganization :many
SELECT
    a.id,
    a.action,
    a.resource_type,
    a.resource_id,
    a.metadata,
    a.ip,
    a.user_agent,
    a.created_at,
    au.email AS admin_email
FROM admin_audit_log a
LEFT JOIN admin_users au ON a.admin_user_id = au.id
WHERE a.organization_id = sqlc.arg('organization_id')::uuid
ORDER BY a.created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountAuditLogsByOrganization :one
SELECT count(*) FROM admin_audit_log
WHERE organization_id = sqlc.arg('organization_id')::uuid;
