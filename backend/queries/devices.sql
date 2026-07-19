-- name: CreateDevice :one
insert into devices(license_id, hwid_hash, hostname) values($1, $2, $3) returning *;

-- name: GetDeviceByLicenseAndHwidHash :one
select * from devices where license_id = $1 and hwid_hash = $2;

-- name: ListAdminDevicesByOrganization :many
SELECT
    d.id AS device_id,
    d.hostname,
    d.created_at AS device_created_at,
    a.id AS activation_id,
    a.created_at AS activated_at,
    a.checked_in_at,
    l.id AS license_id,
    l.customer_email,
    l.customer_name,
    l.is_active AS license_active,
    p.id AS product_id,
    p.name AS product_name
FROM devices d
JOIN activations a ON a.device_id = d.id
JOIN licenses l ON l.id = d.license_id
JOIN products p ON p.id = l.product_id
WHERE l.organization_id = sqlc.arg('organization_id')::uuid
  AND a.deactivated_at IS NULL
  AND (sqlc.narg('q')::text IS NULL
       OR d.hostname ILIKE '%' || sqlc.narg('q')::text || '%'
       OR l.customer_email ILIKE '%' || sqlc.narg('q')::text || '%'
       OR l.customer_name ILIKE '%' || sqlc.narg('q')::text || '%'
       OR p.name ILIKE '%' || sqlc.narg('q')::text || '%'
       OR d.id::text ILIKE '%' || sqlc.narg('q')::text || '%'
       OR l.id::text ILIKE '%' || sqlc.narg('q')::text || '%')
  AND (sqlc.narg('product_id')::uuid IS NULL OR p.id = sqlc.narg('product_id')::uuid)
  AND (
      sqlc.narg('status')::text IS NULL
      OR sqlc.narg('status')::text = 'all'
      OR (sqlc.narg('status')::text = 'seen' AND a.checked_in_at IS NOT NULL)
      OR (sqlc.narg('status')::text = 'never_seen' AND a.checked_in_at IS NULL)
  )
ORDER BY d.created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountAdminDevicesByOrganization :one
SELECT count(*)
FROM devices d
JOIN activations a ON a.device_id = d.id
JOIN licenses l ON l.id = d.license_id
JOIN products p ON p.id = l.product_id
WHERE l.organization_id = sqlc.arg('organization_id')::uuid
  AND a.deactivated_at IS NULL
  AND (sqlc.narg('q')::text IS NULL
       OR d.hostname ILIKE '%' || sqlc.narg('q')::text || '%'
       OR l.customer_email ILIKE '%' || sqlc.narg('q')::text || '%'
       OR l.customer_name ILIKE '%' || sqlc.narg('q')::text || '%'
       OR p.name ILIKE '%' || sqlc.narg('q')::text || '%'
       OR d.id::text ILIKE '%' || sqlc.narg('q')::text || '%'
       OR l.id::text ILIKE '%' || sqlc.narg('q')::text || '%')
  AND (sqlc.narg('product_id')::uuid IS NULL OR p.id = sqlc.narg('product_id')::uuid)
  AND (
      sqlc.narg('status')::text IS NULL
      OR sqlc.narg('status')::text = 'all'
      OR (sqlc.narg('status')::text = 'seen' AND a.checked_in_at IS NOT NULL)
      OR (sqlc.narg('status')::text = 'never_seen' AND a.checked_in_at IS NULL)
  );

-- name: DeactivateAdminDeviceByOrganization :one
UPDATE activations a
SET deactivated_at = now(),
    deactivation_reason = sqlc.arg('reason')
FROM devices d
JOIN licenses l ON l.id = d.license_id
WHERE a.device_id = d.id
  AND a.license_id = l.id
  AND d.id = sqlc.arg('device_id')
  AND l.organization_id = sqlc.arg('organization_id')::uuid
  AND a.deactivated_at IS NULL
RETURNING d.id;
