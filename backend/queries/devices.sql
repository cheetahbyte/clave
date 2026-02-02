-- name: CreateDevice :one
insert into devices(license_id, hwid_hash, hostname) values($1, $2, $3) returning *;
