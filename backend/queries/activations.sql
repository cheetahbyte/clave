-- name: GetActivationsForLicense :many
select * from activations where license_id = $1 and deactivated_at is null;

-- name: CountActivations :one
select count(*) from activations where license_id = $1 and deactivated_at is null;

-- name: ActivateLicense :one
insert into activations (device_id, license_id) values($1, $2) returning *;

-- name: GetActivationByLicenseAndDevice :one
select * from activations where license_id = $1 and device_id = $2 and deactivated_at is null;

-- name: GetActiveActivationByID :one
select a.*
from activations a
join devices d on d.id = a.device_id
where a.id = sqlc.arg('activation_id')
  and a.license_id = sqlc.arg('license_id')
  and d.hwid_hash = sqlc.arg('hwid_hash')
  and a.deactivated_at is null;

-- name: GetActiveActivationByLicenseAndHwidHash :one
select a.*
from activations a
join devices d on d.id = a.device_id
where a.license_id = sqlc.arg('license_id')
  and d.hwid_hash = sqlc.arg('hwid_hash')
  and a.deactivated_at is null;

-- name: DeactivateActivationByLicenseAndHwid :execrows
UPDATE activations a
SET deactivated_at = now(),
    deactivation_reason = 'client_unregistration'
FROM devices d
WHERE a.device_id = d.id
  AND a.license_id = sqlc.arg('license_id')
  AND d.license_id = sqlc.arg('license_id')
  AND d.hwid_hash = sqlc.arg('hwid_hash')
  AND a.deactivated_at IS NULL;
