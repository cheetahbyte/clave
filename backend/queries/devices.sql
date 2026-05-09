-- name: CreateDevice :one
insert into devices(license_id, hwid_hash, hostname) values($1, $2, $3) returning *;

-- name: GetDeviceByLicenseAndHwidHash :one
select * from devices where license_id = $1 and hwid_hash = $2;
