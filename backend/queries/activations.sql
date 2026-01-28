-- name: GetActivationsForLicense :many
select * from activations where license_id = $1;

-- name: ActivateLicense :one
insert into activations (license_id, hwid) values($1, $2) returning id;
