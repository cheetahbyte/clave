-- name: GetActivationsForLicense :many
select * from activations where license_id = $1;
