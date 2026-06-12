-- name: GetProductsByOrganization :many
select * from products where organization_id = sqlc.arg('organization_id')::uuid;

-- name: GetProductById :one
select * from products where id = $1;

-- name: GetOneByIdForOrganization :one
select * from products where id = $1 and organization_id = sqlc.arg('organization_id')::uuid;

-- name: CreateProduct :one
INSERT INTO products (organization_id, name, version, logo_url)
VALUES (sqlc.arg('organization_id')::uuid, sqlc.arg('name'), sqlc.arg('version'), sqlc.arg('logo_url'))
RETURNING *;

-- name: CountLicensesByProduct :one
SELECT count(*) FROM licenses
WHERE product_id = $1 AND organization_id = sqlc.arg('organization_id')::uuid;

-- name: DeleteProduct :one
DELETE FROM products
WHERE id = $1 AND organization_id = sqlc.arg('organization_id')::uuid
RETURNING id;

-- name: UpdateProduct :one
UPDATE products
SET name = sqlc.arg('name'), version = sqlc.arg('version'), logo_url = sqlc.arg('logo_url')
WHERE id = $1 AND organization_id = sqlc.arg('organization_id')::uuid
RETURNING *;
