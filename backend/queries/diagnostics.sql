-- name: InsertClientCheckin :one
INSERT INTO client_checkins (
    organization_id, product_id, license_id, activation_id,
    version, build, platform, arch, os_version
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: DeleteExpiredClientCheckins :execrows
DELETE FROM client_checkins
WHERE created_at < now() - make_interval(days => sqlc.arg('retention_days')::int);

-- name: ListLatestClientCheckins :many
WITH ranked AS (
    SELECT
        c.*,
        row_number() OVER (
            PARTITION BY c.activation_id
            ORDER BY c.created_at DESC, c.id DESC
        ) AS position
    FROM client_checkins c
    WHERE c.organization_id = sqlc.arg('organization_id')::uuid
      AND c.created_at >= now() - make_interval(days => sqlc.arg('days')::int)
      AND (
          sqlc.narg('product_id')::uuid IS NULL
          OR c.product_id = sqlc.narg('product_id')::uuid
      )
)
SELECT
    ranked.activation_id,
    d.hostname,
    ranked.version,
    ranked.build,
    ranked.platform,
    ranked.arch,
    ranked.os_version,
    ranked.created_at
FROM ranked
JOIN activations a ON a.id = ranked.activation_id
JOIN devices d ON d.id = a.device_id
WHERE ranked.position = 1
  AND a.deactivated_at IS NULL
ORDER BY ranked.created_at DESC;

-- name: ListDailyLatestClientVersions :many
SELECT date, version, count(*)::bigint AS device_count
FROM (
    SELECT DISTINCT ON (c.created_at::date, c.activation_id)
        c.created_at::date AS date,
        c.activation_id,
        c.version
    FROM client_checkins c
    JOIN activations a ON a.id = c.activation_id
    WHERE c.organization_id = sqlc.arg('organization_id')::uuid
      AND c.created_at >= now() - make_interval(days => sqlc.arg('days')::int)
      AND (
          sqlc.narg('product_id')::uuid IS NULL
          OR c.product_id = sqlc.narg('product_id')::uuid
      )
      AND a.deactivated_at IS NULL
    ORDER BY c.created_at::date, c.activation_id, c.created_at DESC, c.id DESC
) latest
GROUP BY date, version
ORDER BY date, version;
