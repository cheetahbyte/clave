-- name: GetLicenseById :one
select * from licenses where id = $1;

-- name: GetLicenseByDigest :one
select * from licenses where lookup_digest = $1;

-- name: CreateLicense :one
INSERT INTO licenses(product_id, max_activations, lookup_digest, key_phc) values($1, $2, $3, $4) returning *;

-- name: ListByCustomerEmail :many
select lt.is_active, lt.id, lt.max_activations, lt.expires_at, p.name from licenses lt join products p on lt.product_id = p.id where customer_email = $1;
