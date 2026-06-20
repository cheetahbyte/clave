-- ============ product_features catalog ============

-- name: GetProductFeaturesByProduct :many
SELECT * FROM product_features
WHERE product_id = $1
ORDER BY key;

-- name: GetProductFeaturesByOrganization :many
SELECT * FROM product_features
WHERE organization_id = $1
ORDER BY product_id, key;

-- name: GetProductFeatureByKey :one
SELECT * FROM product_features
WHERE organization_id = $1 AND product_id = $2 AND key = $3;

-- name: GetProductFeatureByID :one
SELECT * FROM product_features WHERE id = $1;

-- name: CreateProductFeature :one
INSERT INTO product_features (organization_id, product_id, key, name, description)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateProductFeature :one
UPDATE product_features
SET name = $2, description = $3
WHERE id = $1
RETURNING *;

-- name: ArchiveProductFeature :one
UPDATE product_features
SET archived_at = now()
WHERE id = $1 AND archived_at IS NULL
RETURNING *;

-- name: DeleteProductFeature :one
DELETE FROM product_features
WHERE id = $1 AND organization_id = $2
RETURNING id;

-- ============ license_features ============

-- name: GetLicenseFeatureKeys :many
SELECT pf.key
FROM license_features lf
JOIN product_features pf ON pf.id = lf.feature_id
WHERE lf.license_id = $1
ORDER BY pf.key;

-- name: SetLicenseFeatures :exec
DELETE FROM license_features WHERE license_id = $1;

-- name: AddLicenseFeature :exec
INSERT INTO license_features (license_id, feature_id, source, source_window_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (license_id, feature_id) DO NOTHING;

-- name: AddLicenseFeatureByKey :exec
INSERT INTO license_features (license_id, feature_id, source)
SELECT $1, pf.id, $3
FROM product_features pf
WHERE pf.organization_id = $2 AND pf.product_id = $4 AND pf.key = $5
ON CONFLICT (license_id, feature_id) DO NOTHING;

-- ============ update_channel_required_features ============

-- name: GetChannelRequiredFeatureKeys :many
SELECT pf.key
FROM update_channel_required_features crf
JOIN product_features pf ON pf.id = crf.feature_id
WHERE crf.channel_id = $1
ORDER BY pf.key;

-- name: SetChannelRequiredFeatures :exec
DELETE FROM update_channel_required_features WHERE channel_id = $1;

-- name: AddChannelRequiredFeature :exec
INSERT INTO update_channel_required_features (channel_id, feature_id)
VALUES ($1, $2)
ON CONFLICT (channel_id, feature_id) DO NOTHING;

-- ============ product_feature_windows ============

-- name: GetFeatureWindowsByProduct :many
SELECT pfw.*, pf.key AS feature_key
FROM product_feature_windows pfw
JOIN product_features pf ON pf.id = pfw.feature_id
WHERE pfw.product_id = $1
ORDER BY pfw.starts_at DESC;

-- name: GetFeatureWindowsByOrganization :many
SELECT pfw.*, pf.key AS feature_key, p.name AS product_name
FROM product_feature_windows pfw
JOIN product_features pf ON pf.id = pfw.feature_id
JOIN products p ON p.id = pfw.product_id
WHERE pfw.organization_id = $1
ORDER BY pfw.starts_at DESC;

-- name: GetActiveFeatureWindowsForProduct :many
SELECT pfw.*, pf.key AS feature_key
FROM product_feature_windows pfw
JOIN product_features pf ON pf.id = pfw.feature_id
WHERE pfw.product_id = $1
  AND pfw.is_active = true
  AND pfw.starts_at <= $2
  AND pfw.ends_at > $2
  AND (pfw.applies_to = 'all' OR pfw.applies_to = $3)
ORDER BY pf.key;

-- name: CreateFeatureWindow :one
INSERT INTO product_feature_windows (organization_id, product_id, feature_id, starts_at, ends_at, applies_to, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateFeatureWindow :one
UPDATE product_feature_windows
SET feature_id = $2,
    starts_at = $3,
    ends_at = $4,
    applies_to = $5,
    is_active = $6
WHERE id = $1 AND organization_id = $7
RETURNING *;

-- name: DeleteFeatureWindow :one
DELETE FROM product_feature_windows
WHERE id = $1 AND organization_id = $2
RETURNING id;
