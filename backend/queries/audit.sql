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
  AND (sqlc.narg('q')::text IS NULL
       OR a.action ILIKE '%' || sqlc.narg('q')::text || '%'
       OR a.resource_type ILIKE '%' || sqlc.narg('q')::text || '%'
       OR a.resource_id::text ILIKE '%' || sqlc.narg('q')::text || '%'
       OR au.email ILIKE '%' || sqlc.narg('q')::text || '%')
  AND (sqlc.narg('action')::text IS NULL OR a.action ILIKE sqlc.narg('action')::text || '%')
  AND (sqlc.narg('resource_type')::text IS NULL OR a.resource_type = sqlc.narg('resource_type')::text)
  AND (sqlc.narg('admin_email')::text IS NULL OR au.email ILIKE '%' || sqlc.narg('admin_email')::text || '%')
  AND (sqlc.narg('from_ts')::timestamptz IS NULL OR a.created_at >= sqlc.narg('from_ts')::timestamptz)
  AND (sqlc.narg('to_ts')::timestamptz IS NULL OR a.created_at <= sqlc.narg('to_ts')::timestamptz)
ORDER BY a.created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountAuditLogsByOrganization :one
SELECT count(*)
FROM admin_audit_log a
LEFT JOIN admin_users au ON a.admin_user_id = au.id
WHERE a.organization_id = sqlc.arg('organization_id')::uuid
  AND (sqlc.narg('q')::text IS NULL
       OR a.action ILIKE '%' || sqlc.narg('q')::text || '%'
       OR a.resource_type ILIKE '%' || sqlc.narg('q')::text || '%'
       OR a.resource_id::text ILIKE '%' || sqlc.narg('q')::text || '%'
       OR au.email ILIKE '%' || sqlc.narg('q')::text || '%')
  AND (sqlc.narg('action')::text IS NULL OR a.action ILIKE sqlc.narg('action')::text || '%')
  AND (sqlc.narg('resource_type')::text IS NULL OR a.resource_type = sqlc.narg('resource_type')::text)
  AND (sqlc.narg('admin_email')::text IS NULL OR au.email ILIKE '%' || sqlc.narg('admin_email')::text || '%')
  AND (sqlc.narg('from_ts')::timestamptz IS NULL OR a.created_at >= sqlc.narg('from_ts')::timestamptz)
  AND (sqlc.narg('to_ts')::timestamptz IS NULL OR a.created_at <= sqlc.narg('to_ts')::timestamptz);
