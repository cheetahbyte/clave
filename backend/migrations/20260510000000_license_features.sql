-- +goose Up
ALTER TABLE licenses ADD COLUMN IF NOT EXISTS features TEXT[] NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE licenses DROP COLUMN IF EXISTS features;
