-- name: GetAdminOverviewStatsByOrganization :one
-- When product_id is provided, license/activation/trial counts are scoped to
-- that product; total_products always reflects the whole organization.
WITH license_stats AS (
    SELECT
        count(*) AS total_licenses,
        count(*) FILTER (WHERE is_active = true AND (expires_at IS NULL OR expires_at > now())) AS active_licenses,
        count(*) FILTER (WHERE expires_at IS NOT NULL AND expires_at <= now()) AS expired_licenses,
        count(*) FILTER (WHERE is_trial = true) AS total_trials,
        count(*) FILTER (WHERE is_trial = true AND is_active = true AND (expires_at IS NULL OR expires_at > now())) AS active_trials
    FROM licenses
    WHERE organization_id = sqlc.arg('organization_id')::uuid
      AND (sqlc.narg('product_id')::uuid IS NULL OR product_id = sqlc.narg('product_id')::uuid)
), activation_stats AS (
    SELECT count(*) AS total_activations
    FROM activations a
    JOIN licenses l ON a.license_id = l.id
    WHERE l.organization_id = sqlc.arg('organization_id')::uuid
      AND a.deactivated_at IS NULL
      AND (sqlc.narg('product_id')::uuid IS NULL OR l.product_id = sqlc.narg('product_id')::uuid)
)
SELECT
    ls.total_licenses,
    ls.active_licenses,
    ls.expired_licenses,
    (SELECT count(*) FROM products WHERE organization_id = sqlc.arg('organization_id')::uuid) AS total_products,
    acts.total_activations,
    ls.total_trials,
    ls.active_trials
FROM license_stats ls
CROSS JOIN activation_stats acts;

-- name: ListAdminRecentLicensesByOrganization :many
SELECT
    lt.id,
    lt.customer_email,
    lt.customer_name,
    lt.is_active,
    lt.max_activations,
    lt.created_at,
    lt.expires_at,
    lt.is_trial,
    lt.features,
    p.name AS product_name,
    (SELECT count(*) FROM activations WHERE license_id = lt.id AND deactivated_at IS NULL) AS activation_count
FROM licenses lt
JOIN products p ON lt.product_id = p.id
WHERE lt.organization_id = sqlc.arg('organization_id')::uuid
ORDER BY lt.created_at DESC
LIMIT sqlc.arg('limit');

-- name: GetAdminLicensesByEmail :many
SELECT
    lt.id,
    lt.customer_email,
    lt.customer_name,
    lt.is_active,
    lt.max_activations,
    lt.created_at,
    lt.expires_at,
    lt.is_trial,
    lt.features,
    p.name AS product_name,
    p.id AS product_id,
    (SELECT count(*) FROM activations WHERE license_id = lt.id AND deactivated_at IS NULL) AS activation_count
FROM licenses lt
JOIN products p ON lt.product_id = p.id
WHERE lt.organization_id = sqlc.arg('organization_id')::uuid
  AND lower(lt.customer_email) = lower(sqlc.arg('customer_email'))
ORDER BY lt.created_at DESC;

-- name: GetAdminLicenseByDigest :one
SELECT
    lt.id,
    lt.customer_email,
    lt.customer_name,
    lt.is_active,
    lt.max_activations,
    lt.created_at,
    lt.expires_at,
    lt.is_trial,
    lt.features,
    p.name AS product_name,
    p.id AS product_id,
    (SELECT count(*) FROM activations WHERE license_id = lt.id AND deactivated_at IS NULL) AS activation_count
FROM licenses lt
JOIN products p ON lt.product_id = p.id
WHERE lt.lookup_digest = sqlc.arg('lookup_digest')
  AND lt.organization_id = sqlc.arg('organization_id')::uuid;

-- name: RevokeAdminLicense :one
UPDATE licenses SET is_active = false
WHERE id = sqlc.arg('id')::uuid
  AND organization_id = sqlc.arg('organization_id')::uuid
  AND is_active = true
RETURNING id;

-- name: RevokeAdminLicenseByDigest :one
UPDATE licenses SET is_active = false
WHERE lookup_digest = sqlc.arg('lookup_digest')
  AND organization_id = sqlc.arg('organization_id')::uuid
  AND is_active = true
RETURNING id;

-- name: CountAdminLicensesByOrganization :one
SELECT count(*)
FROM licenses lt
JOIN products p ON lt.product_id = p.id
WHERE lt.organization_id = sqlc.arg('organization_id')::uuid
    AND (sqlc.arg('q')::text = '' OR lt.customer_email ILIKE '%' || sqlc.arg('q')::text || '%' OR lt.customer_name ILIKE '%' || sqlc.arg('q')::text || '%' OR p.name ILIKE '%' || sqlc.arg('q')::text || '%')
    AND (
        sqlc.arg('status')::text = 'all'
        OR (sqlc.arg('status')::text = 'active' AND lt.is_active = true AND (lt.expires_at IS NULL OR lt.expires_at > now()))
        OR (sqlc.arg('status')::text = 'inactive' AND lt.is_active = false)
        OR (sqlc.arg('status')::text = 'expired' AND lt.expires_at IS NOT NULL AND lt.expires_at <= now())
    )
    AND (sqlc.arg('product_id')::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR lt.product_id = sqlc.arg('product_id')::uuid)
    AND (
        sqlc.arg('type')::text = 'all'
        OR (sqlc.arg('type')::text = 'trial' AND lt.is_trial = true)
        OR (sqlc.arg('type')::text = 'standard' AND lt.is_trial = false)
    );

-- name: ListAdminLicensesByOrganization :many
SELECT
    lt.id,
    lt.customer_email,
    lt.customer_name,
    lt.is_active,
    lt.max_activations,
    lt.created_at,
    lt.expires_at,
    lt.is_trial,
    p.name AS product_name,
    (SELECT count(*) FROM activations WHERE license_id = lt.id AND deactivated_at IS NULL) AS activation_count
FROM licenses lt
JOIN products p ON lt.product_id = p.id
WHERE lt.organization_id = sqlc.arg('organization_id')::uuid
    AND (sqlc.arg('q')::text = '' OR lt.customer_email ILIKE '%' || sqlc.arg('q')::text || '%' OR lt.customer_name ILIKE '%' || sqlc.arg('q')::text || '%' OR p.name ILIKE '%' || sqlc.arg('q')::text || '%')
    AND (
        sqlc.arg('status')::text = 'all'
        OR (sqlc.arg('status')::text = 'active' AND lt.is_active = true AND (lt.expires_at IS NULL OR lt.expires_at > now()))
        OR (sqlc.arg('status')::text = 'inactive' AND lt.is_active = false)
        OR (sqlc.arg('status')::text = 'expired' AND lt.expires_at IS NOT NULL AND lt.expires_at <= now())
    )
    AND (sqlc.arg('product_id')::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR lt.product_id = sqlc.arg('product_id')::uuid)
    AND (
        sqlc.arg('type')::text = 'all'
        OR (sqlc.arg('type')::text = 'trial' AND lt.is_trial = true)
        OR (sqlc.arg('type')::text = 'standard' AND lt.is_trial = false)
    )
ORDER BY lt.created_at DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: GetAdminLicenseDetailByOrganization :one
SELECT
    lt.id,
    lt.customer_email,
    lt.customer_name,
    lt.is_active,
    lt.max_activations,
    lt.created_at,
    lt.expires_at,
    lt.is_trial,
    lt.features,
    p.name AS product_name,
    p.id AS product_id,
    (SELECT count(*) FROM activations WHERE license_id = lt.id AND deactivated_at IS NULL) AS activation_count
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
WHERE a.license_id = $1 AND l.organization_id = sqlc.arg('organization_id')::uuid AND a.deactivated_at IS NULL
ORDER BY a.created_at DESC;

-- name: GetAdminTimeseriesByOrganization :many
WITH days AS (
    SELECT d::date AS day FROM generate_series(
        (now() - make_interval(days => sqlc.arg('days')::int - 1))::date,
        now()::date,
        interval '1 day'
    ) d
),
license_counts AS (
    SELECT
        l.created_at::date AS day,
        COUNT(*) FILTER (WHERE NOT l.is_trial) AS licenses,
        COUNT(*) FILTER (WHERE l.is_trial) AS trials
    FROM licenses l
    WHERE l.organization_id = sqlc.arg('organization_id')::uuid
      AND (sqlc.narg('product_id')::uuid IS NULL OR l.product_id = sqlc.narg('product_id')::uuid)
      AND l.created_at >= (now() - make_interval(days => sqlc.arg('days')::int))
    GROUP BY l.created_at::date
),
activation_counts AS (
    SELECT
        a.created_at::date AS day,
        COUNT(*) AS activations
    FROM activations a
    JOIN licenses l ON a.license_id = l.id
    WHERE l.organization_id = sqlc.arg('organization_id')::uuid
      AND (sqlc.narg('product_id')::uuid IS NULL OR l.product_id = sqlc.narg('product_id')::uuid)
      AND a.created_at >= (now() - make_interval(days => sqlc.arg('days')::int))
    GROUP BY a.created_at::date
)
SELECT
    d.day,
    COALESCE(lc.licenses, 0)::bigint AS licenses,
    COALESCE(lc.trials, 0)::bigint AS trials,
    COALESCE(ac.activations, 0)::bigint AS activations
FROM days d
LEFT JOIN license_counts lc ON d.day = lc.day
LEFT JOIN activation_counts ac ON d.day = ac.day
ORDER BY d.day ASC;

-- name: ListAdminTrialsByOrganization :many
SELECT
    lt.id,
    lt.customer_email,
    lt.customer_name,
    lt.is_active,
    lt.max_activations,
    lt.created_at,
    lt.expires_at,
    lt.is_trial,
    p.name AS product_name,
    (SELECT count(*) FROM activations WHERE license_id = lt.id AND deactivated_at IS NULL) AS activation_count
FROM licenses lt
JOIN products p ON lt.product_id = p.id
WHERE lt.organization_id = sqlc.arg('organization_id')::uuid
    AND lt.is_trial = true
    AND (sqlc.narg('product_id')::uuid IS NULL OR lt.product_id = sqlc.narg('product_id')::uuid)
    AND (sqlc.arg('q')::text = '' OR lt.customer_email ILIKE '%' || sqlc.arg('q')::text || '%' OR lt.customer_name ILIKE '%' || sqlc.arg('q')::text || '%' OR p.name ILIKE '%' || sqlc.arg('q')::text || '%')
    AND (
        sqlc.arg('status')::text = 'all'
        OR (sqlc.arg('status')::text = 'active' AND lt.is_active = true AND (lt.expires_at IS NULL OR lt.expires_at > now()))
        OR (sqlc.arg('status')::text = 'expired' AND lt.expires_at IS NOT NULL AND lt.expires_at <= now())
    )
ORDER BY lt.created_at DESC
LIMIT 500;
