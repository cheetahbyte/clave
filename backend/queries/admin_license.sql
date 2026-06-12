-- name: GetAdminOverviewStatsByOrganization :one
SELECT
    (SELECT count(*) FROM licenses WHERE organization_id = sqlc.arg('organization_id')::uuid) AS total_licenses,
    (SELECT count(*) FROM licenses WHERE organization_id = sqlc.arg('organization_id')::uuid AND is_active = true AND (expires_at IS NULL OR expires_at > now())) AS active_licenses,
    (SELECT count(*) FROM licenses WHERE organization_id = sqlc.arg('organization_id')::uuid AND expires_at IS NOT NULL AND expires_at <= now()) AS expired_licenses,
    (SELECT count(*) FROM products WHERE organization_id = sqlc.arg('organization_id')::uuid) AS total_products,
    (SELECT count(*) FROM activations a JOIN licenses l ON a.license_id = l.id WHERE l.organization_id = sqlc.arg('organization_id')::uuid) AS total_activations;

-- name: ListAdminRecentLicensesByOrganization :many
SELECT
    lt.id,
    lt.customer_email,
    lt.is_active,
    lt.max_activations,
    lt.created_at,
    lt.expires_at,
    lt.is_trial,
    p.name AS product_name,
    (SELECT count(*) FROM activations WHERE license_id = lt.id) AS activation_count
FROM licenses lt
JOIN products p ON lt.product_id = p.id
WHERE lt.organization_id = sqlc.arg('organization_id')::uuid
ORDER BY lt.created_at DESC
LIMIT sqlc.arg('limit');

-- name: CountAdminLicensesByOrganization :one
SELECT count(*)
FROM licenses lt
JOIN products p ON lt.product_id = p.id
WHERE lt.organization_id = sqlc.arg('organization_id')::uuid
    AND (sqlc.arg('q')::text = '' OR lt.customer_email ILIKE '%' || sqlc.arg('q')::text || '%' OR p.name ILIKE '%' || sqlc.arg('q')::text || '%')
    AND (
        sqlc.arg('status')::text = 'all'
        OR (sqlc.arg('status')::text = 'active' AND lt.is_active = true AND (lt.expires_at IS NULL OR lt.expires_at > now()))
        OR (sqlc.arg('status')::text = 'inactive' AND lt.is_active = false)
        OR (sqlc.arg('status')::text = 'expired' AND lt.expires_at IS NOT NULL AND lt.expires_at <= now())
    )
    AND (sqlc.arg('product_id')::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR lt.product_id = sqlc.arg('product_id')::uuid);

-- name: ListAdminLicensesByOrganization :many
SELECT
    lt.id,
    lt.customer_email,
    lt.is_active,
    lt.max_activations,
    lt.created_at,
    lt.expires_at,
    lt.is_trial,
    p.name AS product_name,
    (SELECT count(*) FROM activations WHERE license_id = lt.id) AS activation_count
FROM licenses lt
JOIN products p ON lt.product_id = p.id
WHERE lt.organization_id = sqlc.arg('organization_id')::uuid
    AND (sqlc.arg('q')::text = '' OR lt.customer_email ILIKE '%' || sqlc.arg('q')::text || '%' OR p.name ILIKE '%' || sqlc.arg('q')::text || '%')
    AND (
        sqlc.arg('status')::text = 'all'
        OR (sqlc.arg('status')::text = 'active' AND lt.is_active = true AND (lt.expires_at IS NULL OR lt.expires_at > now()))
        OR (sqlc.arg('status')::text = 'inactive' AND lt.is_active = false)
        OR (sqlc.arg('status')::text = 'expired' AND lt.expires_at IS NOT NULL AND lt.expires_at <= now())
    )
    AND (sqlc.arg('product_id')::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR lt.product_id = sqlc.arg('product_id')::uuid)
ORDER BY lt.created_at DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: GetAdminLicenseDetailByOrganization :one
SELECT
    lt.id,
    lt.customer_email,
    lt.is_active,
    lt.max_activations,
    lt.created_at,
    lt.expires_at,
    lt.is_trial,
    lt.features,
    p.name AS product_name,
    p.id AS product_id,
    (SELECT count(*) FROM activations WHERE license_id = lt.id) AS activation_count
FROM licenses lt
JOIN products p ON lt.product_id = p.id
WHERE lt.id = $1 AND lt.organization_id = sqlc.arg('organization_id')::uuid;

-- name: ListAdminLicenseActivationsByOrganization :many
SELECT
    a.id,
    a.license_id,
    a.created_at,
    a.checked_in_at,
    d.id AS device_id,
    d.hwid_hash,
    d.hostname,
    d.created_at AS device_created_at
FROM activations a
JOIN devices d ON a.device_id = d.id
JOIN licenses l ON a.license_id = l.id
WHERE a.license_id = $1 AND l.organization_id = sqlc.arg('organization_id')::uuid
ORDER BY a.created_at DESC;
