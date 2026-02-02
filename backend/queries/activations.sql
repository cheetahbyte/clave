-- name: GetActivationsForLicense :many
select * from activations where license_id = $1;

-- name: CountActivations :one
select count(*) from activations where license_id = $1;

-- name: ActivateLicense :one
insert into activations (device_id, license_id) values($1, $2) returning *;
