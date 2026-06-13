-- name: GetDefaultChannelForProduct :one
SELECT * FROM update_channels
WHERE product_id = $1 AND is_default = true
LIMIT 1;

-- name: GetChannelsForProduct :many
SELECT * FROM update_channels
WHERE product_id = $1
ORDER BY name;

-- name: GetChannelByProductAndName :one
SELECT * FROM update_channels
WHERE product_id = $1 AND name = $2;

-- name: GetUpdateChannelByID :one
SELECT * FROM update_channels WHERE id = $1;

-- name: UpsertUpdateChannel :one
INSERT INTO update_channels (organization_id, product_id, name, is_default)
VALUES ($1, $2, $3, $4)
ON CONFLICT (product_id, name)
DO UPDATE SET is_default = EXCLUDED.is_default, updated_at = now()
RETURNING *;

-- name: CreateUpdateChannel :one
INSERT INTO update_channels (organization_id, product_id, name, is_default, required_features, description)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateUpdateChannel :one
UPDATE update_channels
SET name = $3,
    is_default = $4,
    required_features = $5,
    description = $6,
    updated_at = now()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: CountReleasesForChannel :one
SELECT count(*) FROM update_releases WHERE channel_id = $1;

-- name: CountConfigsForChannel :one
SELECT count(*) FROM product_update_configs WHERE channel_id = $1;

-- name: DeleteUpdateChannel :one
DELETE FROM update_channels
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: GetProductUpdateConfig :one
SELECT puc.*
FROM product_update_configs puc
JOIN update_channels c ON c.id = puc.channel_id
WHERE puc.product_id = $1
  AND puc.platform = $2
  AND c.name = $3
  AND puc.enabled = true;

-- name: GetProductUpdateConfigs :many
SELECT puc.*, c.name AS channel_name
FROM product_update_configs puc
JOIN update_channels c ON c.id = puc.channel_id
WHERE puc.product_id = $1
ORDER BY puc.platform, c.name;

-- name: DeleteProductUpdateConfig :one
DELETE FROM product_update_configs
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: UpsertProductUpdateConfig :one
INSERT INTO product_update_configs (
    organization_id, product_id, platform, channel_id, provider_key, config, enabled
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (product_id, platform, channel_id)
DO UPDATE SET
    provider_key = EXCLUDED.provider_key,
    config = EXCLUDED.config,
    enabled = EXCLUDED.enabled,
    updated_at = now()
RETURNING *;

-- name: LatestPublishedUpdateRelease :one
SELECT *
FROM update_releases
WHERE product_id = $1
  AND platform = $2
  AND channel_id = $3
  AND status = 'published'
  AND published_at <= now()
ORDER BY published_at DESC, created_at DESC
LIMIT 1;

-- name: GetUpdateRelease :one
SELECT * FROM update_releases WHERE id = $1;

-- name: InsertUpdateRelease :one
INSERT INTO update_releases (
    organization_id, product_id, channel_id, platform,
    version, build_number, status, release_notes, published_at, changelog_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: PublishUpdateRelease :one
UPDATE update_releases
SET status = 'published', published_at = now(), updated_at = now()
WHERE id = $1
RETURNING *;

-- name: YankUpdateRelease :one
UPDATE update_releases
SET status = 'yanked', updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteUpdateRelease :one
DELETE FROM update_releases
WHERE id = $1
RETURNING *;

-- name: ListReleasesForOrganization :many
SELECT ur.*, p.name AS product_name, c.name AS channel_name
FROM update_releases ur
JOIN products p ON p.id = ur.product_id
JOIN update_channels c ON c.id = ur.channel_id
WHERE ur.organization_id = $1
ORDER BY ur.created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListReleasesForOrganizationByProduct :many
SELECT ur.*, p.name AS product_name, c.name AS channel_name
FROM update_releases ur
JOIN products p ON p.id = ur.product_id
JOIN update_channels c ON c.id = ur.channel_id
WHERE ur.organization_id = $1 AND ur.product_id = $2
ORDER BY ur.created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListArtifactsForRelease :many
SELECT * FROM update_artifacts
WHERE release_id = $1;

-- name: GetUpdateArtifact :one
SELECT * FROM update_artifacts WHERE id = $1;

-- name: InsertUpdateArtifact :one
INSERT INTO update_artifacts (
    id, release_id, artifact_type, os, arch, url, size_bytes, checksum_sha256, signature, metadata,
    filename, mime_type, minimum_system_version, storage_backend, storage_key
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
RETURNING *;

-- name: GetReleasePolicy :one
SELECT * FROM update_release_policies
WHERE release_id = $1;

-- name: InsertUpdateCheck :one
INSERT INTO update_checks (
    organization_id, product_id, license_id, platform, channel,
    provider_key, current_version, current_build, arch, os_version, decision, selected_release_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: ListPublishedReleasesForFeed :many
SELECT *
FROM update_releases
WHERE product_id = $1
  AND platform = $2
  AND channel_id = $3
  AND status = 'published'
  AND published_at <= now()
ORDER BY published_at DESC
LIMIT 20;

-- name: ListArtifactsForReleases :many
SELECT * FROM update_artifacts
WHERE release_id = ANY($1::uuid[]);

-- name: GetChangelogsByReleaseIDs :many
SELECT r.id AS release_id, c.body AS changelog_body
FROM update_releases r
JOIN changelogs c ON c.id = r.changelog_id
WHERE r.id = ANY($1::uuid[]);
