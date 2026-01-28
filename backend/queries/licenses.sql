-- name: GetLicenseById :one
select * from licenses where id = $1;

-- name: GetLicenseByDigest :one
select * from licenses where lookup_digest = $1;

-- name: CreateLicense :one
INSERT INTO licenses(product_id, max_activations, lookup_digest, key_phc) values($1, $2, $3, $4) returning *;
