-- +goose Up
-- +goose StatementBegin

ALTER TABLE update_releases ADD COLUMN IF NOT EXISTS changelog TEXT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE update_releases DROP COLUMN IF EXISTS changelog;

-- +goose StatementEnd
