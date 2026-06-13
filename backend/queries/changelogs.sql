-- name: ListChangelogsForProduct :many
SELECT * FROM changelogs
WHERE product_id = $1
ORDER BY updated_at DESC;

-- name: GetChangelog :one
SELECT * FROM changelogs WHERE id = $1;

-- name: CreateChangelog :one
INSERT INTO changelogs (organization_id, product_id, title, body)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateChangelog :one
UPDATE changelogs
SET title = $3, body = $4, updated_at = now()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: DeleteChangelog :one
DELETE FROM changelogs
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: CountReleasesForChangelog :one
SELECT count(*) FROM update_releases WHERE changelog_id = $1;

-- name: SetReleaseChangelog :one
UPDATE update_releases
SET changelog_id = $2, updated_at = now()
WHERE id = $1
RETURNING *;
