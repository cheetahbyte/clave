-- name: GetLicenseById :one
select * from licenses where id = $1;

-- name: GetLicenseByDigest :one
select * from licenses where lookup_digest = $1;

-- name: CreateLicense :one
INSERT INTO licenses(organization_id, product_id, max_activations, lookup_digest, key_phc, customer_email, expires_at, is_trial, trial_hwid_hash)
values(sqlc.arg('organization_id')::uuid, sqlc.arg('product_id'), sqlc.arg('max_activations'), sqlc.arg('lookup_digest'), sqlc.arg('key_phc'), sqlc.arg('customer_email'), sqlc.arg('expires_at'), sqlc.arg('is_trial'), sqlc.arg('trial_hwid_hash'))
returning *;

-- name: CountTrialsByEmailProduct :one
SELECT count(*) FROM licenses
WHERE organization_id = sqlc.arg('organization_id')::uuid
  AND product_id = sqlc.arg('product_id')
  AND lower(customer_email) = lower(sqlc.arg('customer_email'))
  AND is_trial = true;

-- name: CountTrialsByHwidProduct :one
SELECT count(*) FROM licenses
WHERE organization_id = sqlc.arg('organization_id')::uuid
  AND product_id = sqlc.arg('product_id')
  AND trial_hwid_hash = sqlc.arg('trial_hwid_hash')
  AND is_trial = true;

-- name: GetAdminLicenseById :one
select * from licenses where id = $1 and organization_id = sqlc.arg('organization_id')::uuid;

-- name: DeleteAdminLicense :one
DELETE FROM licenses
WHERE id = $1 AND organization_id = sqlc.arg('organization_id')::uuid
RETURNING id;

-- name: UpdateAdminLicense :one
UPDATE licenses SET
    customer_email = sqlc.arg('customer_email'),
    max_activations = sqlc.arg('max_activations'),
    is_active = sqlc.arg('is_active'),
    expires_at = sqlc.arg('expires_at'),
    features = sqlc.arg('features')
WHERE id = $1 AND organization_id = sqlc.arg('organization_id')::uuid
RETURNING *;

-- name: GetLicenseByIdForUpdate :one
select * from licenses where id = $1 for update;

-- name: ListByCustomerEmail :many
select lt.is_active, lt.id, lt.max_activations, lt.expires_at, p.name, p.logo_url from licenses lt join products p on lt.product_id = p.id where customer_email = $1;

-- name: ListByCustomerEmailAndOrganization :many
select lt.is_active, lt.id, lt.max_activations, lt.expires_at, p.name, p.logo_url
from licenses lt
join products p on lt.product_id = p.id
where lt.customer_email = $1 and lt.organization_id = $2;

-- name: ListSelfServiceDevices :many
select
    d.id,
    d.hostname,
    d.created_at,
    a.checked_in_at
from activations a
join devices d on a.device_id = d.id
join licenses l on a.license_id = l.id
where a.license_id = $1
  and l.customer_email = $2
  and l.organization_id = $3
order by a.checked_in_at desc nulls last;

-- name: DeleteSelfServiceDevice :one
delete from devices d
where d.id = sqlc.arg('device_id')
  and d.license_id in (
    select l.id from licenses l
    where l.id = sqlc.arg('license_id')
      and l.customer_email = sqlc.arg('customer_email')
      and l.organization_id = sqlc.arg('organization_id')
  )
returning d.id;

-- name: RevokeSelfServiceLicense :one
update licenses set is_active = false
where id = sqlc.arg('license_id')
  and customer_email = sqlc.arg('customer_email')
  and organization_id = sqlc.arg('organization_id')
returning id;

-- name: DeactivateActiveTrialsByEmailProduct :exec
UPDATE licenses
SET
  is_active = false,
  expires_at = CASE
    WHEN expires_at IS NULL OR expires_at > now() THEN now()
    ELSE expires_at
  END
WHERE organization_id = sqlc.arg('organization_id')::uuid
  AND product_id = sqlc.arg('product_id')
  AND lower(customer_email) = lower(sqlc.arg('customer_email'))
  AND is_trial = true
  AND is_active = true
  AND (expires_at IS NULL OR expires_at > now());

-- name: TransferActiveTrialActivationsByEmailProduct :exec
WITH trial_activations AS (
    SELECT DISTINCT ON (d.hwid_hash)
        d.hwid_hash,
        d.hostname
    FROM licenses l
    JOIN devices d ON d.license_id = l.id
    JOIN activations a ON a.license_id = l.id AND a.device_id = d.id
    WHERE l.organization_id = sqlc.arg('organization_id')::uuid
      AND l.product_id = sqlc.arg('product_id')
      AND lower(l.customer_email) = lower(sqlc.arg('customer_email'))
      AND l.is_trial = true
      AND l.is_active = true
      AND (l.expires_at IS NULL OR l.expires_at > now())
    ORDER BY d.hwid_hash, a.checked_in_at DESC NULLS LAST
),
limited AS (
    SELECT * FROM trial_activations
    LIMIT sqlc.arg('max_activations')
),
new_devices AS (
    INSERT INTO devices (license_id, hwid_hash, hostname)
    SELECT sqlc.arg('paid_license_id')::uuid, tr.hwid_hash, tr.hostname
    FROM limited tr
    RETURNING id, hwid_hash
)
INSERT INTO activations (device_id, license_id)
SELECT nd.id, sqlc.arg('paid_license_id')::uuid
FROM new_devices nd;
