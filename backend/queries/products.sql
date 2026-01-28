-- name: GetProducts :many
select * from products;

-- name: GetOneById :one
select * from products where id = $1;
