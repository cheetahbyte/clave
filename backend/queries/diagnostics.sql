-- name: RecordClientCheckin :exec
WITH updated AS (
    UPDATE activations a
    SET last_seen_at = now(),
        current_version = CASE WHEN sqlc.arg('version') <> '' THEN sqlc.arg('version') ELSE a.current_version END,
        current_build = CASE WHEN sqlc.arg('version') <> '' THEN sqlc.narg('build') ELSE a.current_build END,
        platform = CASE WHEN sqlc.arg('version') <> '' THEN sqlc.narg('platform') ELSE a.platform END,
        arch = CASE WHEN sqlc.arg('version') <> '' THEN sqlc.narg('arch') ELSE a.arch END,
        os_version = CASE WHEN sqlc.arg('version') <> '' THEN sqlc.narg('os_version') ELSE a.os_version END
    WHERE a.id = sqlc.arg('activation_id')
      AND a.license_id = sqlc.arg('license_id')
      AND a.deactivated_at IS NULL
    RETURNING a.id
)
INSERT INTO client_checkins (
    organization_id, product_id, license_id, activation_id,
    version, build, platform, arch, os_version
)
SELECT
    sqlc.arg('organization_id'), sqlc.arg('product_id'), sqlc.arg('license_id'), updated.id,
    sqlc.arg('version'), sqlc.narg('build'), sqlc.narg('platform'),
    sqlc.narg('arch'), sqlc.narg('os_version')
FROM updated
WHERE sqlc.arg('version') <> '';

-- name: ListClosedClientCheckinDates :many
SELECT DISTINCT (created_at AT TIME ZONE 'UTC')::date AS date
FROM client_checkins
WHERE (created_at AT TIME ZONE 'UTC')::date < (now() AT TIME ZONE 'UTC')::date
ORDER BY date;

-- name: DeleteDailyVersionAdoptionForDate :exec
DELETE FROM daily_version_adoption
WHERE date = sqlc.arg('date')::date;

-- name: InsertDailyVersionAdoptionForDate :exec
INSERT INTO daily_version_adoption (date, organization_id, product_id, version, device_count)
SELECT observed_date, organization_id, product_id, version, count(*)
FROM (
    SELECT DISTINCT ON (activation_id)
        (created_at AT TIME ZONE 'UTC')::date AS observed_date,
        organization_id,
        product_id,
        activation_id,
        version
    FROM client_checkins
    WHERE (created_at AT TIME ZONE 'UTC')::date = sqlc.arg('date')::date
    ORDER BY activation_id, created_at DESC, id DESC
) latest_daily
GROUP BY observed_date, organization_id, product_id, version;

-- name: DeleteExpiredClientCheckins :execrows
DELETE FROM client_checkins
WHERE created_at < now() - make_interval(days => sqlc.arg('retention_days')::int);

-- name: ListCurrentClientStates :many
SELECT
    a.id AS activation_id,
    d.hostname,
    a.current_version AS version,
    a.current_build AS build,
    a.platform,
    a.arch,
    a.os_version,
    a.last_seen_at
FROM activations a
JOIN devices d ON d.id = a.device_id
JOIN licenses l ON l.id = a.license_id
WHERE l.organization_id = sqlc.arg('organization_id')::uuid
  AND a.deactivated_at IS NULL
  AND a.last_seen_at >= now() - make_interval(days => sqlc.arg('days')::int)
  AND a.current_version IS NOT NULL
  AND (
      sqlc.narg('product_id')::uuid IS NULL
      OR l.product_id = sqlc.narg('product_id')::uuid
  )
ORDER BY a.last_seen_at DESC, a.id;

-- name: ListDailyVersionAdoption :many
SELECT date, version, sum(device_count)::bigint AS device_count
FROM daily_version_adoption
WHERE organization_id = sqlc.arg('organization_id')::uuid
  AND date >= ((now() AT TIME ZONE 'UTC')::date - (sqlc.arg('days')::int - 1))
  AND (
      sqlc.narg('product_id')::uuid IS NULL
      OR product_id = sqlc.narg('product_id')::uuid
  )
GROUP BY date, version
ORDER BY date, version;
