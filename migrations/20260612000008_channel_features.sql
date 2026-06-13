-- +goose Up
-- +goose StatementBegin

ALTER TABLE update_channels ADD COLUMN IF NOT EXISTS required_features TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE update_channels ADD COLUMN IF NOT EXISTS description TEXT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE update_channels DROP COLUMN IF EXISTS description;
ALTER TABLE update_channels DROP COLUMN IF EXISTS required_features;

-- +goose StatementEnd
