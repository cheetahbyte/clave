-- name: GetProductStorageConfig :one
SELECT * FROM product_storage_configs
WHERE product_id = $1;

-- name: UpsertProductStorageConfig :one
INSERT INTO product_storage_configs (organization_id, product_id, backend, config)
VALUES ($1, $2, $3, $4)
ON CONFLICT (product_id)
DO UPDATE SET
    backend = EXCLUDED.backend,
    config = EXCLUDED.config,
    updated_at = now()
RETURNING *;
