-- name: GetLicenseById :one
select * from licenses where id = $1;
