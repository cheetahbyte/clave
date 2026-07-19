-- +goose Up
ALTER TABLE licenses ADD COLUMN customer_name TEXT;

-- +goose Down
ALTER TABLE licenses DROP COLUMN customer_name;
